package dialogs

import (
	"errors"
	"fmt"
	"strconv"

	"xengate/internal/common"
	"xengate/internal/models"
	"xengate/ui/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type EditDialog struct {
	widget.BaseWidget
	conn          *models.Connection
	proxy         *models.ProxyConfig
	configManager common.ConfigManager
	window        fyne.Window
	OnDismiss     func()
	content       fyne.CanvasObject

	// form fields
	nameEntry        *widget.Entry
	addressEntry     *widget.Entry
	portEntry        *widget.Entry
	userEntry        *widget.Entry
	passwordEntry    *widget.Entry
	connectionsEntry *widget.Entry
	maxRetriesEntry  *widget.Entry
	// proxyAddrEntry   *widget.Entry
	// proxyPortEntry   *widget.Entry
	proxyModeSelect *widget.RadioGroup
}

func NewEditDialog(conn *models.Connection, configManager common.ConfigManager, window fyne.Window) *EditDialog {
	d := &EditDialog{
		conn:          conn,
		configManager: configManager,
		window:        window,
	}
	d.ExtendBaseWidget(d)

	d.createFormFields()
	d.initializeFields()
	if conn != nil {
		d.populateFields()
	}

	d.content = d.createDialogContent()

	return d
}

func (d *EditDialog) createFormFields() {
	d.nameEntry = widget.NewEntry()
	d.addressEntry = widget.NewEntry()
	d.portEntry = widget.NewEntry()
	d.userEntry = widget.NewEntry()
	d.passwordEntry = widget.NewPasswordEntry()
	d.connectionsEntry = widget.NewEntry()
	d.maxRetriesEntry = widget.NewEntry()
	// d.proxyAddrEntry = widget.NewEntry()
	// d.proxyPortEntry = widget.NewEntry()
	// d.proxyModeSelect = widget.NewSelect([]string{"socks5", "http"}, nil)
	d.proxyModeSelect = widget.NewRadioGroup([]string{"socks5", "http"}, nil)
	d.proxyModeSelect.Horizontal = true // Make radio buttons horizontal
	d.proxyModeSelect.Required = true   // Make selection required
}

func (d *EditDialog) initializeFields() {
	d.connectionsEntry.SetText("3")
	d.maxRetriesEntry.SetText("3")
	// d.proxyAddrEntry.SetText("127.0.0.1")
	// d.proxyPortEntry.SetText("1080")
	d.proxyModeSelect.SetSelected("socks5")
}

func (d *EditDialog) populateFields() {
	d.nameEntry.SetText(d.conn.Name)
	d.addressEntry.SetText(d.conn.Address)
	d.portEntry.SetText(d.conn.Port)

	if d.conn.Config != nil {
		d.userEntry.SetText(d.conn.Config.User)
		d.passwordEntry.SetText(d.conn.Config.Password)
		d.connectionsEntry.SetText(fmt.Sprintf("%d", d.conn.Config.Connections))
		d.maxRetriesEntry.SetText(fmt.Sprintf("%d", d.conn.Config.MaxRetries))
		// d.proxyAddrEntry.SetText(d.proxy.ListenAddr)
		// d.proxyPortEntry.SetText(fmt.Sprintf("%d", d.proxy.ListenPort))
		d.proxyModeSelect.SetSelected(d.conn.Config.Mode)
	}
}

