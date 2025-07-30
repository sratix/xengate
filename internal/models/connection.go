package models

type ConnectionStatus int

const (
	StatusInactive ConnectionStatus = iota
	StatusActive
	StatusPending
)

type ProxyConfig struct {
	ListenAddr string `json:"listen_addr"`
	ListenPort int    `json:"listen_port"`
}

type ServerConfig struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	Password    string `json:"password,omitempty"`
	Connections int    `json:"connections"`
	MaxRetries  int    `json:"max_retries"`
	Mode        string `json:"mode"`
}

type Connection struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Address   string           `json:"address"`
	Port      string           `json:"port"`
	Type      string           `json:"type"`
	Config    *ServerConfig    `json:"config"`
	Status    ConnectionStatus `json:"status"`
	TunConfig *TunConfig       `json:"tun_config,omitempty"`
	Stats     *Stats
}

type Stats struct {
	ServerName    string `json:"server_name"`
	TotalTunnels  int    `json:"total_tunnels"`
	TotalRequests int64  `json:"total_requests"`
	TotalBytes    int64  `json:"total_bytes"`
	Active        int64  `json:"active"`
	Connected     int    `json:"connected"`
}

type TunConfig struct {
	DeviceName string
	Address    string
	Gateway    string
	MTU        int
	DNSServers []string
}
