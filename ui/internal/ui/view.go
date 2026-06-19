package ui

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"strings"
	"time"

	"charm.land/glamour/v2"
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

	if m.multiDetailSession != "" {
		return m.viewTmuxPeek()
	}
	if m.multiDetailFile != "" {
		return m.viewFileContent()
	}
	if m.multiDetailMode && sel >= 0 && sel < n {
		if tasks[sel].Preset == "coding" {
			return m.viewCodingDetail(tasks[sel])
		}
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


// countMsgsInFile counts occurrences of "### [" in the given file (message headers).
func countMsgsInFile(path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return strings.Count(string(raw), "### [")
}

// renderNetworkGraph renders the Command Center table view for the given task.
// Returns a slice of lines of width w.
func (m Model) renderNetworkGraph(t task.Task, w, h, cursorIdx int) []string {
	if w < 10 {
		w = 10
	}

	channels := listMsgChannels(t.Workspace)
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

	// ══ COMMAND CENTER ═...═ ◉ N AGENTE ══
	nUnits := len(agents)
	leftPart := "══ COMMAND CENTER "
	rightPart := fmt.Sprintf(" ◉ %d AGENTE ══", nUnits)
	fillW := w - len([]rune(leftPart)) - len([]rune(rightPart))
	if fillW < 2 {
		fillW = 2
	}
	lines = append(lines, ViewActive.Render(leftPart+strings.Repeat("═", fillW)+rightPart))
	lines = append(lines, "")

	// Tabla de agentes (todos al mismo nivel, incluido director)
	lines = append(lines, "  "+
		SectionHeader.Render(rpad("AGENTE", 16))+
		SectionHeader.Render(rpad("STATUS", 14))+
		SectionHeader.Render(rpad("IN", 6))+
		SectionHeader.Render("OUT"))

	for i, a := range agents {
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
		prefix := "  "
		if i == cursorIdx {
			prefix = ViewActive.Render("❯ ")
		}
		lines = append(lines, prefix+aStyle.Render(bullet)+" "+
			DimText.Render(rpad(idTrunc, 15))+
			" "+aStyle.Render(rpad(status, 13))+
			" "+DimText.Render(fmt.Sprintf("%-6d", msgsIn(a.ID)))+
			DimText.Render(fmt.Sprintf("%d", msgsOut(a.ID))))
	}

	lines = append(lines, "")

	// Archivos del workspace y de comunicación
	docs := listWorkspaceFiles(t.Workspace)
	msgs := listMsgChannels(t.Workspace)

	lines = append(lines, "  "+SectionHeader.Render("ARCHIVOS"))
	if len(docs) == 0 {
		lines = append(lines, "  "+DimText.Render("sin archivos"))
	} else {
		for i, d := range docs {
			globalIdx := len(agents) + i
			prefix := "  "
			if globalIdx == cursorIdx {
				prefix = ViewActive.Render("❯ ")
			}
			lines = append(lines, prefix+DimText.Render("· "+d))
		}
	}

	if len(msgs) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+SectionHeader.Render("MENSAJES"))
		for i, d := range msgs {
			globalIdx := len(agents) + len(docs) + i
			prefix := "  "
			if globalIdx == cursorIdx {
				prefix = ViewActive.Render("❯ ")
			}
			lines = append(lines, prefix+DimText.Render("· "+strings.TrimSuffix(d, ".md")))
		}
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

func (m Model) viewTmuxPeek() string {
	h := m.height - 2
	headerH := 2
	footerH := 1
	contentH := h - headerH - footerH
	if contentH < 1 {
		contentH = 1
	}

	label := m.multiDetailSession
	header1 := " " + ViewActive.Render("← "+label)
	header2 := " " + SepDim.Render(strings.Repeat("─", m.width-2))

	var out []string
	out = append(out, header1, header2)

	if m.multiDetailTmuxRendering {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		sp := spinner[m.tickCount%len(spinner)]
		for i := 0; i < contentH; i++ {
			if i == contentH/2 {
				out = append(out, "  "+DimText.Render(sp+" capturando sesión…"))
			} else {
				out = append(out, "")
			}
		}
	} else {
		allLines := m.multiDetailTmuxLines
		total := len(allLines)
		off := m.multiDetailScrollOffset
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
		out = append(out, visible...)
	}

	out = append(out, FooterStyle.Width(m.width).Render("↑/↓ scroll   PgUp/PgDn saltar   Esc volver   Ctrl+Q salir"))
	if len(out) > h {
		out = out[:h]
	}
	return strings.Join(out, "\n")
}

// captureSessionCmd runs tmux capture-pane in a goroutine and returns a tmuxCapturedMsg.
func captureSessionCmd(sessionID string, width int) tea.Cmd {
	return func() tea.Msg {
		raw, err := osexec.Command("tmux", "capture-pane", "-t", sessionID, "-p", "-e", "-S", "-500").Output()
		var lines []string
		if err != nil {
			lines = []string{"", "  sesión no disponible: " + sessionID}
		} else {
			stripped := ansi.Strip(string(raw))
			contentWidth := width - 2
			if contentWidth < 1 {
				contentWidth = 1
			}
			for _, l := range strings.Split(stripped, "\n") {
				wrapped := ansi.Wrap(l, contentWidth, " ")
				for _, wl := range strings.Split(wrapped, "\n") {
					lines = append(lines, " "+wl)
				}
			}
		}
		return tmuxCapturedMsg{sessionID: sessionID, lines: lines}
	}
}

func (m Model) viewFileContent() string {
	h := m.height - 2
	headerH := 2
	footerH := 1
	contentH := h - headerH - footerH
	if contentH < 1 {
		contentH = 1
	}

	filePath := m.multiDetailFile
	label := filePath
	if idx := strings.LastIndex(filePath, "/"); idx >= 0 {
		label = filePath[idx+1:]
	}
	label = strings.TrimSuffix(label, ".md")

	header1 := " " + ViewActive.Render("← "+label)
	header2 := " " + SepDim.Render(strings.Repeat("─", m.width-2))

	var out []string
	out = append(out, header1, header2)

	if m.multiDetailRendering {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		sp := spinner[m.tickCount%len(spinner)]
		for i := 0; i < contentH; i++ {
			if i == contentH/2 {
				out = append(out, "  "+DimText.Render(sp+" renderizando…"))
			} else {
				out = append(out, "")
			}
		}
	} else {
		allLines := m.multiDetailRendered
		total := len(allLines)
		off := m.multiDetailScrollOffset
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
		out = append(out, visible...)
	}

	out = append(out, FooterStyle.Width(m.width).Render("↑/↓ scroll   PgUp/PgDn saltar   Esc volver   Ctrl+Q salir"))
	if len(out) > h {
		out = out[:h]
	}
	return strings.Join(out, "\n")
}

// renderFileCmd lee y renderiza un .md con glamour en un goroutine y devuelve fileRenderedMsg.
func renderFileCmd(path string, width int, dark bool) tea.Cmd {
	return func() tea.Msg {
		raw, err := os.ReadFile(path)
		var lines []string
		if err != nil {
			lines = []string{"", "  no se pudo leer el archivo"}
			return fileRenderedMsg{path: path, lines: lines}
		}
		glamourStyle := "light"
		if dark {
			glamourStyle = "dark"
		}
		renderer, rerr := glamour.NewTermRenderer(
			glamour.WithStandardStyle(glamourStyle),
			glamour.WithWordWrap(width),
		)
		var rendered string
		if rerr == nil {
			rendered, _ = renderer.Render(string(raw))
		} else {
			rendered = string(raw)
		}
		for _, l := range strings.Split(rendered, "\n") {
			lines = append(lines, l)
		}
		return fileRenderedMsg{path: path, lines: lines}
	}
}

// viewNetworkDetail — full-width animated network graph view.
func (m Model) viewNetworkDetail(t task.Task) string {
	h := m.height - 2
	headerH := 2
	footerH := 1
	contentH := h - headerH - footerH
	if contentH < 3 {
		contentH = 3
	}

	statusLabel := t.CoordinatorStatus
	if statusLabel == "" {
		statusLabel = "—"
	}

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

	graphLines := m.renderNetworkGraph(t, m.width-2, contentH, m.multiDetailCursorIdx)

	var all []string
	all = append(all, header1, header2)
	for _, gl := range graphLines {
		all = append(all, " "+gl)
	}
	all = append(all, FooterStyle.Width(m.width).Render("Esc volver   Ctrl+Q salir"))

	if len(all) > h {
		all = all[:h]
	}
	return strings.Join(all, "\n")
}

// viewCodingDetail — vista de dos columnas para tareas con Preset=="coding".
// Columna izquierda (~60%): COMMAND CENTER animado.
// Columna derecha: panel git (log + status) y panel terminal (path + tecla t).
func (m Model) viewCodingDetail(t task.Task) string {
	h := m.height - 2
	headerH := 2
	footerH := 1
	contentH := h - headerH - footerH
	if contentH < 3 {
		contentH = 3
	}

	leftW := m.width * 6 / 10
	rightW := m.width - leftW - 1

	// Encabezado idéntico a viewNetworkDetail
	statusLabel := t.CoordinatorStatus
	if statusLabel == "" {
		statusLabel = "—"
	}
	agents := task.AllAgents(t)
	nWorking, nDone := 0, 0
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

	// Columna izquierda: renderNetworkGraph al ancho leftW-2
	leftLines := m.renderNetworkGraph(t, leftW-2, contentH, m.multiDetailCursorIdx)

	// Worktree path convencional
	worktreePath := "/tmp/greg-worktree-" + t.TaskID

	// Columna derecha
	rightLines := m.buildCodingRightPanel(worktreePath, rightW, contentH)

	// Separador y merge de columnas
	sep := SepDim.Render("│")
	var all []string
	all = append(all, header1, header2)
	for i := 0; i < contentH; i++ {
		var ll, rl string
		if i < len(leftLines) {
			ll = leftLines[i]
		}
		if i < len(rightLines) {
			rl = rightLines[i]
		}
		all = append(all, lipgloss.NewStyle().Width(leftW).Render(ll)+sep+rl)
	}
	all = append(all, FooterStyle.Width(m.width).Render("t shell  g refresh git  Esc volver  Ctrl+Q salir"))

	if len(all) > h {
		all = all[:h]
	}
	return strings.Join(all, "\n")
}

// buildCodingRightPanel construye el panel derecho de viewCodingDetail.
// Muestra git log + status (con timeout 1s, fallback a m.multiDetailGitLines)
// y la info del worktree (path + tecla t).
func (m Model) buildCodingRightPanel(worktreePath string, w, h int) []string {
	var lines []string

	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		lines = append(lines, " "+DimText.Render("worktree no disponible"))
		for len(lines) < h {
			lines = append(lines, "")
		}
		if len(lines) > h {
			lines = lines[:h]
		}
		return lines
	}

	// Obtener info git con timeout
	gitLines := m.fetchCodingGitLines(worktreePath)

	// Reservar 3 líneas para el panel terminal al fondo
	termH := 3
	gitPanelH := h - termH
	if gitPanelH < 2 {
		gitPanelH = 2
	}

	// Panel git
	lines = append(lines, " "+SectionHeader.Render("GIT"))
	maxLineW := w - 2
	if maxLineW < 1 {
		maxLineW = 1
	}
	for _, gl := range gitLines {
		if len(lines) >= gitPanelH {
			break
		}
		r := []rune(gl)
		if len(r) > maxLineW {
			gl = string(r[:maxLineW-1]) + "…"
		}
		lines = append(lines, " "+DimText.Render(gl))
	}

	// Relleno hasta el límite del panel git
	for len(lines) < gitPanelH {
		lines = append(lines, "")
	}

	// Panel terminal
	lines = append(lines, " "+SepDim.Render(strings.Repeat("─", w-2)))
	pathStr := worktreePath
	r := []rune(pathStr)
	if len(r) > maxLineW {
		pathStr = "…" + string(r[len(r)-maxLineW+1:])
	}
	lines = append(lines, " "+DimText.Render(pathStr))
	lines = append(lines, " "+ViewActive.Render("t")+" "+DimText.Render("→ abrir shell"))

	if len(lines) > h {
		lines = lines[:h]
	}
	return lines
}

