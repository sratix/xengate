package tun

// import (
// 	"context"
// 	"encoding/binary"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net"
// 	"os/exec"
// 	"runtime"
// 	"strings"
// 	"sync"
// 	"sync/atomic"
// 	"time"

// 	"xengate/internal/models"

// 	"github.com/songgao/water"
// 	"golang.org/x/net/proxy"
// )

// const (
// 	// IP Protocol Numbers
// 	ICMP = 1
// 	TCP  = 6
// 	UDP  = 17
// )

// type Device struct {
// 	iface         *water.Interface
// 	config        *water.Config
// 	nat           *NAT
// 	proxy         *SOCKS5Proxy
// 	stopChan      chan struct{}
// 	ctx           context.Context
// 	cancel        context.CancelFunc
// 	stats         *Stats
// 	verbose       bool
// 	respondToIcmp bool
// 	handleDNS     bool
// 	dnsServers    []string
// 	mtu           int
// 	isRunning     atomic.Bool
// }

// type Stats struct {
// 	BytesIn    uint64
// 	BytesOut   uint64
// 	PacketsIn  uint64
// 	PacketsOut uint64
// 	mu         sync.RWMutex
// }

// type NAT struct {
// 	enabled     bool
// 	connections map[string]*NATEntry
// 	mu          sync.RWMutex
// }

// type NATEntry struct {
// 	originalSrc net.IP
// 	originalDst net.IP
// 	srcPort     uint16
// 	dstPort     uint16
// 	protocol    uint8
// 	proxyConn   net.Conn
// 	lastSeen    time.Time
// }

// type SOCKS5Proxy struct {
// 	server   string
// 	username string
// 	password string
// 	dialer   proxy.Dialer
// }

// type DeviceOption func(*Device)

// func WithVerbose(verbose bool) DeviceOption {
// 	return func(d *Device) {
// 		d.verbose = verbose
// 	}
// }

// func WithICMPResponse(enabled bool) DeviceOption {
// 	return func(d *Device) {
// 		d.respondToIcmp = enabled
// 	}
// }

// func WithDNSHandling(enabled bool, servers ...string) DeviceOption {
// 	return func(d *Device) {
// 		d.handleDNS = enabled
// 		d.dnsServers = servers
// 	}
// }

// func WithNAT(enabled bool) DeviceOption {
// 	return func(d *Device) {
// 		if d.nat != nil {
// 			d.nat.enabled = enabled
// 		}
// 	}
// }

// func WithSOCKS5Proxy(config *models.ProxyConfig) DeviceOption {
// 	return func(d *Device) {
// 		if config != nil {
// 			proxy, err := NewSOCKS5Proxy(config)
// 			if err == nil {
// 				d.proxy = proxy
// 			}
// 		}
// 	}
// }

// func NewDevice(name string, options ...DeviceOption) (*Device, error) {
// 	ctx, cancel := context.WithCancel(context.Background())

// 	dev := &Device{
// 		stopChan: make(chan struct{}),
// 		nat: &NAT{
// 			enabled:     true,
// 			connections: make(map[string]*NATEntry),
// 		},
// 		stats:  &Stats{},
// 		ctx:    ctx,
// 		cancel: cancel,
// 		mtu:    1500,
// 	}

// 	for _, opt := range options {
// 		opt(dev)
// 	}

// 	config := water.Config{
// 		DeviceType: water.TUN,
// 	}
// 	if runtime.GOOS != "windows" {
// 		config.Name = name
// 	}

// 	iface, err := water.New(config)
// 	if err != nil {
// 		return nil, err
// 	}

// 	dev.iface = iface
// 	return dev, nil
// }

// func (d *Device) Configure(address, gateway string, mtu int) error {
// 	d.mtu = mtu

// 	switch runtime.GOOS {
// 	case "linux":
// 		return d.configureLinux(address, gateway, mtu)
// 	case "darwin":
// 		return d.configureDarwin(address, gateway, mtu)
// 	case "windows":
// 		return d.configureWindows(address, gateway, mtu)
// 	default:
// 		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
// 	}
// }

// func (d *Device) Start(ctx context.Context) error {
// 	if d.isRunning.Load() {
// 		return nil
// 	}

// 	d.isRunning.Store(true)
// 	go d.readPackets()
// 	go d.cleanupNAT()

// 	return nil
// }

// func (d *Device) Stop() error {
// 	if !d.isRunning.Load() {
// 		return nil
// 	}

