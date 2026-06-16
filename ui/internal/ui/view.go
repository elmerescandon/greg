package ui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/elmerescandon/greg-ui/internal/session"
	"github.com/elmerescandon/greg-ui/internal/task"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	topBar := m.viewTopBar()

	var content string
	switch m.viewMode {
	case ViewMultiple:
		content = m.viewMultiple()
	case ViewGraficas:
		content = m.viewGraficas()
	case ViewConfig:
		content = m.viewConfig()
	default:
		content = m.viewMetricas()
	}

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, topBar, content))
	v.WindowTitle = "Greg"
	v.AltScreen = true
	v.MouseMode = tea.MouseModeNone
	return v
}

func (m Model) viewBarButtonPositions() (metStart, metEnd, mulStart, mulEnd, grafStart, grafEnd, confStart, confEnd int) {
	gregW := lipgloss.Width(GregLabel.Render(" GREG"))
	metW := lipgloss.Width(ViewActive.Render("Chat"))
	mulW := lipgloss.Width(ViewActive.Render("Agente"))
	grafW := lipgloss.Width(ViewActive.Render("Métricas"))
	confW := lipgloss.Width(ViewActive.Render("Config"))
	gap := 2
	margin := 1
	rightTotal := metW + gap + mulW + gap + grafW + gap + confW + margin
	leftPad := m.width - gregW - rightTotal
	if leftPad < 0 {
		leftPad = 0
	}
	metStart = gregW + leftPad
	metEnd = metStart + metW
	mulStart = metEnd + gap
	mulEnd = mulStart + mulW
	grafStart = mulEnd + gap
	grafEnd = grafStart + grafW
	confStart = grafEnd + gap
	confEnd = confStart + confW
	return
}

func (m Model) viewTopBar() string {
	gregRendered := GregLabel.Render(" GREG")
	gregW := lipgloss.Width(gregRendered)

	styleFor := func(mode ViewMode) lipgloss.Style {
		if m.viewMode == mode {
			return ViewActive
		}
		return ViewInactive
	}

	metRendered := styleFor(ViewMetricas).Render("Chat")
	mulRendered := styleFor(ViewMultiple).Render("Agente")
	grafRendered := styleFor(ViewGraficas).Render("Métricas")
	confRendered := styleFor(ViewConfig).Render("Config")
	metW := lipgloss.Width(metRendered)
	mulW := lipgloss.Width(mulRendered)
	grafW := lipgloss.Width(grafRendered)
	confW := lipgloss.Width(confRendered)
	gap := 2
	margin := 1
	rightTotal := metW + gap + mulW + gap + grafW + gap + confW + margin

	leftPad := m.width - gregW - rightTotal
	if leftPad < 0 {
		leftPad = 0
	}

	row1 := gregRendered + strings.Repeat(" ", leftPad) +
		metRendered + strings.Repeat(" ", gap) +
		mulRendered + strings.Repeat(" ", gap) +
		grafRendered + strings.Repeat(" ", gap) +
		confRendered + strings.Repeat(" ", margin)

	// Row 2: thick ━ under the active button
	leftThinW := gregW + leftPad
	var row2 string
	switch m.viewMode {
	case ViewMultiple:
		row2 = SepDim.Render(strings.Repeat("─", leftThinW+metW+gap)) +
			SepActive.Render(strings.Repeat("━", mulW)) +
			SepDim.Render(strings.Repeat("─", gap+grafW+gap+confW+margin))
	case ViewGraficas:
		row2 = SepDim.Render(strings.Repeat("─", leftThinW+metW+gap+mulW+gap)) +
			SepActive.Render(strings.Repeat("━", grafW)) +
			SepDim.Render(strings.Repeat("─", gap+confW+margin))
	case ViewConfig:
		row2 = SepDim.Render(strings.Repeat("─", leftThinW+metW+gap+mulW+gap+grafW+gap)) +
			SepActive.Render(strings.Repeat("━", confW)) +
			SepDim.Render(strings.Repeat("─", margin))
	default: // ViewMetricas / Chat
		row2 = SepDim.Render(strings.Repeat("─", leftThinW)) +
			SepActive.Render(strings.Repeat("━", metW)) +
			SepDim.Render(strings.Repeat("─", gap+mulW+gap+grafW+gap+confW+margin))
	}

	return row1 + "\n" + row2
}

func (m Model) viewMultiple() string {
	h := m.height - 2
	headerH := 2
	footerH := 2
	listH := h - headerH - footerH

	tasks, _ := task.LoadTasks()
	n := len(tasks)

	// Clamp selection
	sel := m.multiSelectedIdx
	if n == 0 {
		sel = -1
	} else {
		if sel >= n {
			sel = n - 1
		}
		if sel < 0 {
			sel = 0
		}
	}

	if m.multiAgentView && sel >= 0 && sel < n {
		return m.viewAgentOutput(tasks[sel], task.Agent{})
	}

	if m.multiDetailMode && sel >= 0 && sel < n {
		return m.viewNetworkDetail(tasks[sel])
	}

	// Compute scrollTop to keep selected row visible
	scrollTop := 0
	if sel >= listH {
		scrollTop = sel - listH + 1
	}
	maxScroll := n - listH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollTop > maxScroll {
		scrollTop = maxScroll
	}

	// Column widths
	const statusW = 12
	const idW = 14
	const dateW = 12
	fixedW := 2 + 1 + 1 + statusW + 2 + idW + 2 + dateW + 2
	goalW := m.width - fixedW
	if goalW < 8 {
		goalW = 8
	}

	// Column header row
	hdr := fmt.Sprintf("  %s %-*s  %-*s  %-*s  %s",
		" ", statusW, "estado", idW, "tarea", dateW, "creado", "objetivo")
	var out []string
	out = append(out, DimText.Render(hdr))
	out = append(out, " "+SepDim.Render(strings.Repeat("─", m.width-2)))

	if n == 0 {
		out = append(out, "")
		out = append(out, "  "+DimText.Render("sin tareas — greg task run --goal \"...\" --agent \"id:rol\""))
		for len(out) < h-footerH {
			out = append(out, "")
		}
	} else {
		for i := scrollTop; i < scrollTop+listH; i++ {
			if i >= n {
				out = append(out, "")
				continue
			}
			out = append(out, m.renderTaskRow(tasks[i], i == sel, statusW, idW, dateW, goalW))
		}
	}

	// Pinned footer
	out = append(out, " "+SepDim.Render(strings.Repeat("─", m.width-2)))
	hint := "↑/↓ navegar  PgUp/PgDn saltar  Ctrl+1 chat  Ctrl+Q salir"
	if n > 0 {
		hint = fmt.Sprintf("↑/↓ navegar  Enter detalle  x cerrar (si todos done)  %d/%d  Ctrl+1 chat  Ctrl+Q salir", sel+1, n)
	}
	out = append(out, " "+DimText.Render(hint))

	return strings.Join(out, "\n")
}

