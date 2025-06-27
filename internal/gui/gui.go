//go:build !(android || ios)
// +build !android,!ios

package gui

import (
	"container/ring"
	"context"
	"embed"
	"errors"
	"os"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

// FyneScreen .
type FyneScreen struct {
	SelectInternalSubs   *widget.Select
	CurrentPos           binding.String
	EndPos               binding.String
	Current              fyne.Window
	cancelEnablePlay     context.CancelFunc
	PlayPause            *widget.Button
	Debug                *debugWriter
	VolumeUp             *widget.Button
	tabs                 *container.AppTabs
	CheckVersion         *widget.Button
	SubsText             *widget.Entry
	CustomSubsCheck      *widget.Check
	NextMediaCheck       *widget.Check
	TranscodeCheckBox    *widget.Check
	Stop                 *widget.Button
	MediaText            *widget.Entry
	ExternalMediaURL     *widget.Check
	MuteUnmute           *widget.Button
	VolumeDown           *widget.Button
	selectedDevice       devType
	State                string
	mediafile            string
	version              string
	eventlURL            string
	subsfile             string
	controlURL           string
	renderingControlURL  string
	connectionManagerURL string
	currentmfolder       string
	ffmpegPath           string
	ffmpegSeek           int
	systemTheme          fyne.ThemeVariant
	mediaFormats         []string
	muError              sync.RWMutex
	mu                   sync.RWMutex
	ffmpegPathChanged    bool
	Medialoop            bool
	sliderActive         bool
	Transcode            bool
	ErrorVisible         bool
	Hotkeys              bool
}

type debugWriter struct {
	ring *ring.Ring
}

type devType struct {
	name string
	addr string
}

type mainButtonsLayout struct {
	buttonHeight  float32
	buttonPadding float32
}

func (f *debugWriter) Write(b []byte) (int, error) {
	f.ring.Value = string(b)
	f.ring = f.ring.Next()
	return len(b), nil
}

//go:embed translations
var translations embed.FS

// Start .
func Start(ctx context.Context, s *FyneScreen) error {
	if s == nil {
		return errors.New("FyneScreen is nil")
	}

	w := s.Current

	tabs := container.NewAppTabs(
		container.NewTabItem("SShProxy", container.NewPadded(mainWindow(s))),
		container.NewTabItem(lang.L("Settings"), container.NewPadded(settingsWindow(s))),
		container.NewTabItem(lang.L("About"), aboutWindow(s)),
	)

	s.Hotkeys = true
	tabs.OnSelected = func(t *container.TabItem) {
		if t.Text == "XenGate" {
			s.Hotkeys = true
			s.TranscodeCheckBox.Enable()
			return
		}
		s.Hotkeys = false
	}

	s.tabs = tabs

	w.SetContent(tabs)
	w.Resize(fyne.NewSize(w.Canvas().Size().Width, w.Canvas().Size().Height*1.2))
	w.CenterOnScreen()
	w.SetMaster()

	go func() {
		<-ctx.Done()
		os.Exit(0)
	}()

	w.ShowAndRun()

	return nil
}

func (p *FyneScreen) Shutdown(ctx context.Context) error {
	if p.Current != nil {
		p.Current.Close()
	}

	return nil
}

// EmitMsg Method to implement the screen interface
func (p *FyneScreen) EmitMsg(a string) {
	switch a {
	default:
		dialog.ShowInformation("?", "Unknown callback value", p.Current)
	}
}

// Fini Method to implement the screen interface.
// Will only be executed when we receive a callback message,
// not when we explicitly click the Stop button.
func (p *FyneScreen) Fini() {
	gaplessOption := fyne.CurrentApp().Preferences().StringWithFallback("Gapless", "Disabled")
	_ = gaplessOption
}

func initFyneNewScreen(version string) *FyneScreen {
	a := app.NewWithID("app.xengate.xengate")

	// Hack. Ongoing discussion in https://github.com/fyne-io/fyne/issues/5333
	var content []byte
	switch a.Preferences().String("Language") {
	case "Persian":
		content, _ = translations.ReadFile("translations/fa.json")
	case "English":
		content, _ = translations.ReadFile("translations/en.json")
	}

	if content != nil {
		name := lang.SystemLocale().LanguageString()
		lang.AddTranslations(fyne.NewStaticResource(name+".json", content))
	} else {
		lang.AddTranslationsFS(translations, "translations")
	}

	a.SetIcon(fyne.NewStaticResource("icon", go2TVIcon512))

	w := a.NewWindow("XenGate")
	currentDir, err := os.Getwd()
	if err != nil {
		currentDir = ""
	}

	dw := &debugWriter{
		ring: ring.New(1000),
	}

	return &FyneScreen{
		Current:        w,
		currentmfolder: currentDir,
		version:        version,
		Debug:          dw,
	}
}

func check(s *FyneScreen, err error) {
	s.muError.Lock()
	defer s.muError.Unlock()

	if err != nil && !s.ErrorVisible {
		s.ErrorVisible = true
		cleanErr := strings.ReplaceAll(err.Error(), ": ", "\n")
		e := dialog.NewError(errors.New(cleanErr), s.Current)
		e.Show()
		e.SetOnClosed(func() {
			s.ErrorVisible = false
		})
	}
}

// updateScreenState updates the screen state based on
// the emitted messages. The State variable is used across
// the GUI interface to control certain flows.
// func (p *FyneScreen) updateScreenState(a string) {
// 	p.mu.Lock()
// 	p.State = a
// 	p.mu.Unlock()
// }

// getScreenState returns the current screen state
func (p *FyneScreen) getScreenState() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.State
}
