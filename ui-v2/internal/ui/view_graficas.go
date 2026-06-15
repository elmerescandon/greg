package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/elmerescandon/greg-ui/internal/metrics"
	"github.com/elmerescandon/greg-ui/internal/session"
)

// ── heatmap colors ────────────────────────────────────────────────────────────

var heatSessionColors = []string{
	colorBorder,  // 0: no activity
	"#0e4429",    // 1: low
	"#006d32",    // 2: medium-low
	"#26a641",    // 3: medium-high
	colorGreen,   // 4: high (#3ddc84)
}

var heatCostColors = []string{
	colorBorder,
	"#2a1500",
	"#5c3600",
	"#b36b00",
	colorAmber, // #f5a32a
}

func heatLevel(val, maxVal float64) int {
	if val <= 0 || maxVal <= 0 {
		return 0
	}
	t := val / maxVal
	switch {
	case t > 0.75:
		return 4
	case t > 0.5:
		return 3
	case t > 0.25:
		return 2
	default:
		return 1
	}
}

// ── heatmap renderer ──────────────────────────────────────────────────────────

var dayLabels = []string{"Lun", "   ", "Mié", "   ", "Vie", "   ", "Dom"}

func renderHeatmap(dayData map[string]float64, numWeeks int, useCost bool) []string {
	colors := heatSessionColors
	if useCost {
		colors = heatCostColors
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Find the Sunday that ends the current week (or today if Sunday)
	endDay := today
	for endDay.Weekday() != time.Sunday {
		endDay = endDay.AddDate(0, 0, 1)
	}
	// Start from (numWeeks-1) weeks before that Monday
	startDay := endDay.AddDate(0, 0, -(numWeeks*7 - 1))
	// Align to Monday
	for startDay.Weekday() != time.Monday {
		startDay = startDay.AddDate(0, 0, -1)
	}

	// Find max value for scaling
	maxVal := 0.0
	for _, v := range dayData {
		if v > maxVal {
			maxVal = v
		}
	}

	// Build grid: 7 rows (Mon=0 .. Sun=6) × numWeeks columns
	grid := make([][]int, 7)
	for i := range grid {
		grid[i] = make([]int, numWeeks)
	}
	dates := make([][]string, 7)
	for i := range dates {
		dates[i] = make([]string, numWeeks)
	}

	d := startDay
	for week := 0; week < numWeeks; week++ {
		for dow := 0; dow < 7; dow++ {
			key := d.Format("2006-01-02")
			dates[dow][week] = key
			if d.After(today) {
				grid[dow][week] = -1 // future: render as empty
			} else {
				grid[dow][week] = heatLevel(dayData[key], maxVal)
			}
			d = d.AddDate(0, 0, 1)
		}
	}

	// Month labels row
	const labelW = 5
	var monthRow strings.Builder
	monthRow.WriteString(strings.Repeat(" ", labelW))
	prevMonth := ""
	for week := 0; week < numWeeks; week++ {
		// Use the Monday of this week to determine the month label
		monDate := startDay.AddDate(0, 0, week*7)
		m := monDate.Format("Jan")
		if m != prevMonth {
			monthRow.WriteString(DimText.Render(m))
			prevMonth = m
			// pad to align next column (month label is 3 chars, cell is 2)
			if week < numWeeks-1 {
				// nothing extra needed — next iteration adds cell width
			}
		} else {
			monthRow.WriteString("  ")
		}
	}

	var lines []string
	lines = append(lines, monthRow.String())

	// Grid rows
	for dow := 0; dow < 7; dow++ {
		var row strings.Builder
		label := dayLabels[dow]
		row.WriteString(DimText.Render(fmt.Sprintf(" %-3s ", label)))
		for week := 0; week < numWeeks; week++ {
			level := grid[dow][week]
			if level < 0 {
				row.WriteString(DimText.Render("░ "))
			} else {
				c := colors[level]
				row.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render("█ "))
			}
		}
		lines = append(lines, row.String())
	}

	return lines
}

// ── legend ────────────────────────────────────────────────────────────────────

func renderHeatLegend(useCost bool) string {
	colors := heatSessionColors
	if useCost {
		colors = heatCostColors
	}
	var sb strings.Builder
	sb.WriteString(DimText.Render("  menos "))
	for _, c := range colors {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render("█ "))
	}
	sb.WriteString(DimText.Render("más"))
	return sb.String()
}

// ── hourly heatmap (single row, 24 columns) ──────────────────────────────────

func renderHourlyStrip(dist [24]float64, useCost bool) []string {
	colors := heatSessionColors
	if useCost {
		colors = heatCostColors
	}
	maxV := 0.0
	for _, v := range dist {
		if v > maxV {
			maxV = v
		}
	}

	// Hour labels
	var labelRow strings.Builder
	labelRow.WriteString("       ")
	for h := 0; h < 24; h++ {
		if h%2 == 0 {
			labelRow.WriteString(DimText.Render(fmt.Sprintf("%-3d", h)))
		}
	}

	// Heat strip
	var strip strings.Builder
	strip.WriteString(DimText.Render("       "))
	for h := 0; h < 24; h++ {
		level := heatLevel(dist[h], maxV)
		c := colors[level]
		strip.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render("██ "))
	}

	return []string{labelRow.String(), strip.String()}
}

// ── summary cards ─────────────────────────────────────────────────────────────

var cardStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(colorBorder)).
	Padding(0, 1)

var cardLabelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorDim)).
	Bold(false)

var cardValueStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorCyan)).
	Bold(true)

var cardCostStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorAmber))

