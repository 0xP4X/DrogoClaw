package router

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// LocalRouter implements intelligent routing using local rules and heuristics
type LocalRouter struct {
	rules     []RoutingRule
	stats     *RoutingStats
	providers map[string]*ProviderInfo
	mu        sync.RWMutex
}

// ProviderInfo tracks provider availability and performance
type ProviderInfo struct {
	Name          string
	Available     bool
	LastChecked   time.Time
	AvgLatency    time.Duration
	SuccessRate   float64
	TotalRequests int
}

// NewLocalRouter creates a new local intelligent router
func NewLocalRouter(rules []RoutingRule) *LocalRouter {
	if len(rules) == 0 {
		rules = DefaultRules
	}

	return &LocalRouter{
		rules: rules,
		stats: &RoutingStats{
			ProviderCounts: make(map[string]int),
			TaskTypeCounts: make(map[TaskType]int),
		},
		providers: map[string]*ProviderInfo{
			"openrouter": {Name: "openrouter", Available: true, SuccessRate: 0.95},
			"openai":     {Name: "openai", Available: true, SuccessRate: 0.98},
			"nvidia":     {Name: "nvidia", Available: true, SuccessRate: 0.92},
			"gemini":     {Name: "gemini", Available: true, SuccessRate: 0.94},
			"ollama":     {Name: "ollama", Available: true, SuccessRate: 0.90},
		},
	}
}

// Route implements the Router interface
func (r *LocalRouter) Route(ctx context.Context, taskType TaskType, prompt string) (*RouteDecision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stats.TotalRequests++
	r.stats.TaskTypeCounts[taskType]++

	// Get routing rule for this task type
	rule := r.getRuleForTask(taskType)

	// Analyze prompt for context-specific routing hints
	promptAnalysis := r.analyzePrompt(prompt)

	// Try preferred providers in order
	for _, preferred := range rule.Preferred {
		provider, model := r.parseProviderModel(preferred)

		if r.isProviderAvailable(provider) && r.meetsConstraints(provider, rule) {
			decision := &RouteDecision{
				Provider:      provider,
				Model:         model,
				EstimatedCost: r.estimateCost(provider, model, len(prompt)),
				Reasoning:     r.buildReasoning(taskType, provider, model, promptAnalysis),
				Timestamp:     time.Now(),
			}

			r.stats.SuccessfulRoutes++
			r.stats.ProviderCounts[provider]++

			return decision, nil
		}
	}

	// Fallback to default provider
	r.stats.FailedRoutes++
	return &RouteDecision{
		Provider:      "openrouter",
		Model:         "meta-llama/llama-3.1-70b-instruct",
		EstimatedCost: 0.30,
		Reasoning:     "Fallback to default provider (all preferred providers unavailable)",
		Timestamp:     time.Now(),
	}, nil
}

// GetStats returns routing statistics
func (r *LocalRouter) GetStats() *RoutingStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to avoid race conditions
	statsCopy := *r.stats
	statsCopy.ProviderCounts = make(map[string]int)
	statsCopy.TaskTypeCounts = make(map[TaskType]int)

	for k, v := range r.stats.ProviderCounts {
		statsCopy.ProviderCounts[k] = v
	}
	for k, v := range r.stats.TaskTypeCounts {
		statsCopy.TaskTypeCounts[k] = v
	}

	return &statsCopy
}

// IsAvailable checks if the router is available
func (r *LocalRouter) IsAvailable() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Router is available if at least one provider is available
	for _, info := range r.providers {
		if info.Available {
			return true
		}
	}
	return false
}

// getRuleForTask finds the rule for a given task type
func (r *LocalRouter) getRuleForTask(taskType TaskType) RoutingRule {
	for _, rule := range r.rules {
		if rule.TaskType == taskType {
			return rule
		}
	}
	return GetRuleForTaskType(taskType)
}

