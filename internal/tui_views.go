package internal

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func tabWidth(total int) int {
	if total > 0 {
		return total
	}
	return 70
}

func (m model) View() string {
	if m.quitting {
		return "🛑 Clipboard Monitor cerrado.\n"
	}

	var b strings.Builder

	b.WriteString(m.headerView())
	b.WriteString("\n")
	b.WriteString(m.tabView())
	b.WriteString("\n")
	b.WriteString(m.renderDivider())
	b.WriteString("\n")

	switch m.activeTab {
	case tabDashboard:
		b.WriteString(m.dashboardView())
	case tabRules:
		b.WriteString(m.rulesView())
	case tabStats:
		b.WriteString(m.statsView())
	case tabHistory:
		b.WriteString(m.historyView())
	}

	b.WriteString("\n")
	b.WriteString(m.renderDivider())
	b.WriteString("\n")

	if m.statusMsg != "" {
		b.WriteString(styleDim.Render(m.statusMsg))
		b.WriteString("\n")
	}

	b.WriteString(m.inputView())

	return b.String()
}

func (m model) headerView() string {
	status := styleSuccess.Render("🟢 ACTIVO")
	if m.paused {
		status = styleWarning.Render("⏸️ PAUSADO")
	}

	mode := styleSuccess.Render("Normal")
	if m.dryRun {
		mode = styleWarning.Render("Dry-run")
	}
	if m.autoDetect {
		mode = styleWarning.Render("Auto")
	}

	rulesCount := len(ListRulesMap())
	replacesCount := m.metrics.GetTotalHits()

	return fmt.Sprintf("  🛡️ Clipboard Monitor %s  |  %s  |  %s  |  📋 %d reglas  |  🔄 %d reemplazos",
		Version, status, mode, rulesCount, replacesCount)
}

func (m model) tabView() string {
	var b strings.Builder
	b.WriteString("  ")
	for i, name := range tabNames {
		if tab(i) == m.activeTab {
			b.WriteString(styleTabActive.Render(name))
		} else {
			b.WriteString(styleTabInactive.Render(name))
		}
		if i < len(tabNames)-1 {
			b.WriteString("  ")
		}
	}
	return b.String()
}

func (m model) renderDivider() string {
	width := m.fullWidth
	if width < 40 {
		width = 40
	}
	return "  " + styleDim.Render(strings.Repeat("─", width-2))
}

func (m model) dashboardView() string {
	totalHits := m.metrics.GetTotalHits()
	ruleHits := m.metrics.GetRuleHits()

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
		boxStyle.Render(fmt.Sprintf("🔄 Reemplazos\n   %s", styleValue.Render(fmt.Sprintf("%d", totalHits)))),
		boxStyle.Render(fmt.Sprintf("📋 Reglas activas\n   %s", styleValue.Render(fmt.Sprintf("%d", len(ListRulesMap()))))),
		boxStyle.Render(fmt.Sprintf("📜 Historial\n   %s", styleValue.Render(fmt.Sprintf("%d", len(m.history.Events()))))),
	}

	b.WriteString("  ")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, stats...))
	b.WriteString("\n\n")

	if totalHits > 0 {
		b.WriteString(fmt.Sprintf("  %sTop reglas activadas:%s\n", styleValue.Render("📊"), styleDim.Render("")))
		b.WriteString("  ")
		count := 0
		for rule, hits := range ruleHits {
			if count >= 5 {
				break
			}
			b.WriteString(styleSuccess.Render(fmt.Sprintf("  • %s", rule)))
			b.WriteString(styleDim.Render(fmt.Sprintf(" (%dx)", hits)))
			count++
		}
		b.WriteString("\n")
	}

	events := m.history.Events()
	if len(events) > 0 {
		b.WriteString(fmt.Sprintf("\n  %sÚltimas detecciones:%s\n", styleValue.Render("🔒"), styleDim.Render("")))
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
		b.WriteString(fmt.Sprintf("\n  %sEsperando detecciones...%s\n", styleDim.Render("🔍"), styleDim.Render("")))
	}

	return b.String()
}

