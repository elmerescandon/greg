package ui

import "charm.land/lipgloss/v2"

// color vars — overwritten by InitStyles on startup and on theme change
var (
	colorBg          = "#080c12"
	colorSurface     = "#0d1320"
	colorBorder      = "#1c2a3a"
	colorDim         = "#4a5a6e"
	colorMuted       = "#8899aa"
	colorText        = "#c8d8e8"
	colorCyan        = "#4de8ff"
	colorAmber       = "#f5a32a"
	colorGreen       = "#3ddc84"
	colorRed         = "#ff6b6b"
	colorViolet      = "#a78bfa"
	colorSelBg       = "#0f1e3a"
	colorSelFg       = "#7dd3fc"
	colorSynthBg     = "#160d2a"
	colorSynthFg     = "#c4a9ff"
	colorTabActiveBg = "#131f30"
	colorFooterBg    = "#090e17"
)

var (
	StatusBarStyle = lipgloss.NewStyle().
			Padding(0, 1)

	StatusGreen = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGreen)).
			Bold(true)

	StatusYellow = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAmber))

	StatusPurple = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorViolet))

	TabActive = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorCyan)).
			Background(lipgloss.Color(colorTabActiveBg)).
			Bold(true).
			Padding(0, 1)

	TabInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim)).
			Padding(0, 1)

	TabBarStyle = lipgloss.NewStyle()

	TabNewBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGreen))

	TabSeparator = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBorder))

	OutputStyle = lipgloss.NewStyle().
			Padding(0, 1)

	InputStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(colorSurface)).
			Padding(0, 1)

	InputPrompt = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim))

	InputCursor = lipgloss.NewStyle().
			Reverse(true)

	UserMessage = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorCyan))

	ToolCall = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAmber))

	ToolResult = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim))

	ErrorText = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorRed))

	DimText = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorDim))

	FooterStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(colorFooterBg)).
			Foreground(lipgloss.Color(colorDim)).
			Padding(0, 1)

	QuestionBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorCyan)).
			Background(lipgloss.Color(colorSurface)).
			Padding(0, 1)

	QuestionSelected = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorGreen))

	QuestionLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Bold(true)

	QuestionLabelDim = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorDim))

	CtxLow = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorDim))

	CtxMed = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorAmber))

	CtxHigh = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorRed))

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAmber))

	CostStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim))

	ModelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted))

	GregLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorCyan))

	ViewActive = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorCyan)).
			Bold(true)

	ViewInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim))

	SepActive = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorCyan))

	SepDim = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBorder))

	SectionHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorMuted))

	TaskRowSelected = lipgloss.NewStyle().
			Background(lipgloss.Color(colorSelBg)).
			Foreground(lipgloss.Color(colorSelFg))

	TaskRowDim = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted))

	SynthesizerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorViolet))

	SynthesizerSelected = lipgloss.NewStyle().
				Background(lipgloss.Color(colorSynthBg)).
				Foreground(lipgloss.Color(colorSynthFg))
)

