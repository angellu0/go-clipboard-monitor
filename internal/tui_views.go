package internal

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.quitting {
		return "🛑 Clipboard Monitor cerrado.\n"
	}

	var b strings.Builder

	b.WriteString(m.bannerView())
	b.WriteString("\n")

	if m.width >= 110 {
		b.WriteString(m.wideLayout())
	} else {
		b.WriteString(m.tabView())
		b.WriteString("\n")
		b.WriteString(m.renderDivider())
		b.WriteString("\n")

		switch m.activeTab {
		case tabDashboard:
			b.WriteString(m.dashboardView())
		case tabRules:
			b.WriteString(m.rulesView())
		case tabHistory:
			b.WriteString(m.historyView())
		}
	}

	b.WriteString("\n")
	b.WriteString(m.renderDivider())
	b.WriteString("\n")

	if m.statusMsg != "" {
		b.WriteString(styleDim.Render(m.statusMsg))
		b.WriteString("\n")
	}

	if m.showPalette {
		b.WriteString(m.paletteView())
		b.WriteString("\n")
	}

	b.WriteString(m.inputView())

	return b.String()
}

func (m model) paletteView() string {
	filtered := m.filteredPalette()
	if len(filtered) == 0 {
		return ""
	}

	boxW := m.fullWidth - 2
	if boxW > 80 {
		boxW = 80
	}

	showSyntax := boxW >= 70

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(" %s Commands %s\n", styleValue.Render(""), styleValue.Render("")))
	sb.WriteString(styleDim.Render(strings.Repeat("\u2500", boxW)))
	sb.WriteString("\n")

	end := m.paletteOffset + paletteVisible
	if end > len(filtered) {
		end = len(filtered)
	}
	for i := m.paletteOffset; i < end; i++ {
		cmd := filtered[i]
		prefix := "  "
		if i == m.paletteCursor {
			prefix = styleSuccess.Render("\u25B8 ")
		}

		var line string
		if showSyntax {
			line = fmt.Sprintf("%s%-12s %-20s %s", prefix, cmd.name, cmd.brief, cmd.syntax)
		} else {
			line = fmt.Sprintf("%s%-12s %s", prefix, cmd.name, cmd.brief)
		}

		if len(line) > boxW {
			line = line[:boxW-3] + "..."
		}
		if i == m.paletteCursor {
			sb.WriteString(styleSuccess.Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	if len(filtered) > paletteVisible {
		scrollInfo := fmt.Sprintf("  %s %d/%d", styleDim.Render("\u2191\u2193"), m.paletteCursor+1, len(filtered))
		sb.WriteString(styleDim.Render(scrollInfo))
		sb.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(boxW + 2).
		Render(sb.String())
}

func (m model) bannerView() string {
	data, err := os.ReadFile("assets/banner.txt")
	banner := ""
	if err == nil {
		banner = strings.TrimRight(string(data), "\n")
	}

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("92")).
		Render(banner)
}

func (m model) wideLayout() string {
	totalW := m.fullWidth - 2
	leftW := int(math.Round(float64(totalW) * 0.55))
	rightW := totalW - leftW

	left := m.wideStatsPanel(leftW)
	right := m.wideRulesPanel(rightW)

	return "  " + lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m model) wideStatsPanel(w int) string {
	totalHits := m.metrics.GetTotalHits()

	box := lipgloss.NewStyle().Width(w)

	var sb strings.Builder

	sb.WriteString(styleValue.Render("  \U0001F4CA Stats"))
	sb.WriteString("\n")
	sb.WriteString("  " + styleDim.Render(strings.Repeat("\u2500", w-4)))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  Replacements: %s\n", styleValue.Render(fmt.Sprintf("%d", totalHits))))
	sb.WriteString(fmt.Sprintf("  Active Rules: %s\n", styleValue.Render(fmt.Sprintf("%d", len(ListRulesMap())))))
	sb.WriteString(fmt.Sprintf("  History:      %s\n", styleValue.Render(fmt.Sprintf("%d", len(m.history.Events())))))

	events := m.history.Events()
	if len(events) > 0 {
		sb.WriteString(fmt.Sprintf("\n  %s Recent Activity\n", styleValue.Render("\U0001F50F")))
		start := len(events) - 4
		if start < 0 {
			start = 0
		}
		for i := start; i < len(events); i++ {
			evt := events[i]
			ts := evt.Timestamp.Format("15:04:05")
			rules := strings.Join(evt.TriggeredRules, ", ")
			sb.WriteString(fmt.Sprintf("  %s  %s\n", styleDim.Render(ts), styleSuccess.Render(rules)))
		}
	} else {
		sb.WriteString(fmt.Sprintf("\n  %s Recent Activity\n", styleValue.Render("\U0001F50F")))
		sb.WriteString(fmt.Sprintf("  %s No activity yet\n", styleDim.Render("\U0001F50D")))
	}

	return box.Render(sb.String())
}

func (m model) wideRulesPanel(w int) string {
	rules := ListRulesMap()

	box := lipgloss.NewStyle().Width(w)

	var sb strings.Builder

	sb.WriteString(styleValue.Render("  \U0001F4CB Active Rules"))
	sb.WriteString("\n")
	sb.WriteString("  " + styleDim.Render(strings.Repeat("\u2500", w-4)))
	sb.WriteString("\n")

	if len(rules) == 0 {
		sb.WriteString(fmt.Sprintf("  %s No rules defined.\n", styleDim.Render("\U0001F4ED")))
		sb.WriteString(fmt.Sprintf("  %s add \"buscar\" \"reemplazo\"\n", styleDim.Render("\U0001F4A1")))
	} else {
		displayed := 0
		for idx, search := range m.ruleKeys {
			if displayed >= 12 {
				sb.WriteString(fmt.Sprintf("\n  %s ...", styleDim.Render("")))
				break
			}
			rule := rules[search]
			icon := "\U0001F7E2"
			if !rule.Enabled {
				icon = "\U0001F534"
			}
			s := search
			if len(s) > w-14 {
				s = s[:w-17] + "..."
			}
			r := rule.Replace
			if len(r) > w-10 {
				r = r[:w-13] + "..."
			}
			suffix := ""
			if rule.Regex {
				suffix = " [regex]"
			}
			cursor := "  "
			if idx == m.ruleCursor {
				cursor = styleSuccess.Render("\u25B8 ")
			}
			line := fmt.Sprintf("%s%s %s%s", cursor, icon, s, suffix)
			if idx == m.ruleCursor {
				sb.WriteString(styleSuccess.Render(line))
			} else if rule.Enabled {
				sb.WriteString(line)
			} else {
				sb.WriteString(styleDim.Render(line))
			}
			sb.WriteString(fmt.Sprintf("\n     \u2192 %s\n", styleDim.Render(r)))
			displayed++
		}

		if len(rules) > 12 {
			sb.WriteString(fmt.Sprintf("\n  %s %d more...\n", styleDim.Render(""), len(rules)-12))
		}
		sb.WriteString(fmt.Sprintf("\n  [\u2191/\u2193] Navegar  [Enter] Toggle\n"))
	}

	return box.Render(sb.String())
}

func (m model) tabView() string {
	active := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("92")).
		Padding(0, 3)

	inactive := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Padding(0, 3)

	var tabs []string
	for i, name := range tabNames {
		label := fmt.Sprintf("[%d] %s", i+1, name)
		if tab(i) == m.activeTab {
			tabs = append(tabs, active.Render(label))
		} else {
			tabs = append(tabs, inactive.Render(label))
		}
	}

	return "  " + lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m model) renderDivider() string {
	width := m.fullWidth
	if width < 40 {
		width = 40
	}
	return "  " + styleDim.Render(strings.Repeat("\u2500", width-2))
}

func (m model) dashboardView() string {
	totalHits := m.metrics.GetTotalHits()

	var b strings.Builder

	boxW := (m.fullWidth - 6) / 3
	if boxW < 18 {
		boxW = 18
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(boxW)

	stats := []string{
		boxStyle.Render(fmt.Sprintf("Reemplazos\n  %s", styleValue.Render(fmt.Sprintf("%d", totalHits)))),
		boxStyle.Render(fmt.Sprintf("Reglas activas\n  %s", styleValue.Render(fmt.Sprintf("%d", len(ListRulesMap()))))),
		boxStyle.Render(fmt.Sprintf("Historial\n  %s", styleValue.Render(fmt.Sprintf("%d", len(m.history.Events()))))),
	}

	b.WriteString("  ")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, stats...))
	b.WriteString("\n\n")

	events := m.history.Events()
	if len(events) > 0 {
		b.WriteString(fmt.Sprintf("\n  %s \u00DAltimas detecciones:\n", styleValue.Render("\U0001F512")))
		start := len(events) - 3
		if start < 0 {
			start = 0
		}
		for i := start; i < len(events); i++ {
			evt := events[i]
			timeStr := evt.Timestamp.Format("15:04:05")
			rules := strings.Join(evt.TriggeredRules, ", ")
			b.WriteString(styleDim.Render(fmt.Sprintf("  [%s]", timeStr)))
			b.WriteString(styleSuccess.Render(fmt.Sprintf(" %s\n", rules)))
		}
	} else {
		b.WriteString(fmt.Sprintf("\n  %s Esperando detecciones...\n", styleDim.Render("\U0001F50D")))
	}

	return b.String()
}

