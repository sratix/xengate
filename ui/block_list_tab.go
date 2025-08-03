package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"xengate/internal/tunnel"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	log "github.com/sirupsen/logrus"
)

type BlockedIP struct {
	IP        string
	Timestamp time.Time
}

type BlockListTab struct {
	window    fyne.Window
	manager   *tunnel.Manager
	container *fyne.Container
	table     *widget.Table
	items     []BlockedIP
}

func NewBlockListTab(window fyne.Window, manager *tunnel.Manager) *BlockListTab {
	tab := &BlockListTab{
		window:  window,
		manager: manager,
		items:   make([]BlockedIP, 0),
	}
	tab.initUI()
	return tab
}

func (b *BlockListTab) initUI() {
	var selectedCell widget.TableCellID

	// Create table
	b.table = widget.NewTable(
		func() (int, int) {
			return len(b.items) + 1, 2 // +1 for header row
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)

			// Handle header row
			if id.Row == 0 {
				headers := []string{"IP Address", "Blocked Since"}
				if id.Col < len(headers) {
					label.SetText(headers[id.Col])
					label.TextStyle = fyne.TextStyle{Bold: true}
				}
				return
			}

			// Adjust row index for data (subtract header row)
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

	// Set column widths
	b.table.SetColumnWidth(0, 150) // IP Address
	b.table.SetColumnWidth(1, 180) // Timestamp

	// Add button
	addButton := widget.NewButtonWithIcon("Add IP", theme.ContentAddIcon(), func() {
		b.showAddDialog()
	})

	// Remove button
	removeButton := widget.NewButtonWithIcon("Remove IP", theme.DeleteIcon(), func() {
		if len(b.items) > 0 && selectedCell.Row > 0 && selectedCell.Row <= len(b.items) {
			ip := b.items[selectedCell.Row-1].IP
			dialog.ShowConfirm("Remove IP",
				"Are you sure you want to unblock this IP?",
				func(ok bool) {
					if ok {
						b.manager.UnblockIP(ip)
						b.refreshList()
					}
				},
				b.window)
		}
	})

	// Clear all button
	clearButton := widget.NewButtonWithIcon("Clear All", theme.DeleteIcon(), func() {
		if len(b.items) > 0 {
			dialog.ShowConfirm("Clear All",
				"Are you sure you want to unblock all IPs?",
				func(ok bool) {
					if ok {
						for _, item := range b.items {
							b.manager.UnblockIP(item.IP)
						}
						b.refreshList()
					}
				},
				b.window)
		}
	})

	// Selection handler
	b.table.OnSelected = func(id widget.TableCellID) {
		selectedCell = id
		if id.Row > 0 && id.Row <= len(b.items) {
			ip := b.items[id.Row-1].IP
			dialog.ShowConfirm("Remove IP",
				fmt.Sprintf("Do you want to unblock %s?", ip),
				func(ok bool) {
					if ok {
						b.manager.UnblockIP(ip)
						b.refreshList()
					}
					// Deselect after handling
					b.table.UnselectAll()
				},
				b.window)
		}
	}

	toolbar := container.NewHBox(
		addButton,
		removeButton,
		clearButton,
	)

	// Layout
	b.container = container.NewBorder(
		toolbar, nil, nil, nil,
		container.NewPadded(b.table),
	)

	b.refreshList()
}

func (b *BlockListTab) showAddDialog() {
	if b.manager == nil {
		log.Error("Manager is nil in BlockListTab")
		return
	}

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter IP address")

	// اضافه کردن validation برای IP
	validate := func(s string) error {
		if s == "" {
			return fmt.Errorf("IP address cannot be empty")
		}
		// اختیاری: اضافه کردن validation برای فرمت IP
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

			// Validate IP
			if err := validate(input.Text); err != nil {
				dialog.ShowError(err, b.window)
				return
			}

			// Block IP
			b.manager.BlockIP(input.Text)

			// Refresh list
			b.refreshList()

			// Optional: Show success message
			dialog.ShowInformation("Success",
				fmt.Sprintf("IP %s has been blocked", input.Text),
				b.window)
		},
		b.window)
}

func (b *BlockListTab) refreshList() {
	if b.manager == nil {
		log.Error("Manager is nil in BlockListTab")
		return
	}

	blockedIPs := b.manager.GetBlockedIPs()
	b.items = make([]BlockedIP, len(blockedIPs))

	for i, item := range blockedIPs {
		b.items[i] = BlockedIP{
			IP:        item.IP,
			Timestamp: item.Timestamp,
		}
	}

	sort.Slice(b.items, func(i, j int) bool {
		return b.items[i].Timestamp.After(b.items[j].Timestamp)
	})

	b.table.Refresh()
}

func (b *BlockListTab) Container() fyne.CanvasObject {
	return b.container
}
