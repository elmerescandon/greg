package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// AgentClaudeSession maps a greg session ID to its claude session UUID via sessions.json.
func AgentClaudeSession(gregSessionID string) string {
	sessFile := filepath.Join(os.Getenv("HOME"), ".greg", "sessions.json")
	data, err := os.ReadFile(sessFile)
	if err != nil {
		return ""
	}
	var sessions []struct {
		ID            string `json:"id"`
		Dir           string `json:"dir"`
		ClaudeSession string `json:"claude_session_id"`
	}
	if err := json.Unmarshal(data, &sessions); err != nil {
		return ""
	}
	for _, s := range sessions {
		if s.ID == gregSessionID {
			return s.ClaudeSession
		}
	}
	return ""
}

// AgentSessionDir returns the working directory for a greg session.
func AgentSessionDir(gregSessionID string) string {
	sessFile := filepath.Join(os.Getenv("HOME"), ".greg", "sessions.json")
	data, err := os.ReadFile(sessFile)
	if err != nil {
		return ""
	}
	var sessions []struct {
		ID  string `json:"id"`
		Dir string `json:"dir"`
	}
	if err := json.Unmarshal(data, &sessions); err != nil {
		return ""
	}
	for _, s := range sessions {
		if s.ID == gregSessionID {
			return s.Dir
		}
	}
	return ""
}

type Agent struct {
	ID            string `json:"id"`
	Role          string `json:"role"`
	SessionID     string `json:"session_id"`
	IsSynthesizer bool   `json:"-"`
}

// AllAgents returns t.Agents followed by the synthesizer as a final entry (if SynthesizerID is set).
func AllAgents(t Task) []Agent {
	agents := make([]Agent, len(t.Agents))
	copy(agents, t.Agents)
	if t.SynthesizerID != "" {
		agents = append(agents, Agent{
			ID:            t.SynthesizerID,
			Role:          "synthesizer",
			IsSynthesizer: true,
		})
	}
	return agents
}

type Task struct {
	TaskID            string  `json:"task_id"`
	Goal              string  `json:"goal"`
	Workspace         string  `json:"workspace"`
	Created           string  `json:"created"`
	CoordinatorStatus string  `json:"coordinator_status"`
	SynthesizerID     string  `json:"synthesizer_id,omitempty"`
	SchemaVersion     int     `json:"schema_version,omitempty"` // 0/1 = legacy (with synthesizer), 2+ = no synthesizer
	Preset            string  `json:"preset,omitempty"`
	Agents            []Agent `json:"agents"`
}

func MultiTasksDir() string {
	return filepath.Join(os.Getenv("HOME"), ".greg", "multi-tasks")
}

func LoadTasks() ([]Task, error) {
	dir := MultiTasksDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var tasks []Task
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var t Task
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		// manifest.json inside the workspace includes session_id per agent — prefer it.
		if t.Workspace != "" {
			if mdata, merr := os.ReadFile(filepath.Join(t.Workspace, "manifest.json")); merr == nil {
				var manifest Task
				if merr := json.Unmarshal(mdata, &manifest); merr == nil && len(manifest.Agents) > 0 {
					t.Agents = manifest.Agents
				}
			}
		}
		tasks = append(tasks, t)
	}
	// Newest first by Created timestamp
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Created > tasks[j].Created
	})
	return tasks, nil
}

func AgentStatus(workspace, agentID string) string {
	sf := filepath.Join(workspace, "status", agentID+".status")
	data, err := os.ReadFile(sf)
	if err != nil {
		return "—"
	}
	return strings.TrimSpace(string(data))
}
