package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Theme holds all color definitions for the TUI
type Theme struct {
	// Primary colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color

	// Status colors
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color

	// Text colors
	Text      lipgloss.Color
	TextMuted lipgloss.Color
	TextDim   lipgloss.Color

	// Background colors
	Background        lipgloss.Color
	BackgroundPanel   lipgloss.Color
	BackgroundSurface lipgloss.Color

	// Border colors
	Border       lipgloss.Color
	BorderActive lipgloss.Color
	BorderSubtle lipgloss.Color
}

// DefaultTheme is the dark theme matching OpenCode's style
var DefaultTheme = Theme{
	Primary:   lipgloss.Color("#58a6ff"),
	Secondary: lipgloss.Color("#3fb950"),
	Accent:    lipgloss.Color("#79c0ff"),

	Success: lipgloss.Color("#3fb950"),
	Warning: lipgloss.Color("#d29922"),
	Error:   lipgloss.Color("#f85149"),
	Info:    lipgloss.Color("#58a6ff"),

	Text:      lipgloss.Color("#c9d1d9"),
	TextMuted: lipgloss.Color("#8b949e"),
	TextDim:   lipgloss.Color("#6e7681"),

	Background:        lipgloss.Color("#0d1117"),
	BackgroundPanel:   lipgloss.Color("#161b22"),
	BackgroundSurface: lipgloss.Color("#21262d"),

	Border:       lipgloss.Color("#30363d"),
	BorderActive: lipgloss.Color("#58a6ff"),
	BorderSubtle: lipgloss.Color("#21262d"),
}

// Style helpers
func (t Theme) HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)
}

func (t Theme) PanelStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 1)
}

// Legacy color variables for backward compatibility
var (
	ColorBg        = DefaultTheme.Background
	ColorBgPanel   = DefaultTheme.BackgroundPanel
	ColorBgSurface = DefaultTheme.BackgroundSurface
	ColorBgAccent  = lipgloss.Color("#30363d")

	ColorBright = DefaultTheme.Text
	ColorWhite  = lipgloss.Color("#e6edf3")
	ColorSubtle = DefaultTheme.TextMuted
	ColorMuted  = lipgloss.Color("#656d76")
	ColorDim    = DefaultTheme.TextDim
	ColorGhost  = DefaultTheme.BorderSubtle

	ColorAccent  = DefaultTheme.Primary
	ColorAccent2 = DefaultTheme.Secondary
	ColorAccent3 = lipgloss.Color("#a5a5a5")

	ColorSuccess = DefaultTheme.Success
	ColorDanger  = DefaultTheme.Error
	ColorWarning = DefaultTheme.Warning
	ColorGold    = lipgloss.Color("#e3b341")
	ColorCyan    = DefaultTheme.Accent
	ColorPurple  = lipgloss.Color("#bc8cff")
	ColorOrange  = lipgloss.Color("#ff7b72")

	ColorOutputInfo    = DefaultTheme.TextMuted
	ColorOutputDebug   = lipgloss.Color("#656d76")
	ColorOutputSuccess = DefaultTheme.Secondary
	ColorOutputError   = lipgloss.Color("#ff7b72")
	ColorOutputWarn    = lipgloss.Color("#e3b341")
	ColorOutputSignal  = DefaultTheme.Primary

	ColorBorder     = DefaultTheme.Border
	ColorBorderSoft = DefaultTheme.BorderSubtle
)

var (
	HeaderBrandStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	HeaderSepStyle = lipgloss.NewStyle().
		Foreground(ColorBorder).
		Bold(true)

	HeaderInfoStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle)

	HeaderDimStyle = lipgloss.NewStyle().
		Foreground(ColorDim)

	StatusLabelStyle = lipgloss.NewStyle().
		Foreground(ColorDim)

	StatusOnStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	StatusOffStyle = lipgloss.NewStyle().
		Foreground(ColorDim)

	StatusNodeStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	StatusAlertStyle = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true)

	HeaderBarStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle)

	StatusBarStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle)
)

