package proxy

import (
	"context"
	"fmt"

	"xengate/internal/tunnel"
)

type Proxy interface {
	Start(ctx context.Context) error
	Stop() error
}

func NewProxy(mode string, ip string, port int16, manager *tunnel.Manager) (Proxy, error) {
	switch mode {
	case "socks5":
		return NewSocks5Server(ip, port, manager)
	case "http", "https":
		return NewHTTPProxy(mode, ip, port, manager), nil
	case "tuntap":
		return NewTunTapProxy("tun0", ip, port, manager)
	default:
		return nil, fmt.Errorf("unsupported proxy mode: %s", mode)
	}
}
