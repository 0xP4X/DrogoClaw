package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/logging"
	"github.com/charmbracelet/lipgloss"
)

// TimelineEntry represents a formatted timeline entry for display
type TimelineEntry struct {
	Timestamp time.Duration
	Tool      string
	Args      string
	Result    string
	Duration  string
	Success   bool
	Finding   *Finding
	Type      string // tool, finding, decision, phase
}

// FindingType constants
const (
	FindingVulnerability = "vulnerability"
	FindingCredential    = "credential"
	FindingFlag          = "flag"
	FindingInfo          = "info"
)

// Timeline styles
var (
	TimelineTimeStyle = lipgloss.NewStyle().
		Foreground(ColorDim).
		Width(8)

	TimelineToolStyle = lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true)

	TimelineArgsStyle = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true)

	TimelineSuccessStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	TimelineErrorStyle = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true)

	TimelineFindingStyle = lipgloss.NewStyle().
		Foreground(ColorGold).
		Bold(true)

	TimelineVulnStyle = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true)

	TimelineCredStyle = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)

	TimelineFlagStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	TimelineInfoStyle = lipgloss.NewStyle().
		Foreground(ColorAccent)
)

// renderTimeline renders the execution timeline in the viewport
func (m Model) renderTimeline() string {
	if m.sessionLog == nil {
		return WarningStyle.Render("  [!] No session log available. Timeline will appear here during execution.")
	}

	entries := m.sessionLog.GetEntries()
	if len(entries) == 0 {
		return InfoStyle.Render("  [i] No entries yet. Timeline will appear here during execution.")
	}

	var sb strings.Builder
	sb.WriteString(SectionHeaderStyle.Render("  EXECUTION TIMELINE"))
	sb.WriteString("\n")
	sb.WriteString(SectionRuleStyle.Render(strings.Repeat("─", max(4, m.viewport.Width-4))))
	sb.WriteString("\n\n")

	startTime := entries[0].Timestamp
	for _, entry := range entries {
		elapsed := entry.Timestamp.Sub(startTime)
		timeStr := formatDuration(elapsed)

		switch entry.Type {
		case "tool_start":
			sb.WriteString(m.renderTimelineToolStart(timeStr, entry))
		case "tool_complete":
			sb.WriteString(m.renderTimelineToolComplete(timeStr, entry))
		case "finding":
			sb.WriteString(m.renderTimelineFinding(timeStr, entry))
		case "decision":
			sb.WriteString(m.renderTimelineDecision(timeStr, entry))
		case "phase_change":
			sb.WriteString(m.renderTimelinePhase(timeStr, entry))
		case "error":
			sb.WriteString(m.renderTimelineError(timeStr, entry))
		}
	}

	return sb.String()
}

