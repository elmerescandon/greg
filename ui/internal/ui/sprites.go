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
	"  ∧    ∧  \n( ◉    ◉ )\n(   ▽    )\n( ·····  )\n \\______/ ",
	"  ∧    ∧  \n( ◉    ◉ )\n(   △    )\n( ·····  )\n \\______/ ",
	"  ∧    ∧  \n( ◎    ◉ )\n(   ▽    )\n( ·····  )\n \\______/ ",
	"  ∧    ∧  \n( ◉    ◎ )\n(   △    )\n( ·····  )\n \\______/ ",
}

var spriteWaiting = []string{
	"  ∧    ∧  \n( ─    ─ )\n(   ω    )\n(       z)\n \\______/ ",
	"  ∧    ∧  \n( ─    ─ )\n(   ω    )\n(      zZ)\n \\______/ ",
	"  ∧    ∧  \n( ╌    ╌ )\n(   ω    )\n(    zZz )\n \\______/ ",
	"  ∧    ∧  \n( ─    ─ )\n(   ω    )\n(   zZzZ )\n \\______/ ",
}

var spriteNeedsHelp = []string{
	"  ∧    ∧  \n( ●    ● )\n(  >o<   )\n(!      !)\n \\______/ ",
	"  ∧    ∧  \n( ●    ◉ )\n(  >o<   )\n(! !  ! !)\n \\______/ ",
	"  ∧    ∧  \n( ◉    ● )\n(  >o<   )\n(!  !!  !)\n \\______/ ",
	"  ∧    ∧  \n( ●    ● )\n(  >O<   )\n(! !!!! !)\n \\______/ ",
}

var spriteDone = []string{
	"  ∧    ∧  \n( ^    ^ )\n(   ▽    )\n(    ✓   )\n \\______/ ",
	"  ∧    ∧  \n( ⌒    ⌒ )\n(   ▽    )\n(  ✓  ✓  )\n \\______/ ",
	"  ∧    ∧  \n( ^    ^ )\n(   ▽    )\n( ✓  ✓ ✓ )\n \\______/ ",
	"  ∧    ∧  \n( ⌒    ⌒ )\n(   ▽    )\n(  ✓  ✓  )\n \\______/ ",
}

var spriteDirector = []string{
	"  ★    ★  \n( ◆    ◆ )\n(   ─    )\n(  ────  )\n \\══════/ ",
	"  ★    ★  \n( ◇    ◆ )\n(   ─    )\n( ─────  )\n \\══════/ ",
	"  ★    ★  \n( ◆    ◇ )\n(   ─    )\n(  ───── )\n \\══════/ ",
	"  ★    ★  \n( ◆    ◆ )\n(   ▽    )\n(  ────  )\n \\══════/ ",
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
func renderDeskBox(a task.Agent, agentStatus string, spriteFrame string, isDirector bool, boxW int, selected bool) []string {
	if boxW < 22 {
		boxW = 22
	}
	inner := boxW - 2

	// Border style based on status (selected overrides to cyan)
	var borderStyle lipgloss.Style
	if selected {
		borderStyle = ViewActive
	} else if isDirector {
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

	spriteLines := strings.Split(spriteFrame, "\n")
	for len(spriteLines) < 5 {
		spriteLines = append(spriteLines, "")
	}

	var content []string

	// Sprite lines (all, centered)
	for _, sl := range spriteLines {
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
