package components

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type TimerPanel struct {
	widget.BaseWidget
	container  *fyne.Container
	background *canvas.Rectangle
	// shadow      *canvas.Rectangle
	playIcon    *widget.Icon
	stopIcon    *widget.Icon
	timeLabel   *canvas.Text
	isRunning   bool
	startTime   time.Time
	updateTimer chan struct{}

	// Position and size tracking
	containerWidth float32
	buttonWidth    float32
	expandedWidth  float32
	currentX       float32
	isAnimating    bool

	onClick func(bool)
}

func NewTimerPanel() *TimerPanel {
	p := &TimerPanel{
		playIcon:       widget.NewIcon(theme.MediaPlayIcon()),
		stopIcon:       widget.NewIcon(theme.MediaStopIcon()),
		timeLabel:      canvas.NewText("00:00:00", color.White),
		updateTimer:    make(chan struct{}),
		buttonWidth:    40,
		expandedWidth:  140,
		containerWidth: 110,
	}

	// Configure background
	p.background = canvas.NewRectangle(theme.ErrorColor())
	p.background.CornerRadius = 20

	// Configure shadow
	// p.shadow = canvas.NewRectangle(theme.ShadowColor())
	// p.shadow.CornerRadius = 22

	// Configure label
	p.timeLabel.Hide()
	p.timeLabel.TextStyle = fyne.TextStyle{
		Bold:      true,
		Monospace: true,
	}

	p.stopIcon.Hide()

	p.ExtendBaseWidget(p)
	p.createUI()
	return p
}

func (p *TimerPanel) createUI() {
	p.container = container.NewWithoutLayout()
	// p.container.Add(p.shadow)
	p.container.Add(p.background)
	p.container.Add(p.playIcon)
	p.container.Add(p.stopIcon)
	p.container.Add(p.timeLabel)

	// Initialize positions
	p.currentX = p.containerWidth - p.buttonWidth
	p.updateLayout()
}

func (p *TimerPanel) updateLayout() {
	width := p.buttonWidth
	if p.isRunning {
		width = p.expandedWidth
	}

	// Update background
	p.background.Resize(fyne.NewSize(width, 40))
	p.background.Move(fyne.NewPos(p.currentX, 0))

	// Update shadow
	// p.shadow.Resize(fyne.NewSize(width+4, 44))
	// p.shadow.Move(fyne.NewPos(p.currentX-2, -2))

	// Update icons
	iconSize := fyne.NewSize(20, 20)
	iconY := (40 - iconSize.Height) / 2

	if p.isRunning {
		p.stopIcon.Resize(iconSize)
		p.stopIcon.Move(fyne.NewPos(p.currentX+10, iconY))

		// Center time label
		labelSize := p.timeLabel.MinSize()
		labelX := p.currentX + (width-labelSize.Width)/2 + 10
		labelY := (40 - labelSize.Height) / 2
		p.timeLabel.Move(fyne.NewPos(labelX, labelY))
	} else {
		p.playIcon.Resize(iconSize)
		p.playIcon.Move(fyne.NewPos(p.currentX+10, iconY))
	}
}

func (p *TimerPanel) animate(expand bool) {
	if p.isAnimating {
		return
	}
	p.isAnimating = true

	targetX := p.containerWidth - p.buttonWidth
	if expand {
		targetX = p.containerWidth - p.expandedWidth
	}

	go func() {
		for p.currentX != targetX {
			step := float32(10)
			if p.currentX > targetX {
				p.currentX -= step
				if p.currentX < targetX {
					p.currentX = targetX
				}
			} else {
				p.currentX += step
				if p.currentX > targetX {
					p.currentX = targetX
				}
			}

			p.updateLayout()
			p.Refresh()
			time.Sleep(6 * time.Millisecond)
		}
		p.isAnimating = false
	}()
}

func (p *TimerPanel) Tapped(*fyne.PointEvent) {
	p.isRunning = !p.isRunning

	if p.isRunning {
		p.startTimer()
		p.playIcon.Hide()
		p.stopIcon.Show()
		p.timeLabel.Show()
		p.animate(true)
		p.onClick(true)
	} else {
		p.stopTimer()
		p.playIcon.Show()
		p.stopIcon.Hide()
		p.timeLabel.Hide()
		p.animate(false)
		p.onClick(false)
	}
}

func (p *TimerPanel) SetOnClick(callback func(bool)) {
	p.onClick = callback
}

func (p *TimerPanel) MinSize() fyne.Size {
	return fyne.NewSize(p.containerWidth, 40)
}

func (p *TimerPanel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.container)
}

func (p *TimerPanel) startTimer() {
	p.startTime = time.Now()

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if p.isRunning {
					elapsed := time.Since(p.startTime)
					p.updateElapsedTime(elapsed)
				}
			case <-p.updateTimer:
				return
			}
		}
	}()
}

func (p *TimerPanel) stopTimer() {
	p.updateTimer <- struct{}{}
	p.timeLabel.Text = "00:00:00"
}

func (p *TimerPanel) updateElapsedTime(elapsed time.Duration) {
	hours := int(elapsed.Hours())
	minutes := int(elapsed.Minutes()) % 60
	seconds := int(elapsed.Seconds()) % 60

	text := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	p.timeLabel.Text = text
	p.updateLayout() // Recenter label after text update
	// p.Refresh()
}

// Implement desktop.Hoverable
func (p *TimerPanel) MouseIn(*desktop.MouseEvent)    {}
func (p *TimerPanel) MouseOut()                      {}
func (p *TimerPanel) MouseMoved(*desktop.MouseEvent) {}
