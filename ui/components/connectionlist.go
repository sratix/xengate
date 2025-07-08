package components

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"os"
	"strconv"

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
	config := loadConfig()
	l.connections = config.Connections
}

func loadConfig() *Config {
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

func saveConfig(config *Config) error {
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

	mainBg := myTheme.NewThemedRectangle(myTheme.ColorNameNowPlayingPanel)

	leftBorder := canvas.NewRectangle(statusColor)
	leftBorder.SetMinSize(fyne.NewSize(6, itemHeight))

	nameLabel := widget.NewLabel(conn.Name)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	addressLabel := widget.NewLabel(fmt.Sprintf("%s:%s", conn.Address, conn.Port))
	addressLabel.TextStyle = fyne.TextStyle{Monospace: true}

	typeLabel := widget.NewLabelWithStyle(
		fmt.Sprintf("(%s)", conn.Type),
		fyne.TextAlignTrailing,
		fyne.TextStyle{Italic: true},
	)

	shareBtn := widget.NewButtonWithIcon("", myTheme.ShareIcon, func() {
		if r.list.onShare != nil {
			r.list.onShare(conn)
		}
	})

	editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
		ShowEditDialog("Edit Connection", r.list.Window, conn, r.list, r.list.App)
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
			shareBtn,
			editBtn,
			deleteBtn,
		),
		container.NewHBox(
			addressLabel,
			layout.NewSpacer(),
			typeLabel,
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

		appConfig := loadConfig()
		for i, c := range appConfig.Connections {
			if c.Address == conn.Address && c.Port == conn.Port {
				appConfig.Connections[i].Status = conn.Status
				break
			}
		}
		if err := saveConfig(appConfig); err != nil {
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

	// ایجاد کانتینر با پدینگ بالا و پایین برای مارجین
	return container.New(
		&layouts.MarginLayout{MarginTop: 6, MarginBottom: 6},
		mainStack,
	)
}

func ShowEditDialog(title string, window fyne.Window, conn *models.Connection, list *ConnectionList, app fyne.App) {
	w := app.NewWindow(title)
	w.CenterOnScreen()

	nameEntry, addressEntry, portEntry := widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	userEntry, passwordEntry := widget.NewEntry(), widget.NewPasswordEntry()
	connectionsEntry, maxRetriesEntry := widget.NewEntry(), widget.NewEntry()
	proxyAddrEntry, proxyPortEntry := widget.NewEntry(), widget.NewEntry()
	proxyModeSelect := widget.NewSelect([]string{"socks5", "http"}, nil)

	connectionsEntry.SetText("3")
	maxRetriesEntry.SetText("3")
	proxyAddrEntry.SetText("127.0.0.1")
	proxyPortEntry.SetText("1080")
	proxyModeSelect.SetSelected("socks5")

	if conn != nil {
		nameEntry.SetText(conn.Name)
		addressEntry.SetText(conn.Address)
		portEntry.SetText(conn.Port)

		if conn.Config != nil {
			userEntry.SetText(conn.Config.User)
			passwordEntry.SetText(conn.Config.Password)
			connectionsEntry.SetText(fmt.Sprintf("%d", conn.Config.Connections))
			maxRetriesEntry.SetText(fmt.Sprintf("%d", conn.Config.MaxRetries))
			proxyAddrEntry.SetText(conn.Config.Proxy.ListenAddr)
			proxyPortEntry.SetText(fmt.Sprintf("%d", conn.Config.Proxy.ListenPort))
			proxyModeSelect.SetSelected(conn.Config.Proxy.Mode)
		}
	}

	saveBtn := widget.NewButtonWithIcon("Save Connection", theme.DocumentSaveIcon(), nil)
	saveBtn.Importance = widget.HighImportance
	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() { w.Close() })
	buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, widget.NewSeparator(), saveBtn)

	basicInfo := widget.NewForm(&widget.FormItem{Text: "Name", Widget: nameEntry})

	serverInfo := container.NewGridWithColumns(2,
		widget.NewForm(&widget.FormItem{Text: "Address", Widget: addressEntry}),
		widget.NewForm(&widget.FormItem{Text: "Port", Widget: portEntry}),
	)
	basicCard := widget.NewCard("Basic Information", "Connection details",
		container.NewVBox(basicInfo, serverInfo),
	)

	authInfo := container.NewGridWithColumns(2,
		widget.NewForm(&widget.FormItem{Text: "Username", Widget: userEntry}),
		widget.NewForm(&widget.FormItem{Text: "Password", Widget: passwordEntry}),
	)
	authCard := widget.NewCard("Authentication", "User credentials",
		container.NewPadded(authInfo),
	)

	connectionSettings := container.NewGridWithColumns(2,
		widget.NewForm(&widget.FormItem{Text: "Parallel Connections", Widget: connectionsEntry}),
		widget.NewForm(&widget.FormItem{Text: "Max Retries", Widget: maxRetriesEntry}),
	)
	connectionCard := widget.NewCard("Connection Settings", "Advanced configuration",
		container.NewPadded(connectionSettings),
	)

	proxySettings := container.NewGridWithColumns(3,
		widget.NewForm(&widget.FormItem{Text: "Address", Widget: proxyAddrEntry}),
		widget.NewForm(&widget.FormItem{Text: "Port", Widget: proxyPortEntry}),
		widget.NewForm(&widget.FormItem{Text: "Mode", Widget: proxyModeSelect}),
	)
	proxyCard := widget.NewCard("Proxy Settings", "Proxy server configuration",
		container.NewPadded(proxySettings),
	)

	saveBtn.OnTapped = func() {
		if nameEntry.Text == "" || addressEntry.Text == "" || portEntry.Text == "" {
			dialog.ShowError(errors.New("name, address and port are required"), w)
			return
		}

		port, err := strconv.Atoi(portEntry.Text)
		if err != nil {
			dialog.ShowError(errors.New("invalid port number"), w)
			return
		}

		connections, _ := strconv.Atoi(connectionsEntry.Text)
		maxRetries, _ := strconv.Atoi(maxRetriesEntry.Text)
		proxyPort, _ := strconv.Atoi(proxyPortEntry.Text)

		if connections <= 0 {
			connections = 3
		}
		if maxRetries <= 0 {
			maxRetries = 3
		}
		if proxyPort <= 0 {
			proxyPort = 1080
		}

		config := &models.ServerConfig{
			Name: nameEntry.Text, Host: addressEntry.Text, Port: port,
			User: userEntry.Text, Password: passwordEntry.Text,
			MaxRetries: maxRetries,
			Proxy: models.ProxyConfig{
				ListenAddr: proxyAddrEntry.Text,
				ListenPort: proxyPort,
				Mode:       proxyModeSelect.Selected,
			},
		}

		conn.Name = nameEntry.Text
		conn.Address = addressEntry.Text
		conn.Port = portEntry.Text
		conn.Config = config

		appConfig := loadConfig()
		found := false
		for i, c := range appConfig.Connections {
			if c.Address == conn.Address && c.Port == conn.Port {
				appConfig.Connections[i] = conn
				found = true
				break
			}
		}

		if !found {
			appConfig.Connections = append(appConfig.Connections, conn)
		}

		if err := saveConfig(appConfig); err != nil {
			dialog.ShowError(err, w)
			return
		}

		list.Refresh()
		w.Close()
		dialog.ShowInformation("Success", "Connection saved successfully", window)
	}

	content := container.NewPadded(container.NewVBox(
		basicCard, widget.NewSeparator(),
		authCard, widget.NewSeparator(),
		connectionCard, widget.NewSeparator(),
		proxyCard, widget.NewSeparator(),
		buttons,
	))

	w.SetContent(content)
	w.Resize(content.MinSize())
	w.Show()
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
	config := loadConfig()
	config.Connections = append(config.Connections, conn)
	if err := saveConfig(config); err != nil {
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

	config := loadConfig()
	for i, c := range config.Connections {
		if c.Address == conn.Address && c.Port == conn.Port {
			config.Connections = append(config.Connections[:i], config.Connections[i+1:]...)
			break
		}
	}

	if err := saveConfig(config); err != nil {
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
