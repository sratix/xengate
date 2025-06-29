//go:build !(android || ios)
// +build !android,!ios

package gui

import (
	"context"
	"embed"
	"errors"
	"os"
	"sync"

	"fyne.io/fyne/theme"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

type FyneUI struct {
	tabs         *container.AppTabs
	CheckVersion *widget.Button
	State        string
	version      string
	systemTheme  fyne.ThemeVariant
	muError      sync.RWMutex
	mu           sync.RWMutex
	ErrorVisible bool
	Hotkeys      bool
	app          fyne.App
	MainWin      fyne.Window
	systrayWin   fyne.Window
	isHidden     bool
}

//go:embed translations
var translations embed.FS

func Start(ctx context.Context, s *FyneUI) error {
	if s == nil {
		return errors.New("FyneUI is nil")
	}

	s.setupMainWindow()
	s.setupSystray()

	go func() {
		<-ctx.Done()
		os.Exit(0)
	}()

	s.MainWin.ShowAndRun()

	return nil
}

func newGUI(version string) *FyneUI {
	a := app.NewWithID("app.xengate.xengate")

	preferredLanguage := a.Preferences().String("Language")
	filename := "translations/en.json"
	if preferredLanguage == "Persian" {
		filename = "translations/fa.json"
	}

	if content, err := translations.ReadFile(filename); err == nil {
		name := lang.SystemLocale().LanguageString()
		lang.AddTranslations(fyne.NewStaticResource(name+".json", content))
	} else {
		lang.AddTranslationsFS(translations, "translations")
	}

	a.SetIcon(fyne.NewStaticResource("icon", go2TVIcon512))

	return &FyneUI{
		app:        a,
		MainWin:    a.NewWindow("XenGate"),
		version:    version,
		systrayWin: a.NewWindow("Systray"),
		isHidden:   false,
	}
}

func (s *FyneUI) setupMainWindow() {
	tabs := container.NewAppTabs(
		container.NewTabItem("SSH-Proxy", mainWindow(s)),
		container.NewTabItem(lang.L("Settings"), settingsWindow(s)),
		container.NewTabItem(lang.L("About"), aboutWindow(s)),
	)

	s.Hotkeys = true
	tabs.OnSelected = func(t *container.TabItem) {
		s.Hotkeys = t.Text != "XenGate"
	}

	s.tabs = tabs

	s.MainWin.SetContent(tabs)
	s.MainWin.Resize(fyne.NewSize(s.MainWin.Canvas().Size().Width, s.MainWin.Canvas().Size().Height*1.2))
	s.MainWin.CenterOnScreen()
	s.MainWin.SetMaster()
	s.MainWin.SetCloseIntercept(func() {
		s.MainWin.Hide()
		s.app.SendNotification(fyne.NewNotification("App Minimized", "Running in system tray"))
	})
}

func (s *FyneUI) setupSystray() {
	if desk, ok := s.app.(desktop.App); ok {
		menu := fyne.NewMenu("XenGate",
			fyne.NewMenuItem("Show Window", func() {
				s.showMain()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				s.quit()
			}),
		)
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(theme.FyneLogo())
	}
}

func (s *FyneUI) showMain() {
	s.MainWin.Show()
	s.isHidden = false
}

func (s *FyneUI) quit() {
	s.app.Quit()
}
