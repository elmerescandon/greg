package ui

import (
	"github.com/elmerescandon/greg-ui/internal/claude"
)

type Tab struct {
	Name           string
	GregSessionID  string
	ClaudeSession  string
	Lines          []string
	ScrollOffset   int
	ScrollLock     bool
	HasNew         bool
	InputBuf       string
	CursorPos      int
	Running        bool
	CurrentAction  string
	Proc           *claude.Process
	Events         <-chan claude.Event
	Errors         <-chan string
	Cost           float64
	ContextPct     int
	ContextTokens  int
	ContextWindow  int
	CompactWarned  bool
	CompactPending bool
	Model          string
	Effort         string
	PendingQuestion *PendingQuestion
}

type PendingQuestion struct {
	ID          string
	ConfigMode  bool
	Questions   []claude.Question
	CurrentQ    int
	SelectedIdx int
	Answers     map[string]string
}

func NewTab(name, claudeSession, gregSessionID string) *Tab {
	return &Tab{
		Name:          name,
		ClaudeSession: claudeSession,
		GregSessionID: gregSessionID,
		Lines:         []string{},
		InputBuf:      "",
		Model:         "claude-sonnet-4-6",
		Effort:        "high",
		ContextPct:    -1,
	}
}
