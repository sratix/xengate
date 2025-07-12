package components

import (
	"xengate/internal/common"
	"xengate/internal/models"
	"xengate/internal/storage"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type ConnectionList struct {
	widget.BaseWidget
	App           fyne.App
	storage       *storage.AppStorage
	Window        fyne.Window
	connections   []*models.Connection
	configManager common.ConfigManager
	onEdit        func(*models.Connection)
	onShare       func(*models.Connection)
	onDelete      func(*models.Connection)
	onSelect      func(*models.Connection)
}

func (l *ConnectionList) GetConfigManager() common.ConfigManager {
	return l.configManager
}

func (l *ConnectionList) LoadConnections() {
	config := l.configManager.LoadConfig()
	l.connections = config.Connections
}

func NewConnectionList(app fyne.App, window fyne.Window) *ConnectionList {
	list := &ConnectionList{
		App:         app,
		Window:      window,
		connections: make([]*models.Connection, 0),
	}
	list.ExtendBaseWidget(list)
	list.storage, _ = storage.NewAppStorage(app)
	list.configManager = &DefaultConfigManager{
		Storage: list.storage,
	}
	// Load connections from the config manager
	list.LoadConnections()
	return list
}

func (l *ConnectionList) CreateRenderer() fyne.WidgetRenderer {
	renderer := &connectionListRenderer{
		list: l,
	}
	renderer.rebuild()
	return renderer
}

func (l *ConnectionList) AddConnection(conn *models.Connection) {
	l.connections = append(l.connections, conn)
	config := l.configManager.LoadConfig()
	config.Connections = append(config.Connections, conn)
	if err := l.configManager.SaveConfig(config); err != nil {
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

	config := l.configManager.LoadConfig()
	for i, c := range config.Connections {
		if c.Address == conn.Address && c.Port == conn.Port {
			config.Connections = append(config.Connections[:i], config.Connections[i+1:]...)
			break
		}
	}

	if err := l.configManager.SaveConfig(config); err != nil {
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

func (l *ConnectionList) SetOnSelect(callback func(*models.Connection)) {
	l.onSelect = callback
}
