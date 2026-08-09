package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/agent"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleAgentEvent(ev agent.Event) []tea.Cmd {
	var cmds []tea.Cmd

	switch ev.Type {
	case agent.EvThinking:
		m.lastStatus = ev.Content
		m.phase = phaseFromStatus(ev.Content, "reasoning")
		m.phaseDetail = ev.Content
		m.lastStatus = "thinking..."
		m.appendLine(ThinkingLineStyle.Render(fmt.Sprintf("  ⟳ %s", ev.Content)))

	case agent.EvPlan:
		m.lastPlan = ev.Plan
		m.phase = "planning"
		if ev.Plan != nil && len(ev.Plan.Steps) > 0 {
			m.phaseDetail = fmt.Sprintf("plan: %d steps", len(ev.Plan.Steps))
			m.appendLine(StatusLineStyle.Render(fmt.Sprintf("  📋 Mission plan (%d steps):", len(ev.Plan.Steps))))
			for i, step := range ev.Plan.Steps {
				target := step.TargetAssetID
				if target == "" {
					target = "—"
				}
				m.appendLine(ToolOutputStyle.Render(fmt.Sprintf("    %d. %s  →  %s", i+1, step.Action, target)))
			}
			m.updateViewportContent()
		} else {
			m.phaseDetail = "planning"
		}

	case agent.EvStatus:
		m.lastStatus = ev.Content
		m.phase = phaseFromStatus(ev.Content, m.phase)
		m.phaseDetail = ev.Content
		// Render a clean, dim status line so the operator can follow progress
		// without it being confused with tool output or final answers.
		m.appendLine(StatusLineStyle.Render(fmt.Sprintf("  » %s", ev.Content)))

	case agent.EvApproval:
		est := ev.Content
		if est == "" {
			est = "a while"
		}
		m.pendingApprovalTool = ev.Tool
		m.pendingApprovalEst = est
		banner := fmt.Sprintf("  ⏱ %s  may take ~%s", ev.Tool, ApprovalClockStyle.Render(est))
		m.appendLine(ApprovalBoxStyle.Render(banner))
		m.appendLine(ApprovalHintStyle.Render(fmt.Sprintf("    Approve running this tool? [y/n]  (Enter = run, n/skip = decline)")))
		m.phase = "waiting"
		m.phaseDetail = fmt.Sprintf("approval: %s", ev.Tool)

	case agent.EvToolStart:
		if m.currentResponse != "" {
			for _, line := range strings.Split(m.renderAgentResponseString(m.currentResponse), "\n") {
				m.lines = append(m.lines, line)
			}
			m.currentResponse = ""
		}

		m.activeToolName = ev.Tool
		m.lastTool = ev.Tool
		m.phase = "executing"
		m.phaseDetail = ev.Tool
		m.toolStartTime = time.Now()
		badge, badgeStyle := toolCategory(ev.Tool)

		m.appendLine(fmt.Sprintf("  ⚡ %s  %s", ToolStartStyle.Render(ev.Tool), badgeStyle.Render(" "+badge+" ")))
		m.activeToolLine = len(m.lines) - 1

	case agent.EvToolDone:
		m.activeToolName = ""
		m.lastToolResult = summarizeResult(ev.Result, 220)
		m.phase = "verifying"
		m.phaseDetail = ev.Tool
		elapsed := time.Since(m.toolStartTime)
		badge, badgeStyle := toolCategory(ev.Tool)
		isError := strings.Contains(strings.ToLower(ev.Result), "error") ||
			strings.Contains(strings.ToLower(ev.Result), "failed") ||
			strings.Contains(strings.ToLower(ev.Result), "exit status 127")

		var statusIcon string
		if isError {
			statusIcon = ToolOutputErrorStyle.Render("✖")
		} else {
			statusIcon = ToolOutputSuccessStyle.Render("✔")
		}

		statusLine := fmt.Sprintf("  %s %s  %s  %s", statusIcon, ToolStartStyle.Render(ev.Tool), badgeStyle.Render(" "+badge+" "), ToolTimingStyle.Render(elapsed.Round(10*time.Millisecond).String()))

		if m.activeToolLine >= 0 && m.activeToolLine < len(m.lines) {
			m.lines[m.activeToolLine] = statusLine
		} else {
			m.lines = append(m.lines, statusLine)
		}

		outputLines := strings.Split(ev.Result, "\n")
		for i, line := range outputLines {
			prefix := "    "
			if i < len(outputLines)-1 || strings.TrimSpace(line) != "" {
				prefix = "    │ "
			}
			m.lines = append(m.lines, colorizeOutputLine(truncateLine(prefix+line)))
		}

		m.activeToolLine = -1
		m.updateViewportContent()

	case agent.EvToken:
		m.currentResponse += ev.Content
		m.updateViewportContent()

	case agent.EvDone:
		if m.currentResponse != "" {
			for _, line := range strings.Split(m.renderAgentResponseString(m.currentResponse), "\n") {
				m.lines = append(m.lines, line)
			}
			m.currentResponse = ""
		} else if ev.Content != "" {
			for _, line := range strings.Split(m.renderAgentResponseString(ev.Content), "\n") {
				m.lines = append(m.lines, line)
			}
		}
		m.executing = false
		m.phase = "complete"
		m.phaseDetail = summarizeResult(ev.Content, 120)
		m.lastPlan = nil
		m.cancelFn = nil
		cmds = append(cmds, textarea.Blink)
		m.updateViewportContent()
		m.processQueue(&cmds)
		return cmds

	case agent.EvError:
		m.activeToolName = ""
		m.activeToolLine = -1
		m.phase = "error"
		m.phaseDetail = ev.Content
		m.lastPlan = nil
		m.appendLine(ErrorStyle.Render(fmt.Sprintf("  ✗ Execution Error: %s", ev.Content)))
		m.executing = false
		m.cancelFn = nil
		m.processQueue(&cmds)
		return cmds
	}

	cmds = append(cmds, waitForEvent(m.eventsCh()))
	return cmds
}

