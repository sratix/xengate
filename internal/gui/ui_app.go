//go:build !windows
// +build !windows

package gui

import (
	"context"
	"embed"
	"errors"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

type AppWindow struct {
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
	window       fyne.Window
	systrayWin   fyne.Window
	isHidden     bool
}

//go:embed translations
var translations embed.FS

func NewApp(title, version string) *AppWindow {
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

	return &AppWindow{
		app:        a,
		window:     a.NewWindow(title),
		systrayWin: a.NewWindow("Systray"),
		version:    version,
		isHidden:   false,
	}
}

func (w *AppWindow) Start(ctx context.Context) error {
	if w == nil {
		return errors.New("AppWindow is nil")
	}

	w.setupMainWindow()
	w.setupSystray()

	go func() {
		<-ctx.Done()
		// os.Exit(0)
		w.Shutdown()
	}()

	w.window.ShowAndRun()

	return nil
}

func (w *AppWindow) Shutdown() {
	if w == nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.window != nil {
		w.window.Close()
	}

	if w.systrayWin != nil {
		w.systrayWin.Close()
	}

	if w.app != nil {
		w.app.Quit()
	}
}
