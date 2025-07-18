package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type Tunnel struct {
	id           string
	serverName   string
	client       *ssh.Client
	config       *ssh.ClientConfig
	addr         string
	mu           sync.RWMutex
	active       int64
	totalBytes   int64
	requestCount int64
	lastError    error
	lastUsed     time.Time
	ctx          context.Context
	cancel       context.CancelFunc
	reconnecting atomic.Bool
}

func NewTunnel(id, serverName, addr string, config *ssh.ClientConfig) *Tunnel {
	ctx, cancel := context.WithCancel(context.Background())
	return &Tunnel{
		id:         id,
		serverName: serverName,
		config:     config,
		addr:       addr,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (t *Tunnel) Connect(ctx context.Context) error {
	if t.reconnecting.Load() {
		return fmt.Errorf("tunnel is already reconnecting")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client != nil {
		return nil
	}

	t.reconnecting.Store(true)
	defer t.reconnecting.Store(false)

	log.WithFields(log.Fields{
		"tunnel": t.id,
		"server": t.serverName,
		"addr":   t.addr,
	}).Debug("Connecting tunnel")

	configCopy := *t.config
	configCopy.ClientVersion = "SSH-2.0-OpenSSH_8.4p1"
	configCopy.Timeout = 30 * time.Second

	dialCtx, cancel := context.WithTimeout(ctx, configCopy.Timeout)
	defer cancel()

	var client *ssh.Client
	var err error

	done := make(chan struct{})
	go func() {
		defer close(done)
		client, err = ssh.Dial("tcp", t.addr, &configCopy)
	}()

	select {
	case <-dialCtx.Done():
		if client != nil {
			client.Close()
		}
		err = fmt.Errorf("connection timeout after %v", configCopy.Timeout)
		log.WithError(err).WithField("tunnel", t.id).Warn("Connection timeout")
		return err
	case <-done:
		if err != nil {
			log.WithError(err).WithField("tunnel", t.id).Error("Connection failed")
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	t.client = client
	t.lastError = nil

	log.WithFields(log.Fields{
		"tunnel": t.id,
		"server": t.serverName,
	}).Info("Tunnel connected successfully")

	go t.keepAlive()
	go t.monitorConnection()

	return nil
}

func (t *Tunnel) monitorConnection() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			if err := t.checkConnection(); err != nil {
				log.WithFields(log.Fields{
					"tunnel": t.id,
					"error":  err,
				}).Warn("Connection check failed")
				t.reconnect()
			}
		}
	}
}

func (t *Tunnel) checkConnection() error {
	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("no client")
	}

	_, _, err := client.SendRequest("keepalive@golang", true, nil)
	return err
}

func (t *Tunnel) reconnect() {
	if !t.reconnecting.CompareAndSwap(false, true) {
		return
	}
	defer t.reconnecting.Store(false)

	t.mu.Lock()
	if t.client != nil {
		t.client.Close()
		t.client = nil
	}
	t.mu.Unlock()

	backoff := time.Second
	maxBackoff := 10 * time.Second
	attempts := 0
	maxAttempts := 3

	for attempts < maxAttempts {
		log.WithFields(log.Fields{
			"tunnel":  t.id,
			"attempt": attempts + 1,
			"backoff": backoff,
		}).Info("Attempting to reconnect")

		if err := t.Connect(context.Background()); err == nil {
			log.WithField("tunnel", t.id).Info("Reconnection successful")
			return
		}

		attempts++
		if attempts < maxAttempts {
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	log.WithFields(log.Fields{
		"tunnel":   t.id,
		"attempts": attempts,
	}).Error("Reconnection failed after all attempts")
}

func (t *Tunnel) keepAlive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			t.mu.RLock()
			client := t.client
			t.mu.RUnlock()

			if client != nil {
				_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
				if err != nil {
					log.WithFields(log.Fields{
						"tunnel": t.id,
						"error":  err,
					}).Warn("Keepalive failed")
					t.mu.Lock()
					t.lastError = err
					if t.client != nil {
						t.client.Close()
						t.client = nil
					}
					t.mu.Unlock()
				}
			}
		}
	}
}

func (t *Tunnel) Forward(localConn net.Conn, targetAddr string) error {
	atomic.AddInt64(&t.requestCount, 1)

	logger := log.WithFields(log.Fields{
		"tunnel": t.id,
		"target": targetAddr,
	})

	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("tunnel not connected")
	}

	atomic.AddInt64(&t.active, 1)
	defer atomic.AddInt64(&t.active, -1)

	t.mu.Lock()
	t.lastUsed = time.Now()
	t.mu.Unlock()

	logger.Debug("Starting forward connection")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if strings.HasPrefix(targetAddr, "[") {
		targetAddr = targetAddr[1 : len(targetAddr)-1]
	}

	remoteConn, err := client.DialContext(ctx, "tcp", targetAddr)
	if err != nil {
		logger.WithError(err).Error("Failed to dial target")
		return fmt.Errorf("failed to dial %s: %w", targetAddr, err)
	}
	defer remoteConn.Close()

	logger.Debug("Connected to target")

	errCh := make(chan error, 2)

	go func() {
		n, err := io.Copy(remoteConn, localConn)
		atomic.AddInt64(&t.totalBytes, n)
		if tcpConn, ok := remoteConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
		errCh <- err
	}()

	go func() {
		n, err := io.Copy(localConn, remoteConn)
		atomic.AddInt64(&t.totalBytes, n)
		if tcpConn, ok := localConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
		errCh <- err
	}()

	err1 := <-errCh
	err2 := <-errCh

	if isNormalError(err1) && isNormalError(err2) {
		logger.Debug("Forward connection completed normally")
		return nil
	}

	if err1 != nil {
		logger.WithError(err1).Error("Forward error (local to remote)")
		return fmt.Errorf("local to remote error: %w", err1)
	}

	logger.WithError(err2).Error("Forward error (remote to local)")
	return fmt.Errorf("remote to local error: %w", err2)
}