func (m Model) renderTaskRow(t task.Task, selected bool, statusW, idW, dateW, goalW int) string {
	selector := "  "
	if selected {
		selector = "❯ "
	}

	status := t.CoordinatorStatus
	if len(status) > statusW {
		status = status[:statusW-1] + "…"
	}

	tid := t.TaskID
	if len(tid) > idW {
		tid = tid[:idW]
	}

	dateStr := formatTaskDate(t.Created)

	goal := t.Goal
	if len(goal) > goalW {
		goal = goal[:goalW-1] + "…"
	}

	if selected {
		line := fmt.Sprintf("%s● %-*s  %-*s  %-*s  %s",
			selector, statusW, status, idW, tid, dateW, dateStr, goal)
		return TaskRowSelected.Width(m.width).Render(line)
	}

	var coloredBullet string
	switch t.CoordinatorStatus {
	case "completed":
		coloredBullet = StatusGreen.Render("✔")
	case "running", "synthesizing": // "synthesizing" only appears in schema_version < 2 tasks
		coloredBullet = StatusYellow.Render("◉")
	default:
		coloredBullet = DimText.Render("◌")
	}
	rest := fmt.Sprintf(" %-*s  %-*s  %-*s  %s",
		statusW, status, idW, tid, dateW, dateStr, goal)
	return selector + coloredBullet + TaskRowDim.Render(rest)
}

// listWorkspaceDocs returns sorted .md filenames inside workspace/workspace/.
func listWorkspaceDocs(workspace string) []string {
	entries, err := os.ReadDir(workspace + "/workspace")
	if err != nil {
		return nil
	}
	var docs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			docs = append(docs, e.Name())
		}
	}
	sort.Strings(docs)
	return docs
}

func (m Model) viewAgentOutput(t task.Task, _ task.Agent) string {
	h := m.height - 2
	headerH := 2
	footerH := 2
	contentH := h - headerH - footerH
	if contentH < 1 {
		contentH = 1
	}

	// Choose doc list and file directory based on source
	var docs []string
	var fileDir string
	var sourceLabel string
	if m.multiDocSource == "messages" {
		docs = listMsgChannels(t.Workspace)
		fileDir = t.Workspace + "/messages/"
		sourceLabel = "messages"
	} else {
		docs = listWorkspaceDocs(t.Workspace)
		fileDir = t.Workspace + "/workspace/"
		sourceLabel = "workspace"
	}

	var filePath, fileName string
	if len(docs) == 0 {
		filePath = ""
		fileName = "—"
	} else {
		idx := m.multiDocIdx
		if idx >= len(docs) {
			idx = 0
		}
		fileName = docs[idx]
		filePath = fileDir + fileName
	}

	// Read file — refreshed on every tick
	raw, err := os.ReadFile(filePath)
	var allLines []string
	if err != nil || filePath == "" {
		allLines = []string{"", "  " + DimText.Render("sin output aún…")}
	} else {
		contentWidth := m.width - 2
		if contentWidth < 1 {
			contentWidth = 1
		}
		for _, l := range strings.Split(string(raw), "\n") {
			wrapped := ansi.Wrap(l, contentWidth, " ")
			for _, wl := range strings.Split(wrapped, "\n") {
				if m.multiDocSource == "messages" && strings.HasPrefix(strings.TrimSpace(wl), "###") {
					allLines = append(allLines, " "+SectionHeader.Render(wl))
				} else {
					allLines = append(allLines, " "+wl)
				}
			}
		}
	}

	// Scroll: offset 0 = tail, increasing offset scrolls up
	total := len(allLines)
	off := m.multiAgentScrollOffset
	start := total - contentH - off
	if start < 0 {
		start = 0
	}
	end := start + contentH
	if end > total {
		end = total
	}
	visible := allLines[start:end]
	for len(visible) < contentH {
		visible = append(visible, "")
	}

	// Header
	docPos := ""
	if len(docs) > 0 {
		idx := m.multiDocIdx
		if idx >= len(docs) {
			idx = 0
		}
		docPos = DimText.Render(fmt.Sprintf("  %d/%d", idx+1, len(docs)))
	}

	breadcrumb := fmt.Sprintf("← %s / %s / %s", t.TaskID, sourceLabel, fileName)
	header1 := " " + ViewActive.Render(breadcrumb) + docPos

	// Show agent status only for workspace docs (messages have no .status file)
	if m.multiDocSource != "messages" {
		agentID := strings.TrimSuffix(fileName, ".md")
		agentStatus := task.AgentStatus(t.Workspace, agentID)
		var statusStr string
		switch agentStatus {
		case "done", "completed":
			statusStr = StatusGreen.Render("✔ " + agentStatus)
		case "working":
			statusStr = StatusYellow.Render("◉ " + agentStatus + " " + spinFrames[m.spinIdx])
		case "—":
			statusStr = ""
		default:
			statusStr = DimText.Render("◌ " + agentStatus)
		}
		if statusStr != "" {
			header1 += "   " + statusStr
		}
	}

	header2 := " " + SepDim.Render(strings.Repeat("─", m.width-2))

	var out []string
	out = append(out, header1)
	out = append(out, header2)
	out = append(out, visible...)

	// Footer
	scrollHint := ""
	if off > 0 {
		scrollHint = fmt.Sprintf("  [+%d líneas]", off)
	}
	navHint := "←/→ documento  ↑/↓ scroll  PgUp/PgDn saltar" + scrollHint + "  Esc volver  Ctrl+Q salir"
	out = append(out, " "+SepDim.Render(strings.Repeat("─", m.width-2)))
	out = append(out, " "+DimText.Render(navHint))

	return strings.Join(out, "\n")
}

// countMsgsInFile counts occurrences of "### [" in the given file (message headers).
func countMsgsInFile(path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return strings.Count(string(raw), "### [")
}

