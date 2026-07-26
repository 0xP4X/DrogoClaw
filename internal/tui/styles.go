package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// DrogonClaw TUI color palette — restrained, high-contrast operator console.
var (
	// ── Core palette ──────────────────────────────────────────────────────────
	ColorAccent   = lipgloss.Color("#7DD3FC") // sky — current focus
	ColorAccent2  = lipgloss.Color("#A5B4FC") // indigo — secondary emphasis
	ColorDim      = lipgloss.Color("#334155") // slate border
	ColorMuted    = lipgloss.Color("#64748B") // secondary text
	ColorDanger   = lipgloss.Color("#FB7185") // errors / alerts
	ColorWarning  = lipgloss.Color("#FBBF24") // warnings
	ColorSuccess  = lipgloss.Color("#86EFAC") // successful / active
	ColorGold     = lipgloss.Color("#FDE68A") // badges
	ColorPurple   = lipgloss.Color("#C4B5FD") // tool names
	ColorCyan     = lipgloss.Color("#67E8F9") // highlights
	ColorWhite    = lipgloss.Color("#F1F5F9") // primary text
	ColorSubtle   = lipgloss.Color("#CBD5E1") // light meta text
	ColorBg       = lipgloss.Color("#020617") // deep navy
	ColorBgPanel  = lipgloss.Color("#0F172A") // panel bg
	ColorBgStatus = lipgloss.Color("#0B1220") // status bar bg
	ColorBgAccent = lipgloss.Color("#162033") // accent panel bg

	// ── Status bar ────────────────────────────────────────────────────────────
	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorBgStatus).
			Foreground(ColorMuted).
			Padding(0, 1)

	StatusLabelStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Background(ColorBgStatus)

	StatusOnStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Background(ColorBgStatus).
			Bold(true)

	StatusOffStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Background(ColorBgStatus)

	StatusNodeStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Background(ColorBgStatus).
			Bold(true)

	StatusAlertStyle = lipgloss.NewStyle().
				Foreground(ColorDanger).
				Background(ColorBgStatus).
				Bold(true)

	// ── Output pane ───────────────────────────────────────────────────────────
	OutputPaneStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1)

	// ── Cards / rails ─────────────────────────────────────────────────────────
	ControlRailStyle = lipgloss.NewStyle().
				Background(ColorBgPanel).
				Foreground(ColorWhite).
				Border(lipgloss.RoundedBorder(), true).
				BorderForeground(ColorDim).
				Padding(0, 1)

	ControlRailMutedStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	ControlRailLabelStyle = lipgloss.NewStyle().
				Foreground(ColorSubtle).
				Bold(true)

	ControlRailValueStyle = lipgloss.NewStyle().
				Foreground(ColorWhite).
				Bold(true)

	ControlRailAccentStyle = lipgloss.NewStyle().
				Foreground(ColorAccent2).
				Bold(true)

	ControlRailSuccessStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	ControlRailWarningStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	ControlRailDangerStyle = lipgloss.NewStyle().
				Foreground(ColorDanger).
				Bold(true)

	ActivityRailStyle = lipgloss.NewStyle().
				Background(ColorBgPanel).
				Foreground(ColorWhite).
				Border(lipgloss.RoundedBorder(), true).
				BorderForeground(ColorDim).
				Padding(0, 1)

	ActivityTitleStyle = lipgloss.NewStyle().
				Foreground(ColorCyan).
				Bold(true)

	ActivityDimStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	PhasePlanningStyle = lipgloss.NewStyle().
				Foreground(ColorAccent2).
				Bold(true)

	PhaseReasoningStyle = lipgloss.NewStyle().
				Foreground(ColorCyan).
				Bold(true)

	PhaseExecutingStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	PhaseVerifyingStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	PhaseCompleteStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	PhaseErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	// ── Agent response box ────────────────────────────────────────────────────
	AgentBoxTopStyle = lipgloss.NewStyle().
				Foreground(ColorAccent2).
				Bold(true)

	AgentBoxSideStyle = lipgloss.NewStyle().
				Foreground(ColorDim)

	AgentTextStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)

	// ── Tool execution panels ─────────────────────────────────────────────────
	ToolStartStyle = lipgloss.NewStyle().
			Foreground(ColorAccent2).
			Bold(true)

	ToolDoneStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	ToolArgsStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	// ── Prompt / input area ───────────────────────────────────────────────────
	//  ┏━ ALIAS@drogonclaw [Session: abc12345]
	//  ┗━❯  _
	PromptBorderStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	// Operator alias — glows cyan so it stands out immediately
	PromptAliasStyle = lipgloss.NewStyle().
				Foreground(ColorCyan).
				Bold(true)

	// Separator "@" between alias and agent name
	PromptAtStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Agent name — dimmer violet
	PromptAgentStyle = lipgloss.NewStyle().
				Foreground(ColorAccent2)

	// Session badge
	PromptSessionStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Italic(true)

	// Submitted user text echo
	PromptUserStyle = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Bold(true)

	// Legacy alias kept for backward compat in model.go
	PromptGlyphStyle = PromptBorderStyle

	// ── Hint / autocomplete ───────────────────────────────────────────────────
	HintBorderStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	HintCmdStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	HintDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// ── Error / warning ───────────────────────────────────────────────────────
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// ── HitL banner ───────────────────────────────────────────────────────────
	HitLBannerStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	// ── Spinner ───────────────────────────────────────────────────────────────
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	// ── Divider ───────────────────────────────────────────────────────────────
	// Gradient-like: uses braille dots for a subtle textured rule
	DividerStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	// ── Session / meta text ───────────────────────────────────────────────────
	SessionStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// ── Tool panel badges (pill labels) ──────────────────────────────────────
	ToolBadgeReconStyle = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorAccent).
				Bold(true).
				Padding(0, 1)

	ToolBadgeExploitStyle = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorDanger).
				Bold(true).
				Padding(0, 1)

	ToolBadgeIntelStyle = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorGold).
				Bold(true).
				Padding(0, 1)

	ToolBadgeMemoryStyle = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorPurple).
				Bold(true).
				Padding(0, 1)

	ToolBadgeSystemStyle = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorSuccess).
				Bold(true).
				Padding(0, 1)

	// ── Tool panel structure ──────────────────────────────────────────────────
	ToolPanelBorderStyle = lipgloss.NewStyle().
				Foreground(ColorDim)

	ToolPanelHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorCyan).
				Bold(true)

	ToolArgKeyStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	ToolArgValStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)

	ToolOutputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8892A4"))

	ToolOutputSuccessStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	ToolOutputErrorStyle = lipgloss.NewStyle().
				Foreground(ColorDanger)

	ToolTimingStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	// ── Dashboard Layout ──────────────────────────────────────────────────────
	DashboardPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), true).
				BorderForeground(ColorDim).
				Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
			Background(ColorBgStatus).
			Foreground(ColorWhite).
			Bold(true).
			Padding(0, 1).
			MarginBottom(0)

	SidebarTitleStyle = lipgloss.NewStyle().
				Foreground(ColorCyan).
				Bold(true).
				MarginBottom(1)
)

// CustomHuhTheme builds a custom, high-tech lipgloss theme for Huh forms.
func CustomHuhTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Customize base layout and selection indicators
	t.Focused.Base = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(ColorAccent).PaddingLeft(2)
	t.Blurred.Base = lipgloss.NewStyle().PaddingLeft(3)

	// Style text fields
	t.Focused.Title = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	t.Blurred.Title = lipgloss.NewStyle().Foreground(ColorMuted)
	t.Focused.Description = lipgloss.NewStyle().Foreground(ColorSubtle)
	t.Blurred.Description = lipgloss.NewStyle().Foreground(ColorMuted)

	// Selection and options styling
	t.Focused.Option = lipgloss.NewStyle().Foreground(ColorWhite)
	t.Focused.SelectedOption = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	t.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(ColorMuted)

	// Input controls (text boxes)
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(ColorAccent)
	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(ColorWhite)

	return t
}