func (m model) rulesView() string {
	rules := ListRulesMap()
	var b strings.Builder

	if len(rules) == 0 {
		b.WriteString(fmt.Sprintf("  %sNo hay reglas definidas.%s\n", styleDim.Render("📭"), styleDim.Render("")))
		b.WriteString(fmt.Sprintf("  %sUsa: add \"buscar\" \"reemplazo\"%s\n", styleDim.Render("💡"), styleDim.Render("")))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("  %-30s %s %s\n", styleValue.Render("Buscar"), styleValue.Render("Reemplazar"), styleDim.Render("")))
	b.WriteString("  " + styleDim.Render(strings.Repeat("─", m.fullWidth-6)) + "\n")

	keys := make([]string, 0, len(rules))
	for k := range rules {
		keys = append(keys, k)
	}

	for idx, k := range keys {
		r := rules[k]
		icon := "🟢"
		if !r.Enabled {
			icon = "🔴"
		}

		suffix := ""
		if r.Regex {
			suffix = " [regex]"
		}

		cursor := "  "
		if idx == m.ruleCursor {
			cursor = styleSuccess.Render("▸ ")
		}

		search := k
		if len(search) > 28 {
			search = search[:25] + "..."
		}

		replace := r.Replace
		if len(replace) > 28 {
			replace = replace[:25] + "..."
		}

		line := fmt.Sprintf("%s%s %-28s %s %s%s",
			cursor, icon, search, "→", replace, suffix)

		if idx == m.ruleCursor {
			b.WriteString(styleSuccess.Render(line))
		} else if !r.Enabled {
			b.WriteString(styleDim.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("\n  %s[↑/↓] Navegar  [Enter] Toggle  [d] Eliminar%s\n",
		styleDim.Render(""), styleDim.Render("")))
	b.WriteString(fmt.Sprintf("  %sComandos: enable/disable/toggle \"buscar\"%s\n",
		styleDim.Render("💡"), styleDim.Render("")))

	return b.String()
}

func (m model) statsView() string {
	totalHits := m.metrics.GetTotalHits()
	ruleHits := m.metrics.GetRuleHits()

	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %sEstadísticas de protección%s\n", styleValue.Render("📊"), styleDim.Render("")))

	boxWidth := m.fullWidth - 8
	if boxWidth < 30 {
		boxWidth = 30
	}

	statsBox := fmt.Sprintf("  ╭─%s─╮\n", strings.Repeat("─", boxWidth))
	statsBox += fmt.Sprintf("  │ %s%-*s%s │\n", styleValue.Render("Total de reemplazos:"), boxWidth-20, "", styleDim.Render(""))
	statsBox += fmt.Sprintf("  │ %s%*d%s │\n", styleSuccess.Render(""), boxWidth-4, totalHits, styleDim.Render(""))

	if totalHits > 0 {
		statsBox += fmt.Sprintf("  │ %s%-*s%s │\n", styleDim.Render(""), boxWidth-4, "Detalle por regla:", styleDim.Render(""))
		for rule, hits := range ruleHits {
			line := fmt.Sprintf("    • %-30s %d veces", rule, hits)
			statsBox += fmt.Sprintf("  │ %s%-*s%s │\n", styleSuccess.Render(""), boxWidth-4, line, styleDim.Render(""))
		}
	}

	statsBox += fmt.Sprintf("  ╰─%s─╯\n", strings.Repeat("─", boxWidth))
	b.WriteString(statsBox)

	b.WriteString(fmt.Sprintf("\n  %sEscribe comandos para gestionar reglas.%s\n",
		styleDim.Render("💡"), styleDim.Render("")))

	return b.String()
}

func (m model) historyView() string {
	events := m.history.Events()
	var b strings.Builder

	if len(events) == 0 {
		b.WriteString(fmt.Sprintf("  %sNo hay detecciones registradas.%s\n",
			styleDim.Render("📭"), styleDim.Render("")))
		b.WriteString(fmt.Sprintf("  %sLas detecciones aparecerán aquí automáticamente.%s\n",
			styleDim.Render("💡"), styleDim.Render("")))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("  %-10s │ %s %s\n",
		styleValue.Render("Hora"),
		styleValue.Render("Reglas activadas"),
		styleDim.Render("")))
	b.WriteString("  " + styleDim.Render(strings.Repeat("─", m.fullWidth-6)) + "\n")

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

		b.WriteString(fmt.Sprintf("  %s │ %s\n  %s │ %s\n",
			styleDim.Render(timeStr),
			styleSuccess.Render(rules),
			styleDim.Render(""),
			styleDim.Render(fmt.Sprintf("  \"%s\"", preview))))
	}

	b.WriteString(fmt.Sprintf("\n  %sComando: clear (limpiar historial)%s\n",
		styleDim.Render("💡"), styleDim.Render("")))

	return b.String()
}

func (m model) inputView() string {
	return m.input.View()
}