// renderNetworkGraph draws an ASCII graph of agent communication for the given task.
// selectedChannel is the basename of the currently selected messages/*.md file.
// Returns a slice of lines of width w.
func renderNetworkGraph(t task.Task, channels []string, selectedChannel string, w, h int) []string {
	if w < 10 {
		w = 10
	}

	agents := task.AllAgents(t)

	// Build a set of agent IDs for quick lookup
	agentIDs := map[string]bool{}
	for _, a := range agents {
		agentIDs[a.ID] = true
	}

	// Determine if "human" node is needed (human→director.md exists)
	hasHuman := false
	for _, ch := range channels {
		base := strings.TrimSuffix(ch, ".md")
		if strings.HasPrefix(base, "human→") || strings.HasPrefix(base, "human->") {
			hasHuman = true
			break
		}
	}

	// Parse channels into edges: from → to with message count
	type edge struct {
		from, to string
		count    int
		channel  string // basename
	}
	var edges []edge
	for _, ch := range channels {
		base := strings.TrimSuffix(ch, ".md")
		// support both → (unicode) and -> (ascii)
		var from, to string
		if idx := strings.Index(base, "→"); idx >= 0 {
			from = base[:idx]
			to = base[idx+len("→"):]
		} else if idx := strings.Index(base, "->"); idx >= 0 {
			from = base[:idx]
			to = base[idx+2:]
		} else {
			continue
		}
		count := countMsgsInFile(t.Workspace + "/messages/" + ch)
		edges = append(edges, edge{from: from, to: to, count: count, channel: ch})
	}

	// Collect non-director, non-human agents that appear in edges
	workerSet := map[string]bool{}
	for _, e := range edges {
		if e.from != "director" && e.from != "human" {
			workerSet[e.from] = true
		}
		if e.to != "director" && e.to != "human" {
			workerSet[e.to] = true
		}
	}
	// Also include all agents from task
	for _, a := range agents {
		if a.ID != "director" {
			workerSet[a.ID] = true
		}
	}
	var workers []string
	for id := range workerSet {
		workers = append(workers, id)
	}
	sort.Strings(workers)

	// Helper: truncate agent label to fit box
	const maxLabelW = 12
	label := func(id string) string {
		if len(id) > maxLabelW {
			return id[:maxLabelW-1] + "…"
		}
		return id
	}

	// Box drawing: "╔═══════╗" / "║ label ║" / "╚═══════╝"
	boxLines := func(id string, selected bool) []string {
		lbl := label(id)
		inner := " " + lbl + " "
		top := "╔" + strings.Repeat("═", len(inner)) + "╗"
		mid := "║" + inner + "║"
		bot := "╚" + strings.Repeat("═", len(inner)) + "╝"
		if selected {
			return []string{
				SepActive.Render(top),
				SepActive.Render(mid),
				SepActive.Render(bot),
			}
		}
		return []string{top, mid, bot}
	}

	// Status bullet next to director
	directorStatus := task.AgentStatus(t.Workspace, "director")
	var dirBullet string
	switch directorStatus {
	case "done", "completed":
		dirBullet = StatusGreen.Render("✔")
	case "working":
		dirBullet = StatusYellow.Render("◉")
	default:
		dirBullet = DimText.Render("◌")
	}

	// Build output lines
	var lines []string

	// Title line
	statusStr := t.CoordinatorStatus
	if statusStr == "" {
		statusStr = "—"
	}
	title := fmt.Sprintf("  COMUNICACIÓN — %s   %s %s", t.TaskID, dirBullet, DimText.Render(statusStr))
	lines = append(lines, title)
	lines = append(lines, "")

	// Row 0: human (if present) → director
	dirBox := boxLines("director", false)
	dirBoxW := len("╔") + 2 + len(label("director")) + len("╗") // 2 = spaces around label
	// Actually compute from the string
	dirBoxW = len([]rune(strings.ReplaceAll(dirBox[0], "\x1b[", "")))
	// Use a simpler approach: just measure the raw string without ANSI
	rawDirTop := "╔" + strings.Repeat("═", len(" "+label("director")+" ")) + "╗"
	dirBoxW = len([]rune(rawDirTop))

	humanLine := ""
	if hasHuman {
		humanLine = DimText.Render("human") + "──▶"
	}
	humanW := 0
	if hasHuman {
		humanW = len("human") + 3 // "human──▶"
	}

	// Center director box
	centerX := w / 2
	dirLeft := centerX - dirBoxW/2
	if dirLeft < humanW {
		dirLeft = humanW
	}

	pad := func(n int) string {
		if n <= 0 {
			return ""
		}
		return strings.Repeat(" ", n)
	}

	if hasHuman {
		lines = append(lines, pad(dirLeft-humanW)+humanLine+dirBox[0])
		lines = append(lines, pad(dirLeft)+dirBox[1])
		lines = append(lines, pad(dirLeft)+dirBox[2])
	} else {
		lines = append(lines, pad(dirLeft)+dirBox[0])
		lines = append(lines, pad(dirLeft)+dirBox[1])
		lines = append(lines, pad(dirLeft)+dirBox[2])
	}

	if len(workers) == 0 {
		for len(lines) < h {
			lines = append(lines, "")
		}
		return lines[:h]
	}

	// Build arrow lines from director to each worker
	// Each worker gets a column slot
	const workerBoxW = 14 // "╔════════════╗" = 14 chars for 12-char label
	const workerGap = 2

	totalWorkerW := len(workers)*workerBoxW + (len(workers)-1)*workerGap
	workerLeft := (w - totalWorkerW) / 2
	if workerLeft < 0 {
		workerLeft = 0
	}

	// Center X of each worker box
	workerCenterX := make([]int, len(workers))
	for i := range workers {
		workerCenterX[i] = workerLeft + i*(workerBoxW+workerGap) + workerBoxW/2
	}

	// Draw arrow column from directorCenter down to worker row
	dirCenterX := dirLeft + dirBoxW/2

	// Count messages on director→worker channels
	msgCount := func(from, to string) (int, string) {
		for _, e := range edges {
			if e.from == from && e.to == to {
				return e.count, e.channel
			}
			if e.from == to && e.to == from {
				return e.count, e.channel
			}
		}
		return 0, ""
	}

	// Arrow rows: we draw 2 rows of arrows between director and workers
	// Row A: count label   Row B: │ char
	arrowRowA := make([]byte, w)
	arrowRowB := make([]byte, w)
	for i := range arrowRowA {
		arrowRowA[i] = ' '
		arrowRowB[i] = ' '
	}

	type arrowSeg struct {
		x       int
		countS  string
		ch      string
		workerI int
	}
	var arrowSegs []arrowSeg

	for i, wid := range workers {
		cx := workerCenterX[i]
		count, ch := msgCount("director", wid)
		countS := ""
		if count > 0 {
			countS = fmt.Sprintf("%d↓", count)
		} else {
			countS = "↓"
		}
		arrowSegs = append(arrowSegs, arrowSeg{x: cx, countS: countS, ch: ch, workerI: i})
	}

	// Build the arrow lines as rendered strings (can't use byte arrays with ANSI)
	// Instead build position-aware rendering
	arrowLineA := pad(w) // placeholder
	arrowLineB := pad(w)

	// We'll build them character by character using a rune slice
	runeA := []rune(strings.Repeat(" ", w))
	runeB := []rune(strings.Repeat(" ", w))
	_ = arrowRowA
	_ = arrowRowB

	for _, seg := range arrowSegs {
		isSelected := (seg.ch == selectedChannel)
		cs := []rune(seg.countS)
		startA := seg.x - len(cs)/2
		for j, r := range cs {
			pos := startA + j
			if pos >= 0 && pos < w {
				runeA[pos] = r
			}
		}
		arrowChar := '│'
		if isSelected {
			arrowChar = '┃'
		}
		if seg.x >= 0 && seg.x < w {
			runeB[seg.x] = arrowChar
		}
		_ = isSelected
	}

	// Now render with color for selected
	// We need to apply SepActive to selected arrows — build by scanning segs
	var arrowAparts []string
	var arrowBparts []string
	lastA, lastB := 0, 0
	for _, seg := range arrowSegs {
		isSelected := (seg.ch == selectedChannel)
		cs := []rune(seg.countS)
		startA := seg.x - len(cs)/2

		// plain text up to startA
		if startA > lastA {
			arrowAparts = append(arrowAparts, string(runeA[lastA:startA]))
		}
		rendered := seg.countS
		if isSelected {
			rendered = SepActive.Render(seg.countS)
		} else {
			rendered = DimText.Render(seg.countS)
		}
		arrowAparts = append(arrowAparts, rendered)
		lastA = startA + len(cs)

		// arrow B
		bx := seg.x
		if bx > lastB {
			arrowBparts = append(arrowBparts, string(runeB[lastB:bx]))
		}
		arrowChar := "│"
		if isSelected {
			arrowChar = SepActive.Render("┃")
		} else {
			arrowChar = DimText.Render("│")
		}
		arrowBparts = append(arrowBparts, arrowChar)
		lastB = bx + 1
	}
	if lastA < w {
		arrowAparts = append(arrowAparts, string(runeA[lastA:]))
	}
	if lastB < w {
		arrowBparts = append(arrowBparts, string(runeB[lastB:]))
	}
	arrowLineA = strings.Join(arrowAparts, "")
	arrowLineB = strings.Join(arrowBparts, "")

	// Draw a horizontal line connecting from directorCenterX to each worker column
	// Only if workers are spread out — draw a simple ↓ line if only one worker
	var connLine string
	if len(workers) == 1 {
		cx := workerCenterX[0]
		runeC := []rune(strings.Repeat(" ", w))
		runeC[cx] = '↓'
		connLine = string(runeC)
	} else {
		// Draw horizontal bar connecting all arrows, with ↓ at worker positions
		// Find leftmost and rightmost worker center
		leftCX := workerCenterX[0]
		rightCX := workerCenterX[len(workerCenterX)-1]
		runeC := []rune(strings.Repeat(" ", w))
		// Vertical drop from director
		if dirCenterX >= 0 && dirCenterX < w {
			runeC[dirCenterX] = '┬'
		}
		// Horizontal bar
		for x := leftCX; x <= rightCX; x++ {
			if x >= 0 && x < w {
				if runeC[x] == ' ' {
					runeC[x] = '─'
				}
			}
		}
		// ↓ at each worker
		for _, cx := range workerCenterX {
			if cx >= 0 && cx < w {
				runeC[cx] = '↓'
			}
		}
		// Connect director to horizontal bar
		if dirCenterX < leftCX {
			for x := dirCenterX; x <= leftCX; x++ {
				if x >= 0 && x < w && runeC[x] == ' ' {
					runeC[x] = '─'
				}
			}
			if dirCenterX >= 0 && dirCenterX < w {
				runeC[dirCenterX] = '└'
			}
		} else if dirCenterX > rightCX {
			for x := rightCX; x <= dirCenterX; x++ {
				if x >= 0 && x < w && runeC[x] == ' ' {
					runeC[x] = '─'
				}
			}
			if dirCenterX >= 0 && dirCenterX < w {
				runeC[dirCenterX] = '┘'
			}
		}
		connLine = DimText.Render(string(runeC))
	}

	lines = append(lines, arrowLineA)
	lines = append(lines, arrowLineB)
	lines = append(lines, connLine)

	// Worker boxes row
	workerBox0 := make([]string, len(workers)) // top
	workerBox1 := make([]string, len(workers)) // mid
	workerBox2 := make([]string, len(workers)) // bot
	for i, wid := range workers {
		isSelected := false
		for _, seg := range arrowSegs {
			if seg.workerI == i && seg.ch == selectedChannel {
				isSelected = true
				break
			}
		}
		// Also check if there's an edge from worker to director that matches selectedChannel
		for _, e := range edges {
			if (e.from == wid || e.to == wid) && e.channel == selectedChannel {
				isSelected = true
				break
			}
		}
		wb := boxLines(wid, isSelected)
		workerBox0[i] = wb[0]
		workerBox1[i] = wb[1]
		workerBox2[i] = wb[2]
	}

	// Build worker row lines
	buildWorkerLine := func(parts []string) string {
		result := pad(workerLeft)
		for i, p := range parts {
			result += p
			if i < len(parts)-1 {
				result += pad(workerGap)
			}
		}
		return result
	}
	lines = append(lines, buildWorkerLine(workerBox0))
	lines = append(lines, buildWorkerLine(workerBox1))
	lines = append(lines, buildWorkerLine(workerBox2))

	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return lines
}

