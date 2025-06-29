//go:build !(android || ios)
// +build !android,!ios

package gui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Connection struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	Password    string `json:"password"`
	Connections int    `json:"connections"`
	MaxRetries  int    `json:"max_retries"`
	IsConnected bool   `json:"is_connected"`
}

type Proxy struct {
	ProxyType string `json:"proxy_type"` // socks5, http, https
	ProxyHost string `json:"proxy_host"`
	ProxyPort int    `json:"proxy_port"`
}

type Config struct {
	Connections []Connection `json:"connections"`
	ProxyInfo   Proxy        `json:"proxyinfo"`
	LastUsed    string       `json:"last_used"`
}

// Global variables
var (
	connections       map[string]*Connection
	connectionList    binding.StringList
	nameEntry         *widget.Entry
	hostEntry         *widget.Entry
	portEntry         *widget.Entry
	userEntry         *widget.Entry
	passwordEntry     *widget.Entry
	connectionsEntry  *widget.Entry
	retriesEntry      *widget.Entry
	proxyTypeSelect   *widget.Select
	proxyHostEntry    *widget.Entry
	proxyPortEntry    *widget.Entry
	logArea           *widget.TextGrid
	statusIndicator   *canvas.Circle
	blinkingAnimation *time.Ticker
	connectBtn        *widget.Button
	allCheckbox       *widget.Check
	connectedCount    int
	window            fyne.Window
	connectionCard    *widget.Card
	proxyCard         *widget.Card
)

const (
	configFile = "connections.json"
	maxLogs    = 1000
	appTitle   = "SSH Connection Manager"
)

// File operations
func loadConfig() Config {
	var config Config
	data, err := os.ReadFile(configFile)
	if err == nil {
		json.Unmarshal(data, &config)
	}
	return config
}

func saveConfig(connections map[string]*Connection) error {
	config := Config{
		Connections: make([]Connection, 0, len(connections)),
		LastUsed:    time.Now().Format(time.RFC3339),
	}
	for _, conn := range connections {
		config.Connections = append(config.Connections, *conn)
	}
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0o644)
}

// UI Helpers
func addLog(text string) {
	current := logArea.Text()
	logs := strings.Split(current, "\n")

	timestamp := time.Now().Format("15:04:05")
	newLog := fmt.Sprintf("[%s] %s", timestamp, text)

	allLogs := append([]string{newLog}, logs...)
	if len(allLogs) > maxLogs {
		allLogs = allLogs[:maxLogs]
	}

	logArea.SetText(strings.Join(allLogs, "\n"))
}

func updateConnectionList() {
	names := make([]string, 0, len(connections))
	for name := range connections {
		names = append(names, name)
	}
	sort.Strings(names)
	connectionList.Set(names)
	saveConfig(connections)

	// Update window title with connection count
	window.SetTitle("") // fmt.Sprintf("%s (%d/%d Connected)", appTitle, connectedCount, len(connections)))
}

func clearForm() {
	nameEntry.SetText("")
	hostEntry.SetText("")
	portEntry.SetText("22")
	userEntry.SetText("")
	passwordEntry.SetText("")
	connectionsEntry.SetText("3")
	retriesEntry.SetText("3")
	allCheckbox.SetChecked(false)
	connectBtn.SetText("Connect")
	stopBlinking()
}

func startBlinking() {
	if blinkingAnimation != nil {
		blinkingAnimation.Stop()
	}
	statusIndicator.FillColor = color.RGBA{R: 0, G: 255, B: 0, A: 255}
	statusIndicator.Refresh()
	blinkingAnimation = time.NewTicker(500 * time.Millisecond)
	go func() {
		isGreen := true
		for range blinkingAnimation.C {
			if isGreen {
				statusIndicator.FillColor = color.RGBA{R: 0, G: 255, B: 0, A: 255}
			} else {
				statusIndicator.FillColor = color.RGBA{R: 0, G: 200, B: 0, A: 255}
			}
			isGreen = !isGreen
			statusIndicator.Refresh()
		}
	}()
}

func stopBlinking() {
	if blinkingAnimation != nil {
		blinkingAnimation.Stop()
		statusIndicator.FillColor = color.RGBA{R: 255, G: 0, B: 0, A: 255}
		statusIndicator.Refresh()
	}
}

func validateForm() error {
	if nameEntry.Text == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if hostEntry.Text == "" {
		return fmt.Errorf("host cannot be empty")
	}
	if userEntry.Text == "" {
		return fmt.Errorf("username cannot be empty")
	}

	port := 0
	fmt.Sscanf(portEntry.Text, "%d", &port)
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid port number")
	}

	return nil
}

func updateConnectedCount() {
	connectedCount = 0
	for _, conn := range connections {
		if conn.IsConnected {
			connectedCount++
		}
	}
	updateConnectionList() // This will update the window title
}