// 	d.cancel()
// 	close(d.stopChan)
// 	d.isRunning.Store(false)

// 	if d.iface != nil {
// 		return d.iface.Close()
// 	}
// 	return nil
// }

// func (d *Device) readPackets() {
// 	buffer := make([]byte, d.mtu)
// 	for {
// 		select {
// 		case <-d.ctx.Done():
// 			return
// 		case <-d.stopChan:
// 			return
// 		default:
// 			n, err := d.iface.Read(buffer)
// 			if err != nil {
// 				if err != io.EOF {
// 					log.Printf("Error reading from TUN: %v", err)
// 				}
// 				continue
// 			}

// 			packet := make([]byte, n)
// 			copy(packet, buffer[:n])
// 			go d.handlePacket(packet)
// 		}
// 	}
// }

// func (d *Device) handlePacket(packet []byte) {
// 	ipPacket, err := parseIPPacket(packet)
// 	if err != nil {
// 		return
// 	}

// 	d.stats.update("in", uint64(len(packet)))

// 	if d.verbose {
// 		log.Printf("Received packet: %s -> %s, Protocol: %d, Length: %d",
// 			ipPacket.SourceIP, ipPacket.DestinationIP, ipPacket.Protocol, len(packet))
// 	}

// 	// Handle NAT if enabled
// 	if d.nat.enabled {
// 		if modified := d.nat.processPacket(ipPacket); modified {
// 			d.Write(packet)
// 			return
// 		}
// 	}

// 	switch ipPacket.Protocol {
// 	case TCP:
// 		d.handleTCP(ipPacket)
// 	case UDP:
// 		d.handleUDP(ipPacket)
// 	case ICMP:
// 		d.handleICMP(ipPacket)
// 	}
// }

// func (d *Device) handleTCP(packet *IPPacket) {
// 	if !d.nat.enabled || d.proxy == nil {
// 		return
// 	}

// 	srcPort := binary.BigEndian.Uint16(packet.Payload[0:2])
// 	dstPort := binary.BigEndian.Uint16(packet.Payload[2:4])
// 	flags := packet.Payload[13]

// 	natKey := fmt.Sprintf("%s:%d-%s:%d",
// 		packet.SourceIP, srcPort,
// 		packet.DestinationIP, dstPort)

// 	d.nat.mu.Lock()
// 	entry, exists := d.nat.connections[natKey]

// 	// New connection (SYN packet)
// 	if !exists && (flags&0x02) != 0 {
// 		entry = &NATEntry{
// 			originalSrc: packet.SourceIP,
// 			originalDst: packet.DestinationIP,
// 			srcPort:     srcPort,
// 			dstPort:     dstPort,
// 			protocol:    TCP,
// 			lastSeen:    time.Now(),
// 		}
// 		d.nat.connections[natKey] = entry
// 		d.nat.mu.Unlock()

// 		// Start proxy connection
// 		go d.handleProxyConnection(natKey, packet)
// 	} else if exists {
// 		entry.lastSeen = time.Now()
// 		proxyConn := entry.proxyConn
// 		d.nat.mu.Unlock()

// 		if proxyConn != nil && len(packet.Payload) > 20 {
// 			// Forward data to proxy
// 			proxyConn.Write(packet.Payload[20:])
// 		}
// 	} else {
// 		d.nat.mu.Unlock()
// 	}
// }

// func (d *Device) handleProxyConnection(natKey string, packet *IPPacket) {
// 	targetAddr := fmt.Sprintf("%s:%d",
// 		packet.DestinationIP,
// 		binary.BigEndian.Uint16(packet.Payload[2:4]))

// 	conn, err := d.proxy.dialer.Dial("tcp", targetAddr)
// 	if err != nil {
// 		log.Printf("Failed to connect to target via proxy: %v", err)
// 		return
// 	}

// 	d.nat.mu.Lock()
// 	entry := d.nat.connections[natKey]
// 	if entry != nil {
// 		entry.proxyConn = conn
// 	}
// 	d.nat.mu.Unlock()

// 	// Handle proxy responses
// 	go d.handleProxyResponses(natKey, conn)
// }

// func (d *Device) handleProxyResponses(natKey string, proxyConn net.Conn) {
// 	defer proxyConn.Close()

// 	buffer := make([]byte, d.mtu-40) // Leave room for IP + TCP headers
// 	for {
// 		n, err := proxyConn.Read(buffer)
// 		if err != nil {
// 			if err != io.EOF {
// 				log.Printf("Error reading from proxy: %v", err)
// 			}
// 			d.nat.mu.Lock()
// 			delete(d.nat.connections, natKey)
// 			d.nat.mu.Unlock()
// 			return
// 		}

