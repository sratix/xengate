package dialogs

import (
	"fmt"
	"strconv"
	"strings"

	"xengate/internal/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type TunSettingsDialog struct {
	dialog   *widget.PopUp
	config   *models.TunConfig
	onSave   func(*models.TunConfig)
	onCancel func()
}

func NewTunSettingsDialog(config *models.TunConfig, parent fyne.Window) *TunSettingsDialog {
	d := &TunSettingsDialog{
		config: &models.TunConfig{ // Create a copy of config
			// Enabled:    config.Enabled,
			DeviceName: config.DeviceName,
			Address:    config.Address,
			Gateway:    config.Gateway,
			MTU:        config.MTU,
			DNSServers: make([]string, len(config.DNSServers)),
		},
	}
	copy(d.config.DNSServers, config.DNSServers)

	enabledCheck := widget.NewCheck("Enable TUN", func(enabled bool) {
		// d.config.Enabled = enabled
	})
	// enabledCheck.Checked = config.Enabled

	deviceEntry := widget.NewEntry()
	deviceEntry.SetText(config.DeviceName)
	deviceEntry.OnChanged = func(s string) {
		d.config.DeviceName = s
	}

	addressEntry := widget.NewEntry()
	addressEntry.SetText(config.Address)
	addressEntry.OnChanged = func(s string) {
		d.config.Address = s
	}

	gatewayEntry := widget.NewEntry()
	gatewayEntry.SetText(config.Gateway)
	gatewayEntry.OnChanged = func(s string) {
		d.config.Gateway = s
	}

	mtuEntry := widget.NewEntry()
	mtuEntry.SetText(fmt.Sprintf("%d", config.MTU))
	mtuEntry.OnChanged = func(s string) {
		val, _ := strconv.Atoi(s)
		d.config.MTU = val
	}

	// DNS Servers Entry
	dnsEntry := widget.NewEntry()
	dnsEntry.SetText(strings.Join(config.DNSServers, ","))
	dnsEntry.OnChanged = func(s string) {
		d.config.DNSServers = strings.Split(s, ",")
	}

	form := widget.NewForm(
		&widget.FormItem{Text: "Device Name", Widget: deviceEntry},
		&widget.FormItem{Text: "IP Address", Widget: addressEntry},
		&widget.FormItem{Text: "Gateway", Widget: gatewayEntry},
		&widget.FormItem{Text: "MTU", Widget: mtuEntry},
		&widget.FormItem{Text: "DNS Servers (comma-separated)", Widget: dnsEntry},
	)

	// Save and Cancel buttons
	saveButton := widget.NewButton("Save", func() {
		if d.onSave != nil {
			d.onSave(d.config)
		}
		d.dialog.Hide()
	})

	cancelButton := widget.NewButton("Cancel", func() {
		if d.onCancel != nil {
			d.onCancel()
		}
		d.dialog.Hide()
	})

	buttons := container.NewHBox(
		saveButton,
		cancelButton,
	)

	content := container.NewVBox(
		enabledCheck,
		form,
		buttons,
	)

	d.dialog = widget.NewModalPopUp(content, parent.Canvas())
	return d
}

func (d *TunSettingsDialog) Show() {
	d.dialog.Show()
}

func (d *TunSettingsDialog) Hide() {
	d.dialog.Hide()
}

func (d *TunSettingsDialog) SetOnSave(callback func(*models.TunConfig)) {
	d.onSave = callback
}

func (d *TunSettingsDialog) SetOnCancel(callback func()) {
	d.onCancel = callback
}
