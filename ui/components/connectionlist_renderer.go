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
	// اگر اتصال خالی باشد، یک کانتینر خالی برگردان
	if conn == nil {
		return container.NewHBox()
	}

	// مقداردهی اولیه آمار اگر nil باشد
	if conn.Stats == nil {
		conn.Stats = &models.Stats{
			Connected:    0,
			TotalTunnels: 0,
			TotalBytes:   0,
		}
	}

	// تنظیم ارتفاع آیتم
	itemHeight := float32(60)

	// تنظیم رنگ نوار وضعیت
	statusColor := color.NRGBA{R: 117, G: 117, B: 117, A: 255} // رنگ غیرفعال
	if conn.Status == models.StatusActive {
		statusColor = color.NRGBA{R: 89, G: 205, B: 144, A: 255} // رنگ فعال
	}

	// ایجاد پس‌زمینه و نوار وضعیت
	mainBg := myTheme.NewThemedRectangle(myTheme.ColorNamePageBackground)
	leftBorder := canvas.NewRectangle(statusColor)
	leftBorder.SetMinSize(fyne.NewSize(6, itemHeight))

	// ایجاد لیبل نام با استایل پررنگ
	nameLabel := widget.NewLabel(conn.Name)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// ایجاد لیبل آدرس با فونت monospace
	addressLabel := widget.NewLabel(fmt.Sprintf("%s:%s", conn.Address, conn.Port))
	addressLabel.TextStyle = fyne.TextStyle{Monospace: true}

	// ایجاد نشان نوع پروکسی
	var proxyLabel fyne.CanvasObject
	if conn.Config != nil {
		proxyLabel = NewBadge(conn.Config.Mode, fyne.CurrentApp().Settings().Theme().Color(myTheme.ColorNameTextMuted, fyne.CurrentApp().Settings().ThemeVariant()))
	} else {
		proxyLabel = NewBadge("unknown", fyne.CurrentApp().Settings().Theme().Color(myTheme.ColorNameTextMuted, fyne.CurrentApp().Settings().ThemeVariant()))
	}

	// ایجاد بخش آمار با آیکون‌ها
	tunnelIcon := widget.NewIcon(theme.ComputerIcon())
	tunnelStats := widget.NewLabel(fmt.Sprintf("%d/%d", conn.Stats.Connected, conn.Stats.TotalTunnels))
	tunnelStats.TextStyle = fyne.TextStyle{Monospace: true}

	uploadIcon := widget.NewIcon(theme.UploadIcon())
	downloadIcon := widget.NewIcon(theme.DownloadIcon())
	trafficStats := widget.NewLabel(util.BytesToSizeString(conn.Stats.TotalBytes))
	trafficStats.TextStyle = fyne.TextStyle{Monospace: true}

	// ذخیره لیبل‌های آمار برای آپدیت‌های بعدی
	if r.statusLabels == nil {
		r.statusLabels = make(map[string]*widget.Label)
	}
	r.statusLabels[conn.ID+"_tunnels"] = tunnelStats
	r.statusLabels[conn.ID+"_traffic"] = trafficStats

	// ایجاد دکمه‌های عملیات
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

	// ایجاد دکمه اجرا/توقف با تغییر آیکون
	var runBtn *widget.Button
	icn := theme.MediaPlayIcon()
	if conn.Status == models.StatusActive {
		icn = theme.MediaStopIcon()
	}

	runBtn = widget.NewButtonWithIcon("", icn, func() {
		if r.list != nil && r.list.onRun != nil {
			if conn.Status == models.StatusInactive {
				runBtn.SetIcon(theme.MediaStopIcon())
				conn.Status = models.StatusActive
			} else {
				runBtn.SetIcon(theme.MediaPlayIcon())
				conn.Status = models.StatusInactive
			}

			r.list.onRun(conn)

			// ذخیره وضعیت در تنظیمات
			if r.list.configManager != nil {
				appConfig := r.list.configManager.LoadConfig()
				for i, c := range appConfig.Connections {
					if c.ID == conn.ID {
						appConfig.Connections[i].Status = conn.Status
						break
					}
				}

				if err := r.list.configManager.SaveConfig(appConfig); err != nil && r.list.Window != nil {
					dialog.ShowError(err, r.list.Window)
					return
				}
			}
		}
	})
	runBtn.Importance = widget.LowImportance

	// ایجاد کانتینر آمار
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

	// ایجاد بخش جزئیات با چیدمان عمودی
	details := container.NewVBox(
		container.NewHBox(
			nameLabel,
			layout.NewSpacer(),
			container.NewPadded(container.NewHBox(
				shareBtn,
				editBtn,
				deleteBtn,
				runBtn)),
		),
		container.NewHBox(
			addressLabel,
			proxyLabel,
			layout.NewSpacer(),
			statsContainer,
		),
	)

	// ایجاد کانتینر اصلی با حاشیه و نوار وضعیت
	content := container.NewBorder(
		nil, nil,
		leftBorder,
		nil,
		container.NewVBox(
			details,
		),
	)

	// ترکیب نهایی با پس‌زمینه
	mainStack := container.NewStack(
		mainBg,
		content,
	)

	// برگرداندن کانتینر نهایی با حاشیه‌های بالا و پایین
	return container.New(
		&layouts.MarginLayout{MarginTop: 6, MarginBottom: 6},
		mainStack,
	)
}

func (r *connectionListRenderer) rebuild() {
	items := make([]fyne.CanvasObject, 0)

	if r.statusLabels == nil {
		r.statusLabels = make(map[string]*widget.Label)
	}

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
