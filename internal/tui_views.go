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
	b.WriteString(m.wideLayout())
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
		start := len(events) - 3
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
		end := m.ruleOffset + ruleVisible
		if end > len(m.ruleKeys) {
			end = len(m.ruleKeys)
		}
		for idx := m.ruleOffset; idx < end; idx++ {
			search := m.ruleKeys[idx]
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
		}

		scrollInfo := fmt.Sprintf("  %s %d/%d  [\u2191/\u2193] Navigate  [Enter] Toggle", styleDim.Render("\u2191\u2193"), m.ruleCursor+1, len(rules))
		sb.WriteString(fmt.Sprintf("\n%s\n", scrollInfo))
	}

	return box.Render(sb.String())
}

func (m model) renderDivider() string {
	width := m.fullWidth
	if width < 40 {
		width = 40
	}
	return "  " + styleDim.Render(strings.Repeat("\u2500", width-2))
}

func (m model) inputView() string {
	return m.input.View()
}
