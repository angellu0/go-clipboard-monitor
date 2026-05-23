package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tab int

const (
	tabDashboard tab = iota
	tabRules
	tabHistory
)

var tabNames = []string{"Dashboard", "Rules", "History"}

type paletteCmd struct {
	name        string
	description string
}

var paletteCommands = []paletteCmd{
	{"add", "Add a replacement rule: add \"buscar\" \"reemplazo\""},
	{"del", "Delete a rule: del \"buscar\""},
	{"enable", "Enable a rule: enable \"buscar\""},
	{"disable", "Disable a rule: disable \"buscar\""},
	{"toggle", "Toggle a rule on/off: toggle \"buscar\""},
	{"pause", "Pause clipboard monitoring"},
	{"resume", "Resume clipboard monitoring"},
	{"dryrun", "Toggle dry-run mode (simulation only)"},
	{"notify", "Toggle desktop notifications"},
	{"clear", "Clear detection history"},
	{"scan", "Scan a file for sensitive data"},
	{"export", "Export rules to JSON file"},
	{"import", "Import rules from JSON file"},
	{"help", "Show available commands"},
	{"quit", "Exit the application"},
}

type model struct {
	activeTab tab

	monitor       *Monitor
	metrics       *Metrics
	engine        *Engine
	history       *History
	desktopNotif  *DesktopNotifier

	paused        bool
	dryRun        bool
	autoDetect    bool
	desktopNotifs bool
	statusMsg     string

	input textinput.Model

	ruleKeys      []string
	ruleCursor    int
	showPalette   bool
	paletteCursor int

	width     int
	height    int
	quitting  bool
	fullWidth int
}

var (
	styleSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	styleWarning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	styleDim = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	styleValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("87"))
)

func NewTUI(monitor *Monitor, metrics *Metrics, engine *Engine, history *History, desktopNotif *DesktopNotifier) *model {
	ti := textinput.New()
	ti.Placeholder = "Escribe un comando (help para ayuda)"
	ti.Prompt = "> "
	ti.CharLimit = 256
	ti.Width = 60
	ti.Focus()

	m := &model{
		monitor:       monitor,
		metrics:       metrics,
		engine:        engine,
		history:       history,
		desktopNotif:  desktopNotif,
		paused:        false,
		dryRun:        monitor.DryRun,
		autoDetect:    false,
		desktopNotifs: desktopNotif != nil && desktopNotif.Enabled,
		input:         ti,
		ruleKeys:      make([]string, 0),
		showPalette:   false,
		paletteCursor: 0,
		width:         80,
		height:        24,
		fullWidth:     70,
	}
	m.refreshRuleKeys()
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		waitForMonitorEvent(m.monitor.Events),
		monitorTick(),
	)
}

func waitForMonitorEvent(events <-chan MonitorEvent) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-events
		if !ok {
			return nil
		}
		return evt
	}
}

func monitorTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.fullWidth = msg.Width - 4
		if m.fullWidth < 40 {
			m.fullWidth = 40
		}
		m.input.Width = m.fullWidth - 4
		return m, nil

	case tea.KeyMsg:
		if m.showPalette {
			switch msg.Type {
			case tea.KeyUp:
				if m.paletteCursor > 0 {
					m.paletteCursor--
				}
				return m, nil

			case tea.KeyDown:
				filtered := m.filteredPalette()
				if m.paletteCursor < len(filtered)-1 {
					m.paletteCursor++
				}
				return m, nil

			case tea.KeyEnter:
				filtered := m.filteredPalette()
				if len(filtered) > 0 {
					cmd := filtered[m.paletteCursor]
					m.input.SetValue(cmd.name + " ")
					m.input.SetCursor(len(cmd.name) + 1)
					m.showPalette = false
				}
				return m, nil

			case tea.KeyEscape:
				m.showPalette = false
				m.input.SetValue("")
				return m, nil

			case tea.KeyBackspace:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				if m.input.Value() == "" || !strings.HasPrefix(m.input.Value(), "/") {
					m.showPalette = false
				}
				return m, cmd

			case tea.KeyRunes:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				if !strings.HasPrefix(m.input.Value(), "/") {
					m.showPalette = false
				}
				return m, cmd

			default:
				return m, nil
			}
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			m.monitor.Stop()
			return m, tea.Quit

		case tea.KeyLeft:
			if m.activeTab > tabDashboard {
				m.activeTab--
			}
			m.refreshRuleKeys()

		case tea.KeyRight:
			if m.activeTab < tabHistory {
				m.activeTab++
			}
			m.refreshRuleKeys()

		case tea.KeyTab:
			if m.activeTab < tabHistory {
				m.activeTab++
			} else {
				m.activeTab = tabDashboard
			}
			m.refreshRuleKeys()

		case tea.KeyShiftTab:
			if m.activeTab > tabDashboard {
				m.activeTab--
			} else {
				m.activeTab = tabHistory
			}
			m.refreshRuleKeys()

		case tea.KeyUp:
			if (m.activeTab == tabRules || m.width >= 110) && len(m.ruleKeys) > 0 {
				if m.ruleCursor > 0 {
					m.ruleCursor--
				}
			}
			return m, nil

		case tea.KeyDown:
			if (m.activeTab == tabRules || m.width >= 110) && len(m.ruleKeys) > 0 {
				if m.ruleCursor < len(m.ruleKeys)-1 {
					m.ruleCursor++
				}
			}
			return m, nil

		case tea.KeyEnter:
			if (m.activeTab == tabRules || m.width >= 110) && m.input.Value() == "" && len(m.ruleKeys) > 0 && m.ruleCursor < len(m.ruleKeys) {
				search := m.ruleKeys[m.ruleCursor]
				enabled := ToggleRule(search)
				m.engine.UpdateRules(GetEnabledRules())
				if enabled {
					m.statusMsg = fmt.Sprintf("🟢 Regla habilitada: %s", search)
				} else {
					m.statusMsg = fmt.Sprintf("🔴 Regla deshabilitada: %s", search)
				}
				return m, nil
			}
			cmd := m.processInput()
			return m, cmd

		case tea.KeyEscape:
			m.input.SetValue("")

		case tea.KeyRunes:
			if !m.showPalette && m.input.Value() == "" && len(msg.Runes) == 1 && msg.Runes[0] == '/' {
				m.showPalette = true
				m.paletteCursor = 0
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}

			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd

		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		return m, nil

	case MonitorEvent:
		switch msg.Type {
		case "detection":
			m.history.Add(DetectionEvent{
				Timestamp:       time.Now(),
				OriginalText:    msg.Result.OriginalText,
				ModifiedText:    msg.Result.ModifiedText,
				TriggeredRules:  msg.Result.TriggeredRules,
			})
			m.statusMsg = fmt.Sprintf("🔒 Detectado: %s", strings.Join(msg.Result.TriggeredRules, ", "))

		case "paused":
			m.paused = true
			m.statusMsg = "⏸️ Monitor pausado"

		case "resumed":
			m.paused = false
			m.statusMsg = "▶️ Monitor reanudado"

		case "error":
			m.statusMsg = fmt.Sprintf("❌ Error: %v", msg.Error)
		}
		return m, waitForMonitorEvent(m.monitor.Events)

	case tickMsg:
		m.refreshState()
		return m, monitorTick()
	}

	return m, nil
}

func (m *model) refreshState() {
	m.paused = m.monitor.IsPaused()
	m.dryRun = m.monitor.IsDryRun()
	m.autoDetect = m.monitor.IsAutoDetect()
}

func (m *model) refreshRuleKeys() {
	rules := ListRulesMap()
	m.ruleKeys = make([]string, 0, len(rules))
	for k := range rules {
		m.ruleKeys = append(m.ruleKeys, k)
	}
	sort.Strings(m.ruleKeys)
	if m.ruleCursor >= len(m.ruleKeys) {
		m.ruleCursor = 0
	}
}

func (m model) filteredPalette() []paletteCmd {
	filter := strings.TrimPrefix(m.input.Value(), "/")
	if filter == "" {
		return paletteCommands
	}
	var result []paletteCmd
	for _, cmd := range paletteCommands {
		if strings.HasPrefix(cmd.name, filter) {
			result = append(result, cmd)
		}
	}
	return result
}

func parseInput(input string) []string {
	re := regexp.MustCompile(`"([^"]+)"|(\S+)`)
	matches := re.FindAllStringSubmatch(input, -1)
	var args []string
	for _, m := range matches {
		if m[1] != "" {
			args = append(args, m[1])
		} else if m[2] != "" {
			args = append(args, m[2])
		}
	}
	return args
}

