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
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

type App struct {
	fyne.App

	version      string
	CheckVersion *widget.Button

	// Main window
	topWindow      fyne.Window
	topWindowTitle binding.String

	systemTheme fyne.ThemeVariant
	muError     sync.RWMutex
	mu          sync.RWMutex
	systrayWin  fyne.Window
	isHidden    bool

	// Top toolbar row
	topRow *fyne.Container

	// Toolbar
	toolBar *widget.Toolbar
	// Toolbar actions
	actAbout            *widget.ToolbarAction
	actMenu             *widget.ToolbarAction
	actSettings         *widget.ToolbarAction
	actAdd              *widget.ToolbarAction
	actToggleView       *widget.ToolbarAction
	actToggleFullScreen *widget.ToolbarAction
	actNoAction         *widget.ToolbarAction

	// Frame view
	frameView *fyne.Container
}

//go:embed translations
var translations embed.FS

func NewApp(title, version string) *App {
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

	app := App{
		topWindow:  a.NewWindow(title),
		systrayWin: a.NewWindow("Systray"),
		version:    version,
		isHidden:   false,
	}

	app.topWindow.CenterOnScreen()
	app.topWindow.SetMaster()

	return &app
}

func (a *App) Start(ctx context.Context) error {
	if a == nil {
		return errors.New("AppWindow is nil")
	}

	a.setupMainWindow()
	a.setupSystray()

	go func() {
		<-ctx.Done()
		a.Shutdown()
	}()

	a.topWindow.ShowAndRun()

	return nil
}

func (a *App) Shutdown() {
	if a == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.topWindow != nil {
		a.topWindow.Close()
	}

	if a.systrayWin != nil {
		a.systrayWin.Close()
	}

	a.Quit()
}
