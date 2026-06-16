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
		agents := task.AllAgents(tasks[sel])
		agentIdx := m.multiAgentIdx
		if agentIdx >= len(agents) {
			agentIdx = len(agents) - 1
		}
		if agentIdx >= 0 {
			return m.viewAgentOutput(tasks[sel], agents[agentIdx])
		}
	}

	if m.multiDetailMode && sel >= 0 && sel < n {
		return m.viewTaskDetail(tasks[sel])
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

func (m Model) viewAgentOutput(t task.Task, a task.Agent) string {
	h := m.height - 2
	headerH := 2
	footerH := 2
	contentH := h - headerH - footerH
	if contentH < 1 {
		contentH = 1
	}

	// Read workspace file — refreshed on every tick
	filePath := t.Workspace + "/workspace/" + a.ID + ".md"
	raw, err := os.ReadFile(filePath)
	var allLines []string
	if err != nil {
		allLines = []string{"", "  " + DimText.Render("sin output aún…")}
	} else {
		contentWidth := m.width - 2
		if contentWidth < 1 {
			contentWidth = 1
		}
		for _, l := range strings.Split(string(raw), "\n") {
			wrapped := ansi.Wrap(l, contentWidth, " ")
			for _, wl := range strings.Split(wrapped, "\n") {
				allLines = append(allLines, " "+wl)
			}
		}
	}

	// Scroll: offset 0 = bottom (tail), increasing offset scrolls up
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
	agentStatus := task.AgentStatus(t.Workspace, a.ID)
	var statusStr string
	switch agentStatus {
	case "done", "completed":
		statusStr = StatusGreen.Render("✔ " + agentStatus)
	case "running", "working":
		statusStr = StatusYellow.Render("◉ " + agentStatus + " " + spinFrames[m.spinIdx])
	default:
		statusStr = DimText.Render("◌ " + agentStatus)
	}
	var breadcrumb string
	if a.IsSynthesizer {
		breadcrumb = fmt.Sprintf("← %s / ⚡ synthesizer", t.TaskID)
	} else {
		breadcrumb = fmt.Sprintf("← %s / %s", t.TaskID, a.ID)
	}
	var header1 string
	if a.IsSynthesizer {
		header1 = " " + SynthesizerStyle.Render(breadcrumb) + "   " + statusStr
	} else {
		header1 = " " + ViewActive.Render(breadcrumb) + "   " + statusStr
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
	out = append(out, " "+SepDim.Render(strings.Repeat("─", m.width-2)))
	out = append(out, " "+DimText.Render("↑/↓ scroll  PgUp/PgDn saltar"+scrollHint+"  Esc volver  Ctrl+Q salir"))

	return strings.Join(out, "\n")
}

func (m Model) viewTaskDetail(t task.Task) string {
	h := m.height - 2
	var lines []string

	// Breadcrumb (2 lines)
	statusLabel := t.CoordinatorStatus
	if statusLabel == "" {
		statusLabel = "—"
	}
	breadcrumb := fmt.Sprintf("← Agente / %s  [%s]", t.TaskID, statusLabel)
	lines = append(lines, " "+ViewActive.Render(breadcrumb))
	lines = append(lines, " "+SepDim.Render(strings.Repeat("─", m.width-2)))

	// Office floor
	floorStr := m.viewOfficeFloor(t)
	for _, l := range strings.Split(floorStr, "\n") {
		lines = append(lines, l)
	}
	lines = append(lines, " "+SepDim.Render(strings.Repeat("─", m.width-2)))

	// Channel tabs (1 line) + separator
	channels := listMsgChannels(t.Workspace)
	activeCh := m.activeMsgChannel
	if len(channels) > 0 {
		activeCh = activeCh % len(channels)
	} else {
		activeCh = 0
	}
	lines = append(lines, m.viewMsgChannelTabs(channels))
	lines = append(lines, " "+SepDim.Render(strings.Repeat("─", m.width-2)))

	// Chat panel fills remaining space (reserve 2 lines: input + footer)
	usedLines := len(lines)
	chatH := h - usedLines - 2
	if chatH < 3 {
		chatH = 3
	}
	var channelFile string
	if len(channels) > 0 && activeCh < len(channels) {
		channelFile = channels[activeCh]
	}
	chatStr := m.viewChatPanel(t.Workspace, channelFile, chatH)
	for _, l := range strings.Split(chatStr, "\n") {
		lines = append(lines, l)
	}

	// Chat input (1 line) + footer separator
	lines = append(lines, m.viewChatInput())
	lines = append(lines, " "+SepDim.Render(strings.Repeat("─", m.width-2)))

	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
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
			status := task.AgentStatus(t.Workspace, a.ID)
			isDirector := a.ID == "director" || a.IsSynthesizer
			var frame string
			if isDirector {
				frame = spriteDirector[m.spriteIdx]
			} else {
				frame = agentSpriteFrame(status, m.spriteIdx)
			}
			deskLines := renderDeskBox(a, status, frame, isDirector, deskW)
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
		return FooterStyle.Width(m.width).Render("f/i escribir  ←/→ canal  ↑↓ scroll  Enter agente  Esc volver  Ctrl+Q salir")
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

	if m.globalCost > 0 {
		parts = append(parts, CostStyle.Render(fmt.Sprintf(" $%.3f", m.globalCost)))
	}

	if t.ContextPct >= 0 {
		parts = append(parts, " "+m.viewCtx(t))
	}

	content := strings.Join(parts, "")
	return StatusBarStyle.Width(m.width).Render(content)
}

func (m Model) viewCtx(t *Tab) string {
	pct := t.ContextPct
	const barW = 10
	filled := barW * pct / 100
	if filled > barW {
		filled = barW
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	label := fmt.Sprintf("%d%% %s", pct, bar)

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
