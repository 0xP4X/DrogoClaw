package tui

import "github.com/0xP4X/drogonclaw-go/internal/agent"

// All Bubbletea messages used in the TUI.

// AgentEventMsg wraps an agent.Event received from the execution goroutine.
type AgentEventMsg struct {
	Event agent.Event
}

// CommandSubmittedMsg is sent when the user presses Enter.
type CommandSubmittedMsg struct {
	Input string
}

// WindowResizedMsg is sent on terminal resize (alias for tea.WindowSizeMsg).
type WindowResizedMsg struct {
	Width, Height int
}

// SlashCommandMsg is triggered when user types a /command.
type SlashCommandMsg struct {
	Command string
	Args    string
}

// StatusUpdateMsg forces a re-render of the status bar.
type StatusUpdateMsg struct{}

// ClearScreenMsg clears the output pane.
type ClearScreenMsg struct{}

// NewSessionMsg wipes agent memory.
type NewSessionMsg struct{}

// HealthResultMsg contains diagnostic output.
type HealthResultMsg struct {
	Output string
}

// SandboxToggleResultMsg reports the result of switching execution runtimes.
type SandboxToggleResultMsg struct {
	Enabled bool
	Err     error
}