// fetchCodingGitLines ejecuta git log y git status síncronos con timeout 1s cada uno.
// Si ambos fallan, retorna m.multiDetailGitLines como fallback.
func (m Model) fetchCodingGitLines(worktreePath string) []string {
	runGit := func(args ...string) []byte {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		out, _ := osexec.CommandContext(ctx, "git", append([]string{"-C", worktreePath}, args...)...).Output()
		return out
	}

	logOut := runGit("log", "--oneline", "-10")
	statusOut := runGit("status", "--short")

	if len(logOut) == 0 && len(statusOut) == 0 {
		return m.multiDetailGitLines
	}

	var result []string
	for _, l := range strings.Split(strings.TrimRight(string(logOut), "\n"), "\n") {
		if l != "" {
			result = append(result, l)
		}
	}
	if trimmed := strings.TrimSpace(string(statusOut)); trimmed != "" {
		result = append(result, "──")
		for _, l := range strings.Split(strings.TrimRight(string(statusOut), "\n"), "\n") {
			if l != "" {
				result = append(result, l)
			}
		}
	}
	return result
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
	lines = append(lines, "")

	// 6: Repos de código
	if m.configCursorIdx == 6 {
		lines = append(lines, "  "+ViewActive.Render("▶")+" "+ModelStyle.Render(fmt.Sprintf("%-22s", "Repos de código")))
		for i, repo := range m.cfg.CodingRepos {
			if m.configRepoCursorIdx == i {
				lines = append(lines, "      "+ViewActive.Render("▸")+" "+QuestionLabel.Render(repo))
			} else {
				lines = append(lines, "        "+QuestionLabelDim.Render(repo))
			}
		}
		addSlot := len(m.cfg.CodingRepos)
		if m.configRepoInputMode {
			lines = append(lines, "      "+ViewActive.Render("▸")+" "+QuestionLabel.Render(m.configRepoInputBuf+"_"))
		} else if m.configRepoCursorIdx == addSlot {
			lines = append(lines, "      "+ViewActive.Render("▸")+" "+DimText.Render("+ agregar ruta..."))
		} else {
			lines = append(lines, "        "+DimText.Render("+ agregar ruta..."))
		}
	} else {
		repoCount := len(m.cfg.CodingRepos)
		repoVal := "ninguno"
		if repoCount > 0 {
			repoVal = fmt.Sprintf("%d repo(s)", repoCount)
		}
		lines = append(lines, renderItem(6, "Repos de código", repoVal, ""))
	}

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
