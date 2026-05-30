package main

import (
	"fmt"
	"go-clipboard-monitor/internal"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	err := internal.AcquireLock()
	if err != nil {
		fmt.Println(internal.BoxTop)
		fmt.Printf("  %s %s %s\n", internal.ColorYellow, err, internal.ColorReset)
		fmt.Println(internal.BoxBottom)
		return
	}
	defer internal.ReleaseLock()

	rules := internal.GetEnabledRules()
	engine := internal.NewEngine(rules)
	metrics := internal.NewMetrics()
	logger := internal.NewLogger()
	desktopNotif := &internal.DesktopNotifier{Enabled: true}
	notifier := &internal.CompositeNotifier{
		Notifiers: []internal.Notifier{
			&internal.TuiNotifier{Logger: logger},
			desktopNotif,
		},
	}
	history := internal.NewHistory(100)

	monitor := internal.NewMonitor(engine, metrics, notifier, false)
	go monitor.Run()

	tui := internal.NewTUI(monitor, metrics, engine, history, desktopNotif)

	p := tea.NewProgram(tui, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error al ejecutar TUI: %v\n", err)
		os.Exit(1)
	}
}
