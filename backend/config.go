package backend

import (
	"os"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

type AppConfig struct {
	WindowWidth                 int
	WindowHeight                int
	LastCheckedVersion          string
	LastLaunchedVersion         string
	EnableSystemTray            bool
	CloseToSystemTray           bool
	StartupPage                 string
	SettingsTab                 string
	AllowMultiInstance          bool
	MaxImageCacheSizeMB         int
	ClearCacheOnExit            bool
	DefaultPlaylistID           string
	AddToPlaylistSkipDuplicates bool
	ShowTrackChangeNotification bool
	Language                    string
	StartAtStartup              bool
	MinimizeAtStartup           bool
	DisableDPIDetection         bool
	RequestTimeoutSeconds       int

	FontNormalTTF string
	FontBoldTTF   string
	UIScaleSize   string
}

type GridViewConfig struct {
	CardSize float32
}

type ThemeConfig struct {
	ThemeFile  string
	Appearance string
}

type Config struct {
	Application AppConfig
	GridView    GridViewConfig
	Theme       ThemeConfig
}

var SupportedStartupPages = []string{"Albums", "Favorites", "Playlists"}

func DefaultConfig(appVersionTag string) *Config {
	return &Config{
		Application: AppConfig{
			WindowWidth:         480,
			WindowHeight:        480,
			LastCheckedVersion:  appVersionTag,
			LastLaunchedVersion: "",
			EnableSystemTray:    true,
			CloseToSystemTray:   false,
			ClearCacheOnExit:    false,
			StartupPage:         "Albums",
			SettingsTab:         "General",
			AllowMultiInstance:  false,
			UIScaleSize:         "Normal",
			Language:            "auto",
		},

		GridView: GridViewConfig{
			CardSize: 200,
		},

		Theme: ThemeConfig{
			Appearance: "Light",
		},
	}
}

func ReadConfigFile(filepath, appVersionTag string) (*Config, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c := DefaultConfig(appVersionTag)
	if err := toml.NewDecoder(f).Decode(c); err != nil {
		return nil, err
	}

	// Backfill Subsonic to empty ServerType fields
	// for updating configs created before multiple MediaProviders were added
	// for _, s := range c.Servers {
	// 	if s.ServerType == "" {
	// s.ServerType = ServerTypeSubsonic
	// 	}
	// }

	return c, nil
}

var writeLock sync.Mutex

func (c *Config) WriteConfigFile(filepath string) error {
	if !writeLock.TryLock() {
		return nil // another write in progress
	}
	defer writeLock.Unlock()

	b, err := toml.Marshal(c)
	if err != nil {
		return err
	}
	os.WriteFile(filepath, b, 0o644)

	return nil
}
