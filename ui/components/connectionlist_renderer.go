package components

import (
	"fmt"
	"image/color"

	"xengate/internal/models"
	"xengate/ui/dialogs"
	"xengate/ui/layouts"
	myTheme "xengate/ui/theme"
	"xengate/ui/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type connectionListRenderer struct {
	list         *ConnectionList
	container    *fyne.Container
	objects      []fyne.CanvasObject
	statusLabels map[string]*widget.Label
}

func newConnectionListRenderer(list *ConnectionList) *connectionListRenderer {
	renderer := &connectionListRenderer{
		list:         list,
		statusLabels: make(map[string]*widget.Label),
	}
	renderer.rebuild()
	return renderer
}

func (r *connectionListRenderer) createConnectionItem(conn *models.Connection) fyne.CanvasObject {
	if conn == nil {
		return container.NewHBox()
	}

	if conn.Stats == nil {
		conn.Stats = &models.Stats{
			Connected:    0,
			TotalTunnels: 0,
			TotalBytes:   0,
		}
	}

	itemHeight := float32(60)

	statusColor := color.NRGBA{R: 117, G: 117, B: 117, A: 255}
	if conn.Status == models.StatusActive {
		statusColor = color.NRGBA{R: 89, G: 205, B: 144, A: 255}
	}

	mainBg := myTheme.NewThemedRectangle(myTheme.ColorNamePageBackground)
	leftBorder := canvas.NewRectangle(statusColor)
	leftBorder.SetMinSize(fyne.NewSize(6, itemHeight))

	nameLabel := widget.NewLabel(conn.Name)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	addressLabel := widget.NewLabel(fmt.Sprintf("%s:%s", conn.Address, conn.Port))
	addressLabel.TextStyle = fyne.TextStyle{Monospace: true}

	var proxyLabel fyne.CanvasObject
	if conn.Config != nil {
		proxyLabel = NewBadge(conn.Config.Mode, fyne.CurrentApp().Settings().Theme().Color(myTheme.ColorNameTextMuted, fyne.CurrentApp().Settings().ThemeVariant()))
	} else {
		proxyLabel = NewBadge("unknown", fyne.CurrentApp().Settings().Theme().Color(myTheme.ColorNameTextMuted, fyne.CurrentApp().Settings().ThemeVariant()))
	}

	// ایجاد لیبل‌های آمار
	tunnelIcon := widget.NewIcon(theme.ComputerIcon())
	tunnelStats := widget.NewLabel(fmt.Sprintf("%d/%d", conn.Stats.Connected, conn.Stats.TotalTunnels))
	tunnelStats.TextStyle = fyne.TextStyle{Monospace: true}

	uploadIcon := widget.NewIcon(theme.UploadIcon())
	downloadIcon := widget.NewIcon(theme.DownloadIcon())
	trafficStats := widget.NewLabel(util.BytesToSizeString(conn.Stats.TotalBytes))
	trafficStats.TextStyle = fyne.TextStyle{Monospace: true}

	// ذخیره لیبل‌ها برای بروزرسانی بعدی
	r.statusLabels[conn.ID+"_tunnels"] = tunnelStats
	r.statusLabels[conn.ID+"_traffic"] = trafficStats

	shareBtn := widget.NewButtonWithIcon("", myTheme.ShareIcon, func() {
		if r.list != nil && r.list.onShare != nil {
			r.list.onShare(conn)
		}
	})
	shareBtn.Importance = widget.LowImportance

	editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
		if r.list != nil && r.list.Window != nil && r.list.configManager != nil {
			dlg := dialogs.NewEditDialog(conn, r.list.configManager, r.list.Window)
			pop := widget.NewModalPopUp(dlg, r.list.Window.Canvas())
			dlg.OnDismiss = func() {
				pop.Hide()
				r.list.LoadConnections()
				r.list.Refresh()
			}
			pop.Show()
		}
	})
	editBtn.Importance = widget.LowImportance

	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if r.list != nil && r.list.onDelete != nil {
			r.list.onDelete(conn)
		}
	})
	deleteBtn.Importance = widget.LowImportance

	var runBtn *widget.Button
	icn := theme.MediaPlayIcon()
	if conn.Status == models.StatusActive {
		icn = theme.MediaStopIcon()
	}

	statsContainer := container.NewHBox(
		container.NewHBox(
			tunnelIcon,
			tunnelStats,
		),
		layout.NewSpacer(),
		container.NewHBox(
			uploadIcon,
			downloadIcon,
			trafficStats,
		),
	)

	runBtn = widget.NewButtonWithIcon("", icn, func() {
		if r.list != nil && r.list.onRun != nil {
			if conn.Status == models.StatusInactive {
				runBtn.SetIcon(theme.MediaStopIcon())
				conn.Status = models.StatusActive
				leftBorder.FillColor = color.NRGBA{R: 89, G: 205, B: 144, A: 255}
			} else {
				runBtn.SetIcon(theme.MediaPlayIcon())
				conn.Status = models.StatusInactive
				leftBorder.FillColor = color.NRGBA{R: 117, G: 117, B: 117, A: 255}
			}

			// فراخوانی callback اصلی
			r.list.onRun(conn)

			// ذخیره تغییرات در کانفیگ
			if r.list.configManager != nil {
				appConfig := r.list.configManager.LoadConfig()
				for i, c := range appConfig.Connections {
					if c.ID == conn.ID {
						appConfig.Connections[i].Status = conn.Status
						appConfig.Connections[i].Stats = conn.Stats
						break
					}
				}

				if err := r.list.configManager.SaveConfig(appConfig); err != nil && r.list.Window != nil {
					dialog.ShowError(err, r.list.Window)
					return
				}
			}

			// بروزرسانی آمار با استفاده از متد UpdateStats
			r.list.UpdateStats(conn)

			// بروزرسانی UI
			canvas.Refresh(leftBorder)
			canvas.Refresh(statsContainer)
		}
	})
	runBtn.Importance = widget.LowImportance

	pingBtn := widget.NewButtonWithIcon("", myTheme.PingIcon, func() {
		if r.list != nil && r.list.Window != nil {
			dlg := dialogs.NewPingDialog(conn, r.list.Window)
			dlg.OnDismiss(func() {
				r.list.LoadConnections()
				r.list.Refresh()
			})
			dlg.Show()
		}
	})
	pingBtn.Importance = widget.LowImportance

	details := container.NewVBox(
		container.NewHBox(
			nameLabel,
			layout.NewSpacer(),
			container.NewPadded(container.NewHBox(
				shareBtn,
				editBtn,
				deleteBtn,
				pingBtn,
				runBtn)),
		),
		container.NewHBox(
			addressLabel,
			proxyLabel,
			// layout.NewSpacer(),
			// statsContainer,
		),
	)

	content := container.NewBorder(
		nil, nil,
		leftBorder,
		nil,
		container.NewVBox(
			details,
		),
	)

	mainStack := container.NewStack(
		mainBg,
		content,
	)

	return container.New(
		&layouts.MarginLayout{MarginTop: 6, MarginBottom: 6},
		mainStack,
	)
}

func (r *connectionListRenderer) rebuild() {
	items := make([]fyne.CanvasObject, 0)

	if r.list != nil && r.list.connections != nil {
		for _, conn := range r.list.connections {
			if conn != nil {
				item := r.createConnectionItem(conn)
				spacer := widget.NewSeparator()
				spacer.Hide()
				items = append(items, item, spacer)
			}
		}
	}

	r.container = container.NewVBox(items...)
	r.objects = []fyne.CanvasObject{r.container}
}

func (r *connectionListRenderer) MinSize() fyne.Size {
	return r.container.MinSize()
}

func (r *connectionListRenderer) Layout(size fyne.Size) {
	r.container.Resize(size)
}

func (r *connectionListRenderer) Refresh() {
	r.rebuild()
	canvas.Refresh(r.container)
}

func (r *connectionListRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *connectionListRenderer) Destroy() {
	// Release any resources held by the renderer
	r.container = nil
	r.objects = nil
}
