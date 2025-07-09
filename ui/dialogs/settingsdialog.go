package dialogs

import (
	"errors"
	"os"
	"slices"
	"strings"

	"xengate/backend"
	"xengate/res"
	"xengate/ui/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	myTheme "xengate/ui/theme"
)

type SettingsDialog struct {
	widget.BaseWidget

	OnThemeSettingChanged func()
	OnDismiss             func()
	OnPageNeedsRefresh    func()

	config     *backend.Config
	themeFiles map[string]string // filename -> displayName
	promptText *widget.RichText

	clientDecidesScrobble bool

	content fyne.CanvasObject
}

// TODO: having this depend on the mpv package for the AudioDevice type is kinda gross. Refactor.
func NewSettingsDialog(
	config *backend.Config,
	themeFileList map[string]string,
	window fyne.Window,
) *SettingsDialog {
	s := &SettingsDialog{config: config, themeFiles: themeFileList}
	s.ExtendBaseWidget(s)

	// TODO: It may be a nicer UX to always create the equalizer tab,
	// but disable it if we are not using an equalizer player
	var tabs *container.AppTabs

	tabs = container.NewAppTabs(
		s.createGeneralTab(),
		s.createAppearanceTab(window),
		s.createAdvancedTab(),
	)

	tabs.SelectIndex(s.getActiveTabNumFromConfig())
	tabs.OnSelected = func(ti *container.TabItem) {
		s.saveSelectedTab(tabs.SelectedIndex())
	}
	s.promptText = widget.NewRichTextWithText("")
	s.content = container.NewVBox(tabs, widget.NewSeparator(),
		container.NewHBox(s.promptText, layout.NewSpacer(), widget.NewButton(lang.L("Close"), func() {
			if s.OnDismiss != nil {
				s.OnDismiss()
			}
		})))

	return s
}

func (s *SettingsDialog) createGeneralTab() *container.TabItem {
	// pages := util.LocalizeSlice(backend.SupportedStartupPages)
	// var startupPage *widget.Select
	// startupPage = widget.NewSelect(pages, func(_ string) {
	// 	s.config.Application.StartupPage = backend.SupportedStartupPages[startupPage.SelectedIndex()]
	// })
	initialIdx := slices.Index(backend.SupportedStartupPages, s.config.Application.StartupPage)
	if initialIdx < 0 {
		initialIdx = 0
	}
	// startupPage.SetSelectedIndex(initialIdx)
	// if startupPage.Selected == "" {
	// 	startupPage.SetSelectedIndex(0)
	// }

	languageList := make([]string, len(res.TranslationsInfo)+1)
	languageList[0] = lang.L("Auto")
	var langSelIndex int
	for i, tr := range res.TranslationsInfo {
		languageList[i+1] = tr.DisplayName
		if tr.Name == s.config.Application.Language {
			langSelIndex = i + 1
		}
	}

	languageSelect := widget.NewSelect(languageList, nil)
	languageSelect.SetSelectedIndex(langSelIndex)
	languageSelect.OnChanged = func(_ string) {
		lang := "auto"
		if i := languageSelect.SelectedIndex(); i > 0 {
			lang = res.TranslationsInfo[i-1].Name
		}
		s.config.Application.Language = lang
		s.setRestartRequired()
	}

	closeToTray := widget.NewCheckWithData(lang.L("Close to system tray"),
		binding.BindBool(&s.config.Application.CloseToSystemTray))
	if !s.config.Application.EnableSystemTray {
		closeToTray.Disable()
	}
	systemTrayEnable := widget.NewCheck(lang.L("Enable system tray"), func(val bool) {
		s.config.Application.EnableSystemTray = val
		// TODO: see https://github.com/fyne-io/fyne/issues/3788
		// Once Fyne supports removing/hiding an existing system tray menu,
		// the restart required prompt can be removed and this dialog
		// can expose a callback for the Controller to show/hide the system tray menu.
		s.setRestartRequired()
		if val {
			closeToTray.Enable()
		} else {
			closeToTray.Disable()
		}
	})
	systemTrayEnable.Checked = s.config.Application.EnableSystemTray

	return container.NewTabItem(lang.L("General"), container.NewVBox(
		util.NewHSpace(0), // insert a theme.Padding amount of space at top
		container.NewHBox(widget.NewLabel(lang.L("Language")), languageSelect),
		container.NewHBox(
		// widget.NewLabel(lang.L("Startup page")), container.NewGridWithColumns(2, startupPage),
		),
		container.NewHBox(systemTrayEnable, closeToTray),
		// saveQueueHBox,
		// trackNotif,
		// albumGridYears,
		s.newSectionSeparator(),

		// widget.NewRichText(&widget.TextSegment{Text: "Scrobbling", Style: util.BoldRichTextStyle}),
		// // scrobbleEnabled,
		// container.NewHBox(
		// 	widget.NewLabel(lang.L("Scrobble when")),
		// 	// percentEntry,
		// 	widget.NewLabel(lang.L("percent of track is played")),
		// ),
		// container.NewHBox(
		// 	// durationEnabled,
		// 	// durationEntry,
		// 	widget.NewLabel(lang.L("minutes of track have been played")),
		// ),
	))
}

