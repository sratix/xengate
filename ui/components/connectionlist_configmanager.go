package components

import (
	"encoding/json"
	"path/filepath"

	"xengate/internal/common"
	"xengate/internal/models"
	"xengate/internal/storage"
)

const (
	configFile = "connections.json"
)

type DefaultConfigManager struct {
	Storage *storage.AppStorage
}

func (m *DefaultConfigManager) LoadConfig() *common.Config {
	path := filepath.Join(m.Storage.ConfigPath(), configFile)
	data, err := m.Storage.ReadFile(path)
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
	path := filepath.Join(m.Storage.ConfigPath(), configFile)
	return m.Storage.WriteFile(path, data)
}
