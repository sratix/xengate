package ui

import (
	"fmt"

	"xengate/backend"
	"xengate/internal/models"
	"xengate/res"
	"xengate/ui/components"
	"xengate/ui/dialogs"
	"xengate/ui/layouts"
	"xengate/ui/util"

	myTheme "xengate/ui/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type MainWindow struct {
	Window fyne.Window

	App *backend.App

	// systrayWin     fyne.Window
	theme          *myTheme.MyTheme
	haveSystemTray bool

	// content *mainWindowContent

	// Top toolbar row
	topRow    *fyne.Container
	bottomRow *fyne.Container

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

	ReloadFunc      func()
	RefreshPageFunc func()

	escapablePopUp   fyne.CanvasObject
	haveModal        bool
	runOnModalClosed func()

	connectionList *components.ConnectionList

	fyneApp fyne.App
}

func NewMainWindow(fyneApp fyne.App, appName, displayAppName, appVersion string, app *backend.App) MainWindow {
	m := MainWindow{
		App:     app,
		Window:  fyneApp.NewWindow(displayAppName),
		theme:   myTheme.NewMyTheme(&app.Config.Theme, app.ThemesDir()),
		fyneApp: fyneApp,
	}

	m.theme.NormalFont = app.Config.Application.FontNormalTTF
	m.theme.BoldFont = app.Config.Application.FontBoldTTF
	fyneApp.Settings().SetTheme(m.theme)

	if app.Config.Application.EnableSystemTray {
		m.SetupSystemTrayMenu(displayAppName, fyneApp)
	}

	m.initUI()

	m.setInitialSize()

	m.Window.SetCloseIntercept(func() {
		m.SaveWindowSize()
		if app.Config.Application.CloseToSystemTray && m.HaveSystemTray() {
			m.sendNotification(appName, "Application minimized to system tray")
			m.Window.Hide()
		} else {
			m.Window.Close()
		}
	})

	return m
}

func (m *MainWindow) initUI() {
	m.initToolbar()

	m.connectionList = components.NewConnectionList(m.fyneApp, m.Window)

	// // Add sample connections
	// m.connectionList.AddConnection(&components.Connection{
	// 	Name:    "migrate - sratix",
	// 	Address: "78.129.220.121",
	// 	Port:    "22",
	// 	Type:    "SSH",
	// 	Status:  components.StatusActive,
	// })

	// m.connectionList.AddConnection(&components.Connection{
	// 	Name:    "migrate - sratix",
	// 	Address: "192.227.134.68",
	// 	Port:    "22",
	// 	Type:    "SSH + NONE",
	// 	Status:  components.StatusInactive,
	// })

	// Add handlers
	m.connectionList.SetOnShare(func(conn *models.Connection) {
		dialog.ShowInformation("Share",
			fmt.Sprintf("Sharing connection: %s", conn.Name),
			m.Window)
	})

	m.connectionList.SetOnDelete(func(conn *models.Connection) {
		dialog.ShowConfirm("Delete Connection",
			"Are you sure you want to delete this connection?",
			func(confirm bool) {
				if confirm {
					m.connectionList.RemoveConnection(conn)
				}
			},
			m.Window,
		)
	})

	content := container.NewPadded(
		container.NewVScroll(m.connectionList),
	)

	m.topRow = container.NewStack(m.toolBar)
	// content := container.NewStack(newFixedSpacer(fyne.NewSize(640, 480)))
	m.bottomRow = container.NewStack(layouts.NewFixedSpacer(fyne.NewSize(640, 50)))
	m.Window.SetContent(container.NewBorder(m.topRow, m.bottomRow, nil, nil, content))
}

func (m *MainWindow) DesiredSize() fyne.Size {
	w := float32(m.App.Config.Application.WindowWidth)
	if w <= 1 {
		w = 1000
	}
	h := float32(m.App.Config.Application.WindowHeight)
	if h <= 1 {
		h = 800
	}
	return fyne.NewSize(w, h)
}

func (m *MainWindow) setInitialSize() {
	m.Window.Resize(m.DesiredSize())
}

