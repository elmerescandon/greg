package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/elmerescandon/greg-ui/internal/metrics"
	"github.com/elmerescandon/greg-ui/internal/session"
)

// Initialized by InitStyles — zero value until then.
var cardStyle lipgloss.Style
var cardLabelStyle lipgloss.Style
var cardValueStyle lipgloss.Style
var cardCostStyle lipgloss.Style

// ── Bloque 1: 4 summary cards ─────────────────────────────────────────────────

func deltaLabel(current, prev float64) (string, bool) {
	if prev <= 0 {
		return "", false
	}
	pct := (current - prev) / prev * 100
	if pct > 0 {
		return fmt.Sprintf("▲ %.0f%%", pct), false // sube → rojo
	}
	return fmt.Sprintf("▼ %.0f%%", -pct), true // baja → verde
}

func fmtTokMetric(n int) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func renderSummaryCards(sum metrics.Summary, _ []session.Session, width int) string {
	sep := " " + SepDim.Render("│") + " "
	sepW := 3
	colW := (width - 2*sepW) / 3
	if colW < 12 {
		colW = 12
	}

	// No bold — some terminals render bold glyphs slightly wider, shifting the separator.
	// Use Width(colW) so lipgloss owns the padding logic.
	tokStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorCyan))

	cell := func(style lipgloss.Style, content string) string {
		return style.Width(colW).Render(content)
	}

	buildTokLines := func(label string, total, in, out, sess int) [4]string {
		return [4]string{
			cell(cardLabelStyle, label),
			cell(tokStyle, fmtTokMetric(total)+" tok"),
			cell(DimText, fmt.Sprintf("%s in · %s out", fmtTokMetric(in), fmtTokMetric(out))),
			cell(DimText, fmt.Sprintf("%d sess", sess)),
		}
	}

	buildMesLines := func(total, in, out int, cost float64, sess int) [4]string {
		costLine := fmt.Sprintf("$%.2f", cost)
		if sess > 0 {
			costLine = fmt.Sprintf("$%.2f  %d sess", cost, sess)
		}
		return [4]string{
			cell(cardLabelStyle, "Mes"),
			cell(tokStyle, fmtTokMetric(total)+" tok"),
			cell(DimText, fmt.Sprintf("%s in · %s out", fmtTokMetric(in), fmtTokMetric(out))),
			cell(cardCostStyle, costLine),
		}
	}

	todayTok := sum.TodayInputTokens + sum.TodayOutputTokens
	weekTok := sum.WeekInputTokens + sum.WeekOutputTokens
	monthTok := sum.MonthInputTokens + sum.MonthOutputTokens

	c1 := buildTokLines("Hoy", todayTok, sum.TodayInputTokens, sum.TodayOutputTokens, sum.TodaySessions)
	c2 := buildTokLines("Semana", weekTok, sum.WeekInputTokens, sum.WeekOutputTokens, sum.WeekSessions)
	c3 := buildMesLines(monthTok, sum.MonthInputTokens, sum.MonthOutputTokens, sum.MonthCost, sum.MonthSessions)

	var rows []string
	for i := 0; i < 4; i++ {
		rows = append(rows, c1[i]+sep+c2[i]+sep+c3[i])
	}
	return strings.Join(rows, "\n")
}

// ── Bloque 2: panel 50/50 ─────────────────────────────────────────────────────

func renderDayBars(daily []metrics.DaySummary, width int) []string {
	maxCost := 0.0
	for _, d := range daily {
		if d.Cost > maxCost {
			maxCost = d.Cost
		}
	}
	if maxCost == 0 {
		maxCost = 1
	}

	now := time.Now()
	todayStr := now.Format("2006-01-02")

	// " Mon* ████ $0.00" → label(5) + bar + " " + cost(6) = barW = width - 13
	barMaxW := width - 13
	if barMaxW < 2 {
		barMaxW = 2
	}

	dayAbbr := map[time.Weekday]string{
		time.Monday:    "Lun",
		time.Tuesday:   "Mar",
		time.Wednesday: "Mié",
		time.Thursday:  "Jue",
		time.Friday:    "Vie",
		time.Saturday:  "Sáb",
		time.Sunday:    "Dom",
	}

	var lines []string
	for _, d := range daily {
		t, err := time.Parse("2006-01-02", d.Date)
		if err != nil {
			continue
		}
		abbr := dayAbbr[t.Weekday()]
		barW := int(math.Round(float64(barMaxW) * d.Cost / maxCost))
		if barW < 1 && d.Cost > 0 {
			barW = 1
		}
		bar := strings.Repeat("█", barW)
		costStr := fmt.Sprintf("$%.2f", d.Cost)

		if d.Date == todayStr {
			day := lipgloss.NewStyle().Foreground(lipgloss.Color(colorCyan)).Bold(true).Render(abbr + "*")
			barRend := lipgloss.NewStyle().Foreground(lipgloss.Color(colorCyan)).Render(bar)
			lines = append(lines, fmt.Sprintf(" %-5s%s %s", day, barRend, DimText.Render(costStr)))
		} else {
			day := DimText.Render(fmt.Sprintf("%-4s ", abbr))
			barRend := lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted)).Render(bar)
			lines = append(lines, fmt.Sprintf(" %s%s %s", day, barRend, DimText.Render(costStr)))
		}
	}
	return lines
}

