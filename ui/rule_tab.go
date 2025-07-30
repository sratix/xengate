// ui/rules_tab.go
package ui

import (
	"fmt"
	"sort"
	"time"

	"xengate/internal/tunnel"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type RulesTab struct {
	window        fyne.Window
	accessControl *tunnel.AccessControl
	container     *fyne.Container
	table         *widget.Table
	rules         []*tunnel.AccessRule
	autoSave      bool
}

func NewRulesTab(window fyne.Window, accessControl *tunnel.AccessControl) *RulesTab {
	tab := &RulesTab{
		window:        window,
		accessControl: accessControl,
		autoSave:      true,
	}
	tab.initUI()
	return tab
}

func (r *RulesTab) initUI() {
	// Create table
	r.table = widget.NewTable(
		func() (int, int) {
			// Add 1 to row count for header
			return len(r.rules) + 1, 5
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)

			// Handle header row
			if id.Row == 0 {
				headers := []string{"Title", "IP Address", "Daily Limit", "Remaining", "Last Access"}
				if id.Col < len(headers) {
					label.SetText(headers[id.Col])
					label.TextStyle = fyne.TextStyle{Bold: true}
				}
				return
			}

			// Adjust row index for data (subtract header row)
			dataRow := id.Row - 1
			if dataRow < len(r.rules) {
				rule := r.rules[dataRow]
				_, status := r.accessControl.GetRule(rule.ID)

				switch id.Col {
				case 0:
					label.SetText(rule.Title)
				case 1:
					label.SetText(rule.IP)
				case 2:
					if rule.IsMaster {
						label.SetText("Unlimited")
					} else {
						label.SetText(rule.DailyLimit.String())
					}
				case 3:
					if rule.IsMaster {
						label.SetText("Master")
					} else if status.IsBlocked {
						label.SetText("Blocked")
					} else {
						remaining := rule.DailyLimit - status.UsedTime
						if remaining < 0 {
							remaining = 0
						}
						label.SetText(remaining.String())
					}
				case 4:
					if !status.LastAccess.IsZero() {
						label.SetText(status.LastAccess.Format("15:04:05"))
					} else {
						label.SetText("-")
					}
				}
			}
		},
	)

	// Set column widths
	r.table.SetColumnWidth(0, 120) // Title
	r.table.SetColumnWidth(1, 120) // IP Address
	r.table.SetColumnWidth(2, 100) // Daily Limit
	r.table.SetColumnWidth(3, 100) // Remaining
	r.table.SetColumnWidth(4, 100) // Last Access

	// Double click handler (adjust row index for header)
	r.table.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 && id.Row <= len(r.rules) {
			r.showDetailsDialog(r.rules[id.Row-1])
		}
		// Deselect after handling
		r.table.Select(widget.TableCellID{})
	}

	// Toolbar
	addButton := widget.NewButtonWithIcon("Add Rule", theme.ContentAddIcon(), func() {
		r.showAddDialog()
	})

	autoSaveCheck := widget.NewCheck("Auto Save", func(checked bool) {
		r.autoSave = checked
	})
	autoSaveCheck.Checked = true

	saveButton := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		if err := r.accessControl.SaveRules(); err != nil {
			dialog.ShowError(err, r.window)
		} else {
			dialog.ShowInformation("Success", "Rules saved successfully", r.window)
		}
	})

	toolbar := container.NewHBox(
		addButton,
		widget.NewSeparator(),
		autoSaveCheck,
		saveButton,
	)

	// Layout
	r.container = container.NewBorder(
		toolbar, nil, nil, nil,
		container.NewPadded(r.table),
	)

	// Initial data load
	r.refreshTable()

	// Start refresh timer
	go r.periodicRefresh()
}

