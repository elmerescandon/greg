package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type HistoryEntry struct {
	Type    string          `json:"type"`
	Message *HistoryMessage `json:"message,omitempty"`
}

type HistoryMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type HistoryBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"` // tool_result content
	ToolUseID string          `json:"tool_use_id,omitempty"`
}

func SessionSummary(vault, claudeSessionID string) string {
	if claudeSessionID == "" {
		return ""
	}
	home, _ := os.UserHomeDir()
	projectKey := strings.ReplaceAll(vault, "/", "-")
	path := filepath.Join(home, ".claude", "projects", projectKey, claudeSessionID+".jsonl")

	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var e HistoryEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if e.Type != "user" || e.Message == nil {
			continue
		}
		var text string
		if err := json.Unmarshal(e.Message.Content, &text); err == nil {
			text = strings.TrimSpace(text)
			if text != "" {
				r := []rune(text)
				if len(r) > 60 {
					return string(r[:60])
				}
				return text
			}
		}
		var blocks []HistoryBlock
		if err := json.Unmarshal(e.Message.Content, &blocks); err == nil {
			for _, b := range blocks {
				if b.Type == "text" && b.Text != "" {
					text = strings.TrimSpace(b.Text)
					r := []rune(text)
					if len(r) > 60 {
						return string(r[:60])
					}
					return text
				}
			}
		}
	}
	return ""
}

func LoadHistory(vault, claudeSessionID string, limit int) []HistoryEntry {
	if claudeSessionID == "" {
		return nil
	}
	home, _ := os.UserHomeDir()
	projectKey := strings.ReplaceAll(vault, "/", "-")
	path := filepath.Join(home, ".claude", "projects", projectKey, claudeSessionID+".jsonl")

	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var e HistoryEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if (e.Type != "user" && e.Type != "assistant") || e.Message == nil {
			continue
		}
		entries = append(entries, e)
	}

	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries
}
