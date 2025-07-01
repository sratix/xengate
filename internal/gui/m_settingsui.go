//go:build !(android || ios)
// +build !android,!ios

package gui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const systemDefault = "System Default"

func settingsWindow(a *App) fyne.CanvasObject {
	themeText := widget.NewLabel(lang.L("Theme"))
	dropdownTheme := widget.NewSelect([]string{lang.L(systemDefault), lang.L("Light"), lang.L("Dark")}, parseTheme)

	languageText := widget.NewLabel(lang.L("Language"))
	dropdownLanguage := widget.NewSelect([]string{lang.L(systemDefault), "English", "Persian"}, parseLanguage(a))
	selectedLanguage := fyne.CurrentApp().Preferences().StringWithFallback("Language", systemDefault)

	if selectedLanguage == systemDefault {
		selectedLanguage = lang.L(systemDefault)
	}

	dropdownLanguage.PlaceHolder = selectedLanguage

	themeName := lang.L(fyne.CurrentApp().Preferences().StringWithFallback("Theme", systemDefault))
	dropdownTheme.PlaceHolder = themeName
	parseTheme(themeName)

	a.systemTheme = fyne.CurrentApp().Settings().ThemeVariant()

	dropdownTheme.Refresh()

	return container.New(layout.NewFormLayout(), themeText, dropdownTheme, languageText, dropdownLanguage)
}

func parseTheme(t string) {
	go func() {
		time.Sleep(10 * time.Millisecond)
		switch t {
		case lang.L("Light"):
			fyne.CurrentApp().Settings().SetTheme(go2tvTheme{"Light"})
			fyne.CurrentApp().Preferences().SetString("Theme", "Light")
		case lang.L("Dark"):
			fyne.CurrentApp().Settings().SetTheme(go2tvTheme{"Dark"})
			fyne.CurrentApp().Preferences().SetString("Theme", "Dark")
		default:
			fyne.CurrentApp().Settings().SetTheme(go2tvTheme{systemDefault})
			fyne.CurrentApp().Preferences().SetString("Theme", systemDefault)
		}
	}()
}

func parseLanguage(a *App) func(string) {
	return func(t string) {
		if t != fyne.CurrentApp().Preferences().StringWithFallback("Language", systemDefault) {
			dialog.ShowInformation(lang.L("Update Language Preferences"), lang.L(`Please restart the application for the changes to take effect.`), a.topWindow)
		}
		go func() {
			switch t {
			case "English":
				fyne.CurrentApp().Preferences().SetString("Language", "English")
			case "Persian":
				fyne.CurrentApp().Preferences().SetString("Language", "Persian")
			default:
				fyne.CurrentApp().Preferences().SetString("Language", systemDefault)
			}
		}()
	}
}
