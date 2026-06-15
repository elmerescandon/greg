package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/elmerescandon/greg-ui/internal/metrics"
	"github.com/elmerescandon/greg-ui/internal/session"
)

// ── heatmap colors (5-level scale, GitHub style) ─────────────────────────────

var heatSessionColors = []string{
	"#161b22", // 0: no activity
	"#0e4429", // 1: low
	"#006d32", // 2: medium-low
	"#26a641", // 3: medium-high
	colorGreen, // 4: high
}

var heatCostColors = []string{
	"#161b22",
	"#2a1500",
	"#5c3600",
	"#b36b00",
	colorAmber,
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

// ── heatmap with rounded cells ───────────────────────────────────────────────

// Each cell is rendered as a colored "▪" (small filled square) which together
// with the spacing gives a GitHub-like rounded dot appearance.
// We use lipgloss per-cell styling with the rounded box border for each cell.

var dayLabels = []string{"Lun", "   ", "Mié", "   ", "Vie", "   ", "Dom"}

var heatCellStyle = lipgloss.NewStyle().
	Width(2).
	Align(lipgloss.Center)

func renderHeatmap(dayData map[string]float64, numWeeks int, useCost bool) []string {
	colors := heatSessionColors
	if useCost {
		colors = heatCostColors
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	endDay := today
	for endDay.Weekday() != time.Sunday {
		endDay = endDay.AddDate(0, 0, 1)
	}
	startDay := endDay.AddDate(0, 0, -(numWeeks*7 - 1))
	for startDay.Weekday() != time.Monday {
		startDay = startDay.AddDate(0, 0, -1)
	}

	maxVal := 0.0
	for _, v := range dayData {
		if v > maxVal {
			maxVal = v
		}
	}

	grid := make([][]int, 7)
	for i := range grid {
		grid[i] = make([]int, numWeeks)
	}

	d := startDay
	for week := 0; week < numWeeks; week++ {
		for dow := 0; dow < 7; dow++ {
			key := d.Format("2006-01-02")
			if d.After(today) {
				grid[dow][week] = -1
			} else {
				grid[dow][week] = heatLevel(dayData[key], maxVal)
			}
			d = d.AddDate(0, 0, 1)
		}
	}

	// Month labels — each dot column is 2 chars wide ("● "), align label to column start
	const labelW = 5
	monthSlots := make([]string, numWeeks)
	prevMonth := ""
	for week := 0; week < numWeeks; week++ {
		monDate := startDay.AddDate(0, 0, week*7)
		m := monDate.Format("Jan")
		if m != prevMonth {
			monthSlots[week] = m
			prevMonth = m
		}
	}
	var monthRow strings.Builder
	monthRow.WriteString(strings.Repeat(" ", labelW))
	for week := 0; week < numWeeks; week++ {
		if monthSlots[week] != "" {
			label := monthSlots[week]
			monthRow.WriteString(DimText.Render(label))
			skip := len(label) / 2
			for skip > 0 && week+1 < numWeeks {
				week++
				skip--
			}
		} else {
			monthRow.WriteString("  ")
		}
	}

	var lines []string
	lines = append(lines, monthRow.String())

	for dow := 0; dow < 7; dow++ {
		var row strings.Builder
		label := dayLabels[dow]
		row.WriteString(DimText.Render(fmt.Sprintf(" %-3s ", label)))
		for week := 0; week < numWeeks; week++ {
			level := grid[dow][week]
			if level < 0 {
				row.WriteString("  ")
			} else {
				c := colors[level]
				row.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color(c)).
					Render("●") + " ")
			}
		}
		lines = append(lines, row.String())
	}

	return lines
}

func renderHeatLegend(useCost bool) string {
	colors := heatSessionColors
	if useCost {
		colors = heatCostColors
	}
	var sb strings.Builder
	sb.WriteString(DimText.Render("      menos "))
	for _, c := range colors {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render("●") + " ")
	}
	sb.WriteString(DimText.Render("más"))
	return sb.String()
}

// ── braille line chart ───────────────────────────────────────────────────────

// brailleBit maps (col, row) → braille dot bit.
// Each braille char is 2 cols × 4 rows.
var brailleBit = [2][4]byte{
	{0x01, 0x02, 0x04, 0x40}, // col 0 (left)
	{0x08, 0x10, 0x20, 0x80}, // col 1 (right)
}

