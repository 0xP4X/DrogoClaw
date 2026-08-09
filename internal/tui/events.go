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

	case agent.EvPlan:
		m.lastPlan = ev.Plan
		m.phase = "planning"
		if ev.Plan != nil && len(ev.Plan.Steps) > 0 {
			m.phaseDetail = fmt.Sprintf("plan: %d steps", len(ev.Plan.Steps))
		} else {
			m.phaseDetail = "planning"
		}

	case agent.EvStatus:
		m.lastStatus = ev.Content
		m.phase = phaseFromStatus(ev.Content, m.phase)
		m.phaseDetail = ev.Content

	case agent.EvApproval:
		est := ev.Content
		if est == "" {
			est = "a while"
		}
		m.pendingApprovalTool = ev.Tool
		m.pendingApprovalEst = est
		m.appendLine(WarningStyle.Render(fmt.Sprintf("  [approval] %s may take ~%s. Type y/n or Enter to run:", ev.Tool, est)))
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

		if ev.Args != "" {
			m.appendLine(fmt.Sprintf("  $ %s %s", ev.Tool, ev.Args))
		} else {
			m.appendLine(fmt.Sprintf("  $ %s", ev.Tool))
		}
		m.activeToolLine = len(m.lines) - 1

	case agent.EvToolDone:
		m.activeToolName = ""
		m.lastToolResult = summarizeResult(ev.Result, 220)
		m.phase = "verifying"
		m.phaseDetail = ev.Tool
		elapsed := time.Since(m.toolStartTime)

		isError := strings.Contains(strings.ToLower(ev.Result), "error") ||
			strings.Contains(strings.ToLower(ev.Result), "failed") ||
			strings.Contains(strings.ToLower(ev.Result), "exit status 127")

		if isError {
			m.appendLine(ErrorStyle.Render(fmt.Sprintf("  [exit %s] %s", elapsed.Round(10*time.Millisecond), ev.Tool)))
		} else {
			m.appendLine(ToolDoneStyle.Render(fmt.Sprintf("  [done] %s  (%s)", ev.Tool, elapsed.Round(10*time.Millisecond))))
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
		m.appendLine(ErrorStyle.Render(fmt.Sprintf("  [error] %s", ev.Content)))
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

// colorizeOutputLine applies minimal styling so tool output stays readable
// without overwhelming the terminal. Errors and warnings get a color; everything
// else is plain.
func colorizeOutputLine(raw string) string {
	inner := strings.TrimSpace(raw)
	lower := strings.ToLower(inner)
	switch {
	case strings.Contains(lower, "error") || strings.Contains(lower, "failed") ||
		strings.Contains(lower, "denied") || strings.Contains(lower, "refused") ||
		strings.Contains(lower, "not found") || strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "exit status"):
		return ErrorStyle.Render(raw)
	case strings.Contains(lower, "warning") || strings.Contains(lower, "warn"):
		return WarningStyle.Render(raw)
	case strings.Contains(lower, "success") || strings.Contains(lower, "complete") ||
		strings.Contains(lower, "found") || strings.Contains(lower, "open"):
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
