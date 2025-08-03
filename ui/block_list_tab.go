package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"xengate/internal/common"
	"xengate/internal/models"
	"xengate/internal/storage"
	"xengate/internal/tunnel"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	log "github.com/sirupsen/logrus"
)

type BlockListTab struct {
	window        fyne.Window
	manager       *tunnel.Manager
	container     *fyne.Container
	table         *widget.Table
	items         []models.BlockedIPInfo
	configManager common.ConfigManager
	storage       *storage.AppStorage
}

func NewBlockListTab(window fyne.Window, manager *tunnel.Manager) *BlockListTab {
	storage, err := storage.NewAppStorage(fyne.CurrentApp())
	if err != nil {
		log.WithError(err).Error("Failed to initialize storage")
		return nil
	}

	tab := &BlockListTab{
		window:  window,
		manager: manager,
		items:   make([]models.BlockedIPInfo, 0),
		storage: storage,
		configManager: &common.DefaultConfigManager{
			Storage: storage,
		},
	}

	tab.initUI()
	tab.loadBlockedIPsFromConfig()

	return tab
}

func (b *BlockListTab) loadBlockedIPsFromConfig() {
	config := b.configManager.LoadConfig()
	if config == nil || config.BlockedList == nil {
		return
	}

	// پاک کردن لیست فعلی
	for _, item := range b.items {
		b.manager.UnblockIP(item.IP)
	}

	b.items = make([]models.BlockedIPInfo, 0)

	// اضافه کردن IP های موجود در فایل کانفیگ
	for _, blockedIP := range config.BlockedList {
		if blockedIP != nil && blockedIP.IP != "" {
			b.manager.BlockIP(blockedIP.IP)
			b.items = append(b.items, models.BlockedIPInfo{
				IP:        blockedIP.IP,
				Timestamp: blockedIP.Timestamp,
			})
		}
	}

	// مرتب سازی بر اساس زمان
	b.sortItems()

	if b.table != nil {
		b.table.Refresh()
	}
}

func (b *BlockListTab) saveToConfig() {
	currentBlocked := b.manager.GetBlockedIPs()
	blockedList := make([]*models.BlockedIPInfo, len(currentBlocked))

	for i, item := range currentBlocked {
		blockedList[i] = &models.BlockedIPInfo{
			IP:        item.IP,
			Timestamp: item.Timestamp,
		}
	}

	config := b.configManager.LoadConfig()
	if config == nil {
		config = &common.Config{}
	}

	config.BlockedList = blockedList

	if err := b.configManager.SaveConfig(config); err != nil {
		log.WithError(err).Error("Failed to save blocked IPs to config")
		dialog.ShowError(fmt.Errorf("Failed to save configuration: %v", err), b.window)
	}
}

func (b *BlockListTab) sortItems() {
	sort.Slice(b.items, func(i, j int) bool {
		return b.items[i].Timestamp.After(b.items[j].Timestamp)
	})
}

func (b *BlockListTab) initUI() {
	var selectedCell widget.TableCellID

	b.table = widget.NewTable(
		func() (int, int) {
			return len(b.items) + 1, 2 // +1 for header row
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)

			if id.Row == 0 {
				headers := []string{"IP Address", "Blocked Since"}
				if id.Col < len(headers) {
					label.SetText(headers[id.Col])
					label.TextStyle = fyne.TextStyle{Bold: true}
				}
				return
			}

			dataRow := id.Row - 1
			if dataRow < len(b.items) {
				item := b.items[dataRow]
				switch id.Col {
				case 0:
					label.SetText(item.IP)
				case 1:
					label.SetText(item.Timestamp.Format("2006-01-02 15:04:05"))
				}
			}
		},
	)

	b.table.SetColumnWidth(0, 150)
	b.table.SetColumnWidth(1, 180)

	addButton := widget.NewButtonWithIcon("Add IP", theme.ContentAddIcon(), func() {
		b.showAddDialog()
	})

	removeButton := widget.NewButtonWithIcon("Remove IP", theme.DeleteIcon(), func() {
		if len(b.items) > 0 && selectedCell.Row > 0 && selectedCell.Row <= len(b.items) {
			b.showRemoveDialog(b.items[selectedCell.Row-1].IP)
		}
	})

	clearButton := widget.NewButtonWithIcon("Clear All", theme.DeleteIcon(), func() {
		if len(b.items) > 0 {
			b.showClearAllDialog()
		}
	})

	b.table.OnSelected = func(id widget.TableCellID) {
		selectedCell = id
		if id.Row > 0 && id.Row <= len(b.items) {
			b.showRemoveDialog(b.items[id.Row-1].IP)
			b.table.UnselectAll()
		}
	}

	toolbar := container.NewHBox(
		addButton,
		removeButton,
		clearButton,
	)

	b.container = container.NewBorder(
		toolbar, nil, nil, nil,
		container.NewPadded(b.table),
	)
}

func (b *BlockListTab) showAddDialog() {
	if b.manager == nil {
		log.Error("Manager is nil in BlockListTab")
		return
	}

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter IP address")

	validate := func(s string) error {
		if s = strings.TrimSpace(s); s == "" {
			return fmt.Errorf("IP address cannot be empty")
		}

		parts := strings.Split(s, ".")
		if len(parts) != 4 {
			return fmt.Errorf("Invalid IP format")
		}

		for _, part := range parts {
			num, err := strconv.Atoi(part)
			if err != nil || num < 0 || num > 255 {
				return fmt.Errorf("Invalid IP format")
			}
		}

		if b.manager.IsIPBlocked(s) {
			return fmt.Errorf("This IP is already blocked")
		}

		return nil
	}

	dialog.ShowForm("Add IP to Blocklist",
		"Block", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("IP Address", input),
		},
		func(ok bool) {
			if !ok || input.Text == "" {
				return
			}

			if err := validate(input.Text); err != nil {
				dialog.ShowError(err, b.window)
				return
			}

			b.manager.BlockIP(input.Text)
			b.refreshList()

			dialog.ShowInformation("Success",
				fmt.Sprintf("IP %s has been blocked", input.Text),
				b.window)
		},
		b.window)
}

func (b *BlockListTab) showRemoveDialog(ip string) {
	dialog.ShowConfirm("Remove IP",
		fmt.Sprintf("Do you want to unblock %s?", ip),
		func(ok bool) {
			if ok {
				b.manager.UnblockIP(ip)
				b.refreshList()
			}
		},
		b.window)
}

func (b *BlockListTab) showClearAllDialog() {
	dialog.ShowConfirm("Clear All",
		"Are you sure you want to unblock all IPs?",
		func(ok bool) {
			if ok {
				b.clearAllBlockedIPs()
			}
		},
		b.window)
}

func (b *BlockListTab) clearAllBlockedIPs() {
	for _, item := range b.items {
		b.manager.UnblockIP(item.IP)
	}

	b.items = make([]models.BlockedIPInfo, 0)
	b.table.Refresh()
	b.saveToConfig()
}

func (b *BlockListTab) refreshList() {
	if b.manager == nil {
		log.Error("Manager is nil in BlockListTab")
		return
	}

	blockedIPs := b.manager.GetBlockedIPs()
	b.items = make([]models.BlockedIPInfo, len(blockedIPs))

	for i, item := range blockedIPs {
		b.items[i] = models.BlockedIPInfo{
			IP:        item.IP,
			Timestamp: item.Timestamp,
		}
	}

	b.sortItems()
	b.table.Refresh()
	b.saveToConfig()
}

func (b *BlockListTab) Container() fyne.CanvasObject {
	return b.container
}
