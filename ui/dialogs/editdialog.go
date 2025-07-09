package dialogs

import (
	"fmt"
	"strconv"

	"xengate/internal/models"
	"xengate/ui/components"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type EditDialog struct {
	widget.BaseWidget

	OnDismiss func()

	content fyne.CanvasObject
}

func NewEditDialog(conn *models.Connection) *EditDialog {
	d := &EditDialog{}
	d.ExtendBaseWidget(d)

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
	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() { d.OnDismiss() })
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
		widget.NewForm(&widget.FormItem{Text: "Connections", Widget: connectionsEntry}),
		widget.NewForm(&widget.FormItem{Text: "Max Retries", Widget: maxRetriesEntry}),
	)
	connectionCard := widget.NewCard("Connection Settings", "Advanced configuration",
		container.NewPadded(connectionSettings),
	)

	proxySettings := container.NewGridWithColumns(3,
		widget.NewForm(&widget.FormItem{Text: "IP", Widget: proxyAddrEntry}),
		widget.NewForm(&widget.FormItem{Text: "Port", Widget: proxyPortEntry}),
		widget.NewForm(&widget.FormItem{Text: "Mode", Widget: proxyModeSelect}),
	)
	proxyCard := widget.NewCard("Proxy Settings", "Proxy server configuration",
		container.NewPadded(proxySettings),
	)

	saveBtn.OnTapped = func() {
		if nameEntry.Text == "" || addressEntry.Text == "" || portEntry.Text == "" {
			// dialog.ShowError(errors.New("name, address and port are required"), w)
			return
		}

		port, err := strconv.Atoi(portEntry.Text)
		if err != nil {
			// dialog.ShowError(errors.New("invalid port number"), w)
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

		appConfig := components.LoadConfig()
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

		if err := components.SaveConfig(appConfig); err != nil {
			// dialog.ShowError(err, w)
			return
		}

		// list.Refresh()
		// w.Close()
		// dialog.ShowInformation("Success", "Connection saved successfully", window)
	}

	d.content = container.NewPadded(container.NewVBox(
		basicCard, widget.NewSeparator(),
		authCard, widget.NewSeparator(),
		connectionCard, widget.NewSeparator(),
		proxyCard, widget.NewSeparator(),
		buttons,
	))

	return d
}

// func ShowEditDialog(title string, window fyne.Window, conn *models.Connection, list *components.ConnectionList, app fyne.App) {
// 	w := app.NewWindow(title)
// 	w.CenterOnScreen()

// 	nameEntry, addressEntry, portEntry := widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
// 	userEntry, passwordEntry := widget.NewEntry(), widget.NewPasswordEntry()
// 	connectionsEntry, maxRetriesEntry := widget.NewEntry(), widget.NewEntry()
// 	proxyAddrEntry, proxyPortEntry := widget.NewEntry(), widget.NewEntry()
// 	proxyModeSelect := widget.NewSelect([]string{"socks5", "http"}, nil)

// 	connectionsEntry.SetText("3")
// 	maxRetriesEntry.SetText("3")
// 	proxyAddrEntry.SetText("127.0.0.1")
// 	proxyPortEntry.SetText("1080")
// 	proxyModeSelect.SetSelected("socks5")

// 	if conn != nil {
// 		nameEntry.SetText(conn.Name)
// 		addressEntry.SetText(conn.Address)
// 		portEntry.SetText(conn.Port)

// 		if conn.Config != nil {
// 			userEntry.SetText(conn.Config.User)
// 			passwordEntry.SetText(conn.Config.Password)
// 			connectionsEntry.SetText(fmt.Sprintf("%d", conn.Config.Connections))
// 			maxRetriesEntry.SetText(fmt.Sprintf("%d", conn.Config.MaxRetries))
// 			proxyAddrEntry.SetText(conn.Config.Proxy.ListenAddr)
// 			proxyPortEntry.SetText(fmt.Sprintf("%d", conn.Config.Proxy.ListenPort))
// 			proxyModeSelect.SetSelected(conn.Config.Proxy.Mode)
// 		}
// 	}

// 	saveBtn := widget.NewButtonWithIcon("Save Connection", theme.DocumentSaveIcon(), nil)
// 	saveBtn.Importance = widget.HighImportance
// 	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() { w.Close() })
// 	buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, widget.NewSeparator(), saveBtn)

// 	basicInfo := widget.NewForm(&widget.FormItem{Text: "Name", Widget: nameEntry})