// viewNetworkDetail replaces viewTaskDetail at Level 2.
// Left panel (2/3): network graph. Right panel (1/3): selected channel thread.
func (m Model) viewNetworkDetail(t task.Task) string {
	h := m.height - 2
	headerH := 2
	footerH := 2
	contentH := h - headerH - footerH
	if contentH < 3 {
		contentH = 3
	}

	statusLabel := t.CoordinatorStatus
	if statusLabel == "" {
		statusLabel = "—"
	}
	breadcrumb := fmt.Sprintf("← Agente / %s  [%s]", t.TaskID, statusLabel)
	header1 := " " + ViewActive.Render(breadcrumb)
	header2 := " " + SepDim.Render(strings.Repeat("─", m.width-2))

	channels := listMsgChannels(t.Workspace)
	activeCh := 0
	if len(channels) > 0 {
		activeCh = m.activeMsgChannel % len(channels)
	}

	selectedChannel := ""
	if activeCh < len(channels) {
		selectedChannel = channels[activeCh]
	}

	// Panel widths
	graphW := (m.width * 2) / 3
	divW := 1
	threadW := m.width - graphW - divW
	if threadW < 20 {
		threadW = 20
		graphW = m.width - threadW - divW
	}

	// Left panel: network graph
	graphLines := renderNetworkGraph(t, channels, selectedChannel, graphW, contentH)

	// Right panel: thread of selected channel
	threadLines := m.buildNetworkThreadPanel(t, selectedChannel, threadW, contentH)

	div := SepDim.Render("│")
	var bodyLines []string
	for i := 0; i < contentH; i++ {
		var l, r string
		if i < len(graphLines) {
			l = graphLines[i]
		}
		if i < len(threadLines) {
			r = threadLines[i]
		}
		bodyLines = append(bodyLines, lipgloss.NewStyle().Width(graphW).Render(l)+div+r)
	}

	// Chat input line
	chatInput := m.viewNetworkChatInput()

	// Footer
	footerHint := "Tab/←/→ canal   Enter leer completo   o output agente   f/i escribir al director   Esc volver   Ctrl+Q salir"
	footer := " " + SepDim.Render(strings.Repeat("─", m.width-2)) + "\n" +
		FooterStyle.Width(m.width).Render(footerHint)

	all := []string{header1, header2}
	all = append(all, bodyLines...)
	all = append(all, chatInput)
	all = append(all, " "+SepDim.Render(strings.Repeat("─", m.width-2)))
	all = append(all, FooterStyle.Width(m.width).Render(footerHint))

	// Trim to height
	if len(all) > h {
		all = all[:h]
	}
	_ = footer
	return strings.Join(all, "\n")
}

