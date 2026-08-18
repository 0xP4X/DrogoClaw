package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ─────────────────────────────────────────────────────────────────────────────
// Color Palette — Dark Security Theme
// ─────────────────────────────────────────────────────────────────────────────

var (
	// Background tones
	ColorBg        = lipgloss.Color("#101010")
	ColorBgPanel   = lipgloss.Color("#171717")
	ColorBgSurface = lipgloss.Color("#202124")
	ColorBgAccent  = lipgloss.Color("#2a2b2e")
	ColorBgInput   = lipgloss.Color("#161616")

	// Text hierarchy
	ColorBright = lipgloss.Color("#f1f3f4")
	ColorWhite  = lipgloss.Color("#d7dadc")
	ColorSubtle = lipgloss.Color("#a6adb4")
	ColorMuted  = lipgloss.Color("#858b91")
	ColorDim    = lipgloss.Color("#62686f")
	ColorGhost  = lipgloss.Color("#3a3f44")

	// Primary accents
	ColorAccent  = lipgloss.Color("#2dd4bf")
	ColorAccent2 = lipgloss.Color("#22c55e")

	// Semantic colors
	ColorSuccess = lipgloss.Color("#3fb950")
	ColorDanger  = lipgloss.Color("#f85149")
	ColorWarning = lipgloss.Color("#f59e0b")
	ColorGold    = lipgloss.Color("#f59e0b")
	ColorCyan    = lipgloss.Color("#38bdf8")
	ColorPurple  = lipgloss.Color("#c084fc")

	// Output severity
	ColorOutputInfo    = lipgloss.Color("#a6adb4")
	ColorOutputDebug   = lipgloss.Color("#62686f")
	ColorOutputSuccess = lipgloss.Color("#3fb950")
	ColorOutputError   = lipgloss.Color("#f85149")
	ColorOutputWarn    = lipgloss.Color("#f59e0b")
	ColorOutputSignal  = lipgloss.Color("#2dd4bf")

	// Border color
	ColorBorder = lipgloss.Color("#3a3f44")
)

// ─────────────────────────────────────────────────────────────────────────────
// Header & Status Bar
// ─────────────────────────────────────────────────────────────────────────────

var (
	HeaderBarStyle = lipgloss.NewStyle().
			Background(ColorBgSurface).
			Foreground(ColorWhite).
			Bold(true).
			Padding(0, 1)

	HeaderBarBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(ColorBorder).
				Background(ColorBgSurface)

	HeaderBrandStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	HeaderSepStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	HeaderInfoStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	HeaderDimStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorBgSurface).
			Foreground(ColorMuted).
			Padding(0, 0)

	StatusLabelStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Background(ColorBgSurface)

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
)

// ─────────────────────────────────────────────────────────────────────────────
// Phase Indicators
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// Output Pane & Viewport
// ─────────────────────────────────────────────────────────────────────────────

var (
	OutputPaneStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingRight(2)

	LineNumberStyle = lipgloss.NewStyle().
			Foreground(ColorDim).
			Italic(true)

	ActivityDimStyle = lipgloss.NewStyle().
				Foreground(ColorDim)

	MainPaneStyle = lipgloss.NewStyle().
			Padding(1, 1).
			Background(ColorBg)
)

// ─────────────────────────────────────────────────────────────────────────────
// Agent Response
// ─────────────────────────────────────────────────────────────────────────────

var (
	AgentTextStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)

	AgentResponseStyle = lipgloss.NewStyle().
				Foreground(ColorWhite).
				PaddingLeft(1)

	AgentDividerStyle = lipgloss.NewStyle().
				Foreground(ColorBorder)
)

// ─────────────────────────────────────────────────────────────────────────────
// Tool Execution Styles
// ─────────────────────────────────────────────────────────────────────────────

var (
	ToolDoneStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	ToolArgsStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	ToolOutputStyle = lipgloss.NewStyle().
			Foreground(ColorOutputInfo)

	ToolOutputSuccessStyle = lipgloss.NewStyle().
				Foreground(ColorOutputSuccess)

	ToolOutputErrorStyle = lipgloss.NewStyle().
				Foreground(ColorOutputError)
)

// ─────────────────────────────────────────────────────────────────────────────
// Prompt & Input Area
// ─────────────────────────────────────────────────────────────────────────────

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
				Bold(true)

	InputPaneStyle = lipgloss.NewStyle()
)

// ─────────────────────────────────────────────────────────────────────────────
// Command Hints & Autocomplete
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// Severity & Alert Styles
// ─────────────────────────────────────────────────────────────────────────────

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

	// ── Prompt Queue ─────────────────────────────────────────────────────
	QueueStyle = lipgloss.NewStyle().
			Foreground(ColorPurple).
			Bold(true)

	QueueItemStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	// ── Status / Thinking stream lines ───────────────────────────────────
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

// ─────────────────────────────────────────────────────────────────────────────
// Evidence & Facts
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// Tree / Hierarchy
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// Welcome Screen
// ─────────────────────────────────────────────────────────────────────────────

var (
	WelcomeTitleStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	WelcomeSubtitleStyle = lipgloss.NewStyle().
				Foreground(ColorSubtle)

	WelcomeHintStyle = lipgloss.NewStyle().
				Foreground(ColorDim)

	WelcomeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorder).
				Padding(1, 3).
				Foreground(ColorAccent).
				Bold(true)

	WelcomeQuickStartStyle = lipgloss.NewStyle().
				Foreground(ColorDim)
)

// ─────────────────────────────────────────────────────────────────────────────
// Sidebar
// ─────────────────────────────────────────────────────────────────────────────

var (
	SidebarPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(ColorBorder).
				Padding(1, 1).
				Background(ColorBgPanel)

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

// ─────────────────────────────────────────────────────────────────────────────
// Section Headers (used in /status, sidebar, help)
// ─────────────────────────────────────────────────────────────────────────────

var (
	SectionHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorSubtle).
				Bold(true)

	SectionRuleStyle = lipgloss.NewStyle().
				Foreground(ColorGhost)
)

// ─────────────────────────────────────────────────────────────────────────────
// Phase Router
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// Huh Form Theme
// ─────────────────────────────────────────────────────────────────────────────

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