// 	serverInfo := container.NewGridWithColumns(2,
// 		widget.NewForm(&widget.FormItem{Text: "Address", Widget: addressEntry}),
// 		widget.NewForm(&widget.FormItem{Text: "Port", Widget: portEntry}),
// 	)
// 	basicCard := widget.NewCard("Basic Information", "Connection details",
// 		container.NewVBox(basicInfo, serverInfo),
// 	)

// 	authInfo := container.NewGridWithColumns(2,
// 		widget.NewForm(&widget.FormItem{Text: "Username", Widget: userEntry}),
// 		widget.NewForm(&widget.FormItem{Text: "Password", Widget: passwordEntry}),
// 	)
// 	authCard := widget.NewCard("Authentication", "User credentials",
// 		container.NewPadded(authInfo),
// 	)

// 	connectionSettings := container.NewGridWithColumns(2,
// 		widget.NewForm(&widget.FormItem{Text: "Parallel Connections", Widget: connectionsEntry}),
// 		widget.NewForm(&widget.FormItem{Text: "Max Retries", Widget: maxRetriesEntry}),
// 	)
// 	connectionCard := widget.NewCard("Connection Settings", "Advanced configuration",
// 		container.NewPadded(connectionSettings),
// 	)

// 	proxySettings := container.NewGridWithColumns(3,
// 		widget.NewForm(&widget.FormItem{Text: "Address", Widget: proxyAddrEntry}),
// 		widget.NewForm(&widget.FormItem{Text: "Port", Widget: proxyPortEntry}),
// 		widget.NewForm(&widget.FormItem{Text: "Mode", Widget: proxyModeSelect}),
// 	)
// 	proxyCard := widget.NewCard("Proxy Settings", "Proxy server configuration",
// 		container.NewPadded(proxySettings),
// 	)

// 	saveBtn.OnTapped = func() {
// 		if nameEntry.Text == "" || addressEntry.Text == "" || portEntry.Text == "" {
// 			dialog.ShowError(errors.New("name, address and port are required"), w)
// 			return
// 		}

// 		port, err := strconv.Atoi(portEntry.Text)
// 		if err != nil {
// 			dialog.ShowError(errors.New("invalid port number"), w)
// 			return
// 		}

// 		connections, _ := strconv.Atoi(connectionsEntry.Text)
// 		maxRetries, _ := strconv.Atoi(maxRetriesEntry.Text)
// 		proxyPort, _ := strconv.Atoi(proxyPortEntry.Text)

// 		if connections <= 0 {
// 			connections = 3
// 		}
// 		if maxRetries <= 0 {
// 			maxRetries = 3
// 		}
// 		if proxyPort <= 0 {
// 			proxyPort = 1080
// 		}

// 		config := &models.ServerConfig{
// 			Name: nameEntry.Text, Host: addressEntry.Text, Port: port,
// 			User: userEntry.Text, Password: passwordEntry.Text,
// 			MaxRetries: maxRetries,
// 			Proxy: models.ProxyConfig{
// 				ListenAddr: proxyAddrEntry.Text,
// 				ListenPort: proxyPort,
// 				Mode:       proxyModeSelect.Selected,
// 			},
// 		}

// 		conn.Name = nameEntry.Text
// 		conn.Address = addressEntry.Text
// 		conn.Port = portEntry.Text
// 		conn.Config = config

// 		appConfig := components.LoadConfig()
// 		found := false
// 		for i, c := range appConfig.Connections {
// 			if c.Address == conn.Address && c.Port == conn.Port {
// 				appConfig.Connections[i] = conn
// 				found = true
// 				break
// 			}
// 		}

// 		if !found {
// 			appConfig.Connections = append(appConfig.Connections, conn)
// 		}

// 		if err := components.SaveConfig(appConfig); err != nil {
// 			dialog.ShowError(err, w)
// 			return
// 		}

// 		list.Refresh()
// 		w.Close()
// 		dialog.ShowInformation("Success", "Connection saved successfully", window)
// 	}

// 	content := container.NewPadded(container.NewVBox(
// 		basicCard, widget.NewSeparator(),
// 		authCard, widget.NewSeparator(),
// 		connectionCard, widget.NewSeparator(),
// 		proxyCard, widget.NewSeparator(),
// 		buttons,
// 	))

// 	w.SetContent(content)
// 	w.Resize(content.MinSize())
// 	w.Show()
// }

func (d *EditDialog) MinSize() fyne.Size {
	return fyne.NewSize(420, d.BaseWidget.MinSize().Height)
}

func (d *EditDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(d.content)
}
