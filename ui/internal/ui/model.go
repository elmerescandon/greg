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
	DarkMode bool `json:"dark_mode"`
}

func gregConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "greg", "config.json")
}

func loadGregConfig() gregConfig {
	data, err := os.ReadFile(gregConfigPath())
	if err != nil {
		return gregConfig{DarkMode: true}
	}
	var c gregConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return gregConfig{DarkMode: true}
	}
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
	multiSelectedIdx       int
	multiDetailMode        bool
	multiAgentIdx          int
	multiAgentView         bool
	multiAgentScrollOffset int
	multiDocIdx            int
	multiDocSource         string // "workspace" or "messages"
	darkMode               bool

	// Task detail chat/office view
	taskChatInput        string
	taskChatCursorPos    int
	taskChatScrollOffset int
	activeMsgChannel     int
	taskChatFocused      bool
	taskSectionFocus     int // 0 = agentes, 1 = docs/mensajes
}

// Messages
type tickMsg struct{}
type claudeEventMsg struct {
	tabIdx int
	event  claude.Event
}
type claudeErrorMsg struct {
	tabIdx int
	text   string
}
type claudeDoneMsg struct{ tabIdx int }
type clipboardPasteMsg string

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
		}
		if active.ClaudeSession != "" {
			entries := claude.LoadHistory(vault, active.ClaudeSession, 10)
			tab.Lines = renderHistory(entries)
		}
	} else {
		s, _ := session.Spawn(vault)
		name := strings.TrimPrefix(s.ID, "greg-")
		tab = NewTab(name, "", s.ID)
	}

	m := Model{
		tabs:       []*Tab{tab},
		tabIdx:     0,
		vault:      vault,
		historyIdx: -1,
		darkMode:   cfg.DarkMode,
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
		return m, tickCmd()

	case claudeEventMsg:
		if msg.tabIdx < len(m.tabs) {
			m.handleEvent(msg.tabIdx, msg.event)
			t := m.tabs[msg.tabIdx]
			return m, waitForEvent(msg.tabIdx, t.Events, t.Errors)
		}
		return m, nil

	case claudeErrorMsg:
		if msg.tabIdx < len(m.tabs) {
			t := m.tabs[msg.tabIdx]
			t.Lines = append(t.Lines, ErrorText.Render(msg.text))
			if msg.tabIdx != m.tabIdx {
				t.HasNew = true
			}
			return m, waitForEvent(msg.tabIdx, t.Events, t.Errors)
		}
		return m, nil

	case claudeDoneMsg:
		if msg.tabIdx < len(m.tabs) {
			t := m.tabs[msg.tabIdx]
			t.Running = false
			t.Proc = nil
			t.CurrentAction = ""
			t.Lines = append(t.Lines, "")

			if t.CompactPending && msg.tabIdx == m.tabIdx {
				t.CompactPending = false
				t.CompactWarned = false
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
	}

	return m, nil
}

func waitForEvent(tabIdx int, events <-chan claude.Event, errors <-chan string) tea.Cmd {
	return func() tea.Msg {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			return claudeEventMsg{tabIdx: tabIdx, event: ev}
		case err, ok := <-errors:
			if !ok {
				return nil
			}
			return claudeErrorMsg{tabIdx: tabIdx, text: err}
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

					if t.ContextPct < 75 {
						t.CompactWarned = false
						t.CompactPending = false
					} else if t.ContextPct >= 95 {
						t.CompactPending = true
					} else if t.ContextPct >= 90 && !t.CompactWarned {
						t.CompactWarned = true
						t.Lines = append(t.Lines,
							CtxMed.Render(fmt.Sprintf("⚠ contexto al %d%% — Ctrl+K para compactar", t.ContextPct)))
					}

					totalOutput := 0
					for _, m := range ev.ModelUsage {
						totalOutput += m.OutputTokens
					}
					if t.GregSessionID != "" {
						session.AccumulateUsage(t.GregSessionID, totalOutput, ev.TotalCostUSD)
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
	tabIdx := m.tabIdx

	proc, events, errors := claude.StartClaude(vault, model, effort, text, claudeSess)
	t.Proc = proc
	t.Events = events
	t.Errors = errors

	return tea.Batch(
		waitForEvent(tabIdx, events, errors),
		func() tea.Msg {
			<-proc.Done
			return claudeDoneMsg{tabIdx: tabIdx}
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
		m.tabs = append(m.tabs, newT)
		m.tabIdx = len(m.tabs) - 1
		t = m.tab()
		m.showConfigSelection(t)
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
		case "up", "down", " ", "enter":
			m.darkMode = !m.darkMode
			InitStyles(m.darkMode)
			saveGregConfig(gregConfig{DarkMode: m.darkMode})
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
			sessions := loadStandaloneSessions()
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

		// Level 3: document browser
		if m.multiAgentView {
			tasks, _ := task.LoadTasks()
			var docWorkspace string
			if m.multiSelectedIdx < len(tasks) {
				docWorkspace = tasks[m.multiSelectedIdx].Workspace
			}
			switch k {
			case "escape", "backspace", "ctrl+shift+up":
				m.multiAgentView = false
				m.multiAgentScrollOffset = 0
			case "left":
				if docWorkspace != "" {
					var docs []string
					if m.multiDocSource == "messages" {
						docs = listMsgChannels(docWorkspace)
					} else {
						docs = listWorkspaceDocs(docWorkspace)
					}
					if len(docs) > 0 {
						m.multiDocIdx = (m.multiDocIdx - 1 + len(docs)) % len(docs)
						m.multiAgentScrollOffset = 0
					}
				}
			case "right", "tab":
				if docWorkspace != "" {
					var docs []string
					if m.multiDocSource == "messages" {
						docs = listMsgChannels(docWorkspace)
					} else {
						docs = listWorkspaceDocs(docWorkspace)
					}
					if len(docs) > 0 {
						m.multiDocIdx = (m.multiDocIdx + 1) % len(docs)
						m.multiAgentScrollOffset = 0
					}
				}
			case "up":
				m.multiAgentScrollOffset++
			case "down":
				if m.multiAgentScrollOffset > 0 {
					m.multiAgentScrollOffset--
				}
			case "pgup":
				m.multiAgentScrollOffset += m.height / 2
			case "pgdown":
				m.multiAgentScrollOffset -= m.height / 2
				if m.multiAgentScrollOffset < 0 {
					m.multiAgentScrollOffset = 0
				}
			}
			return m, nil
		}

		// Level 2: task detail (office view)
		if m.multiDetailMode {
			tasks, _ := task.LoadTasks()
			n := len(tasks)
			sel := m.multiSelectedIdx
			if sel >= n {
				sel = n - 1
			}
			var curTask *task.Task
			if sel >= 0 && sel < n {
				t := tasks[sel]
				curTask = &t
			}

			// Input-focused mode: text editing
			if m.taskChatFocused {
				switch k {
				case "escape":
					m.taskChatFocused = false
					m.taskChatInput = ""
					m.taskChatCursorPos = 0
				case "enter":
					if strings.TrimSpace(m.taskChatInput) == "" {
						m.taskChatFocused = false
						m.taskChatInput = ""
						m.taskChatCursorPos = 0
					} else if curTask != nil {
						m.sendTaskMessage(curTask.TaskID, m.taskChatInput)
						m.taskChatInput = ""
						m.taskChatCursorPos = 0
						m.taskChatFocused = false
					}
				case "backspace":
					if m.taskChatCursorPos > 0 {
						m.taskChatInput = m.taskChatInput[:m.taskChatCursorPos-1] + m.taskChatInput[m.taskChatCursorPos:]
						m.taskChatCursorPos--
					}
				default:
					if msg.Text != "" {
						m.taskChatInput = m.taskChatInput[:m.taskChatCursorPos] + msg.Text + m.taskChatInput[m.taskChatCursorPos:]
						m.taskChatCursorPos += len(msg.Text)
					}
				}
				return m, nil
			}

			// Navigation mode
			switch k {
			case "escape", "backspace":
				m.multiDetailMode = false
				m.multiAgentIdx = 0
				m.multiDocIdx = 0
				m.multiDocSource = ""
				m.taskChatInput = ""
				m.taskChatCursorPos = 0
				m.taskChatScrollOffset = 0
				m.activeMsgChannel = 0
				m.taskChatFocused = false
				m.taskSectionFocus = 0
			case "tab", "right":
				if curTask != nil {
					channels := listMsgChannels(curTask.Workspace)
					if len(channels) > 0 {
						m.activeMsgChannel = (m.activeMsgChannel + 1) % len(channels)
						m.taskChatScrollOffset = 0
					}
				}
			case "left":
				if curTask != nil {
					channels := listMsgChannels(curTask.Workspace)
					if len(channels) > 0 {
						m.activeMsgChannel = (m.activeMsgChannel - 1 + len(channels)) % len(channels)
						m.taskChatScrollOffset = 0
					}
				}
			case "up":
				if m.taskSectionFocus == 1 {
					m.taskChatScrollOffset++
				} else if curTask != nil {
					if m.multiAgentIdx > 0 {
						m.multiAgentIdx--
					}
				}
			case "down":
				if m.taskSectionFocus == 1 {
					if m.taskChatScrollOffset > 0 {
						m.taskChatScrollOffset--
					}
				} else if curTask != nil {
					agents := task.AllAgents(*curTask)
					if m.multiAgentIdx < len(agents)-1 {
						m.multiAgentIdx++
					}
				}
			case "ctrl+shift+up":
				m.taskSectionFocus = 0
			case "ctrl+shift+down":
				m.taskSectionFocus = 1
			case "pgup":
				m.taskChatScrollOffset += m.height / 2
			case "pgdown":
				m.taskChatScrollOffset -= m.height / 2
				if m.taskChatScrollOffset < 0 {
					m.taskChatScrollOffset = 0
				}
			case "f", "i":
				m.taskChatFocused = true
			case "enter":
				// Open selected message channel in full-screen doc browser
				if curTask != nil {
					channels := listMsgChannels(curTask.Workspace)
					if len(channels) > 0 {
						activeCh := m.activeMsgChannel
						if activeCh >= len(channels) {
							activeCh = 0
						}
						m.multiDocSource = "messages"
						m.multiDocIdx = activeCh
						m.multiAgentView = true
						m.multiAgentScrollOffset = 0
					}
				}
			case "o":
				// Open selected agent workspace doc in full-screen doc browser
				if curTask != nil {
					agents := task.AllAgents(*curTask)
					if m.multiAgentIdx < len(agents) {
						agentFile := agents[m.multiAgentIdx].ID + ".md"
						docs := listWorkspaceDocs(curTask.Workspace)
						m.multiDocIdx = 0
						for i, d := range docs {
							if d == agentFile {
								m.multiDocIdx = i
								break
							}
						}
						m.multiDocSource = "workspace"
						m.multiAgentView = true
						m.multiAgentScrollOffset = 0
					}
				}
			case "x":
				if curTask != nil {
					agents := task.AllAgents(*curTask)
					if m.multiAgentIdx < len(agents) {
						tid := curTask.TaskID
						aid := agents[m.multiAgentIdx].ID
						go func(taskID, agentID string) {
							c := osexec.Command("greg", "task", "done", taskID, agentID)
							c.Run()
						}(tid, aid)
					}
				}
			}
			return m, nil
		}

		// Level 1: task list
		switch k {
		case "enter":
			m.multiDetailMode = true
			m.multiAgentIdx = 0
			m.taskChatInput = ""
			m.taskChatCursorPos = 0
			m.taskChatScrollOffset = 0
			m.activeMsgChannel = 0
			m.taskChatFocused = false
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
	if m.viewMode == ViewMultiple && m.multiDetailMode && m.taskChatFocused {
		m.taskChatInput = m.taskChatInput[:m.taskChatCursorPos] + text + m.taskChatInput[m.taskChatCursorPos:]
		m.taskChatCursorPos += len(text)
		return
	}
	t := m.tab()
	if !t.Running {
		t.InputBuf = t.InputBuf[:t.CursorPos] + text + t.InputBuf[t.CursorPos:]
		t.CursorPos += len(text)
	}
}

func (m *Model) sendTaskMessage(taskID, msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}
	go func(tid, text string) {
		c := osexec.Command("greg", "task", "message", tid, text)
		c.Run()
	}(taskID, msg)
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

func loadStandaloneSessions() []session.Session {
	taskSessions := map[string]bool{}
	var completedTaskSessionIDs []string
	if tasks, err := task.LoadTasks(); err == nil {
		for _, t := range tasks {
			var ids []string
			if t.SynthesizerID != "" {
				taskSessions[t.SynthesizerID] = true
				ids = append(ids, t.SynthesizerID)
			}
			for _, a := range t.Agents {
				if a.SessionID != "" {
					taskSessions[a.SessionID] = true
					ids = append(ids, a.SessionID)
				}
			}
			if t.CoordinatorStatus == "completed" {
				completedTaskSessionIDs = append(completedTaskSessionIDs, ids...)
			}
		}
	}
	session.ArchiveTaskSessions(completedTaskSessionIDs)
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
	// Active sessions first, then finished; within each group newest first — matches sidebar display order
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
			return ai
		}
		return tsKey(all[i]) > tsKey(all[j])
	})
	return all
}

func (m Model) openSidebarSession() (tea.Model, tea.Cmd) {
	sessions := loadStandaloneSessions()
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
	}
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