func (r *RulesTab) showDetailsDialog(rule *tunnel.AccessRule) {
	_, status := r.accessControl.GetRule(rule.ID)

	details := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Title: %s", rule.Title)),
		widget.NewLabel(fmt.Sprintf("IP: %s", rule.IP)),
		widget.NewLabel(fmt.Sprintf("Daily Limit: %s", rule.DailyLimit)),
		widget.NewLabel(fmt.Sprintf("Description: %s", rule.Description)),
		widget.NewLabel(fmt.Sprintf("Created: %s", rule.CreatedAt.Format("2006-01-02 15:04:05"))),
		widget.NewLabel(fmt.Sprintf("Updated: %s", rule.UpdatedAt.Format("2006-01-02 15:04:05"))),
		widget.NewSeparator(),
		widget.NewLabel(fmt.Sprintf("Used Time: %s", status.UsedTime)),
		widget.NewLabel(fmt.Sprintf("Status: %s", getStatusText(rule, status))),
	)

	actions := container.NewHBox(
		widget.NewButton("Edit", func() {
			dialog.ShowCustom("", "", nil, r.window) // Hide details dialog
			r.showEditDialog(rule)
		}),
		widget.NewButton("Reset Time", func() {
			if err := r.accessControl.ResetRule(rule.ID); err != nil {
				dialog.ShowError(err, r.window)
			} else {
				r.refreshTable()
				dialog.ShowInformation("Success", "Time has been reset", r.window)
			}
		}),
		widget.NewButton("Delete", func() {
			dialog.ShowConfirm("Confirm Delete",
				"Are you sure you want to delete this rule?",
				func(confirm bool) {
					if confirm {
						if err := r.accessControl.DeleteRule(rule.ID); err != nil {
							dialog.ShowError(err, r.window)
						} else {
							r.refreshTable()
						}
					}
				}, r.window)
		}),
	)

	content := container.NewVBox(
		details,
		widget.NewSeparator(),
		actions,
	)

	dialog.ShowCustom("Rule Details", "Close", content, r.window)
}

func (r *RulesTab) showAddDialog() {
	titleEntry := widget.NewEntry()
	ipEntry := widget.NewEntry()
	isMasterCheck := widget.NewCheck("Master IP", nil)
	limitEntry := widget.NewEntry()
	limitEntry.SetText("1h")
	descEntry := widget.NewMultiLineEntry()

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Title", Widget: titleEntry},
			{Text: "IP Address", Widget: ipEntry},
			{Text: "Is Master", Widget: isMasterCheck},
			{Text: "Daily Limit", Widget: limitEntry},
			{Text: "Description", Widget: descEntry},
		},
		OnSubmit: func() {
			limit, err := time.ParseDuration(limitEntry.Text)
			if err != nil {
				dialog.ShowError(err, r.window)
				return
			}

			rule := &tunnel.AccessRule{
				Title:       titleEntry.Text,
				IP:          ipEntry.Text,
				IsMaster:    isMasterCheck.Checked,
				DailyLimit:  limit,
				Description: descEntry.Text,
			}

			if err := r.accessControl.AddRule(rule); err != nil {
				dialog.ShowError(err, r.window)
				return
			}

			r.refreshTable()
			if r.autoSave {
				r.accessControl.SaveRules()
			}
		},
	}

	dialog.ShowCustom("Add New Rule", "Add", form, r.window)
}

func (r *RulesTab) showEditDialog(rule *tunnel.AccessRule) {
	titleEntry := widget.NewEntry()
	titleEntry.SetText(rule.Title)

	ipEntry := widget.NewEntry()
	ipEntry.SetText(rule.IP)

	isMasterCheck := widget.NewCheck("Master IP", nil)
	isMasterCheck.Checked = rule.IsMaster

	limitEntry := widget.NewEntry()
	limitEntry.SetText(rule.DailyLimit.String())

	descEntry := widget.NewMultiLineEntry()
	descEntry.SetText(rule.Description)

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Title", Widget: titleEntry},
			{Text: "IP Address", Widget: ipEntry},
			{Text: "Is Master", Widget: isMasterCheck},
			{Text: "Daily Limit", Widget: limitEntry},
			{Text: "Description", Widget: descEntry},
		},
		OnSubmit: func() {
			limit, err := time.ParseDuration(limitEntry.Text)
			if err != nil {
				dialog.ShowError(err, r.window)
				return
			}

			updatedRule := &tunnel.AccessRule{
				ID:          rule.ID,
				Title:       titleEntry.Text,
				IP:          ipEntry.Text,
				IsMaster:    isMasterCheck.Checked,
				DailyLimit:  limit,
				Description: descEntry.Text,
			}

			if err := r.accessControl.UpdateRule(updatedRule); err != nil {
				dialog.ShowError(err, r.window)
				return
			}

			r.refreshTable()
			if r.autoSave {
				r.accessControl.SaveRules()
			}
		},
	}

	dialog.ShowCustom("Edit Rule", "Save", form, r.window)
}

func (r *RulesTab) refreshTable() {
	r.rules = r.accessControl.GetAllRules()
	sort.Slice(r.rules, func(i, j int) bool {
		return r.rules[i].Title < r.rules[j].Title
	})
	r.table.Refresh()
}

func (r *RulesTab) periodicRefresh() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		r.table.Refresh()
	}
}

func (r *RulesTab) Container() fyne.CanvasObject {
	return r.container
}

func getStatusText(rule *tunnel.AccessRule, status *tunnel.AccessStatus) string {
	if rule.IsMaster {
		return "Master (Unlimited)"
	}
	if status.IsBlocked {
		return "Blocked"
	}
	if status.ActiveSince != nil {
		return "Active"
	}
	return "Inactive"
}