// buildNetworkThreadPanel renders the right-side thread panel for the selected channel.
func (m Model) buildNetworkThreadPanel(t task.Task, selectedChannel string, w, h int) []string {
	var lines []string

	// Header
	chName := selectedChannel
	if chName == "" {
		chName = "—"
	} else {
		chName = strings.TrimSuffix(chName, ".md")
	}
	lines = append(lines, ViewActive.Render(" ❯ "+chName))
	lines = append(lines, " "+SepActive.Render(strings.Repeat("━", w-2)))

	chatH := h - len(lines)
	if chatH < 1 {
		chatH = 1
	}

	var allLines []string
	if selectedChannel == "" {
		allLines = []string{" " + DimText.Render("sin canal seleccionado")}
	} else {
		filePath := t.Workspace + "/messages/" + selectedChannel
		raw, err := os.ReadFile(filePath)
		if err != nil {
			allLines = []string{" " + DimText.Render("sin mensajes aún…")}
		} else {
			contentWidth := w - 2
			if contentWidth < 1 {
				contentWidth = 1
			}
			for _, l := range strings.Split(string(raw), "\n") {
				wrapped := ansi.Wrap(l, contentWidth, " ")
				for _, wl := range strings.Split(wrapped, "\n") {
					if strings.HasPrefix(strings.TrimSpace(wl), "###") {
						allLines = append(allLines, " "+SectionHeader.Render(wl))
					} else if wl == "" {
						allLines = append(allLines, "")
					} else {
						allLines = append(allLines, " "+DimText.Render(wl))
					}
				}
			}
		}
	}

	total := len(allLines)
	off := m.taskChatScrollOffset
	start := total - chatH - off
	if start < 0 {
		start = 0
	}
	end := start + chatH
	if end > total {
		end = total
	}
	visible := make([]string, end-start)
	copy(visible, allLines[start:end])
	for len(visible) < chatH {
		visible = append(visible, "")
	}
	lines = append(lines, visible...)

	for len(lines) < h {
		lines = append(lines, "")
	}
	return lines[:h]
}

// viewNetworkChatInput renders the chat input bar for the network view.
func (m Model) viewNetworkChatInput() string {
	if !m.taskChatFocused {
		return FooterStyle.Width(m.width).Render("Tab/←/→ canal   ↑/↓ scroll   f/i escribir")
	}

	buf := m.taskChatInput
	cursor := m.taskChatCursorPos
	if cursor > len(buf) {
		cursor = len(buf)
	}

	before := buf[:cursor]
	var cursorChar, after string
	if cursor < len(buf) {
		cursorChar = string(buf[cursor])
		after = buf[cursor+1:]
	}

	var rendered string
	if cursorChar != "" {
		rendered = before + InputCursor.Render(cursorChar) + after
	} else {
		rendered = before + InputCursor.Render(" ")
	}

	content := InputPrompt.Render("> ") + rendered
	return InputStyle.Width(m.width).Render(content)
}

func (m Model) viewTaskDetail(t task.Task) string {
	h := m.height - 2

	statusLabel := t.CoordinatorStatus
	if statusLabel == "" {
		statusLabel = "—"
	}
	breadcrumb := fmt.Sprintf("← Agente / %s  [%s]", t.TaskID, statusLabel)
	header1 := " " + ViewActive.Render(breadcrumb)
	header2 := " " + SepDim.Render(strings.Repeat("─", m.width-2))

	// Content area: total height minus header(2) + input(1) + footer-sep(1)
	contentH := h - 4
	if contentH < 3 {
		contentH = 3
	}

	listW := 30
	divW := 1
	msgW := m.width - listW - divW
	if msgW < 24 {
		msgW = 24
		listW = m.width - msgW - divW
	}

	leftLines := m.buildAgentListPanel(t, listW, contentH)
	rightLines := m.buildMsgPanel(t, msgW, contentH)

	div := SepDim.Render("│")
	var bodyLines []string
	for i := 0; i < contentH; i++ {
		var l, r string
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		bodyLines = append(bodyLines, lipgloss.NewStyle().Width(listW).Render(l)+div+r)
	}

	all := []string{header1, header2}
	all = append(all, bodyLines...)
	all = append(all, m.viewChatInput())
	all = append(all, " "+SepDim.Render(strings.Repeat("─", m.width-2)))

	if len(all) > h {
		all = all[:h]
	}
	return strings.Join(all, "\n")
}

func (m Model) buildAgentListPanel(t task.Task, w int, h int) []string {
	agents := task.AllAgents(t)
	var lines []string

	if m.taskSectionFocus == 0 {
		lines = append(lines, ViewActive.Render(" ❯ AGENTES"))
		lines = append(lines, " "+SepActive.Render(strings.Repeat("━", w-2)))
	} else {
		lines = append(lines, DimText.Render("   AGENTES"))
		lines = append(lines, " "+SepDim.Render(strings.Repeat("─", w-2)))
	}

	for i, a := range agents {
		status := task.AgentStatus(t.Workspace, a.ID)
		var bullet string
		switch status {
		case "done", "completed":
			bullet = StatusGreen.Render("✔")
		case "working":
			bullet = StatusYellow.Render("◉")
		default:
			bullet = DimText.Render("◌")
		}

		id := a.ID
		maxLen := w - 7
		if maxLen < 4 {
			maxLen = 4
		}
		if len(id) > maxLen {
			id = id[:maxLen-1] + "…"
		}

		selected := i == m.multiAgentIdx && m.taskSectionFocus == 0
		if selected {
			lines = append(lines, fmt.Sprintf(" ❯ %s %s", bullet, ViewActive.Render(id)))
		} else {
			lines = append(lines, fmt.Sprintf("   %s %s", bullet, id))
		}
	}

	for len(lines) < h {
		lines = append(lines, "")
	}
	return lines[:h]
}

