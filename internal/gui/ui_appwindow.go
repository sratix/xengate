//go:build !(android || ios)
// +build !android,!ios

package gui

import (
	"fyne.io/fyne/theme"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
)

func (w *AppWindow) setupMainWindow() {
	tabs := container.NewAppTabs(
		container.NewTabItem("SSH-Proxy", mainWindow(w)),
		container.NewTabItem(lang.L("Settings"), settingsWindow(w)),
		container.NewTabItem(lang.L("About"), aboutWindow(w)),
	)

	w.Hotkeys = true
	tabs.OnSelected = func(t *container.TabItem) {
		w.Hotkeys = t.Text != w.window.Title()
	}

	w.tabs = tabs

	w.window.SetContent(tabs)
	w.window.Resize(fyne.NewSize(w.window.Canvas().Size().Width, w.window.Canvas().Size().Height*1.2))
	w.window.CenterOnScreen()
	w.window.SetMaster()
	w.window.SetCloseIntercept(func() {
		w.window.Hide()
		w.app.SendNotification(fyne.NewNotification("App Minimized", "Running in system tray"))
	})
}

func (w *AppWindow) setupSystray() {
	if desk, ok := w.app.(desktop.App); ok {
		menu := fyne.NewMenu(w.window.Title(),
			fyne.NewMenuItem("Show Window", func() {
				w.showMain()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				w.quit()
			}),
		)
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(theme.FyneLogo())
	}
}

func (w *AppWindow) showMain() {
	w.window.Show()
	w.isHidden = false
}

func (w *AppWindow) quit() {
	w.app.Quit()
}
