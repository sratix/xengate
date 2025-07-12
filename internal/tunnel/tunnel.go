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
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client != nil {
		return nil // Already connected
	}

	// Create a copy of config to modify client version
	configCopy := *t.config
	configCopy.ClientVersion = "SSH-2.0-OpenSSH_8.4p1" // Modern client version
	configCopy.Timeout = 15 * time.Second              // Shorter timeout

	log.Debugf("Connecting tunnel %s to %s", t.id, t.addr)

	// Use DialContext for proper cancellation
	dialCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var client *ssh.Client
	var err error

	// Dial with context
	go func() {
		client, err = ssh.Dial("tcp", t.addr, &configCopy)
		cancel() // Ensure cancel is called when dial completes
	}()

	<-dialCtx.Done()

	if err != nil {
		t.lastError = err
		return fmt.Errorf("failed to connect: %w", err)
	}
	if client == nil {
		return fmt.Errorf("connection timeout")
	}

	t.client = client
	t.lastError = nil
	log.Infof("Tunnel %s connected to %s", t.id, t.addr)

	// Start keepalive
	go t.keepAlive()

	return nil
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

func (t *Tunnel) Forward(localConn net.Conn, targetAddr string) error {
	atomic.AddInt64(&t.requestCount, 1)
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

	// Use a context with timeout for the SSH dial
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Strip IPv6 brackets if present
	if strings.HasPrefix(targetAddr, "[") {
		targetAddr = targetAddr[1 : len(targetAddr)-1]
	}

	// Open a TCP connection through the SSH tunnel
	remoteConn, err := client.DialContext(ctx, "tcp", targetAddr)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %w", targetAddr, err)
	}
	defer remoteConn.Close()

	log.Debugf("Tunnel %s: connected to %s", t.id, targetAddr)

	// Start bidirectional copy
	errCh := make(chan error, 2)

	go func() {
		n, err := io.Copy(remoteConn, localConn)
		atomic.AddInt64(&t.totalBytes, n)
		// Close the write half of the remote connection
		if tcpConn, ok := remoteConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
		errCh <- err
	}()

	go func() {
		n, err := io.Copy(localConn, remoteConn)
		atomic.AddInt64(&t.totalBytes, n)
		// Close the write half of the local connection
		if tcpConn, ok := localConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
		errCh <- err
	}()

	// Wait for both directions to finish
	err1 := <-errCh
	err2 := <-errCh

	if isNormalError(err1) && isNormalError(err2) {
		return nil
	}

	if err1 != nil {
		return fmt.Errorf("local to remote error: %w", err1)
	}
	return fmt.Errorf("remote to local error: %w", err2)
}

func isNormalError(err error) bool {
	return err == nil || err == io.EOF ||
		strings.Contains(err.Error(), "closed") ||
		strings.Contains(err.Error(), "reset") ||
		strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "use of closed network connection")
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
				// Send a keepalive request
				_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
				if err != nil {
					log.Warnf("Tunnel %s keepalive failed: %v", t.id, err)
					t.mu.Lock()
					t.lastError = err
					t.client.Close()
					t.client = nil
					t.mu.Unlock()
				}
			}
		}
	}
}

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