func renderLineChart(values [24]float64, width, height int, useCost bool) []string {
	maxV := 0.0
	for _, v := range values {
		if v > maxV {
			maxV = v
		}
	}
	if maxV == 0 {
		maxV = 1
	}

	pixW := width * 2
	pixH := height * 4
	canvas := make([][]bool, pixH)
	for i := range canvas {
		canvas[i] = make([]bool, pixW)
	}

	// Catmull-Rom spline interpolation for smooth curve
	clampHour := func(i int) int {
		if i < 0 {
			return 0
		}
		if i > 23 {
			return 23
		}
		return i
	}
	catmullRom := func(p0, p1, p2, p3, t float64) float64 {
		t2 := t * t
		t3 := t2 * t
		return 0.5 * ((2 * p1) +
			(-p0+p2)*t +
			(2*p0-5*p1+4*p2-p3)*t2 +
			(-p0+3*p1-3*p2+p3)*t3)
	}
	points := make([]float64, pixW)
	for px := 0; px < pixW; px++ {
		frac := float64(px) / float64(pixW) * 23.0
		i1 := int(frac)
		t := frac - float64(i1)
		p0 := values[clampHour(i1-1)]
		p1 := values[clampHour(i1)]
		p2 := values[clampHour(i1+1)]
		p3 := values[clampHour(i1+2)]
		v := catmullRom(p0, p1, p2, p3, t)
		if v < 0 {
			v = 0
		}
		points[px] = v
	}

	// Plot the line: for each x, compute y and draw a dot
	for px := 0; px < pixW; px++ {
		normalized := points[px] / maxV
		py := pixH - 1 - int(math.Round(normalized*float64(pixH-1)))
		if py < 0 {
			py = 0
		}
		if py >= pixH {
			py = pixH - 1
		}
		canvas[py][px] = true
	}

	lineColor := colorCyan
	if useCost {
		lineColor = colorAmber
	}

	rows := make([]string, height)
	for row := 0; row < height; row++ {
		var sb strings.Builder
		for col := 0; col < width; col++ {
			var bits byte
			for dc := 0; dc < 2; dc++ {
				for dr := 0; dr < 4; dr++ {
					py := row*4 + dr
					px := col*2 + dc
					if py < pixH && px < pixW && canvas[py][px] {
						bits |= brailleBit[dc][dr]
					}
				}
			}
			ch := rune(0x2800 + int(bits))
			if bits == 0 {
				sb.WriteString(" ")
			} else {
				sb.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color(lineColor)).
					Render(string(ch)))
			}
		}
		rows[row] = sb.String()
	}
	return rows
}

func renderHourLabels(width int) string {
	labels := []struct {
		hour  int
		label string
	}{
		{0, "0h"}, {4, "4h"}, {8, "8h"}, {12, "12h"}, {16, "16h"}, {20, "20h"},
	}
	buf := make([]byte, width)
	for i := range buf {
		buf[i] = ' '
	}
	for _, l := range labels {
		pos := l.hour * width / 24
		if pos+len(l.label) <= width {
			copy(buf[pos:], l.label)
		}
	}
	return DimText.Render(string(buf))
}

// ── summary cards ─────────────────────────────────────────────────────────────

var cardStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(colorBorder)).
	Padding(0, 1)

var cardLabelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color(colorDim))

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

const lineChartRows = 8

func (m Model) viewGraficas() string {
	allSess := metrics.AllSessions()
	sum := metrics.GetSummary(allSess)

	h := m.height - 2
	w := m.width
	useCost := m.metricsShowCost

	var lines []string

	// ── Summary cards ──────────────────────────────────────────────────────
	lines = append(lines, "")
	lines = append(lines, " "+renderSummaryCards(sum, w-2))
	lines = append(lines, "")

	// ── Toggle hint ────────────────────────────────────────────────────────
	toggleHint := "[s] sesiones  [c] costo"
	if useCost {
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

	dayData := make(map[string]float64)
	for _, s := range allSess {
		if len(s.Started) < 10 {
			continue
		}
		key := s.Started[:10]
		if useCost {
			dayData[key] += s.CostUSD
		} else {
			dayData[key]++
		}
	}

	var hourDist [24]float64
	if useCost {
		hourDist = metrics.HourlyCostDist(allSess)
	} else {
		dist := metrics.HourlyDist(allSess)
		for i, v := range dist {
			hourDist[i] = float64(v)
		}
	}

	gap := 4
	halfW := (w - gap) / 2

	numWeeks := (halfW - 5) / 2
	if numWeeks > 52 {
		numWeeks = 52
	}
	if numWeeks < 4 {
		numWeeks = 4
	}

	heatLines := renderHeatmap(dayData, numWeeks, useCost)
	heatLines = append(heatLines, renderHeatLegend(useCost))

	chartW := halfW - 2
	if chartW < 20 {
		chartW = 20
	}
	var chartPanel []string
	chartPanel = append(chartPanel, SectionHeader.Render(" Horas de actividad"))
	chartPanel = append(chartPanel, "")
	chartLines := renderLineChart(hourDist, chartW, lineChartRows, useCost)
	chartPanel = append(chartPanel, chartLines...)
	chartPanel = append(chartPanel, renderHourLabels(chartW))

	maxRows := len(heatLines)
	if len(chartPanel) > maxRows {
		maxRows = len(chartPanel)
	}
	for len(heatLines) < maxRows {
		heatLines = append(heatLines, "")
	}
	for len(chartPanel) < maxRows {
		chartPanel = append(chartPanel, "")
	}

	separator := strings.Repeat(" ", gap)
	for i := 0; i < maxRows; i++ {
		left := heatLines[i]
		leftW := lipgloss.Width(left)
		if leftW < halfW {
			left += strings.Repeat(" ", halfW-leftW)
		}
		lines = append(lines, left+separator+chartPanel[i])
	}
	lines = append(lines, "")

	// ── Top sessions table ─────────────────────────────────────────────────
	tableLines := m.renderSessionsTable(allSess, w)
	lines = append(lines, tableLines...)

	// ── Fill + footer ──────────────────────────────────────────────────────
	for len(lines) < h-1 {
		lines = append(lines, "")
	}
	footer := DimText.Render("  s sesiones  c costo  Ctrl+1 chat  Ctrl+2 agente  Ctrl+Q salir")
	if len(lines) >= h {
		lines[h-1] = footer
	} else {
		lines = append(lines, footer)
	}

	return strings.Join(lines, "\n")
}

// ── top sessions table ───────────────────────────────────────────────────────

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