var (
	PhaseIdleStyle = lipgloss.NewStyle().
		Foreground(ColorDim).
		Bold(true)

	PhasePlanningStyle = lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true)

	PhaseReasoningStyle = lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)

	PhaseExecutingStyle = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)

	PhaseVerifyingStyle = lipgloss.NewStyle().
		Foreground(ColorAccent2).
		Bold(true)

	PhaseCompleteStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	PhaseErrorStyle = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true)

	PhaseHitLStyle = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)
)

var (
	OutputPaneStyle = lipgloss.NewStyle().
		PaddingLeft(1)

	LineNumberStyle = lipgloss.NewStyle().
		Foreground(ColorDim).
		Width(4).
		Align(lipgloss.Right)

	ActivityDimStyle = lipgloss.NewStyle().
		Foreground(ColorDim)

	MainPaneStyle = lipgloss.NewStyle()

	SidebarPaneStyle = lipgloss.NewStyle()
)

var (
	AgentTextStyle = lipgloss.NewStyle().
		Foreground(ColorWhite)

	AgentHeaderStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	AgentSubheaderStyle = lipgloss.NewStyle().
		Foreground(ColorAccent2).
		Bold(true)

	AgentListStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle).
		MarginLeft(2)

	AgentQuoteStyle = lipgloss.NewStyle().
		Foreground(ColorDim).
		Italic(true).
		BorderLeft(true).
		BorderForeground(ColorBorder).
		PaddingLeft(2).
		MarginLeft(1)

	CodeBlockStyle = lipgloss.NewStyle().
		Foreground(ColorBright).
		Background(ColorBgAccent).
		Padding(0, 1)

	InlineCodeStyle = lipgloss.NewStyle().
		Foreground(ColorOrange).
		Background(ColorBgAccent).
		Padding(0, 1)

	KeywordStyle = lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)

	StringStyle = lipgloss.NewStyle().
		Foreground(ColorAccent2)

	CommentStyle = lipgloss.NewStyle().
		Foreground(ColorDim).
		Italic(true)

	AgentResponseStyle = lipgloss.NewStyle().
		Foreground(ColorWhite).
		PaddingLeft(1)

	AgentDividerStyle = lipgloss.NewStyle().
		Foreground(ColorBorder)
)

var (
	ToolStartStyle = lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true).
		MarginTop(1)

	ToolBorderStyle = lipgloss.NewStyle().
		Foreground(ColorBorder)

	ToolErrorStyle = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true).
		MarginTop(1)

	ToolDoneStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true).
		MarginTop(1)

	ToolArgsStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle).
		Italic(true).
		MarginLeft(2)

	ToolOutputStyle = lipgloss.NewStyle().
		Foreground(ColorOutputInfo).
		MarginLeft(2)

	ToolOutputSuccessStyle = lipgloss.NewStyle().
		Foreground(ColorOutputSuccess).
		MarginLeft(2)

	ToolOutputErrorStyle = lipgloss.NewStyle().
		Foreground(ColorOutputError).
		MarginLeft(2)
)

var (
	PromptBorderStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	PromptAliasStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	PromptAtStyle = lipgloss.NewStyle().
		Foreground(ColorDim)

	PromptAgentStyle = lipgloss.NewStyle().
		Foreground(ColorAccent2)

	PromptSessionStyle = lipgloss.NewStyle().
		Foreground(ColorDim).
		Italic(true)

	PromptUserStyle = lipgloss.NewStyle().
		Foreground(ColorBright).
		Bold(true)

	PromptGlyphStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true).
		MarginRight(1)

	InputPaneStyle = lipgloss.NewStyle().
		Padding(0, 1)
)

var (
	HintBorderStyle = lipgloss.NewStyle().
		Foreground(ColorDim)

	HintCmdStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	HintDescStyle = lipgloss.NewStyle().
		Foreground(ColorMuted)

	HintSelectedStyle = lipgloss.NewStyle().
		Foreground(ColorBright).
		Bold(true)
)

var (
	ErrorStyle = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true)

	WarningStyle = lipgloss.NewStyle().
		Foreground(ColorWarning)

	InfoStyle = lipgloss.NewStyle().
		Foreground(ColorCyan)

	SpinnerStyle = lipgloss.NewStyle().
		Foreground(ColorAccent)

	QueueStyle = lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)

	QueueItemStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle)

	StatusLineStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle).
		Italic(true)

	SignalLineStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	DividerStyle = lipgloss.NewStyle().
		Foreground(ColorGhost)

	SessionStyle = lipgloss.NewStyle().
		Foreground(ColorDim)
)