// 		d.nat.mu.RLock()
// 		entry := d.nat.connections[natKey]
// 		d.nat.mu.RUnlock()

// 		if entry == nil {
// 			return
// 		}

// 		// Create response packet
// 		response := d.createTCPResponse(
// 			entry.originalDst,
// 			entry.originalSrc,
// 			entry.dstPort,
// 			entry.srcPort,
// 			buffer[:n])

// 		d.Write(response)
// 	}
// }

// func (d *Device) handleUDP(packet *IPPacket) {
// 	if !d.handleDNS {
// 		return
// 	}

// 	dstPort := binary.BigEndian.Uint16(packet.Payload[2:4])
// 	if dstPort == 53 {
// 		d.handleDNSPacket(packet)
// 	}
// }

// func (d *Device) handleICMP(packet *IPPacket) {
// 	if !d.respondToIcmp {
// 		return
// 	}

// 	if len(packet.Payload) < 8 {
// 		return
// 	}

// 	icmpType := packet.Payload[0]
// 	if icmpType == 8 { // Echo Request
// 		response := d.createICMPResponse(packet)
// 		d.Write(response)
// 	}
// }

// func (d *Device) Write(packet []byte) error {
// 	_, err := d.iface.Write(packet)
// 	if err == nil {
// 		d.stats.update("out", uint64(len(packet)))
// 	}
// 	return err
// }

// func (d *Device) cleanupNAT() {
// 	ticker := time.NewTicker(30 * time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-d.ctx.Done():
// 			return
// 		case <-d.stopChan:
// 			return
// 		case <-ticker.C:
// 			now := time.Now()
// 			d.nat.mu.Lock()
// 			for key, entry := range d.nat.connections {
// 				if now.Sub(entry.lastSeen) > 5*time.Minute {
// 					if entry.proxyConn != nil {
// 						entry.proxyConn.Close()
// 					}
// 					delete(d.nat.connections, key)
// 				}
// 			}
// 			d.nat.mu.Unlock()
// 		}
// 	}
// }

// func (s *Stats) update(direction string, bytes uint64) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	if direction == "in" {
// 		s.BytesIn += bytes
// 		s.PacketsIn++
// 	} else {
// 		s.BytesOut += bytes
// 		s.PacketsOut++
// 	}
// }

// func NewSOCKS5Proxy(config *models.ProxyConfig) (*SOCKS5Proxy, error) {
// 	// auth := &proxy.Auth{
// 	// 	User:     config.Username,
// 	// 	Password: config.Password,
// 	// }

// 	addr := fmt.Sprintf("%s:%d", config.ListenAddr, config.ListenPort)

// 	dialer, err := proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &SOCKS5Proxy{
// 		server: addr,
// 		// username: config.Username,
// 		// password: config.Password,
// 		dialer: dialer,
// 	}, nil
// }

// type ProxyConfig struct {
// 	Type     string
// 	Server   string
// 	Username string
// 	Password string
// }

// // Helper functions for packet handling
// func parseIPPacket(data []byte) (*IPPacket, error) {
// 	if len(data) < 20 {
// 		return nil, fmt.Errorf("packet too short")
// 	}

// 	packet := &IPPacket{
// 		Version:       uint8(data[0] >> 4),
// 		IHL:           uint8(data[0] & 0x0f),
// 		TOS:           data[1],
// 		TotalLength:   binary.BigEndian.Uint16(data[2:4]),
// 		ID:            binary.BigEndian.Uint16(data[4:6]),
// 		Flags:         uint16(data[6] >> 5),
// 		FragOffset:    binary.BigEndian.Uint16(data[6:8]) & 0x1fff,
// 		TTL:           data[8],
// 		Protocol:      data[9],
// 		Checksum:      binary.BigEndian.Uint16(data[10:12]),
// 		SourceIP:      net.IP(data[12:16]),
// 		DestinationIP: net.IP(data[16:20]),
// 	}

// 	headerLen := int(packet.IHL * 4)
// 	if headerLen > 20 {
// 		packet.Options = data[20:headerLen]
// 	}

// 	if len(data) > headerLen {
// 		packet.Payload = data[headerLen:]
// 	}

// 	return packet, nil
// }

