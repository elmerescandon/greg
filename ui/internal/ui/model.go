package ui

import (
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/elmerescandon/greg-ui/internal/claude"
	"github.com/elmerescandon/greg-ui/internal/session"
	"github.com/elmerescandon/greg-ui/internal/task"

	tea "charm.land/bubbletea/v2"
)

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type ViewMode int

const (
	ViewMetricas ViewMode = iota
	ViewMultiple
	ViewGraficas
	ViewConfig
)

// gregConfig holds persistent user preferences.
type gregConfig struct {
	DarkMode         bool   `json:"dark_mode"`
	DefaultModel     string `json:"default_model"`
	DefaultEffort    string `json:"default_effort"`
	CompactWarnPct   int    `json:"compact_warn_pct"`
	AutoCompact      bool   `json:"auto_compact"`
	IdleTimeoutHours int    `json:"idle_timeout_hours"`
}

func defaultGregConfig() gregConfig {
	return gregConfig{
		DarkMode:         true,
		DefaultModel:     "claude-sonnet-4-6",
		DefaultEffort:    "high",
		CompactWarnPct:   90,
		AutoCompact:      false,
		IdleTimeoutHours: 0,
	}
}

func applyGregConfigDefaults(c *gregConfig) {
	if c.DefaultModel == "" {
		c.DefaultModel = "claude-sonnet-4-6"
	}
	if c.DefaultEffort == "" {
		c.DefaultEffort = "high"
	}
	if c.CompactWarnPct == 0 {
		c.CompactWarnPct = 90
	}
}

func gregConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "greg", "config.json")
}

func loadGregConfig() gregConfig {
	data, err := os.ReadFile(gregConfigPath())
	if err != nil {
		return defaultGregConfig()
	}
	var c gregConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return defaultGregConfig()
	}
	applyGregConfigDefaults(&c)
	return c
}

func saveGregConfig(c gregConfig) {
	p := gregConfigPath()
	os.MkdirAll(filepath.Dir(p), 0755)
	data, _ := json.Marshal(c)
	os.WriteFile(p, data, 0644)
}

type ModelOption struct {
	ID            string
	Label         string
	Desc          string
	ContextWindow int
}

type EffortOption struct {
	ID    string
	Label string
	Desc  string
}

var ModelOptions = []ModelOption{
	{"claude-opus-4-6", "Opus 4.6", "1M ctx · máxima calidad", 1_000_000},
	{"claude-opus-4-7", "Opus 4.7", "200k ctx · más reciente", 200_000},
	{"claude-sonnet-4-6", "Sonnet 4.6", "balanceado · rápido", 200_000},
	{"claude-haiku-4-5-20251001", "Haiku 4.5", "económico · muy rápido", 200_000},
}

var EffortOptions = []EffortOption{
	{"low", "low", "mínimo razonamiento · más rápido"},
	{"medium", "medium", "balanceado"},
	{"high", "high", "mayor razonamiento · recomendado"},
	{"xhigh", "xhigh", "razonamiento extendido"},
	{"max", "max", "máximo esfuerzo · más lento"},
}

type Model struct {
	tabs           []*Tab
	tabIdx         int
	width          int
	height         int
	globalCost     float64
	spinIdx        int
	spriteTick     int
	spriteIdx      int
	inputHistory   []string
	historyIdx     int
	savedInput     string
	vault          string
	viewMode              ViewMode
	metricsShowCost       bool
	sidebarFocused        bool
	sidebarIdx            int
	multiSelectedIdx        int
	multiDetailMode         bool
	multiDetailCursorIdx    int
	multiDetailFile         string
	multiDetailSession      string
	multiDetailScrollOffset int
	multiDetailRendered     []string
	multiDetailRendering    bool
	multiDetailTmuxLines    []string
	multiDetailTmuxRendering bool
	cfg                   gregConfig
	configCursorIdx       int
	tickCount             int
}

// Messages
type tickMsg struct{}
type claudeEventMsg struct {
	tabID  string
	event  claude.Event
	events <-chan claude.Event
	errors <-chan string
}
type claudeErrorMsg struct {
	tabID  string
	text   string
	events <-chan claude.Event
	errors <-chan string
}
type claudeDoneMsg struct{ tabID string }
type clipboardPasteMsg string
type fileRenderedMsg struct {
	path  string
	lines []string
}
type tmuxCapturedMsg struct {
	sessionID string
	lines     []string
}

func readClipboardCmd() tea.Cmd {
	return func() tea.Msg {
		out, err := osexec.Command("pbpaste").Output()
		if err != nil {
			return nil
		}
		return clipboardPasteMsg(out)
	}
}

