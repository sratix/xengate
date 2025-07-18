package components

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	log "github.com/sirupsen/logrus"
)

type LogEvent struct {
	Time    time.Time
	Level   log.Level
	Message string
}

type LogHandler struct {
	events    chan LogEvent
	logView   *widget.Entry
	scroll    *container.Scroll
	container fyne.CanvasObject
	maxLines  int
	lines     []string
}

func NewLogHandler(maxLines int) *LogHandler {
	handler := &LogHandler{
		events:   make(chan LogEvent, 1000),
		maxLines: maxLines,
		lines:    make([]string, 0, maxLines),
	}

	// ایجاد Entry چند خطی با اسکرول
	handler.logView = widget.NewMultiLineEntry()
	handler.logView.Disable()                                   // غیرقابل ویرایش
	handler.logView.TextStyle = fyne.TextStyle{Monospace: true} // فونت مونواسپیس برای نمایش بهتر

	// ایجاد اسکرول
	handler.scroll = container.NewScroll(handler.logView)
	handler.scroll.SetMinSize(fyne.NewSize(300, 400))

	// قرار دادن اسکرول در یک کارت
	handler.container = widget.NewCard(
		"Logs",
		"",
		handler.scroll,
	)

	go handler.processEvents()

	// اتصال به logrus
	log.AddHook(&logrusHook{handler: handler})

	return handler
}

func (h *LogHandler) processEvents() {
	for event := range h.events {
		timeStr := event.Time.Format("15:04:05")
		levelStr := fmt.Sprintf("[%-5s]", event.Level.String())
		line := fmt.Sprintf("%s %s %s", timeStr, levelStr, event.Message)

		h.lines = append(h.lines, line)
		if len(h.lines) > h.maxLines {
			h.lines = h.lines[1:]
		}

		// آپدیت UI در goroutine اصلی
		if h.logView != nil {
			text := strings.Join(h.lines, "\n")
			h.logView.SetText(text)

			// اسکرول به انتها
			h.scroll.ScrollToBottom()
		}
	}
}

func (h *LogHandler) AddEvent(level log.Level, msg string) {
	h.events <- LogEvent{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
	}
}

func (h *LogHandler) GetContainer() fyne.CanvasObject {
	return h.container
}

// پیاده‌سازی hook برای logrus
type logrusHook struct {
	handler *LogHandler
}

func (h *logrusHook) Levels() []log.Level {
	return log.AllLevels
}

func (h *logrusHook) Fire(entry *log.Entry) error {
	msg := entry.Message
	if len(entry.Data) > 0 {
		msg = fmt.Sprintf("%s %v", msg, entry.Data)
	}
	h.handler.AddEvent(entry.Level, msg)
	return nil
}
