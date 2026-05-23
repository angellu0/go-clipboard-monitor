package internal

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
)

type Notifier interface {
	Notify(result Result)
}

type ConsoleNotifier struct {
	Logger *log.Logger
}

func (c *ConsoleNotifier) Notify(result Result) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Print("\033[2K\r")
	fmt.Println(BoxTop)
	fmt.Printf("  🔒 Contenido sensible detectado - %s\n", timestamp)
	fmt.Printf("\n  Reglas activadas: %d\n", len(result.TriggeredRules))
	for _, rule := range result.TriggeredRules {
		fmt.Printf("   • %s\n", rule)
	}
	fmt.Printf("  Contenido protegido y reemplazado\n")
	fmt.Println(BoxBottom)
	fmt.Print("> ")

	logMessage := strings.Builder{}
	logMessage.WriteString(fmt.Sprintf("Contenido sensible detectado - Reglas: [%s]",
		strings.Join(result.TriggeredRules, ", ")))
	c.Logger.Println(logMessage.String())
}

type TuiNotifier struct {
	Logger *log.Logger
}

func (n *TuiNotifier) Notify(result Result) {
	logMessage := strings.Builder{}
	logMessage.WriteString(fmt.Sprintf("Contenido sensible detectado - Reglas: [%s]",
		strings.Join(result.TriggeredRules, ", ")))
	n.Logger.Println(logMessage.String())
}

type DesktopNotifier struct {
	Enabled bool
}

func (n *DesktopNotifier) Notify(result Result) {
	if !n.Enabled {
		return
	}
	title := "🔒 Clipboard Monitor"
	message := fmt.Sprintf("Detectado: %s", strings.Join(result.TriggeredRules, ", "))
	beeep.Notify(title, message, "")
}

type CompositeNotifier struct {
	Notifiers []Notifier
}

func (c *CompositeNotifier) Notify(result Result) {
	for _, n := range c.Notifiers {
		n.Notify(result)
	}
}

type DetectionEvent struct {
	Timestamp      time.Time
	OriginalText   string
	ModifiedText   string
	TriggeredRules []string
}

type History struct {
	events []DetectionEvent
	max    int
}

func NewHistory(max int) *History {
	return &History{
		events: make([]DetectionEvent, 0, max),
		max:    max,
	}
}

func (h *History) Add(event DetectionEvent) {
	if len(h.events) >= h.max {
		h.events = h.events[1:]
	}
	h.events = append(h.events, event)
}

func (h *History) Events() []DetectionEvent {
	return h.events
}

func (h *History) Clear() {
	h.events = make([]DetectionEvent, 0, h.max)
}