func NewModel(vault string) Model {
	cfg := loadGregConfig()
	InitStyles(cfg.DarkMode)

	var tab *Tab

	if active := session.FindActiveSession(); active != nil {
		name := strings.TrimPrefix(active.ID, "greg-")
		tab = NewTab(name, active.ClaudeSession, active.ID)
		if active.Model != "" {
			tab.Model = active.Model
		} else {
			tab.Model = cfg.DefaultModel
		}
		tab.Effort = cfg.DefaultEffort
		if active.ClaudeSession != "" {
			entries := claude.LoadHistory(vault, active.ClaudeSession, 10)
			tab.Lines = renderHistory(entries)
		}
	} else {
		s, _ := session.Spawn(vault)
		name := strings.TrimPrefix(s.ID, "greg-")
		tab = NewTab(name, "", s.ID)
		tab.Model = cfg.DefaultModel
		tab.Effort = cfg.DefaultEffort
	}

	m := Model{
		tabs:       []*Tab{tab},
		tabIdx:     0,
		vault:      vault,
		historyIdx: -1,
		cfg:        cfg,
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) tab() *Tab { return m.tabs[m.tabIdx] }

func (m *Model) tabByID(id string) (int, *Tab) {
	for i, t := range m.tabs {
		if t.GregSessionID == id {
			return i, t
		}
	}
	return -1, nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.spinIdx = (m.spinIdx + 1) % len(spinFrames)
		m.spriteTick = (m.spriteTick + 1) % 4
		if m.spriteTick == 0 {
			m.spriteIdx = (m.spriteIdx + 1) % 4
		}
		m.tickCount++
		if m.tickCount%62 == 0 {
			m.checkIdleTimeout()
		}
		return m, tickCmd()

	case claudeEventMsg:
		if idx, _ := m.tabByID(msg.tabID); idx >= 0 {
			m.handleEvent(idx, msg.event)
			return m, waitForEvent(msg.tabID, msg.events, msg.errors)
		}
		return m, nil

	case claudeErrorMsg:
		if idx, t := m.tabByID(msg.tabID); idx >= 0 {
			t.Lines = append(t.Lines, ErrorText.Render(msg.text))
			if idx != m.tabIdx {
				t.HasNew = true
			}
			return m, waitForEvent(msg.tabID, msg.events, msg.errors)
		}
		return m, nil

	case claudeDoneMsg:
		if idx, t := m.tabByID(msg.tabID); idx >= 0 {
			t.Running = false
			t.Proc = nil
			t.CurrentAction = ""
			t.Lines = append(t.Lines, "")

			if t.CompactPending && idx == m.tabIdx {
				t.CompactPending = false
				t.CompactWarned = false
				if m.cfg.AutoCompact {
					t.Lines = append(t.Lines, DimText.Render("⚡ compactando contexto automáticamente…"))
					return m, m.send("/compact")
				}
				t.Lines = append(t.Lines, ErrorText.Render("⚡ contexto al límite — escribe qué quieres preservar y presiona Enter"))
				t.InputBuf = "/compact "
				t.CursorPos = len(t.InputBuf)
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.MouseWheelMsg:
		t := m.tab()
		if msg.Button == tea.MouseWheelUp {
			t.ScrollLock = true
			t.ScrollOffset += 3
			maxScroll := len(t.Lines) - m.outputHeight()
			if t.ScrollOffset > maxScroll {
				t.ScrollOffset = maxScroll
			}
			if t.ScrollOffset < 0 {
				t.ScrollOffset = 0
			}
		} else if msg.Button == tea.MouseWheelDown {
			t.ScrollOffset -= 3
			if t.ScrollOffset <= 0 {
				t.ScrollOffset = 0
				t.ScrollLock = false
			}
		}
		return m, nil

	case tea.MouseClickMsg:
		if msg.Y <= 1 {
			ms, me, us, ue, gs, ge, cs, ce := m.viewBarButtonPositions()
			if msg.X >= ms && msg.X < me {
				m.viewMode = ViewMetricas
			} else if msg.X >= us && msg.X < ue {
				m.viewMode = ViewMultiple
			} else if msg.X >= gs && msg.X < ge {
				m.viewMode = ViewGraficas
			} else if msg.X >= cs && msg.X < ce {
				m.viewMode = ViewConfig
			}
		} else if msg.Y == 2 {
			if i := m.tabAtX(msg.X); i >= 0 {
				m.tabIdx = i
				m.tab().HasNew = false
			}
		}
		return m, nil

	case tea.PasteMsg:
		m.insertText(msg.Content)
		return m, nil

	case clipboardPasteMsg:
		m.insertText(string(msg))
		return m, nil

	case fileRenderedMsg:
		if msg.path == m.multiDetailFile {
			m.multiDetailRendered = msg.lines
			m.multiDetailRendering = false
		}
		return m, nil

	case tmuxCapturedMsg:
		if msg.sessionID == m.multiDetailSession {
			m.multiDetailTmuxLines = msg.lines
			m.multiDetailTmuxRendering = false
		}
		return m, nil
	}

	return m, nil
}

func waitForEvent(tabID string, events <-chan claude.Event, errors <-chan string) tea.Cmd {
	return func() tea.Msg {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			return claudeEventMsg{tabID: tabID, event: ev, events: events, errors: errors}
		case err, ok := <-errors:
			if !ok {
				return nil
			}
			return claudeErrorMsg{tabID: tabID, text: err, events: events, errors: errors}
		}
	}
}

func (m *Model) handleEvent(tabIdx int, ev claude.Event) {
	t := m.tabs[tabIdx]

	if ev.SessionID != "" && ev.SessionID != t.ClaudeSession {
		t.ClaudeSession = ev.SessionID
		if t.GregSessionID != "" {
			session.UpdateClaudeSession(t.GregSessionID, ev.SessionID)
		}
	}

	if tabIdx != m.tabIdx {
		t.HasNew = true
	}

	switch ev.Type {
	case "assistant":
		if ev.Message == nil {
			return
		}
		for _, b := range ev.Message.Content {
			if b.Type == "text" && b.Text != "" {
				t.CurrentAction = ""
				for _, line := range strings.Split(b.Text, "\n") {
					t.Lines = append(t.Lines, line)
				}
			}
			if b.Type == "tool_use" {
				if b.Name == "AskUserQuestion" {
					var input claude.AskUserInput
					if err := json.Unmarshal(b.Input, &input); err == nil && len(input.Questions) > 0 {
						m.showQuestion(b.ID, input.Questions, t)
					}
				} else {
					label := claude.FormatToolLabel(b.Name, b.Input)
					t.CurrentAction = b.Name + "…"
					t.Lines = append(t.Lines,
						ToolCall.Render("⚙ "+b.Name)+" "+DimText.Render(label))
				}
			}
		}

	case "user":
		if ev.Message == nil {
			return
		}
		t.CurrentAction = "pensando…"
		for _, b := range ev.Message.Content {
			if b.Type != "tool_result" {
				continue
			}
			// Content is a JSON array of blocks or a plain string
			text := extractToolResultText(b.Content)
			if text == "" {
				text = b.Text
			}
			if text == "" {
				continue
			}
			const maxLines = 20
			lines := strings.Split(text, "\n")
			shown := len(lines)
			if shown > maxLines {
				shown = maxLines
			}
			for _, l := range lines[:shown] {
				t.Lines = append(t.Lines, ToolResult.Render("  "+l))
			}
			if len(lines) > maxLines {
				t.Lines = append(t.Lines,
					ToolResult.Render(fmt.Sprintf("  …(%d líneas más)", len(lines)-maxLines)))
			}
		}

	case "result":
		if ev.TotalCostUSD > 0 {
			m.globalCost += ev.TotalCostUSD
		}
		if ev.ModelUsage != nil {
			for _, mu := range ev.ModelUsage {
				if mu.ContextWindow > 0 {
					used := mu.InputTokens + mu.CacheReadInputTokens + mu.CacheCreationInputTokens
					if ev.Usage != nil && len(ev.Usage.Iterations) > 0 {
						last := ev.Usage.Iterations[len(ev.Usage.Iterations)-1]
						used = last.InputTokens + last.CacheReadInputTokens + last.CacheCreationInputTokens
					}
					contextWindow := mu.ContextWindow
					for _, mo := range ModelOptions {
						if mo.ID == t.Model && mo.ContextWindow > contextWindow {
							contextWindow = mo.ContextWindow
							break
						}
					}
					t.ContextPct = int(float64(used) / float64(contextWindow) * 100)
					t.ContextTokens = used
					t.ContextWindow = contextWindow

					resetPct := m.cfg.CompactWarnPct - 15
					if resetPct < 60 {
						resetPct = 60
					}
					pendingPct := m.cfg.CompactWarnPct + 5
					if pendingPct > 99 {
						pendingPct = 99
					}
					if t.ContextPct < resetPct {
						t.CompactWarned = false
						t.CompactPending = false
					} else if t.ContextPct >= pendingPct {
						t.CompactPending = true
					} else if t.ContextPct >= m.cfg.CompactWarnPct && !t.CompactWarned {
						t.CompactWarned = true
						t.Lines = append(t.Lines,
							CtxMed.Render(fmt.Sprintf("⚠ contexto al %d%% — Ctrl+K para compactar", t.ContextPct)))
					}

					totalInput := 0
					totalOutput := 0
					for _, m := range ev.ModelUsage {
						totalInput += m.InputTokens
						totalOutput += m.OutputTokens
					}
					if t.GregSessionID != "" {
						session.AccumulateUsage(t.GregSessionID, totalInput, totalOutput, ev.TotalCostUSD)
					}
					break
				}
			}
		}
		if ev.Subtype == "error" {
			t.Lines = append(t.Lines, ErrorText.Render("Error: "+ev.Error))
		}
	}
}

func (m *Model) showQuestion(id string, questions []claude.Question, t *Tab) {
	t.PendingQuestion = &PendingQuestion{
		ID:        id,
		Questions: questions,
		CurrentQ:  0,
		SelectedIdx: 0,
		Answers:   make(map[string]string),
	}
	t.Lines = append(t.Lines, "")
	t.Lines = append(t.Lines, UserMessage.Render("? "+questions[0].Question))
	t.Lines = append(t.Lines, "")
}

func (m *Model) showConfigSelection(t *Tab) {
	modelQs := make([]claude.Option, len(ModelOptions))
	for i, mo := range ModelOptions {
		modelQs[i] = claude.Option{Label: mo.Label, Description: mo.Desc}
	}
	effortQs := make([]claude.Option, len(EffortOptions))
	for i, eo := range EffortOptions {
		effortQs[i] = claude.Option{Label: eo.Label, Description: eo.Desc}
	}

	selectedModelIdx := 0
	for i, mo := range ModelOptions {
		if mo.ID == t.Model {
			selectedModelIdx = i
			break
		}
	}

	t.PendingQuestion = &PendingQuestion{
		ID:         "__config__",
		ConfigMode: true,
		Questions: []claude.Question{
			{Question: "¿Qué modelo quieres usar?", Header: "Modelo", Options: modelQs},
			{Question: "¿Qué nivel de esfuerzo?", Header: "Esfuerzo", Options: effortQs},
		},
		CurrentQ:    0,
		SelectedIdx: selectedModelIdx,
		Answers:     make(map[string]string),
	}
	t.Lines = append(t.Lines, UserMessage.Render("? ¿Qué modelo quieres usar?"))
	t.Lines = append(t.Lines, "")
}

func (m *Model) send(text string) tea.Cmd {
	t := m.tab()
	text = strings.TrimSpace(text)
	if text == "" || t.Running {
		return nil
	}

	m.inputHistory = append(m.inputHistory, text)
	m.historyIdx = -1
	t.Running = true
	t.CurrentAction = "pensando…"
	t.InputBuf = ""
	t.CursorPos = 0

	t.Lines = append(t.Lines, "")
	t.Lines = append(t.Lines, UserMessage.Render("▶ "+text))
	t.Lines = append(t.Lines, "")

	vault := m.vault
	model := t.Model
	effort := t.Effort
	claudeSess := t.ClaudeSession
	tabID := t.GregSessionID

	proc, events, errors := claude.StartClaude(vault, model, effort, text, claudeSess)
	t.Proc = proc
	t.Events = events
	t.Errors = errors

	return tea.Batch(
		waitForEvent(tabID, events, errors),
		func() tea.Msg {
			<-proc.Done
			return claudeDoneMsg{tabID: tabID}
		},
	)
}

func (m *Model) submitAnswer() {
	t := m.tab()
	q := t.PendingQuestion
	if q == nil {
		return
	}

	qData := q.Questions[q.CurrentQ]
	selected := qData.Options[q.SelectedIdx]
	q.Answers[qData.Question] = selected.Label
	t.Lines = append(t.Lines, QuestionSelected.Render("  ✓ "+selected.Label))

	if q.ConfigMode {
		if q.CurrentQ == 0 {
			for _, mo := range ModelOptions {
				if mo.Label == selected.Label {
					t.Model = mo.ID
					session.UpdateModel(t.GregSessionID, t.Model)
					break
				}
			}
		} else if q.CurrentQ == 1 {
			for _, eo := range EffortOptions {
				if eo.Label == selected.Label {
					t.Effort = eo.ID
					break
				}
			}
		}

		q.CurrentQ++
		if q.CurrentQ < len(q.Questions) {
			q.SelectedIdx = 0
			if q.CurrentQ == 1 {
				q.SelectedIdx = 2
			}
			next := q.Questions[q.CurrentQ]
			t.Lines = append(t.Lines, "")
			t.Lines = append(t.Lines, UserMessage.Render("? "+next.Question))
			t.Lines = append(t.Lines, "")
		} else {
			t.PendingQuestion = nil
			t.Lines = append(t.Lines, "")
			t.Lines = append(t.Lines, DimText.Render(fmt.Sprintf("listo: %s · esfuerzo %s", t.Model, t.Effort)))
			t.Lines = append(t.Lines, "")
		}
		return
	}

	q.CurrentQ++
	if q.CurrentQ < len(q.Questions) {
		q.SelectedIdx = 0
		next := q.Questions[q.CurrentQ]
		t.Lines = append(t.Lines, "")
		t.Lines = append(t.Lines, UserMessage.Render("? "+next.Question))
		t.Lines = append(t.Lines, "")
	} else {
		t.PendingQuestion = nil
		if t.Proc != nil && t.Proc.Stdin != nil {
			data, _ := json.Marshal(map[string]interface{}{"answers": q.Answers})
			t.Proc.Stdin.Write(append(data, '\n'))
		}
		t.Lines = append(t.Lines, "")
	}
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t := m.tab()
	k := msg.String()

	switch k {
	case "ctrl+q":
		return m, tea.Quit

	case "ctrl+1":
		m.viewMode = ViewMetricas
		return m, nil

	case "ctrl+2", "ctrl+space":
		m.viewMode = ViewMultiple
		return m, nil

	case "ctrl+3":
		m.viewMode = ViewGraficas
		return m, nil

	case "ctrl+4":
		m.viewMode = ViewConfig
		return m, nil

	case "ctrl+c":
		if t.Proc != nil && t.Proc.Cmd != nil && t.Proc.Cmd.Process != nil {
			t.Proc.Cmd.Process.Signal(os.Interrupt)
			t.Lines = append(t.Lines, DimText.Render("cancelado"))
		}
		return m, nil

	case "ctrl+k":
		if !t.Running {
			t.CompactWarned = false
			t.CompactPending = false
			t.InputBuf = "/compact "
			t.CursorPos = len(t.InputBuf)
		}
		return m, nil

	case "ctrl+t":
		s, _ := session.Spawn(m.vault)
		name := strings.TrimPrefix(s.ID, "greg-")
		newT := NewTab(name, "", s.ID)
		newT.Model = m.cfg.DefaultModel
		newT.Effort = m.cfg.DefaultEffort
		m.tabs = append(m.tabs, newT)
		m.tabIdx = len(m.tabs) - 1
		return m, nil

	case "ctrl+w":
		if len(m.tabs) <= 1 {
			return m, nil
		}
		if t.GregSessionID != "" {
			go func(id string) {
				c := osexec.Command("greg", "kill", id)
				c.Run()
			}(t.GregSessionID)
		}
		newIdx := m.tabIdx
		if newIdx > 0 {
			newIdx--
		}
		m.tabs = append(m.tabs[:m.tabIdx], m.tabs[m.tabIdx+1:]...)
		m.tabIdx = newIdx
		return m, nil

	case "ctrl+v":
		return m, readClipboardCmd()

	}

	if t.PendingQuestion != nil {
		return m.handleQuestionKey(msg)
	}

	if m.viewMode == ViewConfig {
		switch k {
		case "up":
			if m.configCursorIdx > 0 {
				m.configCursorIdx--
			}
		case "down":
			if m.configCursorIdx < numConfigItems-1 {
				m.configCursorIdx++
			}
		case "left":
			m.cycleConfigItem(false)
		case "right", " ", "enter":
			m.cycleConfigItem(true)
		case "escape":
			m.viewMode = ViewMetricas
		}
		return m, nil
	}

	if m.viewMode == ViewGraficas {
		switch k {
		case "s":
			m.metricsShowCost = false
			return m, nil
		case "c":
			m.metricsShowCost = true
			return m, nil
		case "1":
			m.viewMode = ViewMetricas
			return m, nil
		case "2":
			m.viewMode = ViewMultiple
			return m, nil
		}
		return m, nil
	}

	if m.viewMode == ViewMetricas {
		switch k {
		case "ctrl+shift+up":
			m.sidebarFocused = true
			if m.sidebarIdx > 0 {
				m.sidebarIdx--
			}
			return m, nil
		case "ctrl+shift+down":
			m.sidebarFocused = true
			sessions := computeStandaloneSessions()
			if m.sidebarIdx < len(sessions)-1 {
				m.sidebarIdx++
			}
			return m, nil
		case "ctrl+shift+enter":
			return m.openSidebarSession()
		case "enter":
			if m.sidebarFocused {
				return m.openSidebarSession()
			}
		case "escape":
			if m.sidebarFocused {
				m.sidebarFocused = false
				return m, nil
			}
		case "up":
			m.sidebarFocused = false
			t.ScrollLock = true
			t.ScrollOffset++
			maxScroll := len(t.Lines) - m.outputHeight()
			if t.ScrollOffset > maxScroll {
				t.ScrollOffset = maxScroll
			}
			if t.ScrollOffset < 0 {
				t.ScrollOffset = 0
			}
			return m, nil
		case "down":
			m.sidebarFocused = false
			if t.ScrollOffset > 0 {
				t.ScrollOffset--
				if t.ScrollOffset == 0 {
					t.ScrollLock = false
				}
			}
			return m, nil
		}
	}

	if m.viewMode == ViewMultiple {
		listH := m.height - 6 // topbar(2) + header(2) + footer(2)

		// Level 3: tmux peek o file viewer
		if m.multiDetailSession != "" || m.multiDetailFile != "" {
			switch k {
			case "escape", "backspace":
				m.multiDetailSession = ""
				m.multiDetailFile = ""
				m.multiDetailScrollOffset = 0
				m.multiDetailRendered = nil
				m.multiDetailRendering = false
				m.multiDetailTmuxLines = nil
				m.multiDetailTmuxRendering = false
			case "up":
				m.multiDetailScrollOffset++
			case "down":
				if m.multiDetailScrollOffset > 0 {
					m.multiDetailScrollOffset--
				}
			case "pgup":
				m.multiDetailScrollOffset += m.height / 2
			case "pgdown":
				m.multiDetailScrollOffset -= m.height / 2
				if m.multiDetailScrollOffset < 0 {
					m.multiDetailScrollOffset = 0
				}
			}
			return m, nil
		}

		// Level 2: command center con navegación
		if m.multiDetailMode {
			tasks, _ := task.LoadTasks()
			var curTask *task.Task
			if m.multiSelectedIdx < len(tasks) {
				t := tasks[m.multiSelectedIdx]
				curTask = &t
			}
			switch k {
			case "escape", "backspace":
				m.multiDetailMode = false
				m.multiDetailCursorIdx = 0
			case "up":
				if m.multiDetailCursorIdx > 0 {
					m.multiDetailCursorIdx--
				}
			case "down":
				if curTask != nil {
					agents := task.AllAgents(*curTask)
					docs := listWorkspaceFiles(curTask.Workspace)
					msgs := listMsgChannels(curTask.Workspace)
					total := len(agents) + len(docs) + len(msgs)
					if m.multiDetailCursorIdx < total-1 {
						m.multiDetailCursorIdx++
					}
				}
			case "enter":
				if curTask != nil {
					agents := task.AllAgents(*curTask)
					idx := m.multiDetailCursorIdx
					if idx < len(agents) {
						sid := agents[idx].SessionID
						if sid != "" {
							m.multiDetailSession = sid
							m.multiDetailScrollOffset = 0
							m.multiDetailTmuxLines = nil
							m.multiDetailTmuxRendering = true
							return m, captureSessionCmd(sid, m.width)
						}
					} else if fp := m.resolveDetailFile(curTask); fp != "" {
						m.multiDetailFile = fp
						m.multiDetailScrollOffset = 0
						m.multiDetailRendered = nil
						m.multiDetailRendering = true
						return m, renderFileCmd(fp, m.width-4, m.cfg.DarkMode)
					}
				}
			}
			return m, nil
		}

		// Level 1: task list
		switch k {
		case "enter":
			m.multiDetailMode = true
			m.multiDetailCursorIdx = 0
		case "up":
			if m.multiSelectedIdx > 0 {
				m.multiSelectedIdx--
			}
		case "down":
			m.multiSelectedIdx++
		case "pgup":
			m.multiSelectedIdx -= listH
			if m.multiSelectedIdx < 0 {
				m.multiSelectedIdx = 0
			}
		case "pgdown":
			m.multiSelectedIdx += listH
		case "x":
			tasks, _ := task.LoadTasks()
			n := len(tasks)
			sel := m.multiSelectedIdx
			if sel >= 0 && sel < n {
				tid := tasks[sel].TaskID
				go func(taskID string) {
					c := osexec.Command("greg", "task", "close", taskID)
					c.Run()
				}(tid)
			}
		case "1":
			m.viewMode = ViewMetricas
			return m, nil
		case "3":
			m.viewMode = ViewGraficas
			return m, nil
		}
		return m, nil
	}

	switch k {
	case "pgup":
		t.ScrollLock = true
		t.ScrollOffset += m.height / 2
		maxScroll := len(t.Lines) - m.outputHeight()
		if t.ScrollOffset > maxScroll {
			t.ScrollOffset = maxScroll
		}
		if t.ScrollOffset < 0 {
			t.ScrollOffset = 0
		}
		return m, nil

	case "pgdown":
		t.ScrollOffset -= m.height / 2
		if t.ScrollOffset <= 0 {
			t.ScrollOffset = 0
			t.ScrollLock = false
		}
		return m, nil

	case "ctrl+up":
		t.ScrollLock = true
		t.ScrollOffset += 3
		maxScroll := len(t.Lines) - m.outputHeight()
		if t.ScrollOffset > maxScroll {
			t.ScrollOffset = maxScroll
		}
		return m, nil

	case "ctrl+down":
		t.ScrollOffset -= 3
		if t.ScrollOffset <= 0 {
			t.ScrollOffset = 0
			t.ScrollLock = false
		}
		return m, nil

	case "ctrl+shift+right":
		if m.tabIdx < len(m.tabs)-1 {
			m.tabIdx++
			m.tab().HasNew = false
		}
		return m, nil

	case "ctrl+shift+left":
		if m.tabIdx > 0 {
			m.tabIdx--
			m.tab().HasNew = false
		}
		return m, nil
	}

	if t.Running {
		return m, nil
	}

	switch k {
	case "alt+up":
		if len(m.inputHistory) == 0 {
			return m, nil
		}
		if m.historyIdx == -1 {
			m.savedInput = t.InputBuf
		}
		m.historyIdx++
		if m.historyIdx >= len(m.inputHistory) {
			m.historyIdx = len(m.inputHistory) - 1
		}
		t.InputBuf = m.inputHistory[len(m.inputHistory)-1-m.historyIdx]
		t.CursorPos = len(t.InputBuf)
		return m, nil

	case "alt+down":
		if m.historyIdx == -1 {
			return m, nil
		}
		m.historyIdx--
		if m.historyIdx < 0 {
			m.historyIdx = -1
			t.InputBuf = m.savedInput
		} else {
			t.InputBuf = m.inputHistory[len(m.inputHistory)-1-m.historyIdx]
		}
		t.CursorPos = len(t.InputBuf)
		return m, nil

	case "left":
		if t.CursorPos > 0 {
			t.CursorPos--
		}
		return m, nil

	case "right":
		if t.CursorPos < len(t.InputBuf) {
			t.CursorPos++
		}
		return m, nil

	case "home":
		t.CursorPos = 0
		return m, nil

	case "end":
		t.CursorPos = len(t.InputBuf)
		return m, nil

	case "alt+enter":
		t.InputBuf = t.InputBuf[:t.CursorPos] + "\n" + t.InputBuf[t.CursorPos:]
		t.CursorPos++
		return m, nil

	case "enter":
		cmd := m.send(t.InputBuf)
		return m, cmd

	case "backspace":
		if t.CursorPos > 0 {
			t.InputBuf = t.InputBuf[:t.CursorPos-1] + t.InputBuf[t.CursorPos:]
			t.CursorPos--
		}
		return m, nil

	default:
		if msg.Text != "" {
			t.InputBuf = t.InputBuf[:t.CursorPos] + msg.Text + t.InputBuf[t.CursorPos:]
			t.CursorPos += len(msg.Text)
		}
		return m, nil
	}
}

func (m *Model) insertText(text string) {
	t := m.tab()
	if !t.Running {
		t.InputBuf = t.InputBuf[:t.CursorPos] + text + t.InputBuf[t.CursorPos:]
		t.CursorPos += len(text)
	}
}

// listWorkspaceFiles returns sorted .md filenames inside workspace/workspace/.
func listWorkspaceFiles(workspace string) []string {
	entries, err := os.ReadDir(workspace + "/workspace")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// listMsgChannels returns sorted basenames of *.md files in workspace/messages/.
func listMsgChannels(workspace string) []string {
	dir := workspace + "/messages"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// computeStandaloneSessions returns the filtered, sorted list of non-task sessions.
// Pure function — no file mutations. Use this everywhere the list is needed.
func computeStandaloneSessions() []session.Session {
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
	active, _ := session.LoadSessions()
	finished, _ := session.LoadFinishedSessions()
	seen := map[string]bool{}
	var all []session.Session
	for _, s := range active {
		if !taskSessions[s.ID] && !seen[s.ID] {
			seen[s.ID] = true
			all = append(all, s)
		}
	}
	for _, s := range finished {
		if !taskSessions[s.ID] && !seen[s.ID] {
			seen[s.ID] = true
			all = append(all, s)
		}
	}
	tsKey := func(s session.Session) string {
		if s.Ended != "" {
			return s.Ended
		}
		return s.Started
	}
	// SliceStable garantiza orden determinista cuando timestamps son iguales.
	sort.SliceStable(all, func(i, j int) bool {
		ai := all[i].Status == "active"
		aj := all[j].Status == "active"
		if ai != aj {
			return ai
		}
		return tsKey(all[i]) > tsKey(all[j])
	})
	return all
}

// loadStandaloneSessions archives completed task sessions and returns the standalone list.
// Úsalo solo en contextos de inicialización o ticks de mantenimiento, no en cada render.
func loadStandaloneSessions() []session.Session {
	var completedTaskSessionIDs []string
	if tasks, err := task.LoadTasks(); err == nil {
		for _, t := range tasks {
			if t.CoordinatorStatus != "completed" {
				continue
			}
			if t.SynthesizerID != "" {
				completedTaskSessionIDs = append(completedTaskSessionIDs, t.SynthesizerID)
			}
			for _, a := range t.Agents {
				if a.SessionID != "" {
					completedTaskSessionIDs = append(completedTaskSessionIDs, a.SessionID)
				}
			}
		}
	}
	session.ArchiveTaskSessions(completedTaskSessionIDs)
	return computeStandaloneSessions()
}

func (m Model) openSidebarSession() (tea.Model, tea.Cmd) {
	sessions := computeStandaloneSessions()
	if m.sidebarIdx >= len(sessions) {
		return m, nil
	}
	s := sessions[m.sidebarIdx]
	m.sidebarFocused = false
	// Switch to existing tab if already open
	for i, t := range m.tabs {
		if t.GregSessionID == s.ID {
			m.tabIdx = i
			return m, nil
		}
	}
	// Revive finished sessions so they appear in sessions.json and can track usage
	if s.Status == "finished" {
		if revived, err := session.ReviveSession(s.ID); err == nil && revived != nil {
			s = *revived
		}
	}
	name := strings.TrimPrefix(s.ID, "greg-")
	newT := NewTab(name, s.ClaudeSession, s.ID)
	if s.Model != "" {
		newT.Model = s.Model
	} else {
		newT.Model = m.cfg.DefaultModel
	}
	newT.Effort = m.cfg.DefaultEffort
	if s.ClaudeSession != "" {
		entries := claude.LoadHistory(m.vault, s.ClaudeSession, 10)
		newT.Lines = renderHistory(entries)
	}
	m.tabs = append(m.tabs, newT)
	m.tabIdx = len(m.tabs) - 1
	return m, nil
}

func (m Model) handleQuestionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t := m.tab()
	q := t.PendingQuestion

	switch msg.String() {
	case "up":
		if q.SelectedIdx > 0 {
			q.SelectedIdx--
		}
	case "down":
		maxIdx := len(q.Questions[q.CurrentQ].Options) - 1
		if q.SelectedIdx < maxIdx {
			q.SelectedIdx++
		}
	case "enter":
		m.submitAnswer()
	case "escape":
		t.PendingQuestion = nil
	}
	return m, nil
}

func (m Model) outputHeight() int {
	// topBar=2  tabBar=1  statusBar=1  input=1  footer=1 → 6 rows overhead
	return m.height - 6
}

const numConfigItems = 6

func modelIdx(id string) int {
	for i, mo := range ModelOptions {
		if mo.ID == id {
			return i
		}
	}
	return 0
}

func effortIdx(id string) int {
	for i, eo := range EffortOptions {
		if eo.ID == id {
			return i
		}
	}
	return 0
}

func warnPctIdx(pct int, opts []int) int {
	for i, o := range opts {
		if o == pct {
			return i
		}
	}
	return len(opts) - 1
}

func timeoutIdx(hours int, opts []int) int {
	for i, o := range opts {
		if o == hours {
			return i
		}
	}
	return 0
}

func (m *Model) cycleConfigItem(forward bool) {
	step := 1
	if !forward {
		step = -1
	}
	switch m.configCursorIdx {
	case 0:
		m.cfg.DarkMode = !m.cfg.DarkMode
		InitStyles(m.cfg.DarkMode)
	case 1:
		opts := ModelOptions
		idx := (modelIdx(m.cfg.DefaultModel) + step + len(opts)) % len(opts)
		m.cfg.DefaultModel = opts[idx].ID
	case 2:
		opts := EffortOptions
		idx := (effortIdx(m.cfg.DefaultEffort) + step + len(opts)) % len(opts)
		m.cfg.DefaultEffort = opts[idx].ID
	case 3:
		warnPcts := []int{75, 80, 85, 90}
		idx := (warnPctIdx(m.cfg.CompactWarnPct, warnPcts) + step + len(warnPcts)) % len(warnPcts)
		m.cfg.CompactWarnPct = warnPcts[idx]
	case 4:
		m.cfg.AutoCompact = !m.cfg.AutoCompact
	case 5:
		timeouts := []int{0, 4, 8, 24}
		idx := (timeoutIdx(m.cfg.IdleTimeoutHours, timeouts) + step + len(timeouts)) % len(timeouts)
		m.cfg.IdleTimeoutHours = timeouts[idx]
	}
	saveGregConfig(m.cfg)
}

func (m *Model) checkIdleTimeout() {
	if m.cfg.IdleTimeoutHours <= 0 {
		return
	}
	openIDs := map[string]bool{}
	for _, t := range m.tabs {
		if t.GregSessionID != "" {
			openIDs[t.GregSessionID] = true
		}
	}
	sessions, err := session.LoadSessions()
	if err != nil {
		return
	}
	threshold := time.Now().Add(-time.Duration(m.cfg.IdleTimeoutHours) * time.Hour)
	var toArchive []string
	for _, s := range sessions {
		if openIDs[s.ID] {
			continue
		}
		t, err := time.Parse("2006-01-02 15:04:05", s.Started)
		if err != nil {
			continue
		}
		if t.Before(threshold) {
			toArchive = append(toArchive, s.ID)
		}
	}
	if len(toArchive) > 0 {
		session.ArchiveTaskSessions(toArchive)
	}
}

func (m *Model) resolveDetailFile(t *task.Task) string {
	agents := task.AllAgents(*t)
	docs := listWorkspaceFiles(t.Workspace)
	msgs := listMsgChannels(t.Workspace)
	idx := m.multiDetailCursorIdx - len(agents)
	if idx < 0 {
		return ""
	}
	if idx < len(docs) {
		return t.Workspace + "/workspace/" + docs[idx]
	}
	idx -= len(docs)
	if idx < len(msgs) {
		return t.Workspace + "/messages/" + msgs[idx]
	}
	return ""
}

// tabAtX returns the tab index at the given X column in the tab bar, or -1.
func (m Model) tabAtX(x int) int {
	pos := 2 // "  " prefix
	for i, t := range m.tabs {
		w := len(t.Name) + 2 // Padding(0, 1) adds 1 space each side
		if i == m.tabIdx && t.Running {
			w += 2 // " ⠋"
		} else if i != m.tabIdx && t.HasNew {
			w += 2 // " ●"
		}
		if x >= pos && x < pos+w {
			return i
		}
		pos += w + 3 // " │ " separator
	}
	return -1
}