// type IPPacket struct {
// 	Version       uint8
// 	IHL           uint8
// 	TOS           uint8
// 	TotalLength   uint16
// 	ID            uint16
// 	Flags         uint16
// 	FragOffset    uint16
// 	TTL           uint8
// 	Protocol      uint8
// 	Checksum      uint16
// 	SourceIP      net.IP
// 	DestinationIP net.IP
// 	Options       []byte
// 	Payload       []byte
// }

// func (d *Device) createTCPResponse(src, dst net.IP, srcPort, dstPort uint16, payload []byte) []byte {
// 	// Create TCP header
// 	tcpHeader := make([]byte, 20)
// 	binary.BigEndian.PutUint16(tcpHeader[0:2], srcPort)
// 	binary.BigEndian.PutUint16(tcpHeader[2:4], dstPort)
// 	binary.BigEndian.PutUint32(tcpHeader[4:8], 0)       // Sequence number
// 	binary.BigEndian.PutUint32(tcpHeader[8:12], 0)      // Ack number
// 	tcpHeader[12] = 5 << 4                              // Data offset
// 	tcpHeader[13] = 0x18                                // PSH + ACK flags
// 	binary.BigEndian.PutUint16(tcpHeader[14:16], 65535) // Window size
// 	binary.BigEndian.PutUint16(tcpHeader[16:18], 0)     // Checksum
// 	binary.BigEndian.PutUint16(tcpHeader[18:20], 0)     // Urgent pointer

// 	// Combine with payload
// 	tcpData := append(tcpHeader, payload...)

// 	// Create IP header
// 	ipHeader := make([]byte, 20)
// 	ipHeader[0] = 0x45 // Version 4, IHL 5
// 	ipHeader[1] = 0    // DSCP/ECN
// 	binary.BigEndian.PutUint16(ipHeader[2:4], uint16(20+len(tcpData)))
// 	binary.BigEndian.PutUint16(ipHeader[4:6], 0) // ID
// 	ipHeader[6] = 0x40                           // Don't fragment
// 	ipHeader[7] = 0                              // Fragment offset
// 	ipHeader[8] = 64                             // TTL
// 	ipHeader[9] = TCP                            // Protocol
// 	copy(ipHeader[12:16], src.To4())
// 	copy(ipHeader[16:20], dst.To4())

// 	// Calculate checksums
// 	checksum := calculateIPChecksum(ipHeader)
// 	binary.BigEndian.PutUint16(ipHeader[10:12], checksum)

// 	// Calculate TCP checksum
// 	tcpChecksum := calculateTCPChecksum(src, dst, tcpData)
// 	binary.BigEndian.PutUint16(tcpData[16:18], tcpChecksum)

// 	return append(ipHeader, tcpData...)
// }

// func calculateIPChecksum(header []byte) uint16 {
// 	var sum uint32
// 	for i := 0; i < len(header); i += 2 {
// 		sum += uint32(binary.BigEndian.Uint16(header[i : i+2]))
// 	}
// 	sum = (sum >> 16) + (sum & 0xffff)
// 	sum += sum >> 16
// 	return ^uint16(sum)
// }

// func calculateTCPChecksum(src, dst net.IP, tcpData []byte) uint16 {
// 	var sum uint32

// 	// Add pseudoheader
// 	sum += uint32(binary.BigEndian.Uint16(src[0:2]))
// 	sum += uint32(binary.BigEndian.Uint16(src[2:4]))
// 	sum += uint32(binary.BigEndian.Uint16(dst[0:2]))
// 	sum += uint32(binary.BigEndian.Uint16(dst[2:4]))
// 	sum += uint32(TCP)
// 	sum += uint32(len(tcpData))

// 	// Add TCP header and data
// 	for i := 0; i < len(tcpData)-1; i += 2 {
// 		sum += uint32(binary.BigEndian.Uint16(tcpData[i : i+2]))
// 	}
// 	if len(tcpData)%2 == 1 {
// 		sum += uint32(tcpData[len(tcpData)-1]) << 8
// 	}

// 	sum = (sum >> 16) + (sum & 0xffff)
// 	sum += sum >> 16
// 	return ^uint16(sum)
// }

// func (d *Device) createICMPResponse(request *IPPacket) []byte {
// 	// Create ICMP response
// 	icmpResponse := make([]byte, len(request.Payload))
// 	copy(icmpResponse, request.Payload)
// 	icmpResponse[0] = 0                              // Change type to Echo Reply
// 	binary.BigEndian.PutUint16(icmpResponse[2:4], 0) // Clear checksum
// 	checksum := calculateICMPChecksum(icmpResponse)
// 	binary.BigEndian.PutUint16(icmpResponse[2:4], checksum)