var (
	EvidenceStyle = lipgloss.NewStyle().
		Foreground(ColorCyan).
		Italic(true)

	FactStyle = lipgloss.NewStyle().
		Foreground(ColorWhite)

	FactKeyStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	FactValStyle = lipgloss.NewStyle().
		Foreground(ColorWhite)

	TruncatedStyle = lipgloss.NewStyle().
		Foreground(ColorDim).
		Italic(true)
)

var (
	TreeBranchStyle = lipgloss.NewStyle().
		Foreground(ColorGhost)

	TreeItemStyle = lipgloss.NewStyle().
		Foreground(ColorWhite)

	TreeItemSelectedStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	TreeIndentStyle = lipgloss.NewStyle().
		Foreground(ColorGhost)
)

var (
	WelcomeTitleStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	WelcomeSubtitleStyle = lipgloss.NewStyle().
		Foreground(ColorMuted)

	WelcomeHintStyle = lipgloss.NewStyle().
		Foreground(ColorDim).
		Italic(true)

	WelcomeBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1).
		Background(ColorBgPanel)

	WelcomeQuickStartStyle = lipgloss.NewStyle().
		Foreground(ColorAccent2).
		Bold(true)
)

var (
	SidebarTitleStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle).
		Bold(true)

	SidebarLabelStyle = lipgloss.NewStyle().
		Foreground(ColorDim)

	SidebarValueStyle = lipgloss.NewStyle().
		Foreground(ColorWhite)

	SidebarRuleStyle = lipgloss.NewStyle().
		Foreground(ColorGhost)
)

var (
	SectionHeaderStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle).
		Bold(true)

	SectionRuleStyle = lipgloss.NewStyle().
		Foreground(ColorGhost)
)

func PhaseStyle(phase string) lipgloss.Style {
	switch phase {
	case "idle":
		return PhaseIdleStyle
	case "planning":
		return PhasePlanningStyle
	case "reasoning":
		return PhaseReasoningStyle
	case "executing":
		return PhaseExecutingStyle
	case "verifying":
		return PhaseVerifyingStyle
	case "complete":
		return PhaseCompleteStyle
	case "error":
		return PhaseErrorStyle
	default:
		return PhaseIdleStyle
	}
}

func CustomHuhTheme() *huh.Theme {
	t := huh.ThemeBase()

	t.Focused.Base = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ColorAccent).
		PaddingLeft(2)
	t.Blurred.Base = lipgloss.NewStyle().PaddingLeft(3)

	t.Focused.Title = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	t.Blurred.Title = lipgloss.NewStyle().Foreground(ColorDim)
	t.Focused.Description = lipgloss.NewStyle().Foreground(ColorSubtle)
	t.Blurred.Description = lipgloss.NewStyle().Foreground(ColorDim)

	t.Focused.Option = lipgloss.NewStyle().Foreground(ColorWhite)
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	t.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(ColorDim)

	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(ColorAccent)
	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(ColorWhite)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(ColorDim)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(ColorAccent2)

	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(ColorAccent)
	t.Focused.MultiSelectSelector = lipgloss.NewStyle().Foreground(ColorAccent)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(ColorSuccess)
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(ColorDim)

	t.Focused.FocusedButton = lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorAccent).
		Bold(true).
		Padding(0, 1)
	t.Focused.BlurredButton = lipgloss.NewStyle().
		Foreground(ColorWhite).
		Background(ColorBgAccent).
		Padding(0, 1)

	t.Help.Ellipsis = lipgloss.NewStyle().Foreground(ColorDim)
	t.Help.ShortKey = lipgloss.NewStyle().Foreground(ColorSubtle)
	t.Help.ShortDesc = lipgloss.NewStyle().Foreground(ColorDim)
	t.Help.ShortSeparator = lipgloss.NewStyle().Foreground(ColorGhost)
	t.Help.FullKey = lipgloss.NewStyle().Foreground(ColorSubtle)
	t.Help.FullDesc = lipgloss.NewStyle().Foreground(ColorDim)
	t.Help.FullSeparator = lipgloss.NewStyle().Foreground(ColorGhost)

	return t
}
