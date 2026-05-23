package internal

import (
	"sync"
	"time"

	"github.com/atotto/clipboard"
)

type MonitorEvent struct {
	Result   Result
	Paused   bool
	DryRun   bool
	Error    error
	Type     string
}

type Monitor struct {
	Engine     *Engine
	Metrics    *Metrics
	Notifier   Notifier
	DryRun     bool
	paused     bool
	Events     chan MonitorEvent
	done       chan struct{}
	pauseCh    chan bool
	dryRunCh   chan bool
	mu         sync.RWMutex
	lastContent string
	autoDetect  bool
}

func NewMonitor(engine *Engine, metrics *Metrics, notifier Notifier, dryRun bool) *Monitor {
	return &Monitor{
		Engine:    engine,
		Metrics:   metrics,
		Notifier:  notifier,
		DryRun:    dryRun,
		Events:    make(chan MonitorEvent, 10),
		done:      make(chan struct{}),
		pauseCh:   make(chan bool),
		dryRunCh:  make(chan bool),
		autoDetect: false,
	}
}

func (m *Monitor) IsPaused() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.paused
}

func (m *Monitor) IsDryRun() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.DryRun
}

func (m *Monitor) IsAutoDetect() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.autoDetect
}

func (m *Monitor) Pause() {
	m.mu.Lock()
	m.paused = true
	m.mu.Unlock()
	m.Events <- MonitorEvent{Type: "paused", Paused: true}
}

func (m *Monitor) Resume() {
	m.mu.Lock()
	m.paused = false
	m.mu.Unlock()
	m.Events <- MonitorEvent{Type: "resumed", Paused: false}
}

func (m *Monitor) TogglePause() bool {
	m.mu.Lock()
	m.paused = !m.paused
	paused := m.paused
	m.mu.Unlock()
	if paused {
		m.Events <- MonitorEvent{Type: "paused", Paused: true}
	} else {
		m.Events <- MonitorEvent{Type: "resumed", Paused: false}
	}
	return paused
}

func (m *Monitor) ToggleDryRun() bool {
	m.mu.Lock()
	m.DryRun = !m.DryRun
	dryRun := m.DryRun
	m.mu.Unlock()
	return dryRun
}

func (m *Monitor) ToggleAutoDetect() bool {
	m.mu.Lock()
	m.autoDetect = !m.autoDetect
	ad := m.autoDetect
	m.mu.Unlock()
	return ad
}

func (m *Monitor) Stop() {
	close(m.done)
}

func (m *Monitor) Run() {
	lastContent, err := clipboard.ReadAll()
	if err != nil {
		m.Events <- MonitorEvent{Type: "error", Error: err}
	}
	m.lastContent = lastContent

	for {
		select {
		case <-m.done:
			return
		default:
		}

		m.mu.RLock()
		paused := m.paused
		m.mu.RUnlock()

		if paused {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		currentContent, err := clipboard.ReadAll()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		if currentContent != "" && currentContent != m.lastContent {
			m.mu.RLock()
			autoDetect := m.autoDetect
			m.mu.RUnlock()

			var result Result
			if autoDetect {
				cfg := GetConfig()
				result = m.Engine.ProcessWithAutoDetect(currentContent, cfg.Detect.Patterns)
			} else {
				result = m.Engine.Process(currentContent)
			}

			if len(result.TriggeredRules) > 0 {
				m.Metrics.Register(result.TriggeredRules)

				m.mu.RLock()
				dryRun := m.DryRun
				m.mu.RUnlock()

				if !dryRun {
					clipboard.WriteAll(result.ModifiedText)
				}

				time.Sleep(100 * time.Millisecond)
				m.lastContent = result.ModifiedText

				m.Notifier.Notify(result)

				m.Events <- MonitorEvent{
					Result: result,
					Paused: false,
					DryRun: dryRun,
					Type:   "detection",
				}
			} else {
				m.lastContent = currentContent
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}