type topEntry struct {
	name string
	cost float64
}

func renderTopBreakdown(byModel, byProject map[string]metrics.Summary, width int) []string {
	totalModel := 0.0
	for _, s := range byModel {
		totalModel += s.MonthCost
	}
	if totalModel == 0 {
		totalModel = 1
	}
	totalProject := 0.0
	for _, s := range byProject {
		totalProject += s.MonthCost
	}
	if totalProject == 0 {
		totalProject = 1
	}

	top3 := func(m map[string]metrics.Summary) []topEntry {
		var entries []topEntry
		for name, s := range m {
			entries = append(entries, topEntry{name, s.MonthCost})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].cost > entries[j].cost })
		if len(entries) > 3 {
			entries = entries[:3]
		}
		return entries
	}

	// " name_______  ████ 62% $5.23" → label(14) + bar + pct(5) + cost(6)
	barMaxW := width - 26
	if barMaxW < 2 {
		barMaxW = 2
	}

	renderRows := func(entries []topEntry, total float64) []string {
		var lines []string
		for _, e := range entries {
			pct := e.cost / total * 100
			barW := int(math.Round(float64(barMaxW) * e.cost / total))
			if barW < 1 && e.cost > 0 {
				barW = 1
			}
			name := e.name
			if len([]rune(name)) > 14 {
				name = string([]rune(name)[:13]) + "…"
			}
			bar := lipgloss.NewStyle().Foreground(lipgloss.Color(colorCyan)).Render(strings.Repeat("█", barW))
			lines = append(lines, fmt.Sprintf(" %-14s%s %2.0f%% %s",
				name, bar, pct, DimText.Render(fmt.Sprintf("$%.2f", e.cost))))
		}
		return lines
	}

	var lines []string
	lines = append(lines, SectionHeader.Render(" Modelos"))
	lines = append(lines, renderRows(top3(byModel), totalModel)...)
	lines = append(lines, "")
	lines = append(lines, SectionHeader.Render(" Proyectos"))
	lines = append(lines, renderRows(top3(byProject), totalProject)...)
	return lines
}

func renderWeekPanel(daily []metrics.DaySummary, byModel, byProject map[string]metrics.Summary, width int) []string {
	half := (width - 2) / 2
	leftLines := renderDayBars(daily, half)
	rightLines := renderTopBreakdown(byModel, byProject, width-half-2)

	maxRows := len(leftLines)
	if len(rightLines) > maxRows {
		maxRows = len(rightLines)
	}
	for len(leftLines) < maxRows {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxRows {
		rightLines = append(rightLines, "")
	}

	var lines []string
	sep := strings.Repeat(" ", 2)
	for i := 0; i < maxRows; i++ {
		left := leftLines[i]
		leftW := lipgloss.Width(left)
		if leftW < half {
			left += strings.Repeat(" ", half-leftW)
		}
		lines = append(lines, left+sep+rightLines[i])
	}
	return lines
}

// ── Bloque 3: historial mensual ───────────────────────────────────────────────

func renderMonthlyHistory(history []metrics.MonthSummary, width int) []string {
	maxCost := 0.0
	for _, m := range history {
		if m.Cost > maxCost {
			maxCost = m.Cost
		}
	}
	if maxCost == 0 {
		maxCost = 1
	}

	// " Jun 2026* ████ $12.45  48 sess" → label(10) + bar + cost(7) + sess(10)
	barMaxW := width - 28
	if barMaxW < 2 {
		barMaxW = 2
	}

	var lines []string
	for _, m := range history {
		barW := int(math.Round(float64(barMaxW) * m.Cost / maxCost))
		if barW < 1 && m.Cost > 0 {
			barW = 1
		}
		mark := " "
		var labelRend string
		var barRend string
		if m.IsCurrent {
			mark = "*"
			labelRend = lipgloss.NewStyle().Foreground(lipgloss.Color(colorCyan)).Bold(true).Render(fmt.Sprintf("%-9s", m.Label))
			barRend = lipgloss.NewStyle().Foreground(lipgloss.Color(colorCyan)).Render(strings.Repeat("█", barW))
		} else {
			labelRend = DimText.Render(fmt.Sprintf("%-9s", m.Label))
			barRend = lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted)).Render(strings.Repeat("█", barW))
		}
		lines = append(lines, fmt.Sprintf(" %s%s %s  %s  %s",
			labelRend,
			mark,
			barRend,
			cardCostStyle.Render(fmt.Sprintf("$%.2f", m.Cost)),
			DimText.Render(fmt.Sprintf("%d sess", m.Sessions)),
		))
	}
	return lines
}

