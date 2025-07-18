package tunnel

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"xengate/internal/models"

	log "github.com/sirupsen/logrus"
)

type Manager struct {
	pools      map[string]*ConnectionPool
	mu         sync.RWMutex
	roundRobin uint32
	// Add waitgroup to track active operations
	wg sync.WaitGroup
}

func NewManager() *Manager {
	return &Manager{
		pools: make(map[string]*ConnectionPool),
	}
}

func (m *Manager) Start(ctx context.Context, server *models.Connection) error {
	// First check if pool already exists
	m.mu.RLock()
	if _, exists := m.pools[server.Name]; exists {
		m.mu.RUnlock()
		return fmt.Errorf("pool for server %s already exists", server.Name)
	}
	m.mu.RUnlock()

	// Create new pool
	log.Infof("Creating connection pool for server %s", server.Name)
	pool, err := NewConnectionPool(server)
	if err != nil {
		log.Errorf("Failed to create pool for %s: %v", server.Name, err)
		return fmt.Errorf("failed to create pool for %s: %w", server.Name, err)
	}

	// Start the pool
	if err := pool.Start(ctx); err != nil {
		log.Errorf("Failed to start pool for %s: %v", server.Name, err)
		// Clean up the pool if start fails
		pool.Stop()
		return fmt.Errorf("failed to start pool for %s: %w", server.Name, err)
	}

	// If successful, add to pools map
	m.mu.Lock()
	// Double-check that pool wasn't added while we were starting
	if _, exists := m.pools[server.Name]; exists {
		m.mu.Unlock()
		// Clean up the new pool since we won't be using it
		pool.Stop()
		return fmt.Errorf("pool for server %s was created concurrently", server.Name)
	}
	m.pools[server.Name] = pool
	m.mu.Unlock()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		<-ctx.Done()
		m.Stop(server.Name)
	}()

	log.Infof("Successfully started pool for server %s", server.Name)
	return nil
}

func (m *Manager) Stop(serverName string) {
	m.mu.Lock()
	pool, exists := m.pools[serverName]
	if exists {
		delete(m.pools, serverName)
	}
	m.mu.Unlock()

	if exists {
		// Stop the pool in a goroutine to avoid blocking GUI
		go func() {
			log.Infof("Stopping connection pool for %s", serverName)
			pool.Stop()
		}()
	}
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	pools := make([]*ConnectionPool, 0, len(m.pools))
	for _, pool := range m.pools {
		pools = append(pools, pool)
	}
	m.pools = make(map[string]*ConnectionPool)
	m.mu.Unlock()

	// Stop all pools in separate goroutines
	for _, pool := range pools {
		go func(p *ConnectionPool) {
			p.Stop()
		}(pool)
	}

	// Wait for all operations to complete
	go func() {
		m.wg.Wait()
		// Notify completion via Fyne
	}()
}

func (m *Manager) Forward(localConn net.Conn, targetAddr string) error {
	logger := log.WithFields(log.Fields{
		"target": targetAddr,
	})

	m.mu.RLock()
	if len(m.pools) == 0 {
		m.mu.RUnlock()
		logger.Error("No available connection pools")
		return fmt.Errorf("no available connection pools")
	}

	availablePools := make([]*ConnectionPool, 0, len(m.pools))
	for _, pool := range m.pools {
		if tunnel := pool.GetTunnel(); tunnel != nil && tunnel.IsConnected() {
			availablePools = append(availablePools, pool)
		}
	}
	m.mu.RUnlock()

	if len(availablePools) == 0 {
		logger.Error("No available tunnels")
		return fmt.Errorf("no available tunnels")
	}

	var selectedPool *ConnectionPool
	minActive := int64(^uint64(0) >> 1)

	for _, pool := range availablePools {
		stats := pool.GetStats()
		if stats.ActiveConnections < minActive {
			minActive = stats.ActiveConnections
			selectedPool = pool
		}
	}

	logger = logger.WithFields(log.Fields{
		"pool":        selectedPool.server.Name,
		"activeConns": minActive,
	})
	logger.Debug("Selected pool for forwarding")

	errCh := make(chan error, 1)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		errCh <- selectedPool.Forward(localConn, targetAddr)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			logger.WithError(err).Error("Forward operation failed")
		} else {
			logger.Debug("Forward operation completed successfully")
		}
		return err
	case <-timeoutCtx.Done():
		logger.Error("Forward operation timed out")
		return fmt.Errorf("forward operation timed out")
	}
}

func (m *Manager) GetStats() map[string]PoolStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]PoolStats)
	for name, pool := range m.pools {
		stats[name] = pool.GetStats()
	}
	return stats
}

func (m *Manager) GetPool(serverName string) *ConnectionPool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pools[serverName]
}

// HasPool checks if a pool exists
func (m *Manager) HasPool(serverName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.pools[serverName]
	return exists
}