func (m Model) buildMsgPanel(t task.Task, w int, h int) []string {
	channels := listMsgChannels(t.Workspace)
	activeCh := 0
	if len(channels) > 0 {
		activeCh = m.activeMsgChannel % len(channels)
	}

	var lines []string

	if m.taskSectionFocus == 1 {
		lines = append(lines, ViewActive.Render(" ❯ MENSAJES"))
		lines = append(lines, " "+SepActive.Render(strings.Repeat("━", w-2)))
	} else {
		lines = append(lines, DimText.Render("   MENSAJES"))
		lines = append(lines, " "+SepDim.Render(strings.Repeat("─", w-2)))
	}

	if len(channels) == 0 {
		lines = append(lines, " "+DimText.Render("sin canales"))
	} else {
		sep := TabSeparator.Render(" │ ")
		var parts []string
		for i, ch := range channels {
			name := strings.TrimSuffix(ch, ".md")
			if len([]rune(name)) > 14 {
				name = string([]rune(name)[:13]) + "…"
			}
			if i == activeCh {
				parts = append(parts, ViewActive.Render(name))
			} else {
				parts = append(parts, ViewInactive.Render(name))
			}
			if i < len(channels)-1 {
				parts = append(parts, sep)
			}
		}
		lines = append(lines, " "+strings.Join(parts, ""))
	}
	lines = append(lines, " "+SepDim.Render(strings.Repeat("─", w-2)))

	chatH := h - len(lines)
	if chatH < 1 {
		chatH = 1
	}

	var allLines []string
	if len(channels) == 0 {
		allLines = []string{" " + DimText.Render("sin mensajes")}
	} else if activeCh < len(channels) {
		filePath := t.Workspace + "/messages/" + channels[activeCh]
		raw, err := os.ReadFile(filePath)
		if err != nil {
			allLines = []string{" " + DimText.Render("sin mensajes aún…")}
		} else {
			contentWidth := w - 2
			if contentWidth < 1 {
				contentWidth = 1
			}
			for _, l := range strings.Split(string(raw), "\n") {
				wrapped := ansi.Wrap(l, contentWidth, " ")
				for _, wl := range strings.Split(wrapped, "\n") {
					if strings.HasPrefix(strings.TrimSpace(wl), "#") {
						allLines = append(allLines, " "+SectionHeader.Render(wl))
					} else if wl == "" {
						allLines = append(allLines, "")
					} else {
						allLines = append(allLines, " "+DimText.Render(wl))
					}
				}
			}
		}
	}

	total := len(allLines)
	off := m.taskChatScrollOffset
	start := total - chatH - off
	if start < 0 {
		start = 0
	}
	end := start + chatH
	if end > total {
		end = total
	}
	visible := make([]string, end-start)
	copy(visible, allLines[start:end])
	for len(visible) < chatH {
		visible = append(visible, "")
	}
	lines = append(lines, visible...)

	for len(lines) < h {
		lines = append(lines, "")
	}
	return lines[:h]
}

// viewOfficeFloor renders an animated grid of agent desks.
func (m Model) viewOfficeFloor(t task.Task) string {
	agents := task.AllAgents(t)
	if len(agents) == 0 {
		return "  " + DimText.Render("sin agentes registrados") + "\n"
	}

	const deskW = 24
	const deskGap = 1
	desksPerRow := m.width / (deskW + deskGap)
	if desksPerRow < 1 {
		desksPerRow = 1
	}

	var out []string
	out = append(out, "  "+SectionHeader.Render(fmt.Sprintf("⬡  OFFICE FLOOR  %d agentes", len(agents))))

	for rowStart := 0; rowStart < len(agents); rowStart += desksPerRow {
		rowEnd := rowStart + desksPerRow
		if rowEnd > len(agents) {
			rowEnd = len(agents)
		}
		rowAgents := agents[rowStart:rowEnd]

		// Render each desk as a multi-line string
		deskStrings := make([]string, len(rowAgents))
		for i, a := range rowAgents {
			globalIdx := rowStart + i
			status := task.AgentStatus(t.Workspace, a.ID)
			isDirector := a.ID == "director" || a.IsSynthesizer
			var frame string
			if isDirector {
				frame = spriteDirector[m.spriteIdx]
			} else {
				frame = agentSpriteFrame(status, m.spriteIdx)
			}
			selected := globalIdx == m.multiAgentIdx
			deskLines := renderDeskBox(a, status, frame, isDirector, deskW, selected)
			deskStrings[i] = strings.Join(deskLines, "\n")
		}

		// Join desks side by side with a gap
		rowStr := lipgloss.JoinHorizontal(lipgloss.Top, deskStrings...)
		out = append(out, rowStr)
	}

	return strings.Join(out, "\n")
}

// viewMsgChannelTabs renders a navigable tab bar for message channels.
func (m Model) viewMsgChannelTabs(channels []string) string {
	if len(channels) == 0 {
		return "  " + DimText.Render("sin canales de mensajes")
	}

	active := m.activeMsgChannel % len(channels)

	sep := TabSeparator.Render(" │ ")
	var parts []string
	for i, ch := range channels {
		name := ch
		if len([]rune(name)) > 28 {
			name = string([]rune(name)[:27]) + "…"
		}
		if i == active {
			parts = append(parts, ViewActive.Render(name))
		} else {
			parts = append(parts, ViewInactive.Render(name))
		}
		if i < len(channels)-1 {
			parts = append(parts, sep)
		}
	}
	return "  " + strings.Join(parts, "")
}

// viewChatPanel renders the selected channel file as a scrollable chat panel.
func (m Model) viewChatPanel(workspace, channelFile string, contentH int) string {
	if contentH < 1 {
		contentH = 1
	}

	var allLines []string

	if channelFile == "" {
		allLines = []string{"  " + DimText.Render("↑ selecciona un canal con Tab")}
	} else {
		filePath := workspace + "/messages/" + channelFile
		raw, err := os.ReadFile(filePath)
		if err != nil {
			allLines = []string{"  " + DimText.Render("sin mensajes aún…")}
		} else {
			contentWidth := m.width - 2
			if contentWidth < 1 {
				contentWidth = 1
			}
			for _, l := range strings.Split(string(raw), "\n") {
				wrapped := ansi.Wrap(l, contentWidth, " ")
				for _, wl := range strings.Split(wrapped, "\n") {
					if strings.HasPrefix(strings.TrimSpace(wl), "#") {
						allLines = append(allLines, " "+SectionHeader.Render(wl))
					} else if wl == "" {
						allLines = append(allLines, "")
					} else {
						allLines = append(allLines, " "+DimText.Render(wl))
					}
				}
			}
		}
	}

	// Scroll: offset 0 = tail, increasing offset scrolls up
	total := len(allLines)
	off := m.taskChatScrollOffset
	start := total - contentH - off
	if start < 0 {
		start = 0
	}
	end := start + contentH
	if end > total {
		end = total
	}

	visible := make([]string, end-start)
	copy(visible, allLines[start:end])
	for len(visible) < contentH {
		visible = append(visible, "")
	}

	return strings.Join(visible, "\n")
}

