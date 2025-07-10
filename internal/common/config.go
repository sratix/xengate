package common

import "xengate/internal/models"

type Config struct {
	Connections []*models.Connection `json:"connections"`
}

type ConfigManager interface {
	LoadConfig() *Config
	SaveConfig(*Config) error
}