func (m model) rulesView() string {
	rules := ListRulesMap()
	var b strings.Builder

	if len(rules) == 0 {
		b.WriteString(fmt.Sprintf("  %s No hay reglas definidas.\n", styleDim.Render("\U0001F4ED")))
		b.WriteString(fmt.Sprintf("  %s add \"buscar\" \"reemplazo\"\n", styleDim.Render("\U0001F4A1")))
		return b.String()
	}

	b.WriteString(styleValue.Render(fmt.Sprintf("  %-30s %s\n", "Buscar", "Reemplazar")))
	b.WriteString("  " + styleDim.Render(strings.Repeat("\u2500", m.fullWidth-6)) + "\n")

	for idx, k := range m.ruleKeys {
		r := rules[k]
		icon := "\U0001F7E2"
		if !r.Enabled {
			icon = "\U0001F534"
		}

		suffix := ""
		if r.Regex {
			suffix = " [regex]"
		}

		cursor := "  "
		if idx == m.ruleCursor {
			cursor = styleSuccess.Render("\u25B8 ")
		}

		search := k
		if len(search) > 28 {
			search = search[:25] + "..."
		}

		replace := r.Replace
		if len(replace) > 28 {
			replace = replace[:25] + "..."
		}

		line := fmt.Sprintf("%s%s %-28s \u2192 %s%s",
			cursor, icon, search, replace, suffix)

		if idx == m.ruleCursor {
			b.WriteString(styleSuccess.Render(line))
		} else if !r.Enabled {
			b.WriteString(styleDim.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("\n  [\u2191/\u2193] Navegar  [Enter] Toggle\n"))
	b.WriteString(fmt.Sprintf("  %s enable/disable/toggle \"buscar\"\n", styleDim.Render("\U0001F4A1")))

	return b.String()
}

func (m model) historyView() string {
	events := m.history.Events()
	var b strings.Builder

	if len(events) == 0 {
		b.WriteString(fmt.Sprintf("  %s No hay detecciones registradas.\n", styleDim.Render("\U0001F4ED")))
		b.WriteString(fmt.Sprintf("  %s Las detecciones aparecer\u00E1n aqu\u00ED autom\u00E1ticamente.\n", styleDim.Render("\U0001F4A1")))
		return b.String()
	}

	b.WriteString(styleValue.Render(fmt.Sprintf("  %-10s \u2502 %s\n", "Hora", "Reglas activadas")))
	b.WriteString("  " + styleDim.Render(strings.Repeat("\u2500", m.fullWidth-6)) + "\n")

	start := len(events) - 20
	if start < 0 {
		start = 0
	}
	for i := start; i < len(events); i++ {
		evt := events[i]
		timeStr := evt.Timestamp.Format("15:04:05")
		rules := strings.Join(evt.TriggeredRules, ", ")
		preview := ""
		if len(evt.OriginalText) > 40 {
			preview = evt.OriginalText[:37] + "..."
		} else {
			preview = evt.OriginalText
		}

		b.WriteString(fmt.Sprintf("  %s \u2502 %s\n", styleDim.Render(timeStr), styleSuccess.Render(rules)))
		b.WriteString(fmt.Sprintf("  %s\u2502 %s\n", strings.Repeat(" ", 10), styleDim.Render(fmt.Sprintf("  \"%s\"", preview))))
	}

	b.WriteString(fmt.Sprintf("\n  %s clear - limpiar historial\n", styleDim.Render("\U0001F4A1")))

	return b.String()
}

func (m model) inputView() string {
	return m.input.View()
}
