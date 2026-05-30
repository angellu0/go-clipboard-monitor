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

const paletteVisible = 4
const ruleVisible = 3

type paletteCmd struct {
	name   string
	brief  string
	syntax string
}

var paletteCommands = []paletteCmd{
	{"add", "Add rule", `add "search" "replace"`},
	{"add -regex", "Add regex", `add -regex "pattern" "replace"`},
	{"del", "Delete rule", `del "search"`},
	{"enable", "Enable rule", `enable "search"`},
	{"disable", "Disable rule", `disable "search"`},
	{"toggle", "Toggle rule", `toggle "search"`},
	{"pause", "Pause monitor", "pause"},
	{"resume", "Resume monitor", "resume"},
	{"dryrun", "Simulation mode", "dryrun"},
	{"notify", "Notifications", "notify"},
	{"clear", "Clear history", "clear"},
	{"scan", "Scan file", `scan "file"`},
	{"export", "Export rules", "export [file]"},
	{"import", "Import rules", `import "file"`},
	{"help", "Show help", "help"},
	{"exit", "Exit", "exit"},
}

type model struct {
	monitor      *Monitor
	metrics      *Metrics
	engine       *Engine
	history      *History
	desktopNotif *DesktopNotifier

	paused        bool
	dryRun        bool
	autoDetect    bool
	desktopNotifs bool
	statusMsg     string

	input textinput.Model

	ruleKeys      []string
	ruleCursor    int
	ruleOffset    int
	showPalette   bool
	paletteCursor int
	paletteOffset int

	width     int
	height    int
	quitting  bool
	fullWidth int
}

var (
	styleSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	styleDim = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	styleValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("87"))
)

func NewTUI(monitor *Monitor, metrics *Metrics, engine *Engine, history *History, desktopNotif *DesktopNotifier) *model {
	ti := textinput.New()
	ti.Placeholder = "Type / to see available commands"
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
		ruleOffset:    0,
		showPalette:   false,
		paletteCursor: 0,
		paletteOffset: 0,
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
					if m.paletteCursor < m.paletteOffset {
						m.paletteOffset--
					}
				}
				return m, nil

			case tea.KeyDown:
				filtered := m.filteredPalette()
				if m.paletteCursor < len(filtered)-1 {
					m.paletteCursor++
					if m.paletteCursor >= m.paletteOffset+paletteVisible {
						m.paletteOffset++
					}
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
				} else {
					m.clampPaletteCursor()
				}
				return m, cmd

			case tea.KeyRunes:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				if !strings.HasPrefix(m.input.Value(), "/") {
					m.showPalette = false
				} else {
					m.clampPaletteCursor()
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

		case tea.KeyUp:
			if len(m.ruleKeys) > 0 && m.ruleCursor > 0 {
				m.ruleCursor--
				if m.ruleCursor < m.ruleOffset {
					m.ruleOffset--
				}
			}
			return m, nil

		case tea.KeyDown:
			if len(m.ruleKeys) > 0 && m.ruleCursor < len(m.ruleKeys)-1 {
				m.ruleCursor++
				if m.ruleCursor >= m.ruleOffset+ruleVisible {
					m.ruleOffset++
				}
			}
			return m, nil

		case tea.KeyEnter:
			if m.input.Value() == "" && len(m.ruleKeys) > 0 && m.ruleCursor < len(m.ruleKeys) {
				search := m.ruleKeys[m.ruleCursor]
				enabled := ToggleRule(search)
				m.engine.UpdateRules(GetEnabledRules())
				if enabled {
					m.statusMsg = fmt.Sprintf("🟢 Rule enabled: %s", search)
				} else {
					m.statusMsg = fmt.Sprintf("🔴 Rule disabled: %s", search)
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
				m.paletteOffset = 0
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
				Timestamp:      time.Now(),
				OriginalText:   msg.Result.OriginalText,
				ModifiedText:   msg.Result.ModifiedText,
				TriggeredRules: msg.Result.TriggeredRules,
			})
			m.statusMsg = fmt.Sprintf("🔒 Detected: %s", strings.Join(msg.Result.TriggeredRules, ", "))

		case "paused":
			m.paused = true
			m.statusMsg = "⏸️ Monitor paused"

		case "resumed":
			m.paused = false
			m.statusMsg = "▶️ Monitor resumed"

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
	if m.ruleOffset >= len(m.ruleKeys) {
		m.ruleOffset = 0
	}
	if m.ruleCursor < m.ruleOffset {
		m.ruleOffset = m.ruleCursor
	}
}

