package components

import (
	"fmt"
	"image/color"

	"xengate/internal/models"
	"xengate/ui/dialogs"
	"xengate/ui/layouts"
	myTheme "xengate/ui/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type connectionListRenderer struct {
	list      *ConnectionList
	container *fyne.Container
	objects   []fyne.CanvasObject
}

func (r *connectionListRenderer) createConnectionItem(conn *models.Connection) fyne.CanvasObject {
	appConfig := r.list.configManager.LoadConfig()
	for i := range appConfig.Connections {
		appConfig.Connections[i].Status = models.StatusInactive
	}
	if err := r.list.configManager.SaveConfig(appConfig); err != nil {
	}

	itemHeight := float32(60)

	statusColor := color.NRGBA{R: 117, G: 117, B: 117, A: 255} // Inactive
	if conn.Status == models.StatusActive {
		statusColor = color.NRGBA{R: 254, G: 138, B: 129, A: 255}
	}

	mainBg := myTheme.NewThemedRectangle(myTheme.ColorNamePageBackground)

	leftBorder := canvas.NewRectangle(statusColor)
	leftBorder.SetMinSize(fyne.NewSize(6, itemHeight))

	nameLabel := widget.NewLabel(conn.Name)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	addressLabel := widget.NewLabel(fmt.Sprintf("%s:%s", conn.Address, conn.Port))
	addressLabel.TextStyle = fyne.TextStyle{Monospace: true}

	proxyLabel := NewBadge(conn.Config.Mode, fyne.CurrentApp().Settings().Theme().Color(myTheme.ColorNameTextMuted, fyne.CurrentApp().Settings().ThemeVariant()))

	shareBtn := widget.NewButtonWithIcon("", myTheme.ShareIcon, func() {
		if r.list.onShare != nil {
			r.list.onShare(conn)
		}
	})

	editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
		dlg := dialogs.NewEditDialog(conn, r.list.configManager, r.list.Window)
		pop := widget.NewModalPopUp(dlg, r.list.Window.Canvas())
		dlg.OnDismiss = func() {
			pop.Hide()
			r.list.LoadConnections()
			r.list.Refresh()
		}
		pop.Show()
	})

	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if r.list.onDelete != nil {
			r.list.onDelete(conn)
		}
	})

	var runBtn *widget.Button
	icn := theme.MediaPlayIcon()
	if conn.Status == models.StatusActive {
		icn = theme.MediaStopIcon()
	}

	runBtn = widget.NewButtonWithIcon("", icn, func() {
		if r.list.onRun != nil {
			if conn.Status == models.StatusInactive {
				runBtn.SetIcon(theme.MediaStopIcon())
				conn.Status = models.StatusActive
			} else {
				runBtn.SetIcon(theme.MediaPlayIcon())
				conn.Status = models.StatusInactive
			}

			r.list.onRun(conn)

			appConfig := r.list.configManager.LoadConfig()
			for i, c := range appConfig.Connections {
				if c.ID == conn.ID {
					appConfig.Connections[i].Status = conn.Status
					break
				}
			}

			if err := r.list.configManager.SaveConfig(appConfig); err != nil {
				dialog.ShowError(err, r.list.Window)
				return
			}

			r.list.Refresh()
		}
	})

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
			addressLabel, proxyLabel,
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

	// tapButton := widget.NewButton("", func() {
	// 	switch conn.Status {
	// 	case models.StatusInactive:
	// 		conn.Status = models.StatusActive

	// 	case models.StatusActive:
	// 		conn.Status = models.StatusInactive
	// 	}

	// 	appConfig := r.list.configManager.LoadConfig()
	// 	for i, c := range appConfig.Connections {
	// 		if c.ID == conn.ID {
	// 			appConfig.Connections[i].Status = conn.Status
	// 			r.list.Refresh()
	// 			// r.list.onSelect(conn)
	// 			break
	// 		}
	// 	}

	// 	if err := r.list.configManager.SaveConfig(appConfig); err != nil {
	// 		dialog.ShowError(err, r.list.Window)
	// 		return
	// 	}

	// 	r.list.Refresh()
	// })

	// tapButton.Importance = widget.LowImportance

	mainStack := container.NewStack(
		mainBg,
		// tapButton,
		content,
	)

	return container.New(
		&layouts.MarginLayout{MarginTop: 6, MarginBottom: 6},
		mainStack,
	)
}

func (r *connectionListRenderer) rebuild() {
	items := make([]fyne.CanvasObject, 0)

	for _, conn := range r.list.connections {
		item := r.createConnectionItem(conn)
		spacer := widget.NewSeparator()
		spacer.Hide()
		items = append(items, item, spacer)
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
