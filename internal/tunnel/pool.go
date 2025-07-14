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
	port, err := strconv.Atoi(p.server.Port)
	if err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", p.server.Address, port)

	// Create and connect tunnels
	for i := 0; i < p.server.Config.Connections; i++ {
		tunnelID := fmt.Sprintf("%s-%d", p.server.Name, i+1)
		tunnel := NewTunnel(tunnelID, p.server.Name, addr, p.sshConfig)

		retries := 0
		for retries < p.server.Config.MaxRetries {
			if err := tunnel.Connect(ctx); err != nil {
				retries++
				log.Warnf("Failed to connect tunnel %s (attempt %d/%d): %v",
					tunnelID, retries, p.server.Config.MaxRetries, err)
				if retries < p.server.Config.MaxRetries {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(time.Duration(retries) * time.Second):
						continue
					}
				}
				return fmt.Errorf("failed to connect tunnel %s after %d attempts", tunnelID, p.server.Config.MaxRetries)
			}
			break
		}

		p.tunnels = append(p.tunnels, tunnel)
	}

	// Start connection monitor
	go p.monitorConnections(ctx)

	log.Infof("Started connection pool for %s with %d connections", p.server.Name, len(p.tunnels))
	return nil
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

func (p *ConnectionPool) monitorConnections(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.CheckAndReconnect(ctx)
		}
	}
}

func (p *ConnectionPool) CheckAndReconnect(ctx context.Context) {
	p.mu.RLock()
	tunnels := make([]*Tunnel, len(p.tunnels))
	copy(tunnels, p.tunnels)
	p.mu.RUnlock()

	for _, tunnel := range tunnels {
		if !tunnel.IsConnected() {
			log.Warnf("Tunnel %s is disconnected, attempting to reconnect", tunnel.id)

			retries := 0
			for retries < p.server.Config.MaxRetries {
				if err := tunnel.Connect(ctx); err != nil {
					retries++
					log.Errorf("Failed to reconnect tunnel %s (attempt %d/%d): %v",
						tunnel.id, retries, p.server.Config.MaxRetries, err)
					if retries < p.server.Config.MaxRetries {
						select {
						case <-ctx.Done():
							return
						case <-time.After(time.Duration(retries*2) * time.Second):
							continue
						}
					}
				} else {
					log.Infof("Successfully reconnected tunnel %s", tunnel.id)
					break
				}
			}
		}
	}
}

func (p *ConnectionPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		ServerName:   fmt.Sprintf("%s:%s:%s", p.server.Name, p.server.Address, p.server.Port),
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
	ServerName        string
	TotalTunnels      int
	Connected         int
	ActiveConnections int64
	TotalBytes        int64
	TotalRequests     int64
	Tunnels           []TunnelStats
}