// InitStyles reinitializes all style variables for the given theme.
// Must be called after changing the theme and before the next render.
func InitStyles(dark bool) {
	if dark {
		colorBg          = "#080c12"
		colorSurface     = "#0d1320"
		colorBorder      = "#1c2a3a"
		colorDim         = "#4a5a6e"
		colorMuted       = "#8899aa"
		colorText        = "#c8d8e8"
		colorCyan        = "#4de8ff"
		colorAmber       = "#f5a32a"
		colorGreen       = "#3ddc84"
		colorRed         = "#ff6b6b"
		colorViolet      = "#a78bfa"
		colorSelBg       = "#0f1e3a"
		colorSelFg       = "#7dd3fc"
		colorSynthBg     = "#160d2a"
		colorSynthFg     = "#c4a9ff"
		colorTabActiveBg = "#131f30"
		colorFooterBg    = "#090e17"

		heatSessionColors = []string{"#161b22", "#0e4429", "#006d32", "#26a641", "#3ddc84"}
		heatCostColors    = []string{"#161b22", "#2a1500", "#5c3600", "#b36b00", "#f5a32a"}
	} else {
		// Gruvbox Light palette
		colorBg          = "#f9f5e7"
		colorSurface     = "#ede8d0"
		colorBorder      = "#c8c2a5"
		colorDim         = "#928374"
		colorMuted       = "#7c6f64"
		colorText        = "#3c3836"
		colorCyan        = "#076678"
		colorAmber       = "#b57614"
		colorGreen       = "#79740e"
		colorRed         = "#9d0006"
		colorViolet      = "#8f3f71"
		colorSelBg       = "#d5c4a1"
		colorSelFg       = "#076678"
		colorSynthBg     = "#f2e5bc"
		colorSynthFg     = "#8f3f71"
		colorTabActiveBg = "#ede8d0"
		colorFooterBg    = "#f2ecd8"

		heatSessionColors = []string{"#ede8d0", "#b8d8b0", "#7db87d", "#3a9a3a", "#79740e"}
		heatCostColors    = []string{"#ede8d0", "#f5deba", "#e8c070", "#d4a020", "#b57614"}
	}

	StatusBarStyle = lipgloss.NewStyle().Padding(0, 1)
	StatusGreen = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen)).Bold(true)
	StatusYellow = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAmber))
	StatusPurple = lipgloss.NewStyle().Foreground(lipgloss.Color(colorViolet))
	TabActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorCyan)).
		Background(lipgloss.Color(colorTabActiveBg)).
		Bold(true).
		Padding(0, 1)
	TabInactive = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim)).Padding(0, 1)
	TabBarStyle = lipgloss.NewStyle()
	TabNewBadge = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen))
	TabSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color(colorBorder))
	OutputStyle = lipgloss.NewStyle().Padding(0, 1)
	InputStyle = lipgloss.NewStyle().Background(lipgloss.Color(colorSurface)).Padding(0, 1)
	InputPrompt = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	InputCursor = lipgloss.NewStyle().Reverse(true)
	UserMessage = lipgloss.NewStyle().Foreground(lipgloss.Color(colorCyan))
	ToolCall = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAmber))
	ToolResult = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	ErrorText = lipgloss.NewStyle().Foreground(lipgloss.Color(colorRed))
	DimText = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	FooterStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(colorFooterBg)).
		Foreground(lipgloss.Color(colorDim)).
		Padding(0, 1)
	QuestionBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorCyan)).
		Background(lipgloss.Color(colorSurface)).
		Padding(0, 1)
	QuestionSelected = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen))
	QuestionLabel = lipgloss.NewStyle().Foreground(lipgloss.Color(colorText)).Bold(true)
	QuestionLabelDim = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	CtxLow = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	CtxMed = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAmber))
	CtxHigh = lipgloss.NewStyle().Foreground(lipgloss.Color(colorRed))
	SpinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAmber))
	CostStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	ModelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted))
	GregLabel = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorCyan))
	ViewActive = lipgloss.NewStyle().Foreground(lipgloss.Color(colorCyan)).Bold(true)
	ViewInactive = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	SepActive = lipgloss.NewStyle().Foreground(lipgloss.Color(colorCyan))
	SepDim = lipgloss.NewStyle().Foreground(lipgloss.Color(colorBorder))
	SectionHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorMuted))
	TaskRowSelected = lipgloss.NewStyle().
		Background(lipgloss.Color(colorSelBg)).
		Foreground(lipgloss.Color(colorSelFg))
	TaskRowDim = lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted))
	SynthesizerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorViolet))
	SynthesizerSelected = lipgloss.NewStyle().
		Background(lipgloss.Color(colorSynthBg)).
		Foreground(lipgloss.Color(colorSynthFg))

	// view_graficas.go
	cardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorBorder)).
		Padding(0, 1)
	cardLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorDim))
	cardValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorCyan)).Bold(true)
	cardCostStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorAmber))
}
