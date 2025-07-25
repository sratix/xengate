package ui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"xengate/backend"
	"xengate/internal/models"
	"xengate/internal/proxy"
	"xengate/internal/tunnel"
	"xengate/res"

	"xengate/ui/components"
	"xengate/ui/dialogs"
	"xengate/ui/util"

	myTheme "xengate/ui/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	log "github.com/sirupsen/logrus"
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

	Man *tunnel.Manager

	wg sync.WaitGroup

	proxyErrCh  chan error
	proxyCtx    context.Context
	proxyCancel context.CancelFunc

	timerPanel *components.TimerPanel
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

	m.Man = tunnel.NewManager()

	socksServer, _ := proxy.NewProxy("socks5", "127.0.0.1", 1080, m.Man)
	httpServer, _ := proxy.NewProxy("http", "127.0.0.1", 1090, m.Man)
	// if err != nil {
	// 	// fmt.Errorf("failed to create proxy: %w", err)
	// }

	// Start proxy in background
	m.proxyErrCh = make(chan error, 1)
	m.proxyCtx, m.proxyCancel = context.WithCancel(context.Background())
	// defer proxyCancel()

	// m.wg.Add(1)
	// go func() {
	// defer m.wg.Done()
	if err := socksServer.Start(m.proxyCtx); err != nil {
		//	m.proxyErrCh <- err
	}
	if err := httpServer.Start(m.proxyCtx); err != nil {
		//	m.proxyErrCh <- err
	}
	//}()

	m.addShortcuts()

	// go m.statsReporter(m.proxyCtx, 10*time.Second)

	return m
}

func (m *MainWindow) statsReporter(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := m.Man.GetStats()

			for serverName, poolStats := range stats {
				for _, conn := range m.connectionList.GetConnections() {
					if conn.ID == poolStats.ID {
						// آپدیت آمار در ساختار Connection
						conn.Stats = &models.Stats{
							ServerName:    serverName,
							TotalTunnels:  poolStats.TotalTunnels,
							TotalRequests: poolStats.TotalRequests,
							TotalBytes:    poolStats.TotalBytes,
							Active:        poolStats.ActiveConnections,
							Connected:     poolStats.Connected,
						}

						// fmt.Printf("XXXXX:   %+v\n", conn.Stats)

						// آپدیت مستقیم UI
						m.connectionList.UpdateStats(conn)

						m.connectionList.Refresh()

						// m.connectionList.RefreshStats(conn.ID, conn.Stats)

						break
					}
				}
			}
		}
	}
}

// func (m *MainWindow) statsReporter(ctx context.Context, interval time.Duration) {
// 	ticker := time.NewTicker(interval)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case <-ticker.C:
// 			stats := m.Man.GetStats()
// 			m.connectionList.BatchUpdateStats(stats)
// 		}
// 	}
// }

func (m *MainWindow) initUI() {
	m.initToolbar()

	m.connectionList = components.NewConnectionList(m.fyneApp, m.Window)

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

	m.connectionList.SetOnRun(func(conn *models.Connection) {
		switch conn.Status {
		case models.StatusActive:
			m.Man.Start(context.Background(), conn)
		case models.StatusInactive:
			m.Man.Stop(conn.Name)
		}
	})

	// Create basic timer panel
	m.timerPanel = components.NewTimerPanel()

	m.timerPanel.SetOnClick(func(status bool) {
		for _, c := range m.connectionList.GetConnections() {
			if status {
				c.Status = models.StatusActive
				m.Man.Start(context.Background(), c)
			} else {
				c.Status = models.StatusInactive
				m.Man.Stop(c.Name)
			}
			m.connectionList.Refresh()
		}
	})

	// logHandler := components.NewLogHandler(1000) // نگهداری 1000 خط آخر

	// // تنظیم logrus
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(log.DebugLevel)

	tabs := container.NewAppTabs(
		container.NewTabItem("Connections", container.NewVScroll(m.connectionList)),
		// container.NewTabItem("Log", logHandler.GetContainer()), // components.NewLogWidget(1000)),
		// container.NewTabItem("Statistics", getStatsContent()),
	)

	m.Window.SetContent(container.NewBorder(m.toolBar, container.NewBorder(nil, nil, container.NewHBox(container.NewPadded(container.NewCenter(widget.NewLabel("SOCKS5: 127.0.0.1:1080 \nHTTP    : 127.0.0.1:1090\t")))), container.NewPadded(m.timerPanel), nil), nil, nil, container.NewPadded(tabs)))
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
			Port:   "22",
			Status: models.StatusInactive,
		}

		dlg := dialogs.NewEditDialog(
			newConn,
			m.connectionList.GetConfigManager(),
			m.Window,
		)

		pop := widget.NewModalPopUp(dlg, m.Canvas())
		dlg.OnDismiss = func() {
			pop.Hide()
			m.doModalClosed()
			m.connectionList.LoadConnections()
			m.connectionList.Refresh()
			m.App.SaveConfigFile()
		}
		m.closePopUpOnEscape(pop)
		m.haveModal = true
		pop.Show()
	})
	m.actToggleFullScreen = widget.NewToolbarAction(theme.ViewFullScreenIcon(), m.toggleFullScreen)
	m.actSettings = widget.NewToolbarAction(theme.SettingsIcon(), func() {
		dlg := dialogs.NewSettingsDialog(m.App.Config, m.theme.ListThemeFiles(), m.Window)

		dlg.OnThemeSettingChanged = func() {
			fyne.CurrentApp().Settings().SetTheme(m.theme)
		}
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

	m.Man.StopAll()
	m.proxyCancel()

	if m.HaveSystemTray() {
		m.sendNotification("Xengate", "Application is quitting")
		m.Window.Hide()
		return
	}
	if m.HaveModal() {
		dialog.ShowInformation("Quit", "Please close the modal dialog before quitting.", m.Window)
		return
	}

	m.App.SaveConfigFile()

	fyne.CurrentApp().Quit()
}

func (m *MainWindow) SaveWindowSize() {
	util.SaveWindowSize(m.Window,
		&m.App.Config.Application.WindowWidth,
		&m.App.Config.Application.WindowHeight)
}

func (m *MainWindow) addShortcuts() {
	m.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		switch e.Name {
		case fyne.KeyEscape:
			m.CloseEscapablePopUp()
		}
	})
}

func (m *MainWindow) CloseEscapablePopUp() {
	if m.escapablePopUp != nil {
		m.escapablePopUp.Hide()
		m.escapablePopUp = nil
		m.doModalClosed()
	}
}
