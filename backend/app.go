package backend

import (
	"context"
	"debug/pe"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"time"

	"xengate/internal/storage"

	"fyne.io/fyne/v2"
)

const (
	configFile     = "config.toml"
	portableDir    = "supersonic_portable"
	savedQueueFile = "saved_queue.json"
	themesDir      = "themes"
)

var (
	ErrNoServers       = errors.New("no servers set up")
	ErrAnotherInstance = errors.New("another instance is running")

	appInstance *App
)

type App struct {
	Config *Config

	// Storage handling
	storage *storage.AppStorage

	// UI callbacks to be set in main
	OnReactivate func()
	OnExit       func()

	appName        string
	displayAppName string
	appVersionTag  string
	portableMode   bool

	isFirstLaunch bool // set by config file reader
	bgrndCtx      context.Context
	cancel        context.CancelFunc

	lastWrittenCfg Config

	logFile *os.File
}

func (a *App) VersionTag() string {
	return a.appVersionTag
}

func StartupApp(app fyne.App, appName, displayAppName, appVersion, appVersionTag, latestReleaseURL string) (*App, error) {
	// Initialize storage
	appStorage, err := storage.NewAppStorage(app)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %v", err)
	}

	portableMode := false
	if p := checkPortablePath(); p != "" {
		portableMode = true
	}

	var logFile *os.File
	if isWindowsGUI() {
		// Can't log to console in Windows GUI app so log to file instead
		logPath := filepath.Join(appStorage.ConfigPath(), "xengate.log")
		if f, err := os.Create(logPath); err == nil {
			log.SetOutput(f)
			logFile = f
		}
	}

	a := &App{
		storage:        appStorage,
		logFile:        logFile,
		appName:        appName,
		displayAppName: displayAppName,
		appVersionTag:  appVersionTag,
		portableMode:   portableMode,
	}

	a.bgrndCtx, a.cancel = context.WithCancel(context.Background())
	a.readConfig()

	log.Printf("Starting %s...", appName)
	log.Printf("Using config dir: %s", appStorage.ConfigPath())
	log.Printf("Using cache dir: %s", appStorage.CachePath())

	a.Config.Application.MaxImageCacheSizeMB = clamp(a.Config.Application.MaxImageCacheSizeMB, 1, 500)

	a.startConfigWriter(a.bgrndCtx)

	appInstance = a
	return a, nil
}

func AppInstance() *App {
	return appInstance
}

func (a *App) IsFirstLaunch() bool {
	return a.isFirstLaunch
}

func (a *App) IsPortableMode() bool {
	return a.portableMode
}

func (a *App) ThemesDir() string {
	return filepath.Join(a.storage.ConfigPath(), themesDir)
}

func checkPortablePath() string {
	if p, err := os.Executable(); err == nil {
		pdirPath := path.Join(filepath.Dir(p), portableDir)
		if s, err := os.Stat(pdirPath); err == nil && s.IsDir() {
			return pdirPath
		}
	}
	return ""
}

func (a *App) readConfig() {
	cfgPath := a.configFilePath()
	a.isFirstLaunch = !a.storage.FileExists(cfgPath)

	cfg, err := ReadConfigFile(cfgPath, a.appVersionTag)
	if err != nil {
		log.Printf("Error reading app config file: %v", err)
		cfg = DefaultConfig(a.appVersionTag)
		if !a.isFirstLaunch {
			backupCfgName := fmt.Sprintf("%s.bak", configFile)
			backupPath := filepath.Join(a.storage.ConfigPath(), backupCfgName)
			log.Printf("Config file may be malformed: copying to %s", backupCfgName)
			_ = a.storage.CopyFile(cfgPath, backupPath)
		}
	}
	a.Config = cfg
}

func (a *App) startConfigWriter(ctx context.Context) {
	tick := time.NewTicker(2 * time.Minute)
	go func() {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		case <-tick.C:
			if !reflect.DeepEqual(&a.lastWrittenCfg, a.Config) {
				a.SaveConfigFile()
			}
		}
	}()
}

func (a *App) callOnReactivate() {
	if a.OnReactivate != nil {
		a.OnReactivate()
	}
}

func (a *App) callOnExit() error {
	if a.OnExit == nil {
		return errors.New("no quit handler registered")
	}
	go func() {
		time.Sleep(10 * time.Millisecond)
		a.OnExit()
	}()
	return nil
}

func (a *App) Shutdown() {
	if a.logFile != nil {
		a.logFile.Close()
	}

	a.SaveConfigFile()

	// بررسی سایز کش قبل از پاکسازی
	if cacheSize, err := a.storage.GetCacheSize(); err == nil {
		maxSize := int64(a.Config.Application.MaxImageCacheSizeMB) * 1024 * 1024
		if cacheSize > maxSize {
			_ = a.storage.ClearCache()
		}
	}

	a.cancel()
}

func (a *App) SaveConfigFile() {
	a.Config.WriteConfigFile(a.configFilePath())
	a.lastWrittenCfg = *a.Config
}

func (a *App) configFilePath() string {
	return filepath.Join(a.storage.ConfigPath(), configFile)
}

// Helper functions remain unchanged
func clamp(i, min, max int) int {
	if i < min {
		i = min
	} else if i > max {
		i = max
	}
	return i
}

func isWindowsGUI() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	fileName, err := os.Executable()
	if err != nil {
		return false
	}
	fl, err := pe.Open(fileName)
	if err != nil {
		return false
	}
	defer fl.Close()

	var subsystem uint16
	if header, ok := fl.OptionalHeader.(*pe.OptionalHeader64); ok {
		subsystem = header.Subsystem
	} else if header, ok := fl.OptionalHeader.(*pe.OptionalHeader32); ok {
		subsystem = header.Subsystem
	}

	return subsystem == 2 /*IMAGE_SUBSYSTEM_WINDOWS_GUI*/
}
