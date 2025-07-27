package tunnel

import (
	"context"
	"fmt"

	"xengate/internal/models"
	"xengate/internal/tun"
)

type TunController struct {
	device    *tun.Device
	config    *models.TunConfig
	proxy     *models.ProxyConfig
	isRunning bool
}

func NewTunController(config *models.TunConfig, proxy *models.ProxyConfig) (*TunController, error) {
	device, err := tun.NewDevice(config.DeviceName,
		tun.WithVerbose(true),
		tun.WithICMPResponse(true),
		tun.WithDNSHandling(true, config.DNSServers...),
		tun.WithNAT(true),
		tun.WithSOCKS5Proxy(proxy),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN device: %w", err)
	}

	return &TunController{
		device: device,
		config: config,
		proxy:  proxy,
	}, nil
}

func (c *TunController) Start(ctx context.Context) error {
	if c.isRunning {
		return nil
	}

	if err := c.device.Configure(c.config.Address, c.config.Gateway, c.config.MTU); err != nil {
		return err
	}

	if err := c.device.Start(ctx); err != nil {
		return err
	}

	c.isRunning = true
	return nil
}

func (c *TunController) Stop() error {
	if !c.isRunning {
		return nil
	}

	err := c.device.Stop()
	if err != nil {
		return err
	}

	c.isRunning = false
	return nil
}
