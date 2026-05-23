package main

import (
	"context"
	"fmt"
	"go-clipboard-monitor/internal"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		<-sigs
		fmt.Println("\n🛑 Cerrando Clipboard Monitor...")
		cancel()
	}()

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

	time.Sleep(200 * time.Millisecond)

	tui := internal.NewTUI(monitor, metrics, engine, history, desktopNotif)

	p := tea.NewProgram(tui, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error al ejecutar TUI: %v\n", err)
		os.Exit(1)
	}
}