// 	// Create IP header
// 	ipHeader := make([]byte, 20)
// 	ipHeader[0] = 0x45 // Version 4, IHL 5
// 	binary.BigEndian.PutUint16(ipHeader[2:4], uint16(20+len(icmpResponse)))
// 	ipHeader[8] = 64   // TTL
// 	ipHeader[9] = ICMP // Protocol
// 	copy(ipHeader[12:16], request.DestinationIP.To4())
// 	copy(ipHeader[16:20], request.SourceIP.To4())

// 	checksum = calculateIPChecksum(ipHeader)
// 	binary.BigEndian.PutUint16(ipHeader[10:12], checksum)

// 	return append(ipHeader, icmpResponse...)
// }

// func calculateICMPChecksum(data []byte) uint16 {
// 	var sum uint32
// 	for i := 0; i < len(data)-1; i += 2 {
// 		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
// 	}
// 	if len(data)%2 == 1 {
// 		sum += uint32(data[len(data)-1]) << 8
// 	}
// 	sum = (sum >> 16) + (sum & 0xffff)
// 	sum += sum >> 16
// 	return ^uint16(sum)
// }

// func (d *Device) configureLinux(address, gateway string, mtu int) error {
// 	// Set IP address
// 	cmd := exec.Command("ip", "addr", "add", address, "dev", d.iface.Name())
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("failed to set IP address: %v", err)
// 	}

// 	// Set MTU
// 	cmd = exec.Command("ip", "link", "set", "dev", d.iface.Name(), "mtu", fmt.Sprintf("%d", mtu))
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("failed to set MTU: %v", err)
// 	}

// 	// Bring up interface
// 	cmd = exec.Command("ip", "link", "set", "dev", d.iface.Name(), "up")
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("failed to bring up interface: %v", err)
// 	}

// 	// Add default route
// 	cmd = exec.Command("ip", "route", "add", "default", "via", gateway, "dev", d.iface.Name())
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("failed to add default route: %v", err)
// 	}

// 	return nil
// }

// func (d *Device) configureDarwin(address, gateway string, mtu int) error {
// 	// Set IP address
// 	cmd := exec.Command("ifconfig", d.iface.Name(), "inet", address, gateway)
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("failed to set IP address: %v", err)
// 	}

// 	// Set MTU
// 	cmd = exec.Command("ifconfig", d.iface.Name(), "mtu", fmt.Sprintf("%d", mtu))
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("failed to set MTU: %v", err)
// 	}

// 	// Add default route
// 	cmd = exec.Command("route", "add", "default", gateway)
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("failed to add default route: %v", err)
// 	}

// 	return nil
// }

// func (d *Device) configureWindows(address, gateway string, mtu int) error {
// 	// Get interface index
// 	cmd := exec.Command("netsh", "interface", "ipv4", "show", "interfaces")
// 	output, err := cmd.Output()
// 	if err != nil {
// 		return fmt.Errorf("failed to get interface index: %v", err)
// 	}

// 	var idx string
// 	lines := strings.Split(string(output), "\n")
// 	for _, line := range lines {
// 		if strings.Contains(line, d.iface.Name()) {
// 			fields := strings.Fields(line)
// 			if len(fields) > 0 {
// 				idx = fields[0]
// 				break
// 			}
// 		}
// 	}

// 	if idx == "" {
// 		return fmt.Errorf("interface not found")
// 	}

// 	// Set IP address
// 	cmd = exec.Command("netsh", "interface", "ipv4", "set", "address",
// 		fmt.Sprintf("name=%s", d.iface.Name()),
// 		"source=static",
// 		fmt.Sprintf("addr=%s", strings.Split(address, "/")[0]),
// 		fmt.Sprintf("mask=%s", getSubnetMask(address)),
// 		fmt.Sprintf("gateway=%s", gateway))
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("failed to set IP address: %v", err)
// 	}

// 	// Set MTU
// 	cmd = exec.Command("netsh", "interface", "ipv4", "set", "subinterface",
// 		d.iface.Name(),
// 		fmt.Sprintf("mtu=%d", mtu))
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("failed to set MTU: %v", err)
// 	}

// 	return nil
// }

// func (n *NAT) processPacket(packet *IPPacket) bool {
// 	if !n.enabled {
// 		return false
// 	}