func (s *SettingsDialog) createAppearanceTab(window fyne.Window) *container.TabItem {
	themeNames := []string{"Default"}
	themeFileNames := []string{""}
	i, selIndex := 1, 0
	for filename, displayname := range s.themeFiles {
		themeFileNames = append(themeFileNames, filename)
		themeNames = append(themeNames, displayname)
		if strings.EqualFold(filename, s.config.Theme.ThemeFile) {
			selIndex = i
		}
		i++
	}

	themeFileSelect := widget.NewSelect(themeNames, nil)
	themeFileSelect.SetSelectedIndex(selIndex)
	themeFileSelect.OnChanged = func(_ string) {
		s.config.Theme.ThemeFile = themeFileNames[themeFileSelect.SelectedIndex()]
		if s.OnThemeSettingChanged != nil {
			s.OnThemeSettingChanged()
		}
	}
	themeModeSelect := widget.NewSelect([]string{
		string(myTheme.AppearanceDark),
		string(myTheme.AppearanceLight),
		string(myTheme.AppearanceAuto),
	}, nil)
	themeModeSelect.OnChanged = func(_ string) {
		s.config.Theme.Appearance = themeModeSelect.Options[themeModeSelect.SelectedIndex()]
		if s.OnThemeSettingChanged != nil {
			s.OnThemeSettingChanged()
		}
	}
	themeModeSelect.SetSelected(s.config.Theme.Appearance)
	if themeModeSelect.Selected == "" {
		themeModeSelect.SetSelectedIndex(0)
	}

	normalFontEntry := widget.NewEntry()
	normalFontEntry.SetPlaceHolder("path to .ttf or empty to use default")
	normalFontEntry.Text = s.config.Application.FontNormalTTF
	normalFontEntry.Validator = s.ttfPathValidator
	normalFontEntry.OnChanged = func(path string) {
		if normalFontEntry.Validate() == nil {
			s.setRestartRequired()
			s.config.Application.FontNormalTTF = path
		}
	}
	normalFontBrowse := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		s.doChooseTTFFile(window, normalFontEntry)
	})

	boldFontEntry := widget.NewEntry()
	boldFontEntry.SetPlaceHolder("path to .ttf or empty to use default")
	boldFontEntry.Text = s.config.Application.FontBoldTTF
	boldFontEntry.Validator = s.ttfPathValidator
	boldFontEntry.OnChanged = func(path string) {
		if boldFontEntry.Validate() == nil {
			s.setRestartRequired()
			s.config.Application.FontBoldTTF = path
		}
	}
	boldFontBrowse := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		s.doChooseTTFFile(window, boldFontEntry)
	})

	uiScaleRadio := widget.NewRadioGroup([]string{lang.L("Smaller"), lang.L("Normal"), lang.L("Larger")}, func(choice string) {
		s.config.Application.UIScaleSize = choice
		s.setRestartRequired()
	})
	uiScaleRadio.Required = true
	uiScaleRadio.Horizontal = true
	if s.config.Application.UIScaleSize == "Smaller" || s.config.Application.UIScaleSize == "Larger" {
		uiScaleRadio.Selected = s.config.Application.UIScaleSize
	} else {
		uiScaleRadio.Selected = "Normal"
	}

	disableDPI := widget.NewCheck(lang.L("Disable automatic DPI adjustment"), func(b bool) {
		s.config.Application.DisableDPIDetection = b
		s.setRestartRequired()
	})
	disableDPI.Checked = s.config.Application.DisableDPIDetection

	// gridCardSize := widget.NewSlider(150, 350)
	// gridCardSize.SetValue(float64(s.config.GridView.CardSize))
	// gridCardSize.Step = 10
	// gridCardSize.OnChanged = func(f float64) {
	// 	s.config.GridView.CardSize = float32(f)
	// 	if s.OnPageNeedsRefresh != nil {
	// 		s.OnPageNeedsRefresh()
	// 	}
	// }

	return container.NewTabItem(lang.L("Appearance"), container.NewVBox(
		util.NewHSpace(0), // insert a theme.Padding amount of space at top
		container.NewBorder(nil, nil, widget.NewLabel(lang.L("Theme")), /*left*/
			container.NewHBox(widget.NewLabel(lang.L("Mode")), themeModeSelect, util.NewHSpace(5)), // right
			themeFileSelect, // center
		),
		widget.NewRichText(&widget.TextSegment{Text: lang.L("UI Scaling"), Style: util.BoldRichTextStyle}),
		uiScaleRadio,
		// container.NewBorder(nil, nil, widget.NewLabel(lang.L("Grid card size")), nil, gridCardSize),
		disableDPI,
		s.newSectionSeparator(),
		widget.NewRichText(&widget.TextSegment{Text: lang.L("Application font"), Style: util.BoldRichTextStyle}),
		container.New(layout.NewFormLayout(),
			widget.NewLabel(lang.L("Normal font")), container.NewBorder(nil, nil, nil, normalFontBrowse, normalFontEntry),
			widget.NewLabel(lang.L("Bold font")), container.NewBorder(nil, nil, nil, boldFontBrowse, boldFontEntry),
		),
	))
}

