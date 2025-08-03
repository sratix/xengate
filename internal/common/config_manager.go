package common

import (
	"encoding/json"
	"path/filepath"

	"xengate/internal/models"
	"xengate/internal/storage"
)

const (
	configFile = "config.json"
)

type DefaultConfigManager struct {
	Storage *storage.AppStorage
}

func (m *DefaultConfigManager) LoadConfig() *Config {
	path := filepath.Join(m.Storage.ConfigPath(), configFile)
	data, err := m.Storage.ReadFile(path)
	if err != nil {
		return &Config{Connections: make([]*models.Connection, 0), Rules: make([]*models.Rule, 0)}
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return &Config{Connections: make([]*models.Connection, 0), Rules: make([]*models.Rule, 0)}
	}

	return &config
}

func (m *DefaultConfigManager) SaveConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	path := filepath.Join(m.Storage.ConfigPath(), configFile)
	return m.Storage.WriteFile(path, data)
}
