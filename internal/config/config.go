package config

import (
	"sync"

	"xengate/pkg/logger"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env           string        `yaml:"env" env:"APP_ENV" env-default:"production" env-description:"Environment [production, local, sandbox]"`
	Logger        logger.Config `yaml:"logger"`
	App           App           `yaml:"app"`
	Debug         bool          `yaml:"debug" env:"APP_DEBUG" env-default:"false" env-description:"Enables debug mode"`
	EmbedFrontend bool
}

type App struct {
	License LicenseConfig `yaml:"license"`
}

type LicenseConfig struct {
	ProductName   string `yaml:"product" env:"PRODUCT_NAME" env-description:"product name"`
	ProductSerial string `yaml:"serial" env:"PRODUCT_SERIAL" env-description:"product license serial"`
	ProductToken  string `yaml:"token" env:"PRODUCT_TOKEN" env-description:"product token"`
	ProductID     string `yaml:"omitempty"`
}

var (
	once   = sync.Once{}
	cfg    = &Config{}
	errCfg error
)

func New(configPath string, skipConfig bool) (*Config, error) {
	once.Do(func() {
		cfg = &Config{}

		if skipConfig {
			errCfg = cleanenv.ReadEnv(cfg)
			return
		}

		errCfg = cleanenv.ReadConfig(configPath, cfg)
	})

	return cfg, errCfg
}
