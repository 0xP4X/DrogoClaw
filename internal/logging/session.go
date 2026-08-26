package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SessionEntry represents a single log entry for the session
type SessionEntry struct {
	Timestamp   time.Time     `json:"timestamp"`
	Type        string        `json:"type"` // tool_start, tool_complete, finding, decision, error
	Tool        string        `json:"tool,omitempty"`
	Args        string        `json:"args,omitempty"`
	Result      string        `json:"result,omitempty"`
	Duration    string        `json:"duration,omitempty"`
	Success     *bool         `json:"success,omitempty"`
	FindingType string        `json:"finding_type,omitempty"`
	Description string        `json:"description,omitempty"`
	Source      string        `json:"source,omitempty"`
	Decision    string        `json:"decision,omitempty"`
	Reasoning   string        `json:"reasoning,omitempty"`
	Error       string        `json:"error,omitempty"`
	Phase       string        `json:"phase,omitempty"`
	Step        int           `json:"step,omitempty"`
	TotalSteps  int           `json:"total_steps,omitempty"`
}

// SessionTimeline represents the full execution timeline for a session
type SessionTimeline struct {
	SessionID string         `json:"session_id"`
	StartedAt time.Time      `json:"started_at"`
	Entries   []SessionEntry `json:"entries"`
	mu        sync.RWMutex
	file      *os.File
	path      string
}

// NewSessionLogger creates a new session logger that persists to JSONL
func NewSessionLogger(sessionID string) (*SessionTimeline, error) {
	logDir := filepath.Join("data", "sessions")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("creating session log dir: %w", err)
	}

	path := filepath.Join(logDir, fmt.Sprintf("%s.jsonl", sessionID))
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening session log: %w", err)
	}

	timeline := &SessionTimeline{
		SessionID: sessionID,
		StartedAt: time.Now(),
		Entries:   make([]SessionEntry, 0),
		file:      f,
		path:      path,
	}

	// Write initial entry
	timeline.writeEntry(SessionEntry{
		Timestamp: time.Now(),
		Type:      "session_start",
	})

	return timeline, nil
}

// LogToolStart logs the start of a tool execution
func (st *SessionTimeline) LogToolStart(tool, args string, step, totalSteps int) {
	entry := SessionEntry{
		Timestamp:  time.Now(),
		Type:       "tool_start",
		Tool:       tool,
		Args:       truncateString(args, 1000),
		Step:       step,
		TotalSteps: totalSteps,
	}
	st.writeEntry(entry)
}

// LogToolComplete logs the completion of a tool execution
func (st *SessionTimeline) LogToolComplete(tool, result, duration string, success bool) {
	entry := SessionEntry{
		Timestamp: time.Now(),
		Type:      "tool_complete",
		Tool:      tool,
		Result:    truncateString(result, 2000),
		Duration:  duration,
		Success:   &success,
	}
	st.writeEntry(entry)
}

// LogFinding logs a discovered finding
func (st *SessionTimeline) LogFinding(findingType, description, source string) {
	entry := SessionEntry{
		Timestamp:   time.Now(),
		Type:        "finding",
		FindingType: findingType,
		Description: description,
		Source:      source,
	}
	st.writeEntry(entry)
}

// LogDecision logs an agent decision
func (st *SessionTimeline) LogDecision(decision, reasoning string) {
	entry := SessionEntry{
		Timestamp: time.Now(),
		Type:      "decision",
		Decision:  decision,
		Reasoning: truncateString(reasoning, 500),
	}
	st.writeEntry(entry)
}

// LogPhase logs a phase change
func (st *SessionTimeline) LogPhase(phase string) {
	entry := SessionEntry{
		Timestamp: time.Now(),
		Type:      "phase_change",
		Phase:     phase,
	}
	st.writeEntry(entry)
}

// LogError logs an error
func (st *SessionTimeline) LogError(err string) {
	entry := SessionEntry{
		Timestamp: time.Now(),
		Type:      "error",
		Error:     err,
	}
	st.writeEntry(entry)
}

// GetEntries returns all log entries (thread-safe)
func (st *SessionTimeline) GetEntries() []SessionEntry {
	st.mu.RLock()
	defer st.mu.RUnlock()
	
	entries := make([]SessionEntry, len(st.Entries))
	copy(entries, st.Entries)
	return entries
}

// GetToolCalls returns only tool call entries
func (st *SessionTimeline) GetToolCalls() []SessionEntry {
	st.mu.RLock()
	defer st.mu.RUnlock()
	
	var toolCalls []SessionEntry
	for _, e := range st.Entries {
		if e.Type == "tool_start" || e.Type == "tool_complete" {
			toolCalls = append(toolCalls, e)
		}
	}
	return toolCalls
}

// GetFindings returns only finding entries
func (st *SessionTimeline) GetFindings() []SessionEntry {
	st.mu.RLock()
	defer st.mu.RUnlock()
	
	var findings []SessionEntry
	for _, e := range st.Entries {
		if e.Type == "finding" {
			findings = append(findings, e)
		}
	}
	return findings
}

// GetDuration returns the total session duration
func (st *SessionTimeline) GetDuration() time.Duration {
	st.mu.RLock()
	defer st.mu.RUnlock()
	
	if len(st.Entries) == 0 {
		return time.Since(st.StartedAt)
	}
	return st.Entries[len(st.Entries)-1].Timestamp.Sub(st.StartedAt)
}

// writeEntry writes an entry to both memory and file (thread-safe)
func (st *SessionTimeline) writeEntry(entry SessionEntry) {
	st.mu.Lock()
	defer st.mu.Unlock()
	
	st.Entries = append(st.Entries, entry)
	
	// Write to JSONL file
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	data = append(data, '\n')
	st.file.Write(data)
}

// Close closes the session log file
func (st *SessionTimeline) Close() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	
	// Write final entry
	entry := SessionEntry{
		Timestamp: time.Now(),
		Type:      "session_end",
	}
	data, _ := json.Marshal(entry)
	data = append(data, '\n')
	st.file.Write(data)
	
	return st.file.Close()
}

// LoadSessionTimeline loads a session timeline from a JSONL file
func LoadSessionTimeline(sessionID string) (*SessionTimeline, error) {
	path := filepath.Join("data", "sessions", fmt.Sprintf("%s.jsonl", sessionID))
	
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading session log: %w", err)
	}
	
	timeline := &SessionTimeline{
		SessionID: sessionID,
		Entries:   make([]SessionEntry, 0),
		path:      path,
	}
	
	// Parse JSONL
	decoder := json.NewDecoder(nil)
	_ = decoder
	
	// Simple line-by-line parsing
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry SessionEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		timeline.Entries = append(timeline.Entries, entry)
		if entry.Type == "session_start" {
			timeline.StartedAt = entry.Timestamp
		}
	}
	
	return timeline, nil
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