func (s *SettingsDialog) createAdvancedTab() *container.TabItem {
	multi := widget.NewCheckWithData(lang.L("Allow multiple app instances"), binding.BindBool(&s.config.Application.AllowMultiInstance))
	return container.NewTabItem(lang.L("Advanced"), container.NewVBox(
		multi,
	))
}

func (s *SettingsDialog) doChooseTTFFile(window fyne.Window, entry *widget.Entry) {
	callback := func(urirc fyne.URIReadCloser, err error) {
		if err == nil && urirc != nil {
			entry.SetText(urirc.URI().Path())
		}
	}
	dlg := dialog.NewFileOpen(callback, window)
	dlg.SetFilter(&storage.ExtensionFileFilter{Extensions: []string{".ttf"}})
	dlg.Show()
}

func (s *SettingsDialog) ttfPathValidator(path string) error {
	if path == "" {
		return nil
	}
	if !strings.HasSuffix(path, ".ttf") {
		return errors.New("only .ttf fonts supported")
	}
	_, err := os.Stat(path)
	return err
}

func (s *SettingsDialog) setRestartRequired() {
	ts := s.promptText.Segments[0].(*widget.TextSegment)
	if ts.Text != "" {
		return
	}
	ts.Text = lang.L("Restart required")
	ts.Style.ColorName = theme.ColorNameError
	s.promptText.Refresh()
}

func (s *SettingsDialog) newSectionSeparator() fyne.CanvasObject {
	return container.New(&layout.CustomPaddedLayout{LeftPadding: 15, RightPadding: 15}, widget.NewSeparator())
}

func (s *SettingsDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.content)
}

func (s *SettingsDialog) saveSelectedTab(tabNum int) {
	var tabName string
	switch tabNum {
	case 0:
		tabName = "General"
	case 1:
		tabName = "Playback"
	case 2:
		tabName = "Equalizer"
	case 3:
		tabName = "Experimental"
	}
	s.config.Application.SettingsTab = tabName
}

func (s *SettingsDialog) getActiveTabNumFromConfig() int {
	switch s.config.Application.SettingsTab {
	case "Playback":
		return 1
	case "Equalizer":
		return 2
	case "Experimental":
		return 3
	default:
		return 0
	}
}
