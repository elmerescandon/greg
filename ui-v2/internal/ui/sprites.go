package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/elmerescandon/greg-ui/internal/task"
)

// Sprite animation frames — 4 frames per status, each a 5-line string joined by \n.
// Frame is selected as spinIdx % 4.

var spriteWorking = []string{
	" (o_o)  \n  ⌨░░  ",
	" (o_o)  \n  ⌨▒░  ",
	" (O_o)  \n  ⌨▒▒  ",
	" (o_O)  \n  ⌨█▒  ",
}

var spriteWaiting = []string{
	" (-_-)  \n   zzZ  ",
	" (-_-)  \n   zZz  ",
	" (-.-) \n   Zzz  ",
	" (-_-)  \n   ZZz  ",
}

var spriteNeedsHelp = []string{
	" (o_O)! \n   ‼‼   ",
	" !(O_o) \n   ‼‼   ",
	" (O_O)! \n   !!   ",
	" !(o_O) \n   !!   ",
}

var spriteDone = []string{
	" (^_^)  \n   ✔✔   ",
	" (^_^)  \n   ✔✔   ",
	" (^_^)  \n   ✔✔   ",
	" (^_^)  \n   ✔✔   ",
}

var spriteDirector = []string{
	"  ]=[   \n (^o^)  ",
	"  ]=]   \n (^o^)  ",
	"  [=[   \n (^O^)  ",
	"  ]=]   \n (^o^)  ",
}

// agentSpriteFrame returns the sprite string for the given status and animation index.
func agentSpriteFrame(status string, spinIdx int) string {
	idx := spinIdx % 4
	switch status {
	case "working":
		return spriteWorking[idx]
	case "waiting":
		return spriteWaiting[idx]
	case "needs-help":
		return spriteNeedsHelp[idx]
	case "done", "completed":
		return spriteDone[idx]
	}
	return spriteWaiting[idx]
}

// renderDeskBox renders a single agent desk as a []string of exactly 10 lines.
// boxW is the total width including borders (minimum 22).
// Lines use ANSI color for borders; content is plain text.
func renderDeskBox(a task.Agent, agentStatus string, spriteFrame string, isDirector bool, boxW int) []string {
	if boxW < 22 {
		boxW = 22
	}
	inner := boxW - 2

	// Border style based on status
	var borderStyle lipgloss.Style
	if isDirector {
		borderStyle = SynthesizerStyle
	} else {
		switch agentStatus {
		case "working":
			borderStyle = StatusYellow
		case "done", "completed":
			borderStyle = StatusGreen
		case "needs-help":
			borderStyle = ErrorText
		default:
			borderStyle = DimText
		}
	}

	// center pads s symmetrically to width w (rune-aware, no ANSI in input)
	center := func(s string, w int) string {
		s = strings.TrimRight(s, " ")
		runes := []rune(s)
		if len(runes) >= w {
			return string(runes[:w])
		}
		pad := w - len(runes)
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	}

	// padRight pads s to exactly w chars, truncating if needed
	padRight := func(s string, w int) string {
		runes := []rune(s)
		if len(runes) >= w {
			return string(runes[:w])
		}
		return s + strings.Repeat(" ", w-len(runes))
	}

	truncate := func(s string, max int) string {
		runes := []rune(s)
		if len(runes) <= max {
			return s
		}
		if max <= 1 {
			return "…"
		}
		return string(runes[:max-1]) + "…"
	}

	// Build 8 content lines
	spriteLines := strings.Split(spriteFrame, "\n")
	for len(spriteLines) < 2 {
		spriteLines = append(spriteLines, "")
	}

	var content []string

	// Lines 1-2: sprite face (centered)
	for _, sl := range spriteLines[:2] {
		content = append(content, center(sl, inner))
	}

	// Line 6: agentID + status on same line
	statusTagW := 10
	agentIDW := inner - statusTagW - 2 // 2 for spaces
	if agentIDW < 4 {
		agentIDW = 4
	}
	id := truncate(a.ID, agentIDW)
	stat := truncate(agentStatus, statusTagW)
	line6 := fmt.Sprintf(" %-*s %s", agentIDW, id, stat)
	content = append(content, padRight(line6, inner))

	// Line 7: role
	role := truncate(a.Role, inner-7)
	content = append(content, padRight(" role: "+role, inner))

	// Line 8: session ID
	sess := a.SessionID
	if sess == "" {
		sess = "—"
	}
	sess = truncate(sess, inner-7)
	content = append(content, padRight(" sess: "+sess, inner))

	// Assemble box
	hLine := strings.Repeat("─", inner)
	top := borderStyle.Render("┌"+hLine+"┐")
	bot := borderStyle.Render("└"+hLine+"┘")

	lines := make([]string, 0, 10)
	lines = append(lines, top)
	for _, c := range content {
		lines = append(lines, borderStyle.Render("│")+c+borderStyle.Render("│"))
	}
	lines = append(lines, bot)

	return lines
}
