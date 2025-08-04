package common

import "xengate/internal/models"

type Config struct {
	Connections []*models.Connection    `json:"connections"`
	AccessRules []*models.AccessRule    `json:"rules"`
	BlockedList []*models.BlockedIPInfo `json:"blocked_list"`
}

type ConfigManager interface {
	LoadConfig() *Config
	SaveConfig(*Config) error
}