// 	var key string
// 	switch packet.Protocol {
// 	case TCP:
// 		if len(packet.Payload) < 4 {
// 			return false
// 		}
// 		srcPort := binary.BigEndian.Uint16(packet.Payload[0:2])
// 		dstPort := binary.BigEndian.Uint16(packet.Payload[2:4])
// 		key = fmt.Sprintf("%s:%d-%s:%d", packet.SourceIP, srcPort, packet.DestinationIP, dstPort)
// 	case UDP:
// 		if len(packet.Payload) < 4 {
// 			return false
// 		}
// 		srcPort := binary.BigEndian.Uint16(packet.Payload[0:2])
// 		dstPort := binary.BigEndian.Uint16(packet.Payload[2:4])
// 		key = fmt.Sprintf("%s:%d-%s:%d", packet.SourceIP, srcPort, packet.DestinationIP, dstPort)
// 	default:
// 		return false
// 	}

// 	n.mu.RLock()
// 	entry, exists := n.connections[key]
// 	n.mu.RUnlock()

// 	if !exists {
// 		return false
// 	}

// 	// Update last seen time
// 	n.mu.Lock()
// 	entry.lastSeen = time.Now()
// 	n.mu.Unlock()

// 	return true
// }

// func (d *Device) handleDNSPacket(packet *IPPacket) {
// 	if len(packet.Payload) < 8 { // UDP header size
// 		return
// 	}

// 	dnsPayload := packet.Payload[8:] // Skip UDP header
// 	if len(dnsPayload) < 12 {        // Minimum DNS header size
// 		return
// 	}

// 	// Extract query ID and flags
// 	queryID := binary.BigEndian.Uint16(dnsPayload[0:2])
// 	flags := binary.BigEndian.Uint16(dnsPayload[2:4])

// 	// Check if it's a query (QR bit = 0)
// 	isQuery := (flags & 0x8000) == 0
// 	if !isQuery {
// 		return
// 	}

// 	// Create DNS response
// 	response := d.createDNSResponse(packet, queryID, dnsPayload)
// 	if response != nil {
// 		d.Write(response)
// 	}
// }

// func (d *Device) createDNSResponse(request *IPPacket, queryID uint16, dnsQuery []byte) []byte {
// 	if len(d.dnsServers) == 0 {
// 		return nil
// 	}

// 	// Create UDP connection to DNS server
// 	conn, err := net.Dial("udp", d.dnsServers[0]+":53")
// 	if err != nil {
// 		return nil
// 	}
// 	defer conn.Close()

// 	// Send query
// 	_, err = conn.Write(dnsQuery)
// 	if err != nil {
// 		return nil
// 	}

// 	// Read response
// 	response := make([]byte, 1024)
// 	n, err := conn.Read(response)
// 	if err != nil {
// 		return nil
// 	}

// 	// Create UDP header
// 	udpHeader := make([]byte, 8)
// 	binary.BigEndian.PutUint16(udpHeader[0:2], binary.BigEndian.Uint16(request.Payload[2:4])) // dst port becomes src
// 	binary.BigEndian.PutUint16(udpHeader[2:4], binary.BigEndian.Uint16(request.Payload[0:2])) // src port becomes dst
// 	binary.BigEndian.PutUint16(udpHeader[4:6], uint16(8+n))                                   // UDP length

// 	// Create IP header
// 	ipHeader := make([]byte, 20)
// 	ipHeader[0] = 0x45 // Version 4, IHL 5
// 	binary.BigEndian.PutUint16(ipHeader[2:4], uint16(20+8+n))
// 	ipHeader[8] = 64  // TTL
// 	ipHeader[9] = UDP // Protocol
// 	copy(ipHeader[12:16], request.DestinationIP.To4())
// 	copy(ipHeader[16:20], request.SourceIP.To4())

// 	// Calculate checksums
// 	checksum := calculateIPChecksum(ipHeader)
// 	binary.BigEndian.PutUint16(ipHeader[10:12], checksum)

// 	// Combine everything
// 	packet := append(ipHeader, udpHeader...)
// 	packet = append(packet, response[:n]...)

// 	return packet
// }

// func getSubnetMask(cidr string) string {
// 	_, ipNet, err := net.ParseCIDR(cidr)
// 	if err != nil {
// 		return "255.255.255.0" // Default to /24 if parsing fails
// 	}
// 	mask := ipNet.Mask
// 	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
// }
