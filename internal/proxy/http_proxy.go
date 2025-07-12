package proxy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"xengate/internal/tunnel"

	log "github.com/sirupsen/logrus"
)

type HTTPProxy struct {
	manager  *tunnel.Manager
	listener net.Listener
	wg       sync.WaitGroup
	mu       sync.RWMutex
	closed   bool
	ip       string
	port     int16
	mode     string
}

func NewHTTPProxy(mode string, ip string, port int16, manager *tunnel.Manager) *HTTPProxy {
	return &HTTPProxy{
		manager: manager,
		ip:      ip,
		port:    port,
		mode:    mode,
	}
}

func (p *HTTPProxy) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", p.ip, p.port)

	var listener net.Listener
	var err error

	if p.mode == "https" {
		// Load TLS certificate
		// cert, err := tls.LoadX509KeyPair(p.config.CertFile, p.config.KeyFile)
		// if err != nil {
		// 	return fmt.Errorf("failed to load TLS certificate: %w", err)
		// }

		// tlsConfig := &tls.Config{
		// 	Certificates: []tls.Certificate{cert},
		// }

		// listener, err = tls.Listen("tcp", addr, tlsConfig)
		// log.Infof("HTTPS proxy listening on %s", addr)
	} else {
		listener, err = net.Listen("tcp", addr)
		log.Infof("HTTP proxy listening on %s", addr)
	}

	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	p.listener = listener

	p.wg.Add(1)
	go p.acceptLoop(ctx)

	// Shutdown handler
	go func() {
		<-ctx.Done()
		p.Stop()
	}()

	return nil
}

func (p *HTTPProxy) acceptLoop(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := p.listener.Accept()
			if err != nil {
				p.mu.RLock()
				closed := p.closed
				p.mu.RUnlock()
				if closed {
					return
				}
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Errorf("Accept error: %v", err)
				continue
			}

			p.wg.Add(1)
			go p.handleConnection(ctx, conn)
		}
	}
}

func (p *HTTPProxy) handleConnection(ctx context.Context, clientConn net.Conn) {
	defer p.wg.Done()
	defer clientConn.Close()

	// Set timeout for initial request
	clientConn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Read the first line to determine the type of request
	reader := bufio.NewReader(clientConn)
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		log.Debugf("Failed to read request: %v", err)
		return
	}

	firstLine = strings.TrimSpace(firstLine)
	if firstLine == "" {
		return
	}

	// Check if it's a CONNECT request (HTTPS)
	if strings.HasPrefix(strings.ToUpper(firstLine), "CONNECT ") {
		p.handleHTTPS(ctx, clientConn, reader, firstLine)
	} else {
		// Handle HTTP request
		p.handleHTTP(ctx, clientConn, reader, firstLine)
	}
}

func (p *HTTPProxy) handleHTTPS(ctx context.Context, clientConn net.Conn, reader *bufio.Reader, firstLine string) {
	// Extract host:port from CONNECT request
	parts := strings.Split(firstLine, " ")
	if len(parts) < 2 {
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	target := parts[1]
	log.Debugf("HTTPS CONNECT request for: %s", target)

	// Read headers until we get an empty line (end of headers)
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
	}

	// Acknowledge CONNECT request
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Clear timeout now that connection is established
	clientConn.SetDeadline(time.Time{})

	// Tunnel the connection
	p.manager.Forward(clientConn, target)
}

func (p *HTTPProxy) handleHTTP(ctx context.Context, clientConn net.Conn, reader *bufio.Reader, firstLine string) {
	// Parse the request line
	parts := strings.Split(firstLine, " ")
	if len(parts) < 3 {
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	method, requestURI, proto := parts[0], parts[1], parts[2]
	log.Debugf("HTTP %s request for: %s", method, requestURI)

	// Parse the URL
	targetURL, err := url.Parse(requestURI)
	if err != nil {
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	// Extract host and port
	host := targetURL.Hostname()
	port := targetURL.Port()
	if port == "" {
		if targetURL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	target := net.JoinHostPort(host, port)

	// Reconstruct the request with relative URL
	requestURL := requestURI
	if targetURL.Opaque == "" && targetURL.Host != "" {
		requestURL = targetURL.RequestURI()
	}

	// Read headers
	var headers bytes.Buffer
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
		headers.WriteString(line)
	}

	// Check for HTTP/2
	isHTTP2 := strings.Contains(headers.String(), "HTTP/2")

	// Connect to target through tunnel
	targetConn, err := p.dialThroughTunnel(ctx, target)
	if err != nil {
		log.Errorf("Failed to connect to target: %v", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer targetConn.Close()

	if isHTTP2 {
		// For HTTP/2, just tunnel the raw connection
		go func() {
			io.Copy(targetConn, io.MultiReader(
				bytes.NewBufferString(firstLine+"\r\n"),
				&headers,
				bytes.NewBufferString("\r\n"),
				reader,
			))
		}()
		io.Copy(clientConn, targetConn)
	} else {
		// Send request to target
		fmt.Fprintf(targetConn, "%s %s %s\r\n", method, requestURL, proto)
		io.Copy(targetConn, &headers)
		targetConn.Write([]byte("\r\n"))

		// Copy any remaining body
		go io.Copy(targetConn, reader)

		// Copy response back to client
		io.Copy(clientConn, targetConn)
	}
}

func (p *HTTPProxy) dialThroughTunnel(ctx context.Context, target string) (net.Conn, error) {
	// Create a connection pair
	clientConn, serverConn := net.Pipe()

	go func() {
		defer serverConn.Close()
		if err := p.manager.Forward(serverConn, target); err != nil {
			log.Debugf("Forward error: %v", err)
		}
	}()

	return clientConn, nil
}

func (p *HTTPProxy) Stop() error {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()

	if p.listener != nil {
		p.listener.Close()
	}

	p.wg.Wait()
	return nil
}
