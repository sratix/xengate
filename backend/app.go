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

	"xengate/backend/util"

	"github.com/20after4/configdir"
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

	Storage           *AppStorage
	
	// UI callbacks to be set in main
	OnReactivate func()
	OnExit       func()

	appName        string
	displayAppName string
	appVersionTag  string
	configDir      string
	cacheDir       string
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

func StartupApp(appName, displayAppName, appVersion, appVersionTag, latestReleaseURL string) (*App, error) {
	var confDir, cacheDir string
	portableMode := false
	if p := checkPortablePath(); p != "" {
		confDir = path.Join(p, "config")
		cacheDir = path.Join(p, "cache")
		portableMode = true
	} else {
		confDir = configdir.LocalConfig(appName)
		cacheDir = configdir.LocalCache(appName)
	}
	// ensure config and cache dirs exist
	configdir.MakePath(confDir)
	configdir.MakePath(cacheDir)

	var logFile *os.File
	if isWindowsGUI() {
		// Can't log to console in Windows GUI app so log to file instead
		if f, err := os.Create(filepath.Join(confDir, "xengate.log")); err == nil {
			log.SetOutput(f)
			logFile = f
		}
	}

	a := &App{
		logFile:        logFile,
		appName:        appName,
		displayAppName: displayAppName,
		appVersionTag:  appVersionTag,
		configDir:      confDir,
		cacheDir:       cacheDir,
		portableMode:   portableMode,
	}
	a.bgrndCtx, a.cancel = context.WithCancel(context.Background())
	a.readConfig()

	log.Printf("Starting %s...", appName)
	log.Printf("Using config dir: %s", confDir)
	log.Printf("Using cache dir: %s", cacheDir)

	a.Config.Application.MaxImageCacheSizeMB = clamp(a.Config.Application.MaxImageCacheSizeMB, 1, 500)

	// // Periodically scan for remote players
	// go func() {
	// 	t := time.NewTicker(5 * time.Minute)
	// 	for {
	// 		select {
	// 		case <-a.bgrndCtx.Done():
	// 			t.Stop()
	// 			return
	// 		case <-t.C:
	// 			a.PlaybackManager.ScanRemotePlayers(a.bgrndCtx, false)
	// 		}
	// 	}
	// }()

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
	return filepath.Join(a.configDir, themesDir)
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
	var cfgExists bool
	if _, err := os.Stat(cfgPath); err == nil {
		cfgExists = true
	}
	a.isFirstLaunch = !cfgExists
	cfg, err := ReadConfigFile(cfgPath, a.appVersionTag)
	if err != nil {
		log.Printf("Error reading app config file: %v", err)
		cfg = DefaultConfig(a.appVersionTag)
		if cfgExists {
			backupCfgName := fmt.Sprintf("%s.bak", configFile)
			log.Printf("Config file may be malformed: copying to %s", backupCfgName)
			_ = util.CopyFile(cfgPath, path.Join(a.configDir, backupCfgName))
		}
	}
	a.Config = cfg
}

// periodically save config file so abnormal exit won't lose settings
func (a *App) startConfigWriter(ctx context.Context) {
	tick := time.NewTicker(2 * time.Minute)
	go func() {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		case <-tick.C:
			if !reflect.DeepEqual(&a.lastWrittenCfg, a.Config) {
				a.Config.WriteConfigFile(a.configFilePath())
				a.lastWrittenCfg = *a.Config
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

	a.cancel()
}

func (a *App) SaveConfigFile() {
	a.Config.WriteConfigFile(a.configFilePath())
	a.lastWrittenCfg = *a.Config
}

func (a *App) configFilePath() string {
	return path.Join(a.configDir, configFile)
}

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

	// check executable for windows GUI flag
	// https://stackoverflow.com/questions/58813512/is-it-possible-to-detect-if-go-binary-was-compiled-with-h-windowsgui-at-runtime
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
