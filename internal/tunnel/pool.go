package tunnel

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"xengate/internal/models"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type ConnectionPool struct {
	server     *models.Connection
	tunnels    []*Tunnel
	sshConfig  *ssh.ClientConfig
	mu         sync.RWMutex
	roundRobin uint32
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewConnectionPool(server *models.Connection) (*ConnectionPool, error) {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &ConnectionPool{
		server:  server,
		tunnels: make([]*Tunnel, 0, server.Config.Connections),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Configure SSH
	pool.sshConfig = &ssh.ClientConfig{
		User: server.Config.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(server.Config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	return pool, nil
}

func (p *ConnectionPool) Start(ctx context.Context) error {
	logger := log.WithFields(log.Fields{
		"server": p.server.Name,
		"addr":   p.server.Address,
	})

	port, err := strconv.Atoi(p.server.Port)
	if err != nil {
		logger.WithError(err).Error("Invalid port number")
		return fmt.Errorf("invalid port number: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", p.server.Address, port)
	logger.Info("Starting connection pool")

	errCh := make(chan error, p.server.Config.Connections)
	var wg sync.WaitGroup

	for i := 0; i < p.server.Config.Connections; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			tunnelID := fmt.Sprintf("%s-%d", p.server.Name, index+1)
			tunnelLogger := logger.WithField("tunnel", tunnelID)

			tunnel := NewTunnel(tunnelID, p.server.Name, addr, p.sshConfig)

			backoff := time.Second
			maxBackoff := time.Second * 10

			for retries := 0; retries < p.server.Config.MaxRetries; retries++ {
				tunnelLogger.WithField("attempt", retries+1).Debug("Attempting to connect tunnel")

				if err := tunnel.Connect(ctx); err != nil {
					tunnelLogger.WithError(err).Warn("Failed to connect tunnel")

					select {
					case <-ctx.Done():
						errCh <- ctx.Err()
						return
					case <-time.After(backoff):
						backoff = time.Duration(float64(backoff) * 1.5)
						if backoff > maxBackoff {
							backoff = maxBackoff
						}
						continue
					}
				}

				p.mu.Lock()
				p.tunnels = append(p.tunnels, tunnel)
				p.mu.Unlock()

				tunnelLogger.Info("Tunnel connected successfully")
				return
			}

			err := fmt.Errorf("failed to connect tunnel after %d attempts", p.server.Config.MaxRetries)
			tunnelLogger.Error(err)
			errCh <- err
		}(i)
	}

	wg.Wait()
	close(errCh)

	var errors []error
	for err := range errCh {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		logger.WithField("errors", errors).Error("Failed to start some tunnels")
		return fmt.Errorf("failed to start some tunnels: %v", errors)
	}

	go p.monitorConnections(ctx)
	logger.Info("Connection pool started successfully")
	return nil
}

func (p *ConnectionPool) monitorConnections(ctx context.Context) {
	logger := log.WithField("server", p.server.Name)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.checkAndReconnectConcurrent(ctx)

			// Log pool statistics
			stats := p.GetStats()
			logger.WithFields(log.Fields{
				"total":     stats.TotalTunnels,
				"connected": stats.Connected,
				"active":    stats.ActiveConnections,
			}).Debug("Pool status")
		}
	}
}

func (p *ConnectionPool) Stop() {
	p.cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, tunnel := range p.tunnels {
		tunnel.Disconnect()
	}
}

func (p *ConnectionPool) GetTunnel() *Tunnel {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.tunnels) == 0 {
		return nil
	}

	// Start from next index using round-robin
	start := atomic.AddUint32(&p.roundRobin, 1) - 1
	attempts := 0

	for attempts < len(p.tunnels) {
		idx := int(start+uint32(attempts)) % len(p.tunnels)
		tunnel := p.tunnels[idx]

		if tunnel.IsConnected() {
			return tunnel
		}

		attempts++
	}

	return nil
}

func (p *ConnectionPool) Forward(localConn net.Conn, targetAddr string) error {
	tunnel := p.GetTunnel()
	if tunnel == nil {
		return fmt.Errorf("no available tunnels for server %s", p.server.Name)
	}

	return tunnel.Forward(localConn, targetAddr)
}

// func (p *ConnectionPool) monitorConnections(ctx context.Context) {
// 	ticker := time.NewTicker(10 * time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case <-p.ctx.Done():
// 			return
// 		case <-ticker.C:
// 			p.CheckAndReconnect(ctx)
// 		}
// 	}
// }

