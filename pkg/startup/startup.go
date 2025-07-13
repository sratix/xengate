package startup

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type StartupManager interface {
	Enable() error
	Disable() error
	IsEnabled() bool
}

func NewStartupManager(appName string) StartupManager {
	switch runtime.GOOS {
	case "windows":
		return &WindowsStartupManager{appName: appName}
	case "darwin":
		return &MacOSStartupManager{appName: appName}
	default:
		return &LinuxStartupManager{appName: appName}
	}
}

type WindowsStartupManager struct {
	appName string
}

func (m *WindowsStartupManager) Enable() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command("reg", "add", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", m.appName, "/t", "REG_SZ", "/d", exePath, "/f")
	return cmd.Run()
}

func (m *WindowsStartupManager) Disable() error {
	cmd := exec.Command("reg", "delete", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", m.appName, "/f")
	return cmd.Run()
}

func (m *WindowsStartupManager) IsEnabled() bool {
	cmd := exec.Command("reg", "query", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", m.appName)
	err := cmd.Run()
	return err == nil
}

type LinuxStartupManager struct {
	appName string
}

func (m *LinuxStartupManager) Enable() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	autostartDir := filepath.Join(homeDir, ".config/autostart")
	err = os.MkdirAll(autostartDir, 0o755)
	if err != nil {
		return err
	}

	desktopFile := filepath.Join(autostartDir, m.appName+".desktop")
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	content := `[Desktop Entry]
Type=Application
Version=1.0
Name=` + m.appName + `
Comment=` + m.appName + ` startup entry
Exec=` + exePath + `
Terminal=false
Categories=Utility;
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true`

	return os.WriteFile(desktopFile, []byte(content), 0o644)
}

func (m *LinuxStartupManager) Disable() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	desktopFile := filepath.Join(homeDir, ".config/autostart", m.appName+".desktop")
	if _, err := os.Stat(desktopFile); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(desktopFile)
}

func (m *LinuxStartupManager) IsEnabled() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	desktopFile := filepath.Join(homeDir, ".config/autostart", m.appName+".desktop")
	_, err = os.Stat(desktopFile)
	return err == nil
}

type MacOSStartupManager struct {
	appName string
}

func (m *MacOSStartupManager) Enable() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	launchAgentsDir := filepath.Join(homeDir, "Library/LaunchAgents")
	err = os.MkdirAll(launchAgentsDir, 0o755)
	if err != nil {
		return err
	}

	plistFile := filepath.Join(launchAgentsDir, "com."+m.appName+".plist")
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	content := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.` + m.appName + `</string>
    <key>ProgramArguments</key>
    <array>
        <string>` + exePath + `</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>`

	if err := os.WriteFile(plistFile, []byte(content), 0o644); err != nil {
		return err
	}

	cmd := exec.Command("launchctl", "load", plistFile)
	return cmd.Run()
}

func (m *MacOSStartupManager) Disable() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	plistFile := filepath.Join(homeDir, "Library/LaunchAgents", "com."+m.appName+".plist")

	if _, err := os.Stat(plistFile); err == nil {
		cmd := exec.Command("launchctl", "unload", plistFile)
		if err := cmd.Run(); err != nil {
			return err
		}
		return os.Remove(plistFile)
	}
	return nil
}

func (m *MacOSStartupManager) IsEnabled() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	plistFile := filepath.Join(homeDir, "Library/LaunchAgents", "com."+m.appName+".plist")
	_, err = os.Stat(plistFile)
	return err == nil
}
