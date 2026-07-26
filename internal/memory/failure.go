package memory

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// FailedAttempt records a failed exploitation attempt.
type FailedAttempt struct {
	ID          string
	Command     string
	ErrorOutput string
	Tool        string
	Target      string
	Timestamp   time.Time
}

// FailureMemory tracks all failed attempts to prevent infinite retry loops.
type FailureMemory struct {
	mu       sync.RWMutex
	failures []FailedAttempt
	counter  int
}

var GlobalFailures = &FailureMemory{}

// Record stores a new failed attempt.
func (f *FailureMemory) Record(tool, command, errorOutput, target string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.counter++
	id := fmt.Sprintf("fail_%d", f.counter)
	f.failures = append(f.failures, FailedAttempt{
		ID:          id,
		Command:     command,
		ErrorOutput: truncate(errorOutput, 500),
		Tool:        tool,
		Target:      target,
		Timestamp:   time.Now(),
	})
	return id
}

// HasFailed checks if a similar command has already been tried against the same target.
func (f *FailureMemory) HasFailed(command, target string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	cmd := normalize(command)
	for _, fa := range f.failures {
		if normalize(fa.Command) == cmd && fa.Target == target {
			return true
		}
	}
	return false
}

// SummaryFor returns a human-readable summary of all failures for a given target.
func (f *FailureMemory) SummaryFor(target string) string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var relevant []FailedAttempt
	for _, fa := range f.failures {
		if target == "" || fa.Target == target {
			relevant = append(relevant, fa)
		}
	}

	if len(relevant) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⚠ Previous failed attempts (%d):\n", len(relevant)))
	for _, fa := range relevant {
		sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", fa.Tool, fa.Target, truncate(fa.Command, 80)))
	}
	return sb.String()
}

// All returns all recorded failures.
func (f *FailureMemory) All() []FailedAttempt {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]FailedAttempt, len(f.failures))
	copy(out, f.failures)
	return out
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