// parseProviderModel splits "provider/model" into components
func (r *LocalRouter) parseProviderModel(preferred string) (provider, model string) {
	parts := strings.SplitN(preferred, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

// isProviderAvailable checks if a provider is currently available
func (r *LocalRouter) isProviderAvailable(provider string) bool {
	info, exists := r.providers[provider]
	if !exists {
		return false
	}
	return info.Available
}

// meetsConstraints checks if a provider meets the routing rule constraints
func (r *LocalRouter) meetsConstraints(provider string, rule RoutingRule) bool {
	info, exists := r.providers[provider]
	if !exists {
		return false
	}

	// Check success rate meets minimum quality
	if info.SuccessRate < rule.MinQuality {
		return false
	}

	// Check latency constraints
	if rule.MaxLatency > 0 && info.AvgLatency > rule.MaxLatency {
		return false
	}

	return true
}

// estimateCost estimates the cost of a request based on prompt length
func (r *LocalRouter) estimateCost(provider, model string, promptLen int) float64 {
	// Rough token estimation: ~4 characters per token
	estimatedTokens := float64(promptLen) / 4.0

	// Cost per 1M tokens (approximate values)
	costPer1M := map[string]float64{
		"openrouter/qwen-2.5-7b-instruct":              0.05,
		"openrouter/llama-3.1-8b-instruct":             0.10,
		"openrouter/llama-3.1-70b-instruct":            0.30,
		"openrouter/anthropic/claude-3.5-sonnet":       3.00,
		"gemini/gemini-2.0-flash-exp":                  0.075,
		"openai/gpt-4o-mini":                           0.15,
		"openai/gpt-4o":                                5.00,
		"nvidia/meta/llama-3.1-70b-instruct":           0.35,
	}

	key := provider + "/" + model
	costRate, exists := costPer1M[key]
	if !exists {
		costRate = 1.0 // Default fallback
	}

	return (estimatedTokens / 1_000_000) * costRate
}

// analyzePrompt extracts context hints from the prompt
func (r *LocalRouter) analyzePrompt(prompt string) map[string]bool {
	lower := strings.ToLower(prompt)

	return map[string]bool{
		"urgent":    strings.Contains(lower, "urgent") || strings.Contains(lower, "asap"),
		"complex":   strings.Contains(lower, "complex") || strings.Contains(lower, "sophisticated"),
		"creative":  strings.Contains(lower, "creative") || strings.Contains(lower, "innovative"),
		"technical": strings.Contains(lower, "technical") || strings.Contains(lower, "code"),
		"simple":    strings.Contains(lower, "simple") || strings.Contains(lower, "quick"),
	}
}

// buildReasoning constructs a human-readable reasoning for the routing decision
func (r *LocalRouter) buildReasoning(taskType TaskType, provider, model string, analysis map[string]bool) string {
	var reasons []string

	reasons = append(reasons, fmt.Sprintf("Task type: %s", taskType.String()))
	reasons = append(reasons, fmt.Sprintf("Selected: %s/%s", provider, model))

	if analysis["urgent"] {
		reasons = append(reasons, "Low latency required")
	}
	if analysis["complex"] {
		reasons = append(reasons, "High quality model needed")
	}
	if analysis["simple"] {
		reasons = append(reasons, "Cost optimized for simple task")
	}

	return strings.Join(reasons, " | ")
}

// UpdateProviderStatus updates the availability status of a provider
func (r *LocalRouter) UpdateProviderStatus(provider string, available bool, latency time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.providers[provider]
	if !exists {
		info = &ProviderInfo{Name: provider}
		r.providers[provider] = info
	}

	info.Available = available
	info.LastChecked = time.Now()
	if latency > 0 {
		// Simple moving average
		if info.AvgLatency == 0 {
			info.AvgLatency = latency
		} else {
			info.AvgLatency = (info.AvgLatency + latency) / 2
		}
	}
}

// RecordSuccess records a successful request
func (r *LocalRouter) RecordSuccess(provider string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.providers[provider]
	if !exists {
		return
	}

	// Derive prior successes from rate, rounding to nearest integer to avoid truncation bias.
	prevTotal := info.TotalRequests
	successesBefore := int(math.Round(info.SuccessRate * float64(prevTotal)))
	info.TotalRequests = prevTotal + 1
	info.SuccessRate = float64(successesBefore+1) / float64(info.TotalRequests)
}

// RecordFailure records a failed request
func (r *LocalRouter) RecordFailure(provider string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.providers[provider]
	if !exists {
		return
	}

	prevTotal := info.TotalRequests
	successesBefore := int(math.Round(info.SuccessRate * float64(prevTotal)))
	info.TotalRequests = prevTotal + 1
	info.SuccessRate = float64(successesBefore) / float64(info.TotalRequests)
}
