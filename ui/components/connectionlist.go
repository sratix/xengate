package components

import (
	"fmt"

	"xengate/internal/common"
	"xengate/internal/models"
	"xengate/internal/storage"
	"xengate/internal/tunnel"
	"xengate/ui/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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
	onRun         func(*models.Connection)
	renderer      *connectionListRenderer // اضافه کردن فیلد renderer
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
		if c.ID == conn.ID {
			l.connections = append(l.connections[:i], l.connections[i+1:]...)
			break
		}
	}

	config := l.configManager.LoadConfig()
	for i, c := range config.Connections {
		if c.ID == conn.ID {
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

func (l *ConnectionList) GetConnections() []*models.Connection {
	return l.connections
}

func (l *ConnectionList) GetConfigManager() common.ConfigManager {
	return l.configManager
}

func (l *ConnectionList) LoadConnections() {
	config := l.configManager.LoadConfig()
	l.connections = config.Connections
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

func (l *ConnectionList) SetOnRun(callback func(*models.Connection)) {
	l.onRun = callback
}

func (l *ConnectionList) UpdateStats(conn *models.Connection) {
	if l.renderer == nil || conn == nil {
		return
	}

	// آپدیت آمار تانل‌ها
	if tunnelLabel, exists := l.renderer.statusLabels[conn.ID+"_tunnels"]; exists && tunnelLabel != nil {
		tunnelLabel.SetText(fmt.Sprintf("%d/%d",
			conn.Stats.Connected,
			conn.Stats.TotalTunnels))
		canvas.Refresh(tunnelLabel)
	}

	// آپدیت آمار ترافیک
	if trafficLabel, exists := l.renderer.statusLabels[conn.ID+"_traffic"]; exists && trafficLabel != nil {
		trafficLabel.SetText(util.BytesToSizeString(conn.Stats.TotalBytes))
		canvas.Refresh(trafficLabel)
	}
}

func (l *ConnectionList) BatchUpdateStats(stats map[string]tunnel.PoolStats) {
	needsRefresh := false

	for _, conn := range l.connections {
		if poolStats, exists := stats[conn.Name]; exists {
			if conn.Stats == nil {
				conn.Stats = &models.Stats{}
			}

			// آپدیت آمار
			// oldBytes := conn.Stats.TotalBytes
			conn.Stats.Connected = poolStats.Connected
			conn.Stats.TotalTunnels = poolStats.TotalTunnels
			conn.Stats.TotalBytes = poolStats.TotalBytes
			conn.Stats.Active = poolStats.ActiveConnections

			// آپدیت labels
			if tunnelLabel, exists := l.renderer.statusLabels[conn.ID+"_tunnels"]; exists && tunnelLabel != nil {
				tunnelLabel.SetText(fmt.Sprintf("%d/%d",
					conn.Stats.Connected,
					conn.Stats.TotalTunnels))
				canvas.Refresh(tunnelLabel)
				needsRefresh = true
			}

			if trafficLabel, exists := l.renderer.statusLabels[conn.ID+"_traffic"]; exists && trafficLabel != nil {
				trafficLabel.SetText(util.BytesToSizeString(conn.Stats.TotalBytes))
				canvas.Refresh(trafficLabel)
				needsRefresh = true
			}
		}
	}

	if needsRefresh {
		l.Refresh()
	}
}
