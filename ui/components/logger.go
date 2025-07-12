package components

import (
	"fmt"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/sirupsen/logrus"
)

type LogWidget struct {
	widget.BaseWidget
	textEntry  *widget.Entry
	maxLines   int
	logs       []string
	mu         sync.Mutex
	autoscroll bool
	content    *fyne.Container
}

type LogHook struct {
	widget *LogWidget
}

func (h *LogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *LogHook) Fire(entry *logrus.Entry) error {
	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	level := strings.ToUpper(entry.Level.String())
	msg := fmt.Sprintf("[%s] [%s] %s", timestamp, level, entry.Message)

	// Add fields if any
	if len(entry.Data) > 0 {
		fields := make([]string, 0, len(entry.Data))
		for k, v := range entry.Data {
			fields = append(fields, fmt.Sprintf("%s=%v", k, v))
		}
		msg += " " + strings.Join(fields, " ")
	}

	h.widget.AddLog(msg)
	return nil
}

func NewLogWidget(maxLines int) *LogWidget {
	w := &LogWidget{
		maxLines:   maxLines,
		autoscroll: true,
	}
	w.ExtendBaseWidget(w)
	w.createUI()

	// Add hook to logrus
	logrus.AddHook(&LogHook{widget: w})

	return w
}

func (w *LogWidget) createUI() {
	w.textEntry = widget.NewMultiLineEntry()
	w.textEntry.TextStyle = fyne.TextStyle{Monospace: true}
	w.textEntry.Wrapping = fyne.TextWrapWord
	w.textEntry.MultiLine = true
	w.textEntry.SetMinRowsVisible(10)

	// Create autoscroll checkbox
	autoScrollCheck := widget.NewCheck("Auto-scroll", func(checked bool) {
		w.autoscroll = checked
	})
	autoScrollCheck.SetChecked(true)

	// Create clear button
	clearBtn := widget.NewButton("Clear Logs", func() {
		w.Clear()
	})

	// Create toolbar
	toolbar := container.NewHBox(
		autoScrollCheck,
		clearBtn,
	)

	w.content = container.NewBorder(
		toolbar,
		nil,
		nil,
		nil,
		container.NewScroll(w.textEntry), // Wrap in scroll container
	)
}

func (w *LogWidget) AddLog(log string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.logs = append(w.logs, log)
	if len(w.logs) > w.maxLines {
		w.logs = w.logs[len(w.logs)-w.maxLines:]
	}

	text := strings.Join(w.logs, "\n")
	w.textEntry.SetText(text)

	if w.autoscroll {
		// Scroll to bottom
		w.textEntry.CursorColumn = len(text)
		w.textEntry.Refresh()
	}
}

func (w *LogWidget) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.logs = nil
	w.textEntry.SetText("")
}

func (w *LogWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.content)
}
