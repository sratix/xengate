// ui/dialogs/pingdialog.go
package dialogs

import (
	"fmt"
	"net"
	"time"

	"xengate/internal/models"
	myTheme "xengate/ui/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type PingDialog struct {
	widget.BaseWidget
	conn        *models.Connection
	window      fyne.Window
	content     *fyne.Container
	statusLabel *widget.Label
	resultLabel *widget.Label
	progressBar *widget.ProgressBarInfinite
	onDismiss   func()
	cancelChan  chan struct{}
}

func NewPingDialog(conn *models.Connection, window fyne.Window) *PingDialog {
	d := &PingDialog{
		conn:       conn,
		window:     window,
		cancelChan: make(chan struct{}),
		// onDismiss:  func() {},
	}
	d.ExtendBaseWidget(d)
	d.createContent()
	return d
}

func (d *PingDialog) SetOnDismiss(callback func()) {
	d.onDismiss = callback
}

func (d *PingDialog) Dismiss() {
	if d.onDismiss != nil {
		d.onDismiss()
	}
}

func (d *PingDialog) createContent() {
	// Create header
	header := widget.NewLabelWithStyle(
		fmt.Sprintf("Pinging %s:%s", d.conn.Address, d.conn.Port),
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	// Create status label
	d.statusLabel = widget.NewLabel("Connecting...")
	d.statusLabel.Alignment = fyne.TextAlignCenter

	// Create result label
	d.resultLabel = widget.NewLabel("")
	d.resultLabel.Hide()
	d.resultLabel.Alignment = fyne.TextAlignCenter

	// Create progress bar
	d.progressBar = widget.NewProgressBarInfinite()

	// Create cancel button
	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		close(d.cancelChan)
		if d.onDismiss != nil {
			d.onDismiss()
		}
	})
	cancelBtn.Importance = widget.DangerImportance

	// Create container
	d.content = container.NewVBox(
		header,
		widget.NewSeparator(),
		container.NewPadded(
			container.NewVBox(
				d.statusLabel,
				d.progressBar,
				d.resultLabel,
			),
		),
		container.NewHBox(
			layout.NewSpacer(),
			cancelBtn,
			layout.NewSpacer(),
		),
	)

	// Style container
	bg := myTheme.NewThemedRectangle(myTheme.ColorNamePageBackground)
	d.content = container.NewStack(bg, d.content)
	d.content.Resize(fyne.NewSize(300, 200))
}

func (d *PingDialog) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(d.content)
}

func (d *PingDialog) Show() {
	pop := widget.NewModalPopUp(d, d.window.Canvas())
	oldDismiss := d.onDismiss
	d.onDismiss = func() {
		pop.Hide()
		if oldDismiss != nil {
			oldDismiss()
		}
	}
	pop.Show()

	// Start pinging
	go d.startPinging()
}

func (d *PingDialog) startPinging() {
	timeout := time.Second * 5
	address := net.JoinHostPort(d.conn.Address, d.conn.Port)

	var results []time.Duration
	maxPings := 4

	for i := 0; i < maxPings; i++ {
		select {
		case <-d.cancelChan:
			return
		default:
			start := time.Now()
			conn, err := net.DialTimeout("tcp", address, timeout)
			if err != nil {
				d.updateStatus(fmt.Sprintf("Failed to connect: %v", err), true)
				return
			}

			duration := time.Since(start)
			results = append(results, duration)

			d.updateStatus(fmt.Sprintf("Ping %d: %dms", i+1, duration.Milliseconds()), false)

			conn.Close()
			time.Sleep(time.Second)
		}
	}

	// Calculate average
	var total time.Duration
	for _, dur := range results {
		total += dur
	}
	avg := total / time.Duration(len(results))

	// Show final results
	d.showResults(fmt.Sprintf(
		"Ping statistics for %s:%s\n"+
			"    Packets: Sent = %d, Received = %d\n"+
			"    Average = %dms",
		d.conn.Address, d.conn.Port,
		len(results), len(results),
		avg.Milliseconds(),
	))
}

func (d *PingDialog) updateStatus(status string, isError bool) {
	// استفاده از Timer برای اجرای تغییرات UI در گوروتین اصلی
	time.AfterFunc(time.Millisecond, func() {
		d.statusLabel.SetText(status)
		if isError {
			d.progressBar.Hide()
			d.statusLabel.TextStyle = fyne.TextStyle{Bold: true}
		}
		canvas.Refresh(d.statusLabel)
	})
}

func (d *PingDialog) showResults(results string) {
	// استفاده از Timer برای اجرای تغییرات UI در گوروتین اصلی
	time.AfterFunc(time.Millisecond, func() {
		d.statusLabel.Hide()
		d.progressBar.Hide()
		d.resultLabel.SetText(results)
		d.resultLabel.Show()
		canvas.Refresh(d.content)
	})
}

func (d *PingDialog) OnDismiss(callback func()) {
	d.onDismiss = callback
}
