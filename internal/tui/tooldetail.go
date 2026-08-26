package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ToolDetail holds detailed information about a tool execution
type ToolDetail struct {
	Tool        string
	Args        string
	Result      string
	Duration    time.Duration
	Success     bool
	StartedAt   time.Time
	CompletedAt time.Time
	FindingType string
	Description string
}

// Tool detail styles
var (
	ToolDetailHeaderStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true).
		MarginBottom(1)

	ToolDetailLabelStyle = lipgloss.NewStyle().
		Foreground(ColorDim).
		Width(12)

	ToolDetailValueStyle = lipgloss.NewStyle().
		Foreground(ColorWhite)

	ToolDetailSuccessStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	ToolDetailErrorStyle = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true)

	ToolDetailOutputStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle).
		MarginLeft(2)
)

// renderToolDetailPanel renders the tool execution detail panel
func (m Model) renderToolDetailPanel(width int) string {
	if m.activeToolDetail == nil {
		return WarningStyle.Render("  [!] No tool execution details available.")
	}

	detail := m.activeToolDetail

	var sb strings.Builder
	sb.WriteString(ToolDetailHeaderStyle.Render("TOOL EXECUTION DETAIL"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", max(4, width-2)))
	sb.WriteString("\n\n")

	// Tool name
	sb.WriteString(fmt.Sprintf("%s %s\n",
		ToolDetailLabelStyle.Render("Tool:"),
		ToolDetailValueStyle.Render(detail.Tool)))

	// Status
	status := "✓ SUCCESS"
	statusStyle := ToolDetailSuccessStyle
	if !detail.Success {
		status = "✗ FAILED"
		statusStyle = ToolDetailErrorStyle
	}
	sb.WriteString(fmt.Sprintf("%s %s\n",
		ToolDetailLabelStyle.Render("Status:"),
		statusStyle.Render(status)))

	// Duration
	if detail.Duration > 0 {
		sb.WriteString(fmt.Sprintf("%s %s\n",
			ToolDetailLabelStyle.Render("Duration:"),
			ToolDetailValueStyle.Render(detail.Duration.Round(time.Millisecond).String())))
	}

	// Timestamp
	if !detail.StartedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("%s %s\n",
			ToolDetailLabelStyle.Render("Started:"),
			ToolDetailValueStyle.Render(detail.StartedAt.Format("15:04:05"))))
	}

	sb.WriteString("\n")

	// Arguments
	if detail.Args != "" {
		sb.WriteString(ToolDetailHeaderStyle.Render("Arguments"))
		args := truncate(detail.Args, max(50, width-10))
		sb.WriteString(fmt.Sprintf("%s\n", ToolDetailOutputStyle.Render(args)))
		sb.WriteString("\n")
	}

	// Output
	if detail.Result != "" {
		sb.WriteString(ToolDetailHeaderStyle.Render("Output"))
		lines := strings.Split(detail.Result, "\n")
		maxLines := min(15, len(lines))
		for i := 0; i < maxLines; i++ {
			line := truncate(lines[i], max(50, width-10))
			sb.WriteString(fmt.Sprintf("%s\n", ToolDetailOutputStyle.Render(line)))
		}
		if len(lines) > maxLines {
			sb.WriteString(fmt.Sprintf("%s\n", ToolDetailOutputStyle.Render(
				fmt.Sprintf("... (%d more lines)", len(lines)-maxLines))))
		}
		sb.WriteString("\n")
	}

	// Finding (if any)
	if detail.FindingType != "" {
		sb.WriteString(ToolDetailHeaderStyle.Render("Finding"))
		sb.WriteString(fmt.Sprintf("%s %s\n",
			ToolDetailLabelStyle.Render("Type:"),
			ToolDetailValueStyle.Render(detail.FindingType)))
		if detail.Description != "" {
			desc := truncate(detail.Description, max(50, width-10))
			sb.WriteString(fmt.Sprintf("%s\n", ToolDetailOutputStyle.Render(desc)))
		}
	}

	return sb.String()
}

// updateToolDetail updates the active tool detail from events
func (m *Model) updateToolDetail(tool, args string, startTime time.Time) {
	m.activeToolDetail = &ToolDetail{
		Tool:      tool,
		Args:      args,
		StartedAt: startTime,
		Success:   true,
	}
}

// completeToolDetail marks the current tool detail as complete
func (m *Model) completeToolDetail(result string, duration time.Duration, success bool) {
	if m.activeToolDetail == nil {
		return
	}

	m.activeToolDetail.Result = result
	m.activeToolDetail.Duration = duration
	m.activeToolDetail.Success = success
	m.activeToolDetail.CompletedAt = time.Now()
}

// renderToolDetailSummary renders a compact summary of the current tool
func (m Model) renderToolDetailSummary() string {
	if m.activeToolDetail == nil {
		return ""
	}

	detail := m.activeToolDetail
	status := "✓"
	if !detail.Success {
		status = "✗"
	}

	return fmt.Sprintf("%s %s %s",
		status,
		detail.Tool,
		ToolDetailValueStyle.Render(truncate(detail.Args, 30)))
}