// processQueue launches the next queued prompt (if any) once the current task
// finishes, so the operator can stack objectives without waiting around.
func (m *Model) processQueue(cmds *[]tea.Cmd) {
	if len(m.promptQueue) == 0 {
		return
	}
	next := m.promptQueue[0]
	m.promptQueue = m.promptQueue[1:]
	m.appendLine(QueueStyle.Render(fmt.Sprintf("  ▶ Running queued task (%d remaining): %s", len(m.promptQueue), next)))
	mm, cmd := m.handleInput(next)
	*m = mm
	if cmd != nil {
		*cmds = append(*cmds, cmd)
	}
}

// colorizeOutputLine shades a tool-output line by severity so errors, warnings,
// and successes stand out from routine scan output. The leading tree prefix is
// preserved; the severity is judged from the inner (un-prefixed) content.
func colorizeOutputLine(raw string) string {
	inner := strings.TrimSpace(raw)
	switch classifyOutputLine(inner) {
	case SeverityError:
		return ErrorStyle.Render(raw)
	case SeverityWarning:
		return WarningStyle.Render(raw)
	case SeveritySuccess:
		return ToolOutputSuccessStyle.Render(raw)
	case SeveritySignal:
		return InfoStyle.Render(raw)
	}
	lower := strings.ToLower(inner)
	switch {
	case strings.Contains(lower, "error") || strings.Contains(lower, "fail") ||
		strings.Contains(lower, "denied") || strings.Contains(lower, "refused") ||
		strings.Contains(lower, "not found") || strings.Contains(lower, "timeout"):
		return ToolOutputErrorStyle.Render(raw)
	case strings.Contains(lower, "open") || strings.Contains(lower, "found") ||
		strings.Contains(lower, "success") || strings.Contains(lower, "complete") ||
		strings.Contains(lower, "✔") || strings.Contains(lower, "✓"):
		return ToolOutputSuccessStyle.Render(raw)
	default:
		return ToolOutputStyle.Render(raw)
	}
}

func waitForEvent(events <-chan agent.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			return AgentEventMsg{Event: agent.Event{Type: agent.EvDone}}
		}
		return AgentEventMsg{Event: ev}
	}
}