// UI Components
func setupFormEntries() {
	nameEntry = widget.NewEntry()
	hostEntry = widget.NewEntry()
	portEntry = widget.NewEntry()
	userEntry = widget.NewEntry()
	passwordEntry = widget.NewPasswordEntry()
	connectionsEntry = widget.NewEntry()
	retriesEntry = widget.NewEntry()

	// Default values
	portEntry.SetText("22")
	connectionsEntry.SetText("3")
	retriesEntry.SetText("3")

	// Placeholders
	nameEntry.SetPlaceHolder("Connection name")
	hostEntry.SetPlaceHolder("Hostname or IP")
	portEntry.SetPlaceHolder("22")
	userEntry.SetPlaceHolder("Username")
	passwordEntry.SetPlaceHolder("Password")

	proxyTypeSelect = widget.NewSelect([]string{"socks5", "http"}, nil)
	proxyHostEntry = widget.NewEntry()
	proxyPortEntry = widget.NewEntry()

	proxyTypeSelect.SetSelected("socks5")
	proxyHostEntry.SetText("0.0.0.0")
	proxyPortEntry.SetText("1080")

	// تنظیم placeholder ها
	proxyHostEntry.SetPlaceHolder("Proxy Host")
	proxyPortEntry.SetPlaceHolder("1080")
}

func setupButtons() *fyne.Container {
	addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		clearForm()
		addLog("Ready to add new connection")
	})

	saveBtn := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		if err := validateForm(); err != nil {
			addLog(fmt.Sprintf("Error: %s", err))
			return
		}

		name := nameEntry.Text
		port := 22
		fmt.Sscanf(portEntry.Text, "%d", &port)
		conns := 3
		fmt.Sscanf(connectionsEntry.Text, "%d", &conns)
		retries := 3
		fmt.Sscanf(retriesEntry.Text, "%d", &retries)

		connections[name] = &Connection{
			Name:        name,
			Host:        hostEntry.Text,
			Port:        port,
			User:        userEntry.Text,
			Password:    passwordEntry.Text,
			Connections: conns,
			MaxRetries:  retries,
		}

		updateConnectionList()
		addLog(fmt.Sprintf("Saved connection: %s", name))
	})

	connectBtn = widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		// if allCheckbox.Checked {
		// 	// Connect/Disconnect all
		// 	isConnecting := connectBtn.Text == "Connect"
		// 	for _, conn := range connections {
		// 		conn.IsConnected = isConnecting
		// 		if isConnecting {
		// 			addLog(fmt.Sprintf("Connected to %s@%s:%d", conn.User, conn.Host, conn.Port))
		// 		} else {
		// 			addLog(fmt.Sprintf("Disconnected from %s@%s:%d", conn.User, conn.Host, conn.Port))
		// 		}
		// 	}
		// 	if isConnecting {
		// 		startBlinking()
		// 		connectBtn.SetText("Disconnect")
		// 		connectBtn.SetIcon(theme.MediaStopIcon())
		// 	} else {
		// 		stopBlinking()
		// 		connectBtn.SetText("Connect")
		// 		connectBtn.SetIcon(theme.MediaPlayIcon())
		// 	}
		// 	updateConnectedCount()
		// } else {
		// 	// Single connection
		// 	name := nameEntry.Text
		// 	if conn, exists := connections[name]; exists {
		// 		conn.IsConnected = !conn.IsConnected
		// 		if conn.IsConnected {
		// 			startBlinking()
		// 			connectBtn.SetText("Disconnect")
		// 			connectBtn.SetIcon(theme.MediaStopIcon())
		// 			addLog(fmt.Sprintf("Connected to %s@%s:%d", conn.User, conn.Host, conn.Port))
		// 		} else {
		// 			stopBlinking()
		// 			connectBtn.SetText("Connect")
		// 			connectBtn.SetIcon(theme.MediaPlayIcon())
		// 			addLog(fmt.Sprintf("Disconnected from %s@%s:%d", conn.User, conn.Host, conn.Port))
		// 		}
		// 		updateConnectedCount()
		// 	}
		// }
		// saveConfig(connections)
	})

	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		name := nameEntry.Text
		if _, exists := connections[name]; exists {
			delete(connections, name)
			updateConnectionList()
			clearForm()
			addLog(fmt.Sprintf("Deleted connection: %s", name))
			updateConnectedCount()
		}
	})

	clearBtn := widget.NewButtonWithIcon("", theme.ContentClearIcon(), func() {
		clearForm()
		addLog("Form cleared")
	})

	// allCheckbox = widget.NewCheck("All Conn.", func(checked bool) {
	// 	if checked {
	// 		addLog("All connections mode enabled")
	// 	} else {
	// 		addLog("Single connection mode enabled")
	// 	}
	// })

	// Status indicator
	statusIndicator = canvas.NewCircle(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	statusIndicator.Resize(fyne.NewSize(30, 30))
	statusIndicatorContainer := container.NewPadded(statusIndicator)

	// Group connect button and checkbox
	// connectGroup := container.NewHBox(connectBtn, allCheckbox)

	return container.NewVBox(
		addBtn,
		saveBtn,
		connectBtn,
		deleteBtn,
		clearBtn,
		statusIndicatorContainer,
	)
}

