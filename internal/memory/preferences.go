package memory

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// OperatorCorrection records when the operator overrode the AI.
type OperatorCorrection struct {
	ID        string
	Context   string // what the AI was trying to do
	Action    string // what was blocked (tool name / command)
	Reason    string // "rejected" or "redirected: <new direction>"
	Timestamp time.Time
}

// PreferenceStore records operator corrections for learning.
type PreferenceStore struct {
	mu          sync.RWMutex
	corrections []OperatorCorrection
	counter     int
}

var GlobalPreferences = &PreferenceStore{}

// RecordRejection stores a HitL rejection.
func (p *PreferenceStore) RecordRejection(context, action string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.counter++
	p.corrections = append(p.corrections, OperatorCorrection{
		ID:        fmt.Sprintf("pref_%d", p.counter),
		Context:   context,
		Action:    action,
		Reason:    "rejected by operator",
		Timestamp: time.Now(),
	})
}

// RecordRedirection stores a case where the operator changed the AI's direction.
func (p *PreferenceStore) RecordRedirection(context, action, newDirection string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.counter++
	p.corrections = append(p.corrections, OperatorCorrection{
		ID:        fmt.Sprintf("pref_%d", p.counter),
		Context:   context,
		Action:    action,
		Reason:    "redirected: " + newDirection,
		Timestamp: time.Now(),
	})
}

// BuildPreferenceBlock returns operator preferences as an injected system prompt block.
func (p *PreferenceStore) BuildPreferenceBlock() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.corrections) == 0 {
		return ""
	}

	// Only use last 10 corrections to avoid prompt bloat
	corrections := p.corrections
	if len(corrections) > 10 {
		corrections = corrections[len(corrections)-10:]
	}

	var sb strings.Builder
	sb.WriteString("\n\n--- OPERATOR PREFERENCES (learned from past sessions) ---\n")
	sb.WriteString("These are things the operator has previously rejected or redirected. Respect them:\n")
	for _, c := range corrections {
		sb.WriteString(fmt.Sprintf("• [%s] %s → %s\n", c.Action, c.Context, c.Reason))
	}
	return sb.String()
}

// All returns all recorded corrections.
func (p *PreferenceStore) All() []OperatorCorrection {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]OperatorCorrection, len(p.corrections))
	copy(out, p.corrections)
	return out
}