func (m *model) processInput() tea.Cmd {
	input := strings.TrimSpace(m.input.Value())
	m.input.SetValue("")
	if input == "" {
		return nil
	}

	args := parseInput(input)
	if len(args) == 0 {
		return nil
	}

	switch args[0] {
	case "add":
		if len(args) < 3 {
			m.statusMsg = "❌ Uso: add \"buscar\" \"reemplazo\" o add -regex \"patrón\" \"reemplazo\""
			return nil
		}
		if args[1] == "-regex" && len(args) >= 4 {
			AddRuleEx(args[2], args[3], true, true)
			m.engine.UpdateRules(GetEnabledRules())
			m.statusMsg = fmt.Sprintf("✅ Regla regex añadida: %s → %s", args[2], args[3])
		} else {
			AddRule(args[1], args[2])
			m.engine.UpdateRules(GetEnabledRules())
			m.statusMsg = fmt.Sprintf("✅ Regla añadida: %s → %s", args[1], args[2])
		}
		m.refreshRuleKeys()

	case "del", "delete":
		if len(args) < 2 {
			m.statusMsg = "❌ Uso: del \"buscar\""
			return nil
		}
		RemoveRule(args[1])
		m.engine.UpdateRules(GetEnabledRules())
		m.statusMsg = fmt.Sprintf("🗑️ Regla eliminada: %s", args[1])
		m.refreshRuleKeys()

	case "enable":
		if len(args) < 2 {
			m.statusMsg = "❌ Uso: enable \"buscar\""
			return nil
		}
		SetRuleEnabled(args[1], true)
		m.engine.UpdateRules(GetEnabledRules())
		m.statusMsg = fmt.Sprintf("🟢 Regla habilitada: %s", args[1])

	case "disable":
		if len(args) < 2 {
			m.statusMsg = "❌ Uso: disable \"buscar\""
			return nil
		}
		SetRuleEnabled(args[1], false)
		m.engine.UpdateRules(GetEnabledRules())
		m.statusMsg = fmt.Sprintf("🔴 Regla deshabilitada: %s", args[1])

	case "toggle":
		if len(args) < 2 {
			m.statusMsg = "❌ Uso: toggle \"buscar\""
			return nil
		}
		enabled := ToggleRule(args[1])
		m.engine.UpdateRules(GetEnabledRules())
		if enabled {
			m.statusMsg = fmt.Sprintf("🟢 Regla habilitada: %s", args[1])
		} else {
			m.statusMsg = fmt.Sprintf("🔴 Regla deshabilitada: %s", args[1])
		}

	case "pause":
		m.monitor.Pause()
		m.paused = true
		m.statusMsg = "⏸️ Monitor pausado"

	case "resume":
		m.monitor.Resume()
		m.paused = false
		m.statusMsg = "▶️ Monitor reanudado"

	case "dryrun":
		dryRun := m.monitor.ToggleDryRun()
		m.dryRun = dryRun
		if dryRun {
			m.statusMsg = "🧪 Modo dry-run ACTIVADO (simulación)"
		} else {
			m.statusMsg = "🧪 Modo dry-run DESACTIVADO"
		}

	case "scan":
		if len(args) < 2 {
			m.statusMsg = "❌ Uso: scan ruta/archivo.txt"
			return nil
		}
		scanner := NewScanner(m.engine)
		result, err := scanner.ScanFile(args[1], false)
		if err != nil {
			m.statusMsg = fmt.Sprintf("❌ Error: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("✅ Escaneado: %s → %s (%d reemplazos)",
				result.OriginalPath, result.OutputPath, result.ReplacedCount)
		}

	case "export":
		path := "rules_export.json"
		if len(args) >= 2 {
			path = args[1]
		}
		if err := ExportRules(path); err != nil {
			m.statusMsg = fmt.Sprintf("❌ Error: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("✅ Exportado: %s", path)
		}

	case "import":
		if len(args) < 2 {
			m.statusMsg = "❌ Uso: import ruta/archivo.json"
			return nil
		}
		if err := ImportRules(args[1]); err != nil {
			m.statusMsg = fmt.Sprintf("❌ Error: %v", err)
		} else {
			m.engine.UpdateRules(GetEnabledRules())
			m.statusMsg = fmt.Sprintf("✅ Importado: %s", args[1])
			m.refreshRuleKeys()
		}

	case "notify":
		if m.desktopNotif != nil {
			m.desktopNotif.Enabled = !m.desktopNotif.Enabled
			m.desktopNotifs = m.desktopNotif.Enabled
			if m.desktopNotif.Enabled {
				m.statusMsg = "🔔 Notificaciones de escritorio ACTIVADAS"
			} else {
				m.statusMsg = "🔕 Notificaciones de escritorio DESACTIVADAS"
			}
		} else {
			m.statusMsg = "❌ Notificaciones de escritorio no disponibles"
		}

	case "clear":
		m.history.Clear()
		m.statusMsg = "🧹 Historial limpiado"

	case "help":
		m.statusMsg = "Comandos: add, del, enable, disable, toggle, pause, resume, dryrun, notify, clear, scan, export, import, help, quit"

	case "quit", "exit":
		m.quitting = true
		m.monitor.Stop()
		return tea.Quit

	default:
		m.statusMsg = fmt.Sprintf("❓ Comando desconocido: %s", args[0])
	}

	return nil
}

func ExportRules(path string) error {
	rules := ListRulesMap()
	data := struct {
		Version string          `json:"version"`
		Rules   map[string]Rule `json:"rules"`
	}{
		Version: Version,
		Rules:   rules,
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func ImportRules(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error al leer archivo: %w", err)
	}

	var imported struct {
		Rules map[string]Rule `json:"rules"`
	}
	if err := json.Unmarshal(b, &imported); err != nil {
		return fmt.Errorf("error al parsear: %w", err)
	}

	if imported.Rules == nil {
		return fmt.Errorf("el archivo no contiene reglas válidas")
	}

	c := GetConfig()
	for k, v := range imported.Rules {
		c.Rules[k] = v
	}
	SaveConfig(c)
	return nil
}