// viewChatInput renders the chat input bar (focused or hint mode).
func (m Model) viewChatInput() string {
	if !m.taskChatFocused {
		sectionHint := "foco: AGENTES"
		if m.taskSectionFocus == 1 {
			sectionHint = "foco: MENSAJES"
		}
		hint := sectionHint + "  Ctrl+Shift+↑↓ cambiar  ↑↓ navegar  ←/→ canal  f/i escribir  Esc volver  Ctrl+Q salir"
		return FooterStyle.Width(m.width).Render(hint)
	}

	buf := m.taskChatInput
	cursor := m.taskChatCursorPos
	if cursor > len(buf) {
		cursor = len(buf)
	}

	before := buf[:cursor]
	var cursorChar, after string
	if cursor < len(buf) {
		cursorChar = string(buf[cursor])
		after = buf[cursor+1:]
	}

	var rendered string
	if cursorChar != "" {
		rendered = before + InputCursor.Render(cursorChar) + after
	} else {
		rendered = before + InputCursor.Render(" ")
	}

	content := InputPrompt.Render("> ") + rendered
	return InputStyle.Width(m.width).Render(content)
}

func formatTaskDate(created string) string {
	t, err := time.Parse("2006-01-02 15:04:05", created)
	if err != nil {
		if len(created) >= 16 {
			return created[5:16]
		}
		return created
	}
	return t.Format("Jan 02 15:04")
}

func (m Model) viewMetricas() string {
	// Collect all session IDs that belong to any greg task
	taskSessions := map[string]bool{}
	if tasks, err := task.LoadTasks(); err == nil {
		for _, t := range tasks {
			if t.SynthesizerID != "" {
				taskSessions[t.SynthesizerID] = true
			}
			for _, a := range t.Agents {
				if a.SessionID != "" {
					taskSessions[a.SessionID] = true
				}
			}
		}
	}

	isStandalone := func(s session.Session) bool { return !taskSessions[s.ID] }

	// Active sessions (from sessions.json)
	active, _ := session.LoadSessions()
	// Finished sessions (from history.json)
	finished, _ := session.LoadFinishedSessions()

	// Merge: active first, then finished; filter task sessions; dedup by ID
	seen := map[string]bool{}
	var all []session.Session
	for _, s := range active {
		if isStandalone(s) && !seen[s.ID] {
			seen[s.ID] = true
			all = append(all, s)
		}
	}
	for _, s := range finished {
		if isStandalone(s) && !seen[s.ID] {
			seen[s.ID] = true
			all = append(all, s)
		}
	}

	// Active sessions first, then finished; within each group newest first
	tsKey := func(s session.Session) string {
		if s.Ended != "" {
			return s.Ended
		}
		return s.Started
	}
	sort.Slice(all, func(i, j int) bool {
		ai := all[i].Status == "active"
		aj := all[j].Status == "active"
		if ai != aj {
			return ai // active before finished
		}
		return tsKey(all[i]) > tsKey(all[j])
	})
	sessions := all

	sidebarW := 22
	divW := 1
	chatW := m.width - sidebarW - divW
	if chatW < 40 {
		chatW = 40
		sidebarW = m.width - chatW - divW
	}
	contentH := m.height - 2

	// Sidebar (left)
	sidebarLines := m.buildSidebarLines(sessions, sidebarW, contentH)

	// Chat (right) — render at chatW by narrowing width on a copy
	chatM := m
	chatM.width = chatW
	question := chatM.viewQuestionOverlay()
	questionH := 0
	if question != "" {
		questionH = strings.Count(question, "\n") + 1
	}
	outputH := contentH - 4 - questionH
	if outputH < 1 {
		outputH = 1
	}
	tabBar := chatM.viewTabBar()
	statusBar := chatM.viewStatusBar()
	output := chatM.viewOutputH(outputH)
	input := chatM.viewInput()
	footer := chatM.viewFooter()
	var chatSections []string
	chatSections = append(chatSections, tabBar, statusBar, output)
	if question != "" {
		chatSections = append(chatSections, question)
	}
	chatSections = append(chatSections, input, footer)
	chatLines := strings.Split(lipgloss.JoinVertical(lipgloss.Left, chatSections...), "\n")

	div := SepDim.Render("│")

	// Merge: sidebar | div | chat
	var lines []string
	for i := 0; i < contentH; i++ {
		sl, cl := "", ""
		if i < len(sidebarLines) {
			sl = sidebarLines[i]
		}
		if i < len(chatLines) {
			cl = chatLines[i]
		}
		lines = append(lines, lipgloss.NewStyle().Width(sidebarW).Render(sl)+div+cl)
	}
	return strings.Join(lines, "\n")
}

func (m Model) buildSidebarLines(sessions []session.Session, w int, h int) []string {
	var lines []string
	lines = append(lines, "")
	lines = append(lines, " "+SectionHeader.Render("Sesiones"))
	lines = append(lines, " "+SepDim.Render(strings.Repeat("─", w-2)))

	if len(sessions) == 0 {
		lines = append(lines, "  "+DimText.Render("sin sesiones"))
	}
	for i, s := range sessions {
		id := strings.TrimPrefix(s.ID, "greg-")
		if len(id) > 8 {
			id = id[:8]
		}
		var bullet string
		if s.Status == "active" {
			bullet = StatusGreen.Render("◉")
		} else {
			bullet = DimText.Render("◌")
		}
		ts := s.Started
		if s.Ended != "" {
			ts = s.Ended
		}
		ago := timeAgo(ts)
		content := fmt.Sprintf("%s  %-8s  %s", bullet, id, ago)

		prefix := " "
		if i == m.sidebarIdx {
			prefix = ViewActive.Render("❯")
			lines = append(lines, prefix+" "+ViewActive.Render(content))
		} else {
			lines = append(lines, prefix+" "+content)
		}
	}

	// Hints pinned near bottom
	hintY := h - 3
	for len(lines) < hintY {
		lines = append(lines, "")
	}
	lines = append(lines, " "+DimText.Render("⇧↑↓ navegar  ⇧↵ abrir"))
	lines = append(lines, "")

	for len(lines) < h {
		lines = append(lines, "")
	}
	return lines[:h]
}

