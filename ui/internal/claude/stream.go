package claude

import (
	"bufio"
	"encoding/json"
	"io"
	"os/exec"
	"strings"
)

type ModelUsageEntry struct {
	ContextWindow          int `json:"contextWindow"`
	InputTokens            int `json:"inputTokens"`
	OutputTokens           int `json:"outputTokens"`
	CacheReadInputTokens   int `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int `json:"cacheCreationInputTokens"`
}

type Iteration struct {
	InputTokens              int `json:"input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

type Usage struct {
	Iterations []Iteration `json:"iterations"`
}

type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Name      string          `json:"name,omitempty"`
	ID        string          `json:"id,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"` // tool_result payload
	ToolUseID string          `json:"tool_use_id,omitempty"`
}

type Message struct {
	Content []ContentBlock `json:"content"`
}

type Event struct {
	Type         string                      `json:"type"`
	SessionID    string                      `json:"session_id,omitempty"`
	Message      *Message                    `json:"message,omitempty"`
	TotalCostUSD float64                     `json:"total_cost_usd,omitempty"`
	ModelUsage   map[string]ModelUsageEntry   `json:"modelUsage,omitempty"`
	Usage        *Usage                      `json:"usage,omitempty"`
	Subtype      string                      `json:"subtype,omitempty"`
	Error        string                      `json:"error,omitempty"`
}

type AskUserInput struct {
	Questions []Question `json:"questions"`
}

type Question struct {
	Question    string   `json:"question"`
	Header      string   `json:"header"`
	Options     []Option `json:"options"`
	MultiSelect bool     `json:"multiSelect"`
}

type Option struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

type ToolInput struct {
	Command     string `json:"command,omitempty"`
	Path        string `json:"path,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	Description string `json:"description,omitempty"`
}

type Process struct {
	Cmd   *exec.Cmd
	Stdin io.WriteCloser
	Done  chan struct{}
}

func StartClaude(vault, model, effort, prompt string, claudeSession string, workdir string) (*Process, <-chan Event, <-chan string) {
	args := []string{
		"--model", model,
		"--effort", effort,
		"-p", prompt,
		"--output-format", "stream-json",
		"--verbose",
		"--dangerously-skip-permissions",
	}
	if claudeSession != "" {
		args = append(args, "--resume", claudeSession)
	}

	cmd := exec.Command("claude", args...)
	if workdir != "" {
		cmd.Dir = workdir
	} else {
		cmd.Dir = vault
	}

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	events := make(chan Event, 64)
	errors := make(chan string, 16)
	done := make(chan struct{})

	cmd.Start()

	go func() {
		scanner := bufio.NewScanner(stderr)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" && !strings.Contains(line, "no stdin data received") {
				errors <- line
			}
		}
	}()

	go func() {
		defer close(events)
		defer close(done)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			var ev Event
			if err := json.Unmarshal([]byte(line), &ev); err != nil {
				continue
			}
			events <- ev
		}
		cmd.Wait()
	}()

	return &Process{Cmd: cmd, Stdin: stdin, Done: done}, events, errors
}

func FormatToolLabel(name string, raw json.RawMessage) string {
	var input ToolInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return ""
	}
	if input.Command != "" {
		r := []rune(input.Command)
		if len(r) > 100 {
			return string(r[:100])
		}
		return input.Command
	}
	if input.Path != "" {
		return input.Path
	}
	if input.FilePath != "" {
		return input.FilePath
	}
	if input.Description != "" {
		r := []rune(input.Description)
		if len(r) > 80 {
			return string(r[:80])
		}
		return input.Description
	}
	r := []rune(string(raw))
	if len(r) > 80 {
		return string(r[:80])
	}
	return string(raw)
}