func (m *MainWindow) initToolbar() {
	m.actMenu = widget.NewToolbarAction(theme.MenuIcon(), func() {
	})
	m.actAdd = widget.NewToolbarAction(theme.ContentAddIcon(), func() {
		newConn := &models.Connection{
			Name:   "New Connection",
			Type:   "SSH",
			Status: models.StatusInactive,
		}
		components.ShowEditDialog("Add New Connection", m.Window, newConn, m.connectionList, m.fyneApp)
	})
	m.actToggleFullScreen = widget.NewToolbarAction(theme.ViewFullScreenIcon(), m.toggleFullScreen)
	m.actSettings = widget.NewToolbarAction(theme.SettingsIcon(), func() {
		dlg := dialogs.NewSettingsDialog(m.App.Config, m.theme.ListThemeFiles(), m.Window)

		dlg.OnThemeSettingChanged = func() {
			fyne.CurrentApp().Settings().SetTheme(m.theme)
		}
		// dlg.OnPageNeedsRefresh = c.RefreshPageFunc
		pop := widget.NewModalPopUp(dlg, m.Canvas())
		dlg.OnDismiss = func() {
			pop.Hide()
			m.doModalClosed()
			m.App.SaveConfigFile()
		}
		m.closePopUpOnEscape(pop)
		m.haveModal = true
		pop.Show()
	})
	m.actAbout = widget.NewToolbarAction(theme.InfoIcon(), func() {
		dlg := dialogs.NewAboutDialog("")

		pop := widget.NewModalPopUp(dlg, m.Canvas())
		dlg.OnDismiss = func() {
			pop.Hide()
			m.doModalClosed()
		}
		m.closePopUpOnEscape(pop)
		m.haveModal = true
		pop.Show()
	})

	m.toolBar = widget.NewToolbar()
	m.toolBar.Items = []widget.ToolbarItem{}
	m.toolBar.Append(m.actMenu)
	m.toolBar.Append(m.actAdd)
	m.toolBar.Append(widget.NewToolbarSeparator())
	m.toolBar.Append(widget.NewToolbarSpacer())
	m.toolBar.Append(widget.NewToolbarSeparator())
	m.toolBar.Append(m.actToggleFullScreen)
	m.toolBar.Append(widget.NewToolbarSeparator())
	m.toolBar.Append(m.actSettings)
	m.toolBar.Append(m.actAbout)
}

func (m *MainWindow) closePopUpOnEscape(pop fyne.CanvasObject) {
	m.escapablePopUp = pop
}

func (m *MainWindow) HaveModal() bool {
	return m.haveModal
}

func (m *MainWindow) doModalClosed() {
	m.haveModal = false
	if m.runOnModalClosed != nil {
		m.runOnModalClosed()
		m.runOnModalClosed = nil
	}
}

func (m *MainWindow) sendNotification(title, content string) {
	fyne.CurrentApp().SendNotification(&fyne.Notification{
		Title:   title,
		Content: content,
	})
}

func (m *MainWindow) SetupSystemTrayMenu(appName string, fyneApp fyne.App) {
	if desk, ok := fyneApp.(desktop.App); ok {
		menu := fyne.NewMenu(appName,
			fyne.NewMenuItem(lang.L("Show"), m.Window.Show),
			fyne.NewMenuItem(lang.L("Hide"), m.Window.Hide),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem(lang.L("Quit"), func() {
				m.Quit()
			}),
		)
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(res.ResAppicon256Png)
		m.haveSystemTray = true
	}
}

func (m *MainWindow) HaveSystemTray() bool {
	return m.haveSystemTray
}

func (m *MainWindow) toggleFullScreen() {
	if m.Window.FullScreen() {
		m.Window.SetFullScreen(false)
		m.actToggleFullScreen.SetIcon(theme.ViewFullScreenIcon())
	} else {
		m.Window.SetFullScreen(true)
		m.actToggleFullScreen.SetIcon(theme.ViewRestoreIcon())
	}
	m.toolBar.Refresh()
}

func (m *MainWindow) CenterOnScreen() {
	m.Window.CenterOnScreen()
}

func (m *MainWindow) SetMaster() {
	m.Window.SetMaster()
}

func (m *MainWindow) Show() {
	m.Window.Show()
}

func (m *MainWindow) ShowAndRun() {
	m.Window.ShowAndRun()
}

func (m *MainWindow) Canvas() fyne.Canvas {
	return m.Window.Canvas()
}

func (m *MainWindow) SetTitle(title string) {
	m.Window.SetTitle(title)
}

func (m *MainWindow) SetContent(c fyne.CanvasObject) {
	m.Window.SetContent(c)
}

func (m *MainWindow) Quit() {
	m.SaveWindowSize()
	fyne.CurrentApp().Quit()
}

func (m *MainWindow) SaveWindowSize() {
	util.SaveWindowSize(m.Window,
		&m.App.Config.Application.WindowWidth,
		&m.App.Config.Application.WindowHeight)
}