func (d *EditDialog) createDialogContent() fyne.CanvasObject {
	basicInfo := widget.NewForm(&widget.FormItem{Text: "Name", Widget: d.nameEntry})
	serverInfo := container.NewGridWithColumns(2,
		widget.NewForm(&widget.FormItem{Text: "IP", Widget: d.addressEntry}),
		widget.NewForm(&widget.FormItem{Text: "Port", Widget: d.portEntry}),
		widget.NewForm(&widget.FormItem{Text: "Mode", Widget: d.proxyModeSelect}),
	)
	basicCard := widget.NewCard("Basic Information", "Connection details",
		container.NewVBox(basicInfo, serverInfo),
	)

	authInfo := container.NewGridWithRows(2,
		widget.NewForm(&widget.FormItem{Text: "Username", Widget: d.userEntry}),
		widget.NewForm(&widget.FormItem{Text: "Password", Widget: d.passwordEntry}),
	)
	authCard := widget.NewCard("Authentication", "User credentials",
		container.NewPadded(authInfo),
	)

	connectionSettings := container.NewGridWithColumns(2,
		widget.NewForm(&widget.FormItem{Text: "Connections", Widget: d.connectionsEntry}),
		widget.NewForm(&widget.FormItem{Text: "Max Retries", Widget: d.maxRetriesEntry}),
	)
	connectionCard := widget.NewCard("Connection Settings", "Advanced configuration",
		container.NewPadded(connectionSettings),
	)

	// proxySettings := container.NewGridWithColumns(2,
	// 	// widget.NewForm(&widget.FormItem{Text: "IP", Widget: d.proxyAddrEntry}),
	// 	// widget.NewForm(&widget.FormItem{Text: "Port", Widget: d.proxyPortEntry}),
	// 	widget.NewForm(&widget.FormItem{Text: "Mode", Widget: d.proxyModeSelect}),
	// )
	// proxyCard := widget.NewCard("Proxy Settings", "Proxy server configuration",
	// 	container.NewPadded(proxySettings),
	// )

	saveBtn := widget.NewButtonWithIcon("Save Connection", theme.DocumentSaveIcon(), func() {
		if err := d.validateAndSave(); err != nil {
			dialog.ShowError(err, d.window)
			return
		}

		dialog.ShowInformation("Success", "Connection saved successfully", d.window)
		if d.OnDismiss != nil {
			d.OnDismiss()
		}
	})
	saveBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		if d.OnDismiss != nil {
			d.OnDismiss()
		}
	})

	buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, widget.NewSeparator(), saveBtn)

	return container.NewPadded(container.NewVBox(
		basicCard, widget.NewSeparator(),
		authCard, widget.NewSeparator(),
		connectionCard, widget.NewSeparator(),
		// proxyCard, widget.NewSeparator(),
		buttons,
	))
}

func (d *EditDialog) validateAndSave() error {
	if d.nameEntry.Text == "" || d.addressEntry.Text == "" || d.portEntry.Text == "" {
		return errors.New("name, address and port are required")
	}

	port, err := strconv.Atoi(d.portEntry.Text)
	if err != nil {
		return errors.New("invalid port number")
	}

	connections, _ := strconv.Atoi(d.connectionsEntry.Text)
	maxRetries, _ := strconv.Atoi(d.maxRetriesEntry.Text)
	// proxyPort, _ := strconv.Atoi(d.proxyPortEntry.Text)

	if connections <= 0 {
		connections = 3
	}
	if maxRetries <= 0 {
		maxRetries = 3
	}
	// if proxyPort <= 0 {
	// 	proxyPort = 1080
	// }

	config := &models.ServerConfig{
		Name:        d.nameEntry.Text,
		Host:        d.addressEntry.Text,
		Port:        port,
		User:        d.userEntry.Text,
		Password:    d.passwordEntry.Text,
		Mode:        d.proxyModeSelect.Selected,
		Connections: connections,
		MaxRetries:  maxRetries,
	}

	d.conn.Name = d.nameEntry.Text
	d.conn.Address = d.addressEntry.Text
	d.conn.Port = d.portEntry.Text
	d.conn.Config = config

	fmt.Printf("%s%d", config.Host, config.Port)

	d.conn.ID = util.HashString(fmt.Sprintf("%s%d", config.Host, config.Port))

	appConfig := d.configManager.LoadConfig()
	found := false
	for i, c := range appConfig.Connections {
		if c.ID == d.conn.ID {
			appConfig.Connections[i] = d.conn
			found = true
			break
		}
	}

	if !found {
		fmt.Println("OMADAM INJA!!!!")
		appConfig.Connections = append(appConfig.Connections, d.conn)
	}

	return d.configManager.SaveConfig(appConfig)
}

func (d *EditDialog) MinSize() fyne.Size {
	return fyne.NewSize(400, d.BaseWidget.MinSize().Height)
}

func (d *EditDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(d.content)
}
