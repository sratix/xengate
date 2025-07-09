package components

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"

	"xengate/internal/models"
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

const (
	configFile = "connections.json"
)

type Config struct {
	Connections []*models.Connection `json:"connections"`
}

type ConnectionList struct {
	widget.BaseWidget
	App         fyne.App
	Window      fyne.Window
	connections []*models.Connection
	onEdit      func(*models.Connection)
	onShare     func(*models.Connection)
	onDelete    func(*models.Connection)
}

type connectionListRenderer struct {
	list      *ConnectionList
	container *fyne.Container
	objects   []fyne.CanvasObject
}

func NewConnectionList(app fyne.App, window fyne.Window) *ConnectionList {
	list := &ConnectionList{
		App:         app,
		Window:      window,
		connections: make([]*models.Connection, 0),
	}
	list.ExtendBaseWidget(list)
	list.loadConnections()
	return list
}

func (l *ConnectionList) loadConnections() {
	config := LoadConfig()
	l.connections = config.Connections
}

func LoadConfig() *Config {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return &Config{Connections: make([]*models.Connection, 0)}
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return &Config{Connections: make([]*models.Connection, 0)}
	}

	return &config
}

func SaveConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0o644)
}

func (l *ConnectionList) CreateRenderer() fyne.WidgetRenderer {
	renderer := &connectionListRenderer{
		list: l,
	}
	renderer.rebuild()
	return renderer
}

func (r *connectionListRenderer) createConnectionItem(conn *models.Connection) fyne.CanvasObject {
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

	typeLabel := NewBadge(conn.Type, fyne.CurrentApp().Settings().Theme().Color(myTheme.ColorNameTextMuted, fyne.CurrentApp().Settings().ThemeVariant()))

	shareBtn := widget.NewButtonWithIcon("", myTheme.ShareIcon, func() {
		if r.list.onShare != nil {
			r.list.onShare(conn)
		}
	})

	editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
		// ShowEditDialog("Edit Connection", r.list.Window, conn, r.list, r.list.App)
	})

	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if r.list.onDelete != nil {
			r.list.onDelete(conn)
		}
	})

	details := container.NewVBox(
		container.NewHBox(
			nameLabel,
			layout.NewSpacer(),
			container.NewPadded(container.NewHBox(
				shareBtn,
				editBtn,
				deleteBtn)),
		),
		container.NewHBox(
			addressLabel, typeLabel,
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

	tapButton := widget.NewButton("", func() {
		switch conn.Status {
		case models.StatusInactive:
			conn.Status = models.StatusActive
		case models.StatusActive:
			conn.Status = models.StatusInactive
		}

		appConfig := LoadConfig()
		for i, c := range appConfig.Connections {
			if c.Address == conn.Address && c.Port == conn.Port {
				appConfig.Connections[i].Status = conn.Status
				break
			}
		}
		if err := SaveConfig(appConfig); err != nil {
			dialog.ShowError(err, r.list.Window)
			return
		}

		r.list.Refresh()
	})
	tapButton.Importance = widget.LowImportance

	mainStack := container.NewStack(
		mainBg,
		tapButton,
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

func (r *connectionListRenderer) Destroy() {}

func (l *ConnectionList) AddConnection(conn *models.Connection) {
	l.connections = append(l.connections, conn)
	config := LoadConfig()
	config.Connections = append(config.Connections, conn)
	if err := SaveConfig(config); err != nil {
		dialog.ShowError(err, l.Window)
		return
	}
	l.Refresh()
}

func (l *ConnectionList) RemoveConnection(conn *models.Connection) {
	for i, c := range l.connections {
		if c.Address == conn.Address && c.Port == conn.Port {
			l.connections = append(l.connections[:i], l.connections[i+1:]...)
			break
		}
	}

	config := LoadConfig()
	for i, c := range config.Connections {
		if c.Address == conn.Address && c.Port == conn.Port {
			config.Connections = append(config.Connections[:i], config.Connections[i+1:]...)
			break
		}
	}

	if err := SaveConfig(config); err != nil {
		dialog.ShowError(err, l.Window)
		return
	}

	l.Refresh()
}

func (l *ConnectionList) SetOnEdit(callback func(*models.Connection)) {
	l.onEdit = callback
}

func (l *ConnectionList) SetOnShare(callback func(*models.Connection)) {
	l.onShare = callback
}

func (l *ConnectionList) SetOnDelete(callback func(*models.Connection)) {
	l.onDelete = callback
}
