package components

import (
	"encoding/json"
	"os"

	"xengate/internal/common"
	"xengate/internal/models"
)

const (
	configFile = "connections.json"
)

type DefaultConfigManager struct{}

func (m *DefaultConfigManager) LoadConfig() *common.Config {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return &common.Config{Connections: make([]*models.Connection, 0)}
	}

	var config common.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return &common.Config{Connections: make([]*models.Connection, 0)}
	}

	return &config
}

func (m *DefaultConfigManager) SaveConfig(config *common.Config) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0o644)
}