func timeAgo(started string) string {
	t, err := time.Parse("2006-01-02 15:04:05", started)
	if err != nil {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "ahora"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func (m Model) viewStatusBar() string {
	t := m.tab()

	var parts []string

	if t.Running {
		parts = append(parts, StatusYellow.Render("◉"))
	} else {
		parts = append(parts, StatusGreen.Render("✔"))
	}

	modelShort := strings.TrimPrefix(t.Model, "claude-")
	parts = append(parts, ModelStyle.Render(fmt.Sprintf(" [%s·%s]", modelShort, t.Effort)))

	if t.ContextPct >= 0 {
		parts = append(parts, " "+m.viewCtx(t))
	}

	content := strings.Join(parts, "")
	return StatusBarStyle.Width(m.width).Render(content)
}

func fmtTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%dk", n/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func (m Model) viewCtx(t *Tab) string {
	pct := t.ContextPct
	const barW = 10
	filled := barW * pct / 100
	if filled > barW {
		filled = barW
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	label := fmt.Sprintf("%d%% %s %s/%s", pct, bar, fmtTokens(t.ContextTokens), fmtTokens(t.ContextWindow))

	if pct >= 90 {
		return CtxHigh.Render(label)
	}
	if pct >= 75 {
		return CtxMed.Render(label)
	}
	return CtxLow.Render(label)
}

func (m Model) viewTabBar() string {
	var parts []string
	sep := TabSeparator.Render(" │ ")

	for i, t := range m.tabs {
		label := t.Name
		if i == m.tabIdx {
			parts = append(parts, TabActive.Render(label))
		} else {
			s := TabInactive.Render(label)
			if t.HasNew {
				s += " " + TabNewBadge.Render("●")
			}
			parts = append(parts, s)
		}
		if i < len(m.tabs)-1 {
			parts = append(parts, sep)
		}
	}

	content := "  " + strings.Join(parts, "")
	return TabBarStyle.Width(m.width).Render(content)
}

func (m Model) viewOutputH(oh int) string {
	t := m.tab()
	if oh <= 0 {
		oh = 1
	}

	// Expand stored lines by word-wrapping at the available content width.
	// OutputStyle has Padding(0, 1) so content width is m.width - 2.
	contentWidth := m.width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}
	raw := t.Lines
	if len(raw) == 0 {
		raw = []string{""}
	}
	var lines []string
	for _, line := range raw {
		wrapped := ansi.Wrap(line, contentWidth, " ")
		for _, wl := range strings.Split(wrapped, "\n") {
			lines = append(lines, wl)
		}
	}
	if len(lines) == 0 {
		lines = []string{""}
	}

	start := len(lines) - oh - t.ScrollOffset
	if start < 0 {
		start = 0
	}
	end := start + oh
	if end > len(lines) {
		end = len(lines)
	}

	visible := make([]string, len(lines[start:end]))
	copy(visible, lines[start:end])

	// Pad to fill height
	for len(visible) < oh {
		visible = append(visible, "")
	}

	// Inject spinner just below the last content line when running and at the bottom
	if t.Running && t.ScrollOffset == 0 {
		action := t.CurrentAction
		if action == "" {
			action = "pensando…"
		}
		spinnerLine := SpinnerStyle.Render("  "+spinFrames[m.spinIdx]+" ") + DimText.Render(action)

		lastContent := -1
		for i := len(visible) - 1; i >= 0; i-- {
			if strings.TrimSpace(visible[i]) != "" {
				lastContent = i
				break
			}
		}
		insertAt := lastContent + 1
		if insertAt >= len(visible) {
			insertAt = len(visible) - 1
		}
		visible[insertAt] = spinnerLine
	}

	content := strings.Join(visible, "\n")
	return OutputStyle.Width(m.width).Height(oh).Render(content)
}

func (m Model) viewQuestionOverlay() string {
	t := m.tab()
	q := t.PendingQuestion
	if q == nil {
		return ""
	}

	qData := q.Questions[q.CurrentQ]
	var lines []string

	lines = append(lines, lipgloss.NewStyle().Bold(true).Render(qData.Question))
	lines = append(lines, "")

	for i, opt := range qData.Options {
		var marker, label string
		if i == q.SelectedIdx {
			marker = QuestionSelected.Render("▶")
			label = QuestionLabel.Render(opt.Label)
		} else {
			marker = " "
			label = QuestionLabelDim.Render(opt.Label)
		}
		desc := ""
		if opt.Description != "" {
			d := opt.Description
			if len(d) > 65 {
				d = d[:65]
			}
			desc = "  " + DimText.Render(d)
		}
		lines = append(lines, fmt.Sprintf("  %s %s%s", marker, label, desc))
	}

	lines = append(lines, "")
	hint := "↑/↓ navegar  Enter confirmar  Esc cancelar"
	if qData.MultiSelect {
		hint = "↑/↓ navegar  Space marcar  Enter confirmar  Esc cancelar"
	}
	lines = append(lines, DimText.Render("  "+hint))

	header := qData.Header
	if header == "" {
		header = "Selección"
	}

	content := strings.Join(lines, "\n")
	return QuestionBorder.Width(m.width - 4).
		BorderTop(true).BorderBottom(true).BorderLeft(true).BorderRight(true).
		Render(lipgloss.NewStyle().Foreground(lipgloss.Color("#3377ff")).Render(" "+header+" ") + "\n" + content)
}

func (m Model) viewInput() string {
	t := m.tab()
	buf := t.InputBuf

	before := buf[:t.CursorPos]
	var cursorChar string
	after := ""
	if t.CursorPos < len(buf) {
		cursorChar = string(buf[t.CursorPos])
		after = buf[t.CursorPos+1:]
	}

	var rendered string
	if cursorChar == "\n" {
		rendered = before + InputCursor.Render(" ") + "\n" + after
	} else if cursorChar != "" {
		rendered = before + InputCursor.Render(cursorChar) + after
	} else {
		rendered = before + InputCursor.Render(" ")
	}

	content := InputPrompt.Render("> ") + rendered
	return InputStyle.Width(m.width).Render(content)
}

func (m Model) viewFooter() string {
	keys := "Enter enviar  Alt+Enter nueva línea  ↑/↓ scroll  PgUp/PgDn saltar  Alt+↑/↓ historial  Ctrl+Shift+←/→ tabs  Ctrl+T nueva  Ctrl+W cerrar  Ctrl+K compactar  Ctrl+Q salir"
	return FooterStyle.Width(m.width).Render(keys)
}

func (m Model) viewConfig() string {
	h := m.height - 2

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  "+SectionHeader.Render("⚙  Configuración"))
	lines = append(lines, "  "+SepDim.Render(strings.Repeat("─", m.width-4)))
	lines = append(lines, "")
	lines = append(lines, "  "+ModelStyle.Render("Tema"))
	lines = append(lines, "")

	darkLabel := "  Dark mode    oscuro — navy + cyan"
	lightLabel := "  Light mode   claro — Gruvbox warm"

	if m.darkMode {
		lines = append(lines, "    "+ViewActive.Render("▶")+" "+QuestionLabel.Render(darkLabel))
		lines = append(lines, "      "+QuestionLabelDim.Render(lightLabel))
	} else {
		lines = append(lines, "      "+QuestionLabelDim.Render(darkLabel))
		lines = append(lines, "    "+ViewActive.Render("▶")+" "+QuestionLabel.Render(lightLabel))
	}

	lines = append(lines, "")

	for len(lines) < h-1 {
		lines = append(lines, "")
	}

	hint := "↑/↓  cambiar tema  Esc  volver  Ctrl+Q  salir"
	lines = append(lines, FooterStyle.Width(m.width).Render(hint))

	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}
