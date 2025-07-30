package proxy

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"xengate/internal/tunnel"

	log "github.com/sirupsen/logrus"
)

const (
	socks5Version = 0x05
	authNone      = 0x00
	cmdConnect    = 0x01
	addrIPv4      = 0x01
	addrDomain    = 0x03
	addrIPv6      = 0x04
	replySuccess  = 0x00
)

type Socks5Server struct {
	manager       *tunnel.Manager
	listener      net.Listener
	wg            sync.WaitGroup
	mu            sync.RWMutex
	closed        bool
	ip            string
	port          int16
	blocklist     *IPBlocklist
	accessControl *AccessControl
}

func NewSocks5Server(ip string, port int16, manager *tunnel.Manager) (*Socks5Server, error) {
	return &Socks5Server{
		manager:       manager,
		ip:            ip,
		port:          port,
		blocklist:     NewIPBlocklist(),
		accessControl: NewAccessControl(1 * time.Hour),
	}, nil
}

func (s *Socks5Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.ip, s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener
	log.Infof("SOCKS5 server listening on %s", addr)

	s.wg.Add(1)
	go s.acceptLoop(ctx)

	// Shutdown handler
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return nil
}

func (s *Socks5Server) acceptLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				s.mu.RLock()
				closed := s.closed
				s.mu.RUnlock()
				if closed {
					return
				}
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Errorf("Accept error: %v", err)
				continue
			}

			s.wg.Add(1)
			go s.handleConnection(ctx, conn)
		}
	}
}

func (s *Socks5Server) handleConnection(ctx context.Context, conn net.Conn) {
	clientIP, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		conn.Close()
		return
	}

	// چک کردن بلک لیست
	if s.blocklist.IsBlocked(clientIP) {
		log.Debugf("Blocked connection from %s", clientIP)
		conn.Close()
		return
	}

	// چک کردن و شروع سشن
	if !s.accessControl.StartSession(clientIP) {
		log.Debugf("Access denied for %s (time limit exceeded)", clientIP)
		conn.Close()
		return
	}

	defer func() {
		s.accessControl.EndSession(clientIP)
		s.wg.Done()
		conn.Close()
	}()

	// Set timeout for handshake
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Step 1: Authentication
	if err := s.handleAuth(conn); err != nil {
		log.Debugf("Auth failed: %v", err)
		return
	}

	// Step 2: Request handling
	if err := s.handleRequest(conn); err != nil {
		log.Debugf("Request failed: %v", err)
	}
}

func (s *Socks5Server) handleAuth(conn net.Conn) error {
	// Read the version and number of methods
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return fmt.Errorf("failed to read auth header: %w", err)
	}

	version := buf[0]
	if version != socks5Version {
		return fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	nMethods := buf[1]
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return fmt.Errorf("failed to read methods: %w", err)
	}

	// We only support no-auth
	response := []byte{socks5Version, authNone}
	if _, err := conn.Write(response); err != nil {
		return fmt.Errorf("failed to write auth response: %w", err)
	}

	return nil
}

func (s *Socks5Server) handleRequest(conn net.Conn) error {
	// Read the request header (4 bytes)
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return fmt.Errorf("failed to read request header: %w", err)
	}

	version := buf[0]
	command := buf[1]
	// reserved := buf[2] // skip
	addressType := buf[3]

	if version != socks5Version {
		return fmt.Errorf("unsupported version in request: %d", version)
	}

	if command != cmdConnect {
		return fmt.Errorf("only CONNECT command is supported")
	}

	// Parse target address
	targetAddr, err := s.parseAddress(conn, addressType)
	if err != nil {
		return fmt.Errorf("failed to parse address: %w", err)
	}

	// Clear the deadline for the connection now that we've read the request
	conn.SetDeadline(time.Time{})

	// Handle the CONNECT request
	return s.handleConnect(conn, targetAddr)
}

func (s *Socks5Server) parseAddress(conn net.Conn, addrType byte) (string, error) {
	var host string
	var port uint16

	switch addrType {
	case addrIPv4:
		ip := make([]byte, 4)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", err
		}
		host = net.IP(ip).String()
	case addrDomain:
		domainLenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, domainLenBuf); err != nil {
			return "", err
		}
		domain := make([]byte, domainLenBuf[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", err
		}
		host = string(domain)
	case addrIPv6:
		ip := make([]byte, 16)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", err
		}
		host = "[" + net.IP(ip).String() + "]" // Wrap IPv6 in brackets
	default:
		return "", fmt.Errorf("unsupported address type: %d", addrType)
	}

	// Read port
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return "", err
	}
	port = binary.BigEndian.Uint16(portBuf)

	return net.JoinHostPort(host, strconv.Itoa(int(port))), nil
}

func (s *Socks5Server) handleConnect(clientConn net.Conn, targetAddr string) error {
	log.Debugf("CONNECT request to %s", targetAddr)

	// Send a success response (with 0.0.0.0:0 as bind address)
	response := []byte{
		socks5Version,
		replySuccess,
		0x00, // reserved
		addrIPv4,
		0x00, 0x00, 0x00, 0x00, // IPv4: 0.0.0.0
		0x00, 0x00, // port: 0
	}
	if _, err := clientConn.Write(response); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	// Now forward the connection
	err := s.manager.Forward(clientConn, targetAddr)

	// Don't log common errors
	if err != nil && err != io.EOF &&
		!strings.Contains(err.Error(), "closed") &&
		!strings.Contains(err.Error(), "reset") {
		log.Debugf("Forward error for %s: %v", targetAddr, err)
	}

	return nil
}

func (s *Socks5Server) Stop() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}

	s.wg.Wait()
	return nil
}

func (s *Socks5Server) BlockIP(ip string) {
	s.blocklist.Add(ip)
}

func (s *Socks5Server) UnblockIP(ip string) {
	s.blocklist.Remove(ip)
}