func (p *ConnectionPool) checkAndReconnectConcurrent(ctx context.Context) {
	logger := log.WithField("server", p.server.Name)

	p.mu.RLock()
	tunnels := make([]*Tunnel, len(p.tunnels))
	copy(tunnels, p.tunnels)
	p.mu.RUnlock()

	var wg sync.WaitGroup
	reconnectCh := make(chan struct{}, p.server.Config.MaxRetries) // Limit concurrent reconnections

	for _, tunnel := range tunnels {
		if !tunnel.IsConnected() {
			wg.Add(1)
			go func(t *Tunnel) {
				defer wg.Done()

				tunnelLogger := logger.WithField("tunnel", t.id)

				// Acquire reconnection slot
				reconnectCh <- struct{}{}
				defer func() {
					<-reconnectCh // Release reconnection slot
				}()

				tunnelLogger.Warn("Tunnel disconnected, attempting to reconnect")

				backoff := time.Second
				maxBackoff := time.Second * 10

				for retries := 0; retries < p.server.Config.MaxRetries; retries++ {
					select {
					case <-ctx.Done():
						tunnelLogger.Info("Context cancelled during reconnection")
						return
					default:
						if err := t.Connect(ctx); err == nil {
							tunnelLogger.Info("Successfully reconnected")
							return
						}

						tunnelLogger.WithFields(log.Fields{
							"attempt": retries + 1,
							"backoff": backoff,
						}).Debug("Reconnection attempt failed")

						// Apply exponential backoff
						time.Sleep(backoff)
						backoff = time.Duration(float64(backoff) * 1.5)
						if backoff > maxBackoff {
							backoff = maxBackoff
						}
					}
				}

				tunnelLogger.Error("Failed to reconnect after all attempts")
			}(tunnel)
		}
	}

	// Wait for all reconnection attempts to complete
	wg.Wait()
	close(reconnectCh)

	// Update pool statistics after reconnection attempts
	stats := p.GetStats()
	logger.WithFields(log.Fields{
		"total":     stats.TotalTunnels,
		"connected": stats.Connected,
		"active":    stats.ActiveConnections,
	}).Info("Pool reconnection status")
}

// func (p *ConnectionPool) CheckAndReconnect(ctx context.Context) {
// 	p.mu.RLock()
// 	tunnels := make([]*Tunnel, len(p.tunnels))
// 	copy(tunnels, p.tunnels)
// 	p.mu.RUnlock()

// 	for _, tunnel := range tunnels {
// 		if !tunnel.IsConnected() {
// 			log.Warnf("Tunnel %s is disconnected, attempting to reconnect", tunnel.id)

// 			retries := 0
// 			for retries < p.server.Config.MaxRetries {
// 				if err := tunnel.Connect(ctx); err != nil {
// 					retries++
// 					log.Errorf("Failed to reconnect tunnel %s (attempt %d/%d): %v",
// 						tunnel.id, retries, p.server.Config.MaxRetries, err)
// 					if retries < p.server.Config.MaxRetries {
// 						select {
// 						case <-ctx.Done():
// 							return
// 						case <-time.After(time.Duration(retries*2) * time.Second):
// 							continue
// 						}
// 					}
// 				} else {
// 					log.Infof("Successfully reconnected tunnel %s", tunnel.id)
// 					break
// 				}
// 			}
// 		}
// 	}
// }

func (p *ConnectionPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		ID:           p.server.ID,
		ServerName:   p.server.Name,
		TotalTunnels: len(p.tunnels),
		Tunnels:      make([]TunnelStats, 0, len(p.tunnels)),
	}

	for _, tunnel := range p.tunnels {
		tunnelStats := tunnel.GetStats()
		stats.Tunnels = append(stats.Tunnels, tunnelStats)
		if tunnelStats.Connected {
			stats.Connected++
		}
		stats.ActiveConnections += tunnelStats.Active
		stats.TotalBytes += tunnelStats.TotalBytes
		stats.TotalRequests += tunnelStats.RequestCount
	}

	return stats
}

type PoolStats struct {
	ID                string
	ServerName        string
	TotalTunnels      int
	Connected         int
	ActiveConnections int64
	TotalBytes        int64
	TotalRequests     int64
	Tunnels           []TunnelStats
}