func renderSummaryCards(sum metrics.Summary, width int) string {
	type card struct {
		label    string
		sessions int
		cost     float64
	}
	cards := []card{
		{"Este mes", sum.MonthSessions, sum.MonthCost},
		{"Esta semana", sum.WeekSessions, sum.WeekCost},
		{"Hoy", sum.TodaySessions, sum.TodayCost},
	}

	cardW := (width - 6) / 3
	if cardW < 16 {
		cardW = 16
	}

	var rendered []string
	for _, c := range cards {
		label := cardLabelStyle.Render(c.label)
		sess := cardValueStyle.Render(fmt.Sprintf("%d sess", c.sessions))
		cost := cardCostStyle.Render(fmt.Sprintf("$%.2f", c.cost))
		content := label + "\n" + sess + "  " + cost
		rendered = append(rendered, cardStyle.Width(cardW).Render(content))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		rendered[0], "  ", rendered[1], "  ", rendered[2])
}

// ── main view ─────────────────────────────────────────────────────────────────

const dailyDays = 30

func (m Model) viewGraficas() string {
	allSess := metrics.AllSessions()
	sum := metrics.GetSummary(allSess)

	h := m.height - 2
	w := m.width

	var lines []string

	// ── Summary cards ──────────────────────────────────────────────────────
	lines = append(lines, "")
	lines = append(lines, " "+renderSummaryCards(sum, w-2))
	lines = append(lines, "")

	// ── Toggle hint ────────────────────────────────────────────────────────
	toggleHint := "[s] sesiones  [c] costo"
	if m.metricsShowCost {
		toggleHint = "[s] sesiones  " + ViewActive.Render("[c] costo")
	} else {
		toggleHint = ViewActive.Render("[s] sesiones") + "  [c] costo"
	}

	// ── Activity heatmap ───────────────────────────────────────────────────
	hdrLeft := SectionHeader.Render("  Actividad")
	hintPad := w - lipgloss.Width(hdrLeft) - lipgloss.Width(toggleHint) - 2
	if hintPad < 1 {
		hintPad = 1
	}
	lines = append(lines, hdrLeft+strings.Repeat(" ", hintPad)+DimText.Render(toggleHint))
	lines = append(lines, "")

	// Build day→value map
	dayData := make(map[string]float64)
	for _, s := range allSess {
		if len(s.Started) < 10 {
			continue
		}
		key := s.Started[:10]
		if m.metricsShowCost {
			dayData[key] += s.CostUSD
		} else {
			dayData[key]++
		}
	}

	// Dynamic number of weeks based on terminal width (each cell = 2 chars, labels = 5)
	numWeeks := (w - 5) / 2
	if numWeeks > 52 {
		numWeeks = 52
	}
	if numWeeks < 4 {
		numWeeks = 4
	}

	heatLines := renderHeatmap(dayData, numWeeks, m.metricsShowCost)
	lines = append(lines, heatLines...)
	lines = append(lines, renderHeatLegend(m.metricsShowCost))
	lines = append(lines, "")

	// ── Hourly distribution ────────────────────────────────────────────────
	lines = append(lines, SectionHeader.Render("  Horas de actividad"))
	lines = append(lines, "")

	var hourDist [24]float64
	if m.metricsShowCost {
		hourDist = metrics.HourlyCostDist(allSess)
	} else {
		dist := metrics.HourlyDist(allSess)
		for i, v := range dist {
			hourDist[i] = float64(v)
		}
	}
	for _, l := range renderHourlyStrip(hourDist, m.metricsShowCost) {
		lines = append(lines, l)
	}
	lines = append(lines, "")

	// ── Session breakdown table ────────────────────────────────────────────
	tableLines := m.renderSessionsTable(allSess, w)
	lines = append(lines, tableLines...)

	// ── Fill remaining space ───────────────────────────────────────────────
	for len(lines) < h-1 {
		lines = append(lines, "")
	}

	// ── Footer ────────────────────────────────────────────────────────────
	footer := DimText.Render("  s sesiones  c costo  Ctrl+1 chat  Ctrl+2 agente  Ctrl+Q salir")
	if len(lines) >= h {
		lines[h-1] = footer
	} else {
		lines = append(lines, footer)
	}

	return strings.Join(lines, "\n")
}

// renderSessionsTable shows top sessions by cost.
func (m Model) renderSessionsTable(sessions []session.Session, width int) []string {
	type entry struct {
		id      string
		started string
		cost    float64
		tokens  int
	}
	var entries []entry
	for _, s := range sessions {
		if s.CostUSD > 0 {
			entries = append(entries, entry{
				id:      strings.TrimPrefix(s.ID, "greg-"),
				started: s.Started[:min(len(s.Started), 10)],
				cost:    s.CostUSD,
				tokens:  s.OutputTokens,
			})
		}
	}
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].cost > entries[j-1].cost; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}

	var lines []string
	lines = append(lines, SectionHeader.Render("  Top sesiones por costo"))
	lines = append(lines, " "+SepDim.Render(strings.Repeat("─", width-2)))

	maxRows := 5
	if len(entries) < maxRows {
		maxRows = len(entries)
	}
	if maxRows == 0 {
		lines = append(lines, "  "+DimText.Render("sin datos de costo aún"))
		return lines
	}

	for _, e := range entries[:maxRows] {
		costStr := cardCostStyle.Render(fmt.Sprintf("$%6.2f", e.cost))
		tokenStr := DimText.Render(fmt.Sprintf("%5dk tok", e.tokens/1000))
		idStr := lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted)).Render(e.id)
		dateStr := DimText.Render(e.started)
		lines = append(lines, fmt.Sprintf("  ● %s  %s  %s  %s", costStr, tokenStr, idStr, dateStr))
	}
	return lines
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