func (t *Tunnel) Disconnect() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cancel()

	if t.client != nil {
		t.client.Close()
		t.client = nil
		log.Infof("Tunnel %s disconnected", t.id)
	}
}

func (t *Tunnel) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.client != nil
}

// func (t *Tunnel) Forward(localConn net.Conn, targetAddr string) error {
// 	atomic.AddInt64(&t.requestCount, 1)
// 	t.mu.RLock()
// 	client := t.client
// 	t.mu.RUnlock()

// 	if client == nil {
// 		return fmt.Errorf("tunnel not connected")
// 	}

// 	atomic.AddInt64(&t.active, 1)
// 	defer atomic.AddInt64(&t.active, -1)

// 	t.mu.Lock()
// 	t.lastUsed = time.Now()
// 	t.mu.Unlock()

// 	// Use a context with timeout for the SSH dial
// 	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
// 	defer cancel()

// 	// Strip IPv6 brackets if present
// 	if strings.HasPrefix(targetAddr, "[") {
// 		targetAddr = targetAddr[1 : len(targetAddr)-1]
// 	}

// 	// Open a TCP connection through the SSH tunnel
// 	remoteConn, err := client.DialContext(ctx, "tcp", targetAddr)
// 	if err != nil {
// 		return fmt.Errorf("failed to dial %s: %w", targetAddr, err)
// 	}
// 	defer remoteConn.Close()

// 	log.Debugf("Tunnel %s: connected to %s", t.id, targetAddr)

// 	// Start bidirectional copy
// 	errCh := make(chan error, 2)

// 	go func() {
// 		n, err := io.Copy(remoteConn, localConn)
// 		atomic.AddInt64(&t.totalBytes, n)
// 		// Close the write half of the remote connection
// 		if tcpConn, ok := remoteConn.(*net.TCPConn); ok {
// 			tcpConn.CloseWrite()
// 		}
// 		errCh <- err
// 	}()

// 	go func() {
// 		n, err := io.Copy(localConn, remoteConn)
// 		atomic.AddInt64(&t.totalBytes, n)
// 		// Close the write half of the local connection
// 		if tcpConn, ok := localConn.(*net.TCPConn); ok {
// 			tcpConn.CloseWrite()
// 		}
// 		errCh <- err
// 	}()

// 	// Wait for both directions to finish
// 	err1 := <-errCh
// 	err2 := <-errCh

// 	if isNormalError(err1) && isNormalError(err2) {
// 		return nil
// 	}

// 	if err1 != nil {
// 		return fmt.Errorf("local to remote error: %w", err1)
// 	}
// 	return fmt.Errorf("remote to local error: %w", err2)
// }

func isNormalError(err error) bool {
	return err == nil || err == io.EOF ||
		strings.Contains(err.Error(), "closed") ||
		strings.Contains(err.Error(), "reset") ||
		strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "use of closed network connection")
}

// func (t *Tunnel) keepAlive() {
// 	ticker := time.NewTicker(30 * time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-t.ctx.Done():
// 			return
// 		case <-ticker.C:
// 			t.mu.RLock()
// 			client := t.client
// 			t.mu.RUnlock()

// 			if client != nil {
// 				// Send a keepalive request
// 				_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
// 				if err != nil {
// 					log.Warnf("Tunnel %s keepalive failed: %v", t.id, err)
// 					t.mu.Lock()
// 					t.lastError = err
// 					t.client.Close()
// 					t.client = nil
// 					t.mu.Unlock()
// 				}
// 			}
// 		}
// 	}
// }

func (t *Tunnel) GetStats() TunnelStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return TunnelStats{
		ID:           t.id,
		ServerName:   t.serverName,
		Connected:    t.client != nil,
		Active:       atomic.LoadInt64(&t.active),
		TotalBytes:   atomic.LoadInt64(&t.totalBytes),
		RequestCount: atomic.LoadInt64(&t.requestCount),
		LastUsed:     t.lastUsed,
		LastError:    t.lastError,
	}
}

type TunnelStats struct {
	ID           string
	ServerName   string
	Connected    bool
	Active       int64
	TotalBytes   int64
	RequestCount int64
	LastUsed     time.Time
	LastError    error
}
