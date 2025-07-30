package ui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/shirou/gopsutil/process"
)

type MonitorTab struct {
	container *fyne.Container
	proc      *process.Process
	cpuValues []float64
	memValues []float64
	cpuChart  *Chart
	memChart  *Chart
}

type Chart struct {
	container  *fyne.Container
	graph      *fyne.Container
	data       []float64
	line       *canvas.Line
	threshold  *canvas.Line
	threshVal  float64
	gridLines  []*canvas.Line
	axisLabels []*canvas.Text
}

const (
	maxPoints   = 60
	chartWidth  = 300
	chartHeight = 200
	padding     = 40
	gridLinesH  = 5 // تعداد خطوط افقی
	gridLinesV  = 6 // تعداد خطوط عمودی
)

func NewMonitorTab(proc *process.Process) *MonitorTab {
	m := &MonitorTab{
		proc:      proc,
		cpuValues: make([]float64, 0, maxPoints),
		memValues: make([]float64, 0, maxPoints),
	}

	m.cpuChart = newChart("CPU Usage (%)",
		color.NRGBA{R: 0, G: 150, B: 255, A: 255},
		80,
		[]string{"0%", "25%", "50%", "75%", "100%"})

	m.memChart = newChart("Memory Usage (MB)",
		color.NRGBA{G: 200, B: 0, A: 255},
		500,
		[]string{"0MB", "250MB", "500MB", "750MB", "1000MB"})

	m.container = container.NewHBox(
		widget.NewLabel("Resource Monitor"),
		container.NewGridWithRows(2,
			m.cpuChart.container,
			m.memChart.container,
		),
	)

	go m.startMonitoring()
	return m
}

func newChart(title string, lineColor color.Color, thresholdValue float64, yLabels []string) *Chart {
	c := &Chart{
		data:       make([]float64, 0, maxPoints),
		gridLines:  make([]*canvas.Line, gridLinesH+gridLinesV),   // تعداد کل خطوط گرید
		axisLabels: make([]*canvas.Text, len(yLabels)+gridLinesV), // تعداد کل لیبل‌ها
		threshVal:  thresholdValue,
		line: &canvas.Line{
			StrokeColor: lineColor,
			StrokeWidth: 2,
		},
		threshold: &canvas.Line{
			StrokeColor: color.NRGBA{R: 255, A: 180},
			StrokeWidth: 1,
		},
	}

	// Background
	bg := canvas.NewRectangle(color.NRGBA{R: 32, G: 32, B: 32, A: 255})
	bg.Resize(fyne.NewSize(chartWidth, chartHeight))

	// Create graph container
	c.graph = container.NewWithoutLayout(bg)

	// Add grid lines
	// Horizontal grid lines
	for i := 0; i < gridLinesH; i++ {
		line := &canvas.Line{
			StrokeColor: color.NRGBA{R: 100, G: 100, B: 100, A: 100},
			StrokeWidth: 1,
		}
		y := float32(i) * chartHeight / float32(gridLinesH-1)
		line.Position1 = fyne.NewPos(padding, y)
		line.Position2 = fyne.NewPos(chartWidth, y)
		c.gridLines[i] = line
		c.graph.Add(line)

		// Y-axis labels
		if i < len(yLabels) {
			label := canvas.NewText(yLabels[len(yLabels)-1-i], color.NRGBA{R: 200, G: 200, B: 200, A: 200})
			label.TextSize = 10
			label.Move(fyne.NewPos(0, y-8))
			c.axisLabels[i] = label
			c.graph.Add(label)
		}
	}

	// Vertical grid lines and time labels
	for i := 0; i < gridLinesV; i++ {
		line := &canvas.Line{
			StrokeColor: color.NRGBA{R: 100, G: 100, B: 100, A: 100},
			StrokeWidth: 1,
		}
		x := padding + float32(i)*(chartWidth-padding)/float32(gridLinesV-1)
		line.Position1 = fyne.NewPos(x, 0)
		line.Position2 = fyne.NewPos(x, chartHeight-padding)
		c.gridLines[i+gridLinesH] = line
		c.graph.Add(line)

		// X-axis time labels
		timeLabel := canvas.NewText(fmt.Sprintf("%ds", (gridLinesV-1-i)*12), color.NRGBA{R: 200, G: 200, B: 200, A: 200})
		timeLabel.TextSize = 10
		timeLabel.Move(fyne.NewPos(x-10, chartHeight-padding+5))
		c.axisLabels[len(yLabels)+i] = timeLabel
		c.graph.Add(timeLabel)
	}

	// Add data line and threshold line
	c.graph.Add(c.line)
	c.graph.Add(c.threshold)

	// Title
	titleLabel := widget.NewLabel(title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	c.container = container.NewVBox(
		titleLabel,
		c.graph,
	)

	return c
}

func (m *MonitorTab) startMonitoring() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		// Get CPU usage
		if cpu, err := m.proc.CPUPercent(); err == nil {
			m.cpuValues = append(m.cpuValues, cpu)
			if len(m.cpuValues) > maxPoints {
				m.cpuValues = m.cpuValues[1:]
			}
			m.updateChart(m.cpuChart, m.cpuValues, 100)
		}

		// Get Memory usage
		if mem, err := m.proc.MemoryInfo(); err == nil {
			memMB := float64(mem.RSS) / 1024 / 1024
			m.memValues = append(m.memValues, memMB)
			if len(m.memValues) > maxPoints {
				m.memValues = m.memValues[1:]
			}
			m.updateChart(m.memChart, m.memValues, 1000)
		}
	}
}

func (m *MonitorTab) updateChart(chart *Chart, values []float64, maxValue float64) {
	if len(values) < 2 {
		return
	}

	// Calculate positions for data line
	points := make([]fyne.Position, len(values))
	for i, v := range values {
		x := padding + float32(i)*(chartWidth-padding)/float32(maxPoints)
		y := float32(chartHeight-padding) - (float32(v) * (chartHeight - padding) / float32(maxValue))
		points[i] = fyne.NewPos(x, y)
	}

	// Update data line
	if len(points) >= 2 {
		chart.line.Position1 = points[0]
		chart.line.Position2 = points[len(points)-1]
		chart.line.Refresh()
	}

	// Update threshold line - استفاده از threshVal به جای threshold
	threshY := float32(chartHeight-padding) - (float32(chart.threshVal) * (chartHeight - padding) / float32(maxValue))
	chart.threshold.Position1 = fyne.NewPos(padding, threshY)
	chart.threshold.Position2 = fyne.NewPos(chartWidth, threshY)
	chart.threshold.Refresh()
}

func (m *MonitorTab) Container() fyne.CanvasObject {
	return m.container
}