// renderTimelineToolStart renders a tool start entry
func (m Model) renderTimelineToolStart(timeStr string, entry logging.SessionEntry) string {
	var sb strings.Builder

	sb.WriteString(TimelineTimeStyle.Render(fmt.Sprintf("[%s]", timeStr)))
	sb.WriteString(" ")
	sb.WriteString(TimelineToolStyle.Render(fmt.Sprintf("▶ %s", entry.Tool)))
	sb.WriteString("\n")

	if entry.Args != "" {
		args := truncate(entry.Args, max(20, m.viewport.Width-20))
		sb.WriteString(TimelineArgsStyle.Render(fmt.Sprintf("  %s", args)))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderTimelineToolComplete renders a tool completion entry
func (m Model) renderTimelineToolComplete(timeStr string, entry logging.SessionEntry) string {
	var sb strings.Builder

	sb.WriteString(TimelineTimeStyle.Render(fmt.Sprintf("[%s]", timeStr)))
	sb.WriteString(" ")

	status := "✓"
	statusStyle := TimelineSuccessStyle
	if entry.Success != nil && !*entry.Success {
		status = "✗"
		statusStyle = TimelineErrorStyle
	}

	sb.WriteString(statusStyle.Render(fmt.Sprintf("%s %s", status, entry.Tool)))

	if entry.Duration != "" {
		sb.WriteString(TimelineTimeStyle.Render(fmt.Sprintf(" (%s)", entry.Duration)))
	}
	sb.WriteString("\n")

	if entry.Result != "" {
		result := truncate(entry.Result, max(30, m.viewport.Width-10))
		lines := strings.Split(result, "\n")
		for _, line := range lines[:min(3, len(lines))] {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
		if len(lines) > 3 {
			sb.WriteString(fmt.Sprintf("  ... (%d more lines)\n", len(lines)-3))
		}
	}

	return sb.String()
}

// renderTimelineFinding renders a finding entry
func (m Model) renderTimelineFinding(timeStr string, entry logging.SessionEntry) string {
	var sb strings.Builder

	sb.WriteString(TimelineTimeStyle.Render(fmt.Sprintf("[%s]", timeStr)))
	sb.WriteString(" ")

	var findingStyle lipgloss.Style
	var icon string
	switch entry.FindingType {
	case FindingVulnerability:
		findingStyle = TimelineVulnStyle
		icon = "🔴"
	case FindingCredential:
		findingStyle = TimelineCredStyle
		icon = "🟡"
	case FindingFlag:
		findingStyle = TimelineFlagStyle
		icon = "🟢"
	default:
		findingStyle = TimelineInfoStyle
		icon = "ℹ️"
	}

	sb.WriteString(findingStyle.Render(fmt.Sprintf("%s %s", icon, entry.FindingType)))
	sb.WriteString("\n")

	if entry.Description != "" {
		desc := truncate(entry.Description, max(30, m.viewport.Width-10))
		sb.WriteString(fmt.Sprintf("  %s\n", desc))
	}

	if entry.Source != "" {
		sb.WriteString(TimelineArgsStyle.Render(fmt.Sprintf("  Source: %s", entry.Source)))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderTimelineDecision renders a decision entry
func (m Model) renderTimelineDecision(timeStr string, entry logging.SessionEntry) string {
	var sb strings.Builder

	sb.WriteString(TimelineTimeStyle.Render(fmt.Sprintf("[%s]", timeStr)))
	sb.WriteString(" ")
	sb.WriteString(InfoStyle.Render(fmt.Sprintf("→ Decision: %s", entry.Decision)))
	sb.WriteString("\n")

	if entry.Reasoning != "" {
		reasoning := truncate(entry.Reasoning, max(30, m.viewport.Width-10))
		sb.WriteString(fmt.Sprintf("  %s\n", reasoning))
	}

	return sb.String()
}

// renderTimelinePhase renders a phase change entry
func (m Model) renderTimelinePhase(timeStr string, entry logging.SessionEntry) string {
	var sb strings.Builder

	sb.WriteString(TimelineTimeStyle.Render(fmt.Sprintf("[%s]", timeStr)))
	sb.WriteString(" ")

	phaseStyle := PhaseIdleStyle
	switch entry.Phase {
	case "planning":
		phaseStyle = PhasePlanningStyle
	case "reasoning":
		phaseStyle = PhaseReasoningStyle
	case "executing":
		phaseStyle = PhaseExecutingStyle
	case "verifying":
		phaseStyle = PhaseVerifyingStyle
	case "complete":
		phaseStyle = PhaseCompleteStyle
	case "error":
		phaseStyle = PhaseErrorStyle
	}

	sb.WriteString(phaseStyle.Render(fmt.Sprintf("◆ Phase: %s", entry.Phase)))
	sb.WriteString("\n")

	return sb.String()
}

// renderTimelineError renders an error entry
func (m Model) renderTimelineError(timeStr string, entry logging.SessionEntry) string {
	var sb strings.Builder

	sb.WriteString(TimelineTimeStyle.Render(fmt.Sprintf("[%s]", timeStr)))
	sb.WriteString(" ")
	sb.WriteString(TimelineErrorStyle.Render(fmt.Sprintf("✗ Error: %s", entry.Error)))
	sb.WriteString("\n")

	return sb.String()
}

// formatDuration formats a duration as MM:SS or HH:MM:SS
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%02ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%02dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%02dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

// renderTimelineSummary renders a compact summary of the timeline
func (m Model) renderTimelineSummary() string {
	if m.sessionLog == nil {
		return ""
	}

	entries := m.sessionLog.GetEntries()
	toolCalls := 0
	findings := 0
	errors := 0

	for _, e := range entries {
		switch e.Type {
		case "tool_start":
			toolCalls++
		case "finding":
			findings++
		case "error":
			errors++
		}
	}

	duration := m.sessionLog.GetDuration()

	var sb strings.Builder
	sb.WriteString(HeaderInfoStyle.Render(fmt.Sprintf(
		"Timeline: %d tools, %d findings, %d errors, %s elapsed",
		toolCalls, findings, errors, formatDuration(duration),
	)))

	return sb.String()
}
