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

// ── braille chart (fallback) ──────────────────────────────────────────────────

// brailleDots maps (col, row) → braille bit position.
// Each braille char covers 2 columns × 4 rows.
var brailleBit = [2][4]byte{
	{0x01, 0x02, 0x04, 0x40}, // col 0
	{0x08, 0x10, 0x20, 0x80}, // col 1
}

func renderBrailleChart(values []float64, width, height int, colorLow, colorHigh string) []string {
	n := len(values)
	if n == 0 || width == 0 || height == 0 {
		return nil
	}
	maxV := 0.0
	for _, v := range values {
		if v > maxV {
			maxV = v
		}
	}
	if maxV == 0 {
		maxV = 1
	}

	// Braille canvas: pixW = width*2, pixH = height*4
	pixW := width * 2
	pixH := height * 4
	canvas := make([][]bool, pixH)
	for i := range canvas {
		canvas[i] = make([]bool, pixW)
	}

	// Map each value to a bar
	barPixW := float64(pixW) / float64(n)
	for i, v := range values {
		t := v / maxV
		barH := int(math.Round(t * float64(pixH)))
		x0 := int(float64(i) * barPixW)
		x1 := int(float64(i+1)*barPixW) - 1
		if x1 >= pixW {
			x1 = pixW - 1
		}
		for y := pixH - barH; y < pixH; y++ {
			for x := x0; x <= x1; x++ {
				canvas[y][x] = true
			}
		}
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
			// Fraction of this cell that's filled → pick color
			filled := 0
			for dc := 0; dc < 2; dc++ {
				for dr := 0; dr < 4; dr++ {
					py := row*4 + dr
					px := col*2 + dc
					if py < pixH && px < pixW && canvas[py][px] {
						filled++
					}
				}
			}
			t := float64(filled) / 8.0
			var colorStr string
			if t > 0.6 {
				colorStr = colorHigh
			} else if t > 0.2 {
				colorStr = colorCyan
			} else {
				colorStr = colorLow
			}
			ch := rune(0x2800 + int(bits))
			if bits == 0 {
				sb.WriteString(DimText.Render(string(ch)))
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(colorStr)).Render(string(ch)))
			}
		}
		rows[row] = sb.String()
	}
	return rows
}

// ── helpers ───────────────────────────────────────────────────────────────────

func hourlyToFloat(dist [24]int) [24]float64 {
	var out [24]float64
	for i, v := range dist {
		out[i] = float64(v)
	}
	return out
}

func dailyToFloat(stats []metrics.DayStat, useCost bool) []float64 {
	out := make([]float64, len(stats))
	for i, s := range stats {
		if useCost {
			out[i] = s.Cost
		} else {
			out[i] = float64(s.Sessions)
		}
	}
	return out
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

// ── axis labels ───────────────────────────────────────────────────────────────

func renderHourLabels(width int) string {
	// 24 labels spread evenly across width
	labels := []string{"00", "02", "04", "06", "08", "10", "12",
		"14", "16", "18", "20", "22"}
	n := len(labels)
	spacing := width / n
	if spacing < 2 {
		spacing = 2
	}
	var sb strings.Builder
	for i, l := range labels {
		if i > 0 {
			pad := spacing - 2
			if pad < 1 {
				pad = 1
			}
			sb.WriteString(strings.Repeat(" ", pad))
		}
		sb.WriteString(DimText.Render(l))
	}
	return sb.String()
}

func renderDayLabels(stats []metrics.DayStat, width int) string {
	if len(stats) == 0 {
		return ""
	}
	first := formatShortDate(stats[0].Date)
	last := formatShortDate(stats[len(stats)-1].Date)
	pad := width - len(first) - len(last)
	if pad < 1 {
		pad = 1
	}
	return DimText.Render(first) + strings.Repeat(" ", pad) + DimText.Render(last)
}

func formatShortDate(d string) string {
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		return d
	}
	return t.Format("Jan 02")
}

// ── subview toggle ────────────────────────────────────────────────────────────

// metricsShowCost is toggled by pressing 'c' / 's' in the Métricas view.
// It's stored on the Model, not here; but we accept it as a param.

// ── main view ─────────────────────────────────────────────────────────────────

const (
	hourlyChartRows = 9  // terminal rows for the hourly chart
	dailyChartRows  = 5  // terminal rows for the daily chart
	dailyDays       = 30
)

func (m Model) viewGraficas() string {
	allSess := metrics.AllSessions()
	sum := metrics.GetSummary(allSess)
	hourDist := metrics.HourlyDist(allSess)
	var hourDistF [24]float64
	if m.metricsShowCost {
		hourDistF = metrics.HourlyCostDist(allSess)
	} else {
		hourDistF = hourlyToFloat(hourDist)
	}
	dailyStats := metrics.DailyStats(allSess, dailyDays)
	dailyVals := dailyToFloat(dailyStats, m.metricsShowCost)

	h := m.height - 2 // content height (minus 2-row top bar)
	w := m.width

	var lines []string

	// ── Summary cards ──────────────────────────────────────────────────────
	lines = append(lines, "")
	lines = append(lines, " "+renderSummaryCards(sum, w-2))
	lines = append(lines, "")

	// ── Toggle hint + section header ───────────────────────────────────────
	toggleHint := "[s] sesiones  [c] costo"
	if m.metricsShowCost {
		toggleHint = "[s] sesiones  " + ViewActive.Render("[c] costo")
	} else {
		toggleHint = ViewActive.Render("[s] sesiones") + "  [c] costo"
	}
	hdrLeft := SectionHeader.Render("  Actividad por hora")
	hintPad := w - lipgloss.Width(hdrLeft) - lipgloss.Width(toggleHint) - 2
	if hintPad < 1 {
		hintPad = 1
	}
	lines = append(lines, hdrLeft+strings.Repeat(" ", hintPad)+DimText.Render(toggleHint))

	// ── Hourly chart ───────────────────────────────────────────────────────
	lines = append(lines, " "+renderHourLabels(w-2))

	if IsKittySupported() && m.kittyReady {
		for _, l := range kittyPlaceholderLines(kittyImgHourly, hourlyChartRows, w) {
			lines = append(lines, l)
		}
	} else {
		colorLow := colorBorder
		colorHigh := colorCyan
		if m.metricsShowCost {
			colorLow = "#2a1a00"
			colorHigh = colorAmber
		}
		chartLines := renderBrailleChart(hourDistF[:], w-2, hourlyChartRows, colorLow, colorHigh)
		for _, l := range chartLines {
			lines = append(lines, " "+l)
		}
	}

	lines = append(lines, "")

	// ── Daily chart ────────────────────────────────────────────────────────
	lines = append(lines, SectionHeader.Render("  Actividad diaria  ")+
		DimText.Render(fmt.Sprintf("últimos %d días", dailyDays)))

	if IsKittySupported() && m.kittyReady {
		for _, l := range kittyPlaceholderLines(kittyImgDaily, dailyChartRows, w) {
			lines = append(lines, l)
		}
	} else {
		colorLow := "#0a1f14"
		colorHigh := colorGreen
		if m.metricsShowCost {
			colorLow = "#2a1a00"
			colorHigh = colorAmber
		}
		chartLines := renderBrailleChart(dailyVals, w-2, dailyChartRows, colorLow, colorHigh)
		for _, l := range chartLines {
			lines = append(lines, " "+l)
		}
	}
	lines = append(lines, " "+renderDayLabels(dailyStats, w-2))
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
	// Filter to sessions with cost data, sort descending
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
	// Simple insertion sort (small slice)
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