func (m *model) clampPaletteCursor() {
	filtered := m.filteredPalette()
	if m.paletteCursor >= len(filtered) {
		m.paletteCursor = len(filtered) - 1
	}
	if m.paletteCursor < 0 {
		m.paletteCursor = 0
	}
	if m.paletteOffset >= len(filtered) {
		m.paletteOffset = 0
	}
	if m.paletteCursor < m.paletteOffset {
		m.paletteOffset = m.paletteCursor
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
			m.statusMsg = "❌ Usage: add \"search\" \"replacement\" or add -regex \"pattern\" \"replacement\""
			return nil
		}
		if args[1] == "-regex" && len(args) >= 4 {
			AddRuleEx(args[2], args[3], true, true)
			m.engine.UpdateRules(GetEnabledRules())
			m.statusMsg = fmt.Sprintf("✅ Regex rule added: %s → %s", args[2], args[3])
		} else {
			AddRule(args[1], args[2])
			m.engine.UpdateRules(GetEnabledRules())
			m.statusMsg = fmt.Sprintf("✅ Rule added: %s → %s", args[1], args[2])
		}
		m.refreshRuleKeys()

	case "del", "delete":
		if len(args) < 2 {
			m.statusMsg = "❌ Usage: del \"search\""
			return nil
		}
		RemoveRule(args[1])
		m.engine.UpdateRules(GetEnabledRules())
		m.statusMsg = fmt.Sprintf("🗑️ Rule deleted: %s", args[1])
		m.refreshRuleKeys()

	case "enable":
		if len(args) < 2 {
			m.statusMsg = "❌ Usage: enable \"search\""
			return nil
		}
		SetRuleEnabled(args[1], true)
		m.engine.UpdateRules(GetEnabledRules())
		m.statusMsg = fmt.Sprintf("🟢 Rule enabled: %s", args[1])

	case "disable":
		if len(args) < 2 {
			m.statusMsg = "❌ Usage: disable \"search\""
			return nil
		}
		SetRuleEnabled(args[1], false)
		m.engine.UpdateRules(GetEnabledRules())
		m.statusMsg = fmt.Sprintf("🔴 Rule disabled: %s", args[1])

	case "toggle":
		if len(args) < 2 {
			m.statusMsg = "❌ Usage: toggle \"search\""
			return nil
		}
		enabled := ToggleRule(args[1])
		m.engine.UpdateRules(GetEnabledRules())
		if enabled {
			m.statusMsg = fmt.Sprintf("🟢 Rule enabled: %s", args[1])
		} else {
			m.statusMsg = fmt.Sprintf("🔴 Rule disabled: %s", args[1])
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
			m.statusMsg = fmt.Sprintf("✅ Scanned: %s → %s (%d replacements)",
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
			m.statusMsg = fmt.Sprintf("✅ Exported: %s", path)
		}

	case "import":
		if len(args) < 2 {
			m.statusMsg = "❌ Usage: import path/to/file.json"
			return nil
		}
		if err := ImportRules(args[1]); err != nil {
			m.statusMsg = fmt.Sprintf("❌ Error: %v", err)
		} else {
			m.engine.UpdateRules(GetEnabledRules())
			m.statusMsg = fmt.Sprintf("✅ Imported: %s", args[1])
			m.refreshRuleKeys()
		}

	case "notify":
		if m.desktopNotif != nil {
			m.desktopNotif.Enabled = !m.desktopNotif.Enabled
			m.desktopNotifs = m.desktopNotif.Enabled
			if m.desktopNotif.Enabled {
				m.statusMsg = "🔔 Desktop notifications ACTIVATED"
			} else {
				m.statusMsg = "🔕 Desktop notifications DEACTIVATED"
			}
		} else {
			m.statusMsg = "❌ Desktop notifications not available"
		}

	case "clear":
		m.history.Clear()
		m.statusMsg = "🧹 History cleared"

	case "help":
		m.statusMsg = "Commands: add, del, enable, disable, toggle, pause, resume, dryrun, notify, clear, scan, export, import, help, quit"

	case "exit":
		m.quitting = true
		m.monitor.Stop()
		return tea.Quit

	default:
		m.statusMsg = fmt.Sprintf("❓ Command not found: %s", args[0])
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
		return fmt.Errorf("error reading file: %w", err)
	}

	var imported struct {
		Rules map[string]Rule `json:"rules"`
	}
	if err := json.Unmarshal(b, &imported); err != nil {
		return fmt.Errorf("error parsing file: %w", err)
	}

	if imported.Rules == nil {
		return fmt.Errorf("the file does not contain valid rules")
	}

	c := GetConfig()
	for k, v := range imported.Rules {
		c.Rules[k] = v
	}
	SaveConfig(c)
	return nil
}
