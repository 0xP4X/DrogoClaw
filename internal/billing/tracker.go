package billing

import (
	"fmt"
	"strings"
	"sync"
)

// Usage holds raw token counts for one LLM call.
type Usage struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
}

// ModelUsage extends Usage with the model identifier and computed cost.
type ModelUsage struct {
	Model string
	Usage
	Cost float64
}

// Pricing is the per-1M-token rate for a model.
type Pricing struct {
	InputPer1M  float64
	OutputPer1M float64
}

// DefaultPricing is a small static catalog. Models not listed cost $0.00,
// which is a safe conservative default — the operator can extend it.
var DefaultPricing = map[string]Pricing{
	"gpt-4o":                     {InputPer1M: 2.50, OutputPer1M: 10.00},
	"gpt-4o-mini":                {InputPer1M: 0.15, OutputPer1M: 0.60},
	"gpt-4-turbo":                {InputPer1M: 10.00, OutputPer1M: 30.00},
	"gpt-3.5-turbo":              {InputPer1M: 0.50, OutputPer1M: 1.50},
	"claude-3-5-sonnet-20240620": {InputPer1M: 3.00, OutputPer1M: 15.00},
	"claude-3-opus-20240229":     {InputPer1M: 15.00, OutputPer1M: 75.00},
	"gemini-1.5-pro":             {InputPer1M: 1.25, OutputPer1M: 5.00},
	"gemini-1.5-flash":           {InputPer1M: 0.075, OutputPer1M: 0.30},
}

// Tracker accumulates token usage and estimated cost for a session.
type Tracker struct {
	mu      sync.Mutex
	total   Usage
	byModel map[string]*ModelUsage
	pricing map[string]Pricing
}

// New constructs a Tracker with the given pricing map (or DefaultPricing if nil).
func New(pricing map[string]Pricing) *Tracker {
	if pricing == nil {
		pricing = DefaultPricing
	}
	return &Tracker{
		byModel: make(map[string]*ModelUsage),
		pricing: pricing,
	}
}

// Record adds a usage sample for the given model.
func (t *Tracker) Record(model string, prompt, completion int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.total.PromptTokens += prompt
	t.total.CompletionTokens += completion
	t.total.TotalTokens += prompt + completion
	m, ok := t.byModel[model]
	if !ok {
		m = &ModelUsage{Model: model}
		t.byModel[model] = m
	}
	m.PromptTokens += prompt
	m.CompletionTokens += completion
	m.TotalTokens += prompt + completion
	p := t.pricing[model]
	m.Cost += (float64(prompt) / 1_000_000) * p.InputPer1M
	m.Cost += (float64(completion) / 1_000_000) * p.OutputPer1M
}

// Total returns the session-wide token counts.
func (t *Tracker) Total() Usage {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.total
}

// TotalCost returns the estimated total cost across all recorded models.
func (t *Tracker) TotalCost() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	var sum float64
	for _, m := range t.byModel {
		sum += m.Cost
	}
	return sum
}

// Breakdown returns a copy of per-model usage for rendering.
func (t *Tracker) Breakdown() map[string]ModelUsage {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[string]ModelUsage, len(t.byModel))
	for k, v := range t.byModel {
		out[k] = *v
	}
	return out
}

// Render returns a human-readable cost summary string.
func (t *Tracker) Render() string {
	total := t.Total()
	cost := t.TotalCost()
	var sb strings.Builder
	sb.WriteString("═══════════════════════════════════════════\n")
	sb.WriteString(" LLM USAGE & ESTIMATED COST\n")
	sb.WriteString("═══════════════════════════════════════════\n\n")
	fmt.Fprintf(&sb, "  Prompt tokens:     %d\n", total.PromptTokens)
	fmt.Fprintf(&sb, "  Completion tokens: %d\n", total.CompletionTokens)
	fmt.Fprintf(&sb, "  Total tokens:      %d\n", total.TotalTokens)
	fmt.Fprintf(&sb, "  Est. cost:         $%.4f\n\n", cost)

	breakdown := t.Breakdown()
	if len(breakdown) > 0 {
		sb.WriteString("  By model:\n")
		for _, m := range breakdown {
			fmt.Fprintf(&sb, "    %-30s  %d / %d tokens  $%.4f\n", m.Model, m.PromptTokens, m.CompletionTokens, m.Cost)
		}
	}
	sb.WriteString("\n═══════════════════════════════════════════\n")
	return sb.String()
}