// ── Bloque 4: top sesiones scrollable ─────────────────────────────────────────

func (m Model) renderSessionsTable(sessions []session.Session, width int) []string {
	type entry struct {
		project   string
		started   string
		cost      float64
		inputTok  int
		outputTok int
	}
	var entries []entry
	for _, s := range sessions {
		if s.CostUSD <= 0 {
			continue
		}
		proj := "desconocido"
		if s.Dir != "" {
			proj = filepath.Base(s.Dir)
		}
		runes := []rune(proj)
		if len(runes) > 16 {
			proj = string(runes[:15]) + "…"
		}
		started := s.Started
		if len(started) > 10 {
			started = started[:10]
		}
		entries = append(entries, entry{
			project:   proj,
			started:   started,
			cost:      s.CostUSD,
			inputTok:  s.InputTokens,
			outputTok: s.OutputTokens,
		})
	}
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].cost > entries[j-1].cost; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}

	total := len(entries)
	const pageSize = 5
	offset := m.metricsTableOffset
	if total > 0 && offset > total-pageSize {
		offset = total - pageSize
	}
	if offset < 0 {
		offset = 0
	}

	var lines []string
	header := SectionHeader.Render("  Top sesiones por costo")
	if total > pageSize {
		indicator := DimText.Render(fmt.Sprintf("(%d/%d)", offset+1, total))
		pad := width - lipgloss.Width(header) - lipgloss.Width(indicator) - 2
		if pad < 1 {
			pad = 1
		}
		header = header + strings.Repeat(" ", pad) + indicator
	}
	lines = append(lines, header)
	lines = append(lines, " "+SepDim.Render(strings.Repeat("─", width-2)))

	if total == 0 {
		lines = append(lines, "  "+DimText.Render("sin datos de costo aún"))
		return lines
	}

	end := offset + pageSize
	if end > total {
		end = total
	}
	for _, e := range entries[offset:end] {
		costStr := cardCostStyle.Render(fmt.Sprintf("$%6.2f", e.cost))
		tokStr := DimText.Render(fmt.Sprintf("%4dk/%3dk", e.inputTok/1000, e.outputTok/1000))
		projStr := lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted)).Render(fmt.Sprintf("%-16s", e.project))
		lines = append(lines, fmt.Sprintf("  ● %s  %s  %s  %s", costStr, tokStr, projStr, e.started))
	}
	if total > pageSize {
		lines = append(lines, "  "+DimText.Render("j/k desplazar"))
	}
	return lines
}

// ── main view ─────────────────────────────────────────────────────────────────

func (m Model) viewGraficas() string {
	allSess := metrics.AllSessions()
	sum := metrics.GetSummary(allSess)

	h := m.height - 2
	w := m.width

	var lines []string

	// ── Bloque 1: 4 cards ──────────────────────────────────────────────────
	lines = append(lines, "")
	lines = append(lines, " "+renderSummaryCards(sum, allSess, w-2))
	lines = append(lines, "")

	// ── Bloque 2: panel 7 días + modelo/proyecto ───────────────────────────
	lines = append(lines, SectionHeader.Render("  Actividad reciente"))
	lines = append(lines, "")
	daily := metrics.DailyCostDist(allSess, 7)
	byModel := metrics.ByModel(allSess)
	byProject := metrics.ByProject(allSess)
	for _, l := range renderWeekPanel(daily, byModel, byProject, w-2) {
		lines = append(lines, " "+l)
	}
	lines = append(lines, "")

	// ── Bloque 3: historial mensual (omitir si solo hay 1 mes) ─────────────
	history := metrics.MonthlyHistory(allSess)
	if len(history) > 1 {
		lines = append(lines, SectionHeader.Render("  Historial mensual"))
		lines = append(lines, "")
		for _, l := range renderMonthlyHistory(history, w-2) {
			lines = append(lines, " "+l)
		}
		lines = append(lines, "")
	}

	// ── Bloque 4: top sesiones scrollable ──────────────────────────────────
	for _, l := range m.renderSessionsTable(allSess, w) {
		lines = append(lines, l)
	}

	// ── Fill + footer ──────────────────────────────────────────────────────
	for len(lines) < h-1 {
		lines = append(lines, "")
	}
	footer := DimText.Render("  j/k desplazar  Ctrl+1 chat  Ctrl+2 agente  Ctrl+3 métricas  Ctrl+Q salir")
	if len(lines) >= h {
		lines[h-1] = footer
	} else {
		lines = append(lines, footer)
	}

	return strings.Join(lines, "\n")
}
