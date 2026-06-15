package ui

import (
	"encoding/json"
	"strings"

	"github.com/elmerescandon/greg-ui/internal/claude"
)

func renderHistory(entries []claude.HistoryEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	var lines []string
	lines = append(lines, DimText.Render("── historial ──"))
	lines = append(lines, "")

	for _, e := range entries {
		if e.Message == nil {
			continue
		}
		switch e.Type {
		case "user":
			appendUserHistory(e.Message, &lines)
		case "assistant":
			appendAssistantHistory(e.Message, &lines)
		}
	}

	lines = append(lines, "")
	return lines
}

func appendUserHistory(msg *claude.HistoryMessage, lines *[]string) {
	// content can be a plain string or an array of blocks
	var text string
	if err := json.Unmarshal(msg.Content, &text); err == nil {
		if text != "" {
			*lines = append(*lines, UserMessage.Render("▶ "+text))
			*lines = append(*lines, "")
		}
		return
	}

	var blocks []claude.HistoryBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return
	}
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				*lines = append(*lines, UserMessage.Render("▶ "+b.Text))
				*lines = append(*lines, "")
			}
		case "tool_result":
			content := extractToolResultText(b.Content)
			if content != "" {
				r := []rune(content)
				if len(r) > 200 {
					content = string(r[:200]) + "…"
				}
				*lines = append(*lines, ToolResult.Render(content))
			}
		}
	}
}

func appendAssistantHistory(msg *claude.HistoryMessage, lines *[]string) {
	var blocks []claude.HistoryBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return
	}
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				for _, l := range strings.Split(b.Text, "\n") {
					*lines = append(*lines, l)
				}
				*lines = append(*lines, "")
			}
		case "tool_use":
			label := claude.FormatToolLabel(b.Name, b.Input)
			entry := "⚙ " + b.Name
			if label != "" {
				entry += ": " + label
			}
			*lines = append(*lines, ToolCall.Render(entry))
		}
		// skip "thinking" blocks
	}
}

func extractToolResultText(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return b.Text
			}
		}
	}
	return ""
}