func setupProxyForm() *widget.Card {
	proxyForm := container.NewGridWithRows(1,
		widget.NewLabel("Type:"), proxyTypeSelect,
		widget.NewLabel("Host:"), proxyHostEntry,
		widget.NewLabel("Port:"), proxyPortEntry,
	)

	return widget.NewCard(
		"Proxy Settings",
		"Configure proxy connection settings",
		proxyForm,
	)
}

func setupConnectionList() *widget.List {
	list := widget.NewListWithData(connectionList,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.ComputerIcon()),
				widget.NewLabel("Template"),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			text, _ := item.(binding.String).Get()
			label := obj.(*fyne.Container).Objects[1].(*widget.Label)
			icon := obj.(*fyne.Container).Objects[0].(*widget.Icon)

			conn := connections[text]
			label.SetText(text)

			if conn.IsConnected {
				icon.SetResource(theme.ComputerIcon())
			} else {
				icon.SetResource(theme.DesktopIcon())
			}
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		items, _ := connectionList.Get()
		name := items[id]
		if conn, exists := connections[name]; exists {
			nameEntry.SetText(conn.Name)
			hostEntry.SetText(conn.Host)
			portEntry.SetText(fmt.Sprintf("%d", conn.Port))
			userEntry.SetText(conn.User)
			passwordEntry.SetText(conn.Password)
			connectionsEntry.SetText(fmt.Sprintf("%d", conn.Connections))
			retriesEntry.SetText(fmt.Sprintf("%d", conn.MaxRetries))

			if conn.IsConnected {
				// connectBtn.SetText("Disconnect")
				connectBtn.SetIcon(theme.MediaStopIcon())
				startBlinking()
			} else {
				// connectBtn.SetText("Connect")
				connectBtn.SetIcon(theme.MediaPlayIcon())
				stopBlinking()
			}
		}
	}

	return list
}

func setupForm() *fyne.Container {
	mainForm := container.NewGridWithColumns(2,
		widget.NewLabel("Name:"), nameEntry,
		widget.NewLabel("Host:"), hostEntry,
		widget.NewLabel("Port:"), portEntry,
		widget.NewLabel("Username:"), userEntry,
		widget.NewLabel("Password:"), passwordEntry,
		widget.NewLabel("Connections:"), connectionsEntry,
		widget.NewLabel("Max Retries:"), retriesEntry,
	)

	connectionCard := widget.NewCard(
		"Connection Settings",
		"Configure main connection settings",
		mainForm,
	)
	proxyForm := setupProxyForm()

	// ترکیب دو فرم با یک separator
	return container.NewHBox(container.NewVBox(
		layout.NewSpacer(),
		connectionCard,
		widget.NewSeparator(),
		proxyForm,
	), container.NewCenter(setupButtons()))
}

func mainWindow(s *FyneUI) fyne.CanvasObject {
	window = s.MainWin

	// Initialize global variables
	connections = make(map[string]*Connection)
	connectionList = binding.NewStringList()
	connectedCount = 0

	// Setup UI components
	setupFormEntries()
	// buttons := setupButtons()
	list := setupConnectionList()
	form := setupForm()

	// Log area
	logArea = widget.NewTextGrid()
	logScroll := container.NewScroll(logArea)

	// Layout
	rightSideContent := container.NewVBox(
		form,
		// buttons,
		widget.NewSeparator(),
	)

	rightSideSplit := container.NewVSplit(
		rightSideContent,
		logScroll,
	)
	rightSideSplit.SetOffset(0.7)

	leftSide := container.NewBorder(nil, nil, nil, nil, list)
	mainSplit := container.NewHSplit(leftSide, rightSideSplit)
	mainSplit.SetOffset(0.3)

	// Load saved connections
	config := loadConfig()
	for _, conn := range config.Connections {
		connections[conn.Name] = &Connection{
			Name:        conn.Name,
			Host:        conn.Host,
			Port:        conn.Port,
			User:        conn.User,
			Password:    conn.Password,
			Connections: conn.Connections,
			MaxRetries:  conn.MaxRetries,
			IsConnected: conn.IsConnected,
		}
		if conn.IsConnected {
			connectedCount++
		}
	}

	updateConnectionList()

	return mainSplit
}
