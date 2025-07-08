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
	Mode       string `json:"mode"`
}

type ServerConfig struct {
	Name        string      `json:"name"`
	Host        string      `json:"host"`
	Port        int         `json:"port"`
	User        string      `json:"user"`
	Password    string      `json:"password,omitempty"`
	Connections int         `json:"connections"`
	MaxRetries  int         `json:"max_retries"`
	Proxy       ProxyConfig `json:"proxy"`
}

type Connection struct {
	ID      int64            `json:"id"`
	Name    string           `json:"name"`
	Address string           `json:"address"`
	Port    string           `json:"port"`
	Type    string           `json:"type"`
	Status  ConnectionStatus `json:"status"`
	Config  *ServerConfig    `json:"config"`
}
