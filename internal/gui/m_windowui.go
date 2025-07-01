//go:build !(android || ios)
// +build !android,!ios

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (a *App) setupMainWindow() {
	// container := container.NewVBox(
	// 	mainWindow(a),
	// )

	a.newToolbar()

	// a.topWindow.SetContent(container)
	// a.topWindow.Resize(fyne.NewSize(a.topWindow.Canvas().Size().Width, a.topWindow.Canvas().Size().Height*1.2))
	// a.topWindow.CenterOnScreen()
	// a.topWindow.SetMaster()

	a.topRow = container.NewStack(a.toolBar)
	content := container.NewStack(newFixedSpacer(fyne.NewSize(640, 360)))
	a.topWindow.SetContent(container.NewBorder(a.topRow, nil, nil, nil, content))

	// a.topWindow.SetCloseIntercept(func() {
	// 	a.topWindow.Hide()
	// 	a.SendNotification(fyne.NewNotification("App Minimized", "Running in system tray"))
	// })
}

func (a *App) newToolbar() {
	a.actMenu = widget.NewToolbarAction(theme.MenuIcon(), func() {
	})
	a.actAdd = widget.NewToolbarAction(theme.ContentAddIcon(), func() {
	})
	a.actToggleFullScreen = widget.NewToolbarAction(theme.ViewFullScreenIcon(), a.toggleFullScreen)
	a.actSettings = widget.NewToolbarAction(theme.SettingsIcon(), func() {
	})
	a.actAbout = widget.NewToolbarAction(theme.InfoIcon(), a.aboutDialog)

	a.toolBar = widget.NewToolbar()
	a.toolBar.Items = []widget.ToolbarItem{}
	a.toolBar.Append(a.actMenu)
	a.toolBar.Append(a.actAdd)
	a.toolBar.Append(widget.NewToolbarSeparator())
	a.toolBar.Append(widget.NewToolbarSpacer())
	a.toolBar.Append(widget.NewToolbarSeparator())
	a.toolBar.Append(a.actToggleFullScreen)
	a.toolBar.Append(widget.NewToolbarSeparator())
	a.toolBar.Append(a.actSettings)
	a.toolBar.Append(a.actAbout)
}

func (a *App) setupSystray() {
	if desk, ok := a.App.(desktop.App); ok {
		menu := fyne.NewMenu(a.topWindow.Title(),
			fyne.NewMenuItem("Show Window", func() {
				a.showMain()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				a.quit()
			}),
		)
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(theme.FyneLogo())
		// iconResource := fyne.NewStaticResource("icon", go2TVIcon512)
		// desk.SetSystemTrayIcon(iconResource)
	}
}

func (a *App) showMain() {
	a.topWindow.Show()
	a.isHidden = false
}

func (a *App) quit() {
	a.Quit()
}

func (a *App) toggleFullScreen() {
	if a.topWindow.FullScreen() {
		a.topWindow.SetFullScreen(false)
		a.actToggleFullScreen.SetIcon(theme.ViewFullScreenIcon())
	} else {
		a.topWindow.SetFullScreen(true)
		a.actToggleFullScreen.SetIcon(theme.ViewRestoreIcon())
	}
	a.toolBar.Refresh()
}
