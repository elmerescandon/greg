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

	// Reserve 2 visual chars for preset tag ("⌨ ")
	tagLen := 0
	if t.Preset == "coding" {
		tagLen = 2
	}
	goal := t.Goal
	if len(goal) > goalW-tagLen {
		goal = goal[:goalW-tagLen-1] + "…"
	}

	if selected {
		presetStr := ""
		if t.Preset == "coding" {
			presetStr = "⌨ "
		}
		line := fmt.Sprintf("%s● %-*s  %-*s  %-*s  %s%s",
			selector, statusW, status, idW, tid, dateW, dateStr, presetStr, goal)
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
	rowPrefix := fmt.Sprintf(" %-*s  %-*s  %-*s  ", statusW, status, idW, tid, dateW, dateStr)
	if t.Preset == "coding" {
		return selector + coloredBullet + TaskRowDim.Render(rowPrefix) + StatusYellow.Render("⌨") + " " + TaskRowDim.Render(goal)
	}
	return selector + coloredBullet + TaskRowDim.Render(rowPrefix+goal)
}

// listLogFiles returns sorted log filenames in the workspace root (coordinator.log, etc.).
func listLogFiles(workspace string) []string {
	entries, err := os.ReadDir(workspace)
	if err != nil {
		return nil
	}
	var logs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			logs = append(logs, e.Name())
		}
	}
	sort.Strings(logs)
	return logs
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
	} else if m.multiDocSource == "logs" {
		docs = listLogFiles(t.Workspace)
		fileDir = t.Workspace + "/"
		sourceLabel = "logs"
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

// renderNetworkGraph renders the Command Center table view for the given task.
// selectedChannel is the basename of the currently selected messages/*.md file.
// Returns a slice of lines of width w.
func (m Model) renderNetworkGraph(t task.Task, channels []string, selectedChannel string, w, h int) []string {
	if w < 10 {
		w = 10
	}

	agents := task.AllAgents(t)

	// Parse channels into from/to pairs with message counts
	type msgEdge struct {
		from, to string
		count    int
	}
	var msgEdges []msgEdge
	for _, ch := range channels {
		base := strings.TrimSuffix(ch, ".md")
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
		msgEdges = append(msgEdges, msgEdge{from: from, to: to, count: count})
	}

	msgsIn := func(id string) int {
		n := 0
		for _, e := range msgEdges {
			if e.to == id {
				n += e.count
			}
		}
		return n
	}
	msgsOut := func(id string) int {
		n := 0
		for _, e := range msgEdges {
			if e.from == id {
				n += e.count
			}
		}
		return n
	}

	rpad := func(s string, n int) string {
		r := []rune(s)
		for len(r) < n {
			r = append(r, ' ')
		}
		return string(r)
	}
	truncate := func(s string, max int) string {
		r := []rune(s)
		if len(r) > max {
			return string(r[:max-1]) + "…"
		}
		return s
	}

	var lines []string

	// ══ COMMAND CENTER ═...═ ◉ N UNITS ══
	nUnits := len(agents)
	leftPart := "══ COMMAND CENTER "
	rightPart := fmt.Sprintf(" ◉ %d UNITS ══", nUnits)
	fillW := w - len([]rune(leftPart)) - len([]rune(rightPart))
	if fillW < 2 {
		fillW = 2
	}
	headerRaw := leftPart + strings.Repeat("═", fillW) + rightPart
	lines = append(lines, ViewActive.Render(headerRaw))

	lines = append(lines, "")

	// Director row
	dirStatus := task.AgentStatus(t.Workspace, "director")
	var dirBullet string
	var dirStyle lipgloss.Style
	switch dirStatus {
	case "done", "completed":
		dirBullet = "✔"
		dirStyle = StatusGreen
	case "working":
		dirBullet = spinFrames[m.spinIdx]
		dirStyle = StatusYellow
	default:
		dirBullet = "◌"
		dirStyle = DimText
	}
	dirLine := "  " + dirStyle.Render(dirBullet) + "  " +
		ViewActive.Render(rpad("DIRECTOR", 14)) +
		"  " + dirStyle.Render(rpad(dirStatus, 12)) +
		"  " + DimText.Render(fmt.Sprintf("%-8d", msgsIn("director"))) +
		"  " + DimText.Render(fmt.Sprintf("%d", msgsOut("director")))
	lines = append(lines, dirLine)

	// Separator
	sepW := w - 4
	if sepW < 2 {
		sepW = 2
	}
	lines = append(lines, "  "+DimText.Render(strings.Repeat("─", sepW)))

	// Table header
	lines = append(lines, "  "+
		SectionHeader.Render(rpad("UNIT", 16))+
		SectionHeader.Render(rpad("STATUS", 14))+
		SectionHeader.Render(rpad("MSGS IN", 10))+
		SectionHeader.Render("MSGS OUT"))

	// Agent rows (non-director)
	for _, a := range agents {
		if a.ID == "director" {
			continue
		}
		status := task.AgentStatus(t.Workspace, a.ID)
		var bullet string
		var aStyle lipgloss.Style
		switch status {
		case "done", "completed":
			bullet = "✔"
			aStyle = StatusGreen
		case "working":
			bullet = spinFrames[m.spinIdx]
			aStyle = StatusYellow
		default:
			bullet = "◌"
			aStyle = DimText
		}
		idTrunc := truncate(a.ID, 14)
		row := "  " + aStyle.Render(bullet) + " " +
			DimText.Render(rpad(idTrunc, 15)) +
			" " + aStyle.Render(rpad(status, 13)) +
			" " + DimText.Render(fmt.Sprintf("%-10d", msgsIn(a.ID))) +
			DimText.Render(fmt.Sprintf("%d", msgsOut(a.ID)))
		lines = append(lines, row)
	}

	lines = append(lines, "")

	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return lines
}

// viewNetworkDetail — full-width animated network graph view.
func (m Model) viewNetworkDetail(t task.Task) string {
	h := m.height - 2
	headerH := 2
	edgeBarH := 2 // edge info line + its separator
	footerH := 3  // chat-input + sep + footer-hint
	contentH := h - headerH - edgeBarH - footerH
	if contentH < 3 {
		contentH = 3
	}

	statusLabel := t.CoordinatorStatus
	if statusLabel == "" {
		statusLabel = "—"
	}

	// Live agent summary for header
	agents := task.AllAgents(t)
	nWorking := 0
	nDone := 0
	for _, a := range agents {
		switch task.AgentStatus(t.Workspace, a.ID) {
		case "working":
			nWorking++
		case "done", "completed":
			nDone++
		}
	}
	var agentSummary string
	switch {
	case nWorking > 0:
		agentSummary = "  " + StatusYellow.Render(spinFrames[m.spinIdx]+" "+fmt.Sprintf("%d activos", nWorking))
		if len(agents) > nWorking {
			agentSummary += DimText.Render(fmt.Sprintf(" · %d total", len(agents)))
		}
	case len(agents) > 0 && nDone == len(agents):
		agentSummary = "  " + StatusGreen.Render(fmt.Sprintf("✔ %d completados", nDone))
	case len(agents) > 0:
		agentSummary = "  " + DimText.Render(fmt.Sprintf("%d agentes", len(agents)))
	}

	breadcrumb := fmt.Sprintf("← Agente / %s", t.TaskID)
	header1 := " " + ViewActive.Render(breadcrumb) + "  " + DimText.Render("["+statusLabel+"]") + agentSummary
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

	// Full-width network graph
	graphLines := m.renderNetworkGraph(t, channels, selectedChannel, m.width-2, contentH)

	// Edge info bar
	edgeSep := " " + SepDim.Render(strings.Repeat("─", m.width-2))
	edgeInfo := m.renderEdgeInfo(t, selectedChannel, channels, activeCh)

	// Chat input + footer
	hint := "Tab/←/→ canal   Enter leer canal   o workspace agente   l logs   f/i escribir   Esc volver   Ctrl+Q salir"

	var all []string
	all = append(all, header1, header2)
	for _, gl := range graphLines {
		all = append(all, " "+gl)
	}
	all = append(all, edgeSep, edgeInfo)
	all = append(all, m.viewNetworkChatInput())
	all = append(all, " "+SepDim.Render(strings.Repeat("─", m.width-2)))
	all = append(all, FooterStyle.Width(m.width).Render(hint))

	if len(all) > h {
		all = all[:h]
	}
	return strings.Join(all, "\n")
}

// renderEdgeInfo renders the edge info bar showing the selected channel and message count.
func (m Model) renderEdgeInfo(t task.Task, selectedChannel string, channels []string, activeCh int) string {
	if len(channels) == 0 || selectedChannel == "" {
		return FooterStyle.Width(m.width).Render("  sin canales de comunicación")
	}

	chName := strings.TrimSuffix(selectedChannel, ".md")
	count := countMsgsInFile(t.Workspace + "/messages/" + selectedChannel)

	var countStr string
	switch count {
	case 0:
		countStr = DimText.Render("sin mensajes")
	case 1:
		countStr = ViewActive.Render("1 mensaje")
	default:
		countStr = ViewActive.Render(fmt.Sprintf("%d mensajes", count))
	}

	pos := DimText.Render(fmt.Sprintf("[%d/%d]", activeCh+1, len(channels)))

	// Pulse animation when one of the channel's agents is working
	pulse := ""
	chBase := strings.TrimSuffix(selectedChannel, ".md")
	var from, to string
	if idx := strings.Index(chBase, "→"); idx >= 0 {
		from = chBase[:idx]
		to = chBase[idx+len("→"):]
	} else if idx := strings.Index(chBase, "->"); idx >= 0 {
		from = chBase[:idx]
		to = chBase[idx+2:]
	}
	if from != "" || to != "" {
		fs := task.AgentStatus(t.Workspace, from)
		ts := task.AgentStatus(t.Workspace, to)
		if fs == "working" || ts == "working" {
			pulse = "  " + StatusYellow.Render(spinFrames[m.spinIdx]+" activo")
		}
	}

	line := fmt.Sprintf("  ❯ %s  %s  %s%s    %s",
		ViewActive.Render(chName),
		countStr,
		pos,
		pulse,
		DimText.Render("Enter para leer completo"),
	)
	return FooterStyle.Width(m.width).Render(line)
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
	sessions := computeStandaloneSessions()

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

	renderItem := func(idx int, label, value, desc string) string {
		cursor := "  "
		if m.configCursorIdx == idx {
			cursor = ViewActive.Render("▶") + " "
		}
		var valueStr string
		if m.configCursorIdx == idx {
			valueStr = QuestionLabel.Render(value)
		} else {
			valueStr = QuestionLabelDim.Render(value)
		}
		descStr := ""
		if desc != "" {
			descStr = "   " + DimText.Render(desc)
		}
		return "  " + cursor + ModelStyle.Render(fmt.Sprintf("%-22s", label)) + " " + valueStr + descStr
	}

	// 0: Tema
	themeVal, themeDesc := "Dark mode", "oscuro — navy + cyan"
	if !m.cfg.DarkMode {
		themeVal, themeDesc = "Light mode", "claro — Gruvbox warm"
	}
	lines = append(lines, renderItem(0, "Tema", themeVal, themeDesc))
	lines = append(lines, "")

	// 1: Modelo por defecto
	modelLabel, modelDesc := m.cfg.DefaultModel, ""
	for _, mo := range ModelOptions {
		if mo.ID == m.cfg.DefaultModel {
			modelLabel = mo.Label
			modelDesc = mo.Desc
			break
		}
	}
	lines = append(lines, renderItem(1, "Modelo por defecto", modelLabel, modelDesc))

	// 2: Esfuerzo por defecto
	effortLabel, effortDesc := m.cfg.DefaultEffort, ""
	for _, eo := range EffortOptions {
		if eo.ID == m.cfg.DefaultEffort {
			effortLabel = eo.Label
			effortDesc = eo.Desc
			break
		}
	}
	lines = append(lines, renderItem(2, "Esfuerzo por defecto", effortLabel, effortDesc))
	lines = append(lines, "")

	// 3: Umbral compactación
	warnStr := fmt.Sprintf("%d%%", m.cfg.CompactWarnPct)
	lines = append(lines, renderItem(3, "Umbral compactación", warnStr, fmt.Sprintf("avisa · compacta al %d%%", m.cfg.CompactWarnPct+5)))

	// 4: Auto-compact
	autoStr := "desactivado"
	if m.cfg.AutoCompact {
		autoStr = "activado"
	}
	lines = append(lines, renderItem(4, "Auto-compact", autoStr, ""))
	lines = append(lines, "")

	// 5: Timeout idle
	timeoutStr := "nunca"
	switch m.cfg.IdleTimeoutHours {
	case 4:
		timeoutStr = "4 horas"
	case 8:
		timeoutStr = "8 horas"
	case 24:
		timeoutStr = "24 horas"
	}
	lines = append(lines, renderItem(5, "Timeout idle", timeoutStr, "sesiones sin abrir se archivan"))

	for len(lines) < h-1 {
		lines = append(lines, "")
	}

	hint := "↑/↓ navegar  ←/→ cambiar  Esc volver  Ctrl+Q salir"
	lines = append(lines, FooterStyle.Width(m.width).Render(hint))

	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}
