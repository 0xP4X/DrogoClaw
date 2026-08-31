package router

import (
	"context"
	"time"
)

// TaskType represents different types of AI tasks with different resource requirements
type TaskType int

const (
	TaskChat TaskType = iota       // Simple chat/queries - fast, cheap models
	TaskRecon                       // Reconnaissance - fast, cheap models
	TaskPlanning                    // Mission planning - medium models
	TaskExploitation               // Exploitation - premium models
	TaskReporting                  // Report generation - writing-focused models
	TaskAnalysis                   // Deep analysis - high-quality models
)

// String returns the string representation of TaskType
func (t TaskType) String() string {
	switch t {
	case TaskChat:
		return "chat"
	case TaskRecon:
		return "recon"
	case TaskPlanning:
		return "planning"
	case TaskExploitation:
		return "exploitation"
	case TaskReporting:
		return "reporting"
	case TaskAnalysis:
		return "analysis"
	default:
		return "unknown"
	}
}

// RouteDecision represents a routing decision made by the router
type RouteDecision struct {
	Provider      string
	Model         string
	EstimatedCost float64
	Reasoning     string
	Timestamp     time.Time
}

// Router is the interface for intelligent model routing
type Router interface {
	// Route determines the best provider and model for a given task
	Route(ctx context.Context, taskType TaskType, prompt string) (*RouteDecision, error)

	// GetStats returns routing statistics
	GetStats() *RoutingStats

	// IsAvailable checks if the router is available
	IsAvailable() bool
}

// RoutingStats tracks routing metrics
type RoutingStats struct {
	TotalRequests   int
	SuccessfulRoutes int
	FailedRoutes    int
	TotalCostSaved  float64
	ProviderCounts  map[string]int
	TaskTypeCounts  map[TaskType]int
}

// RoutingRule defines constraints and preferences for routing
type RoutingRule struct {
	TaskType    TaskType
	MaxCost     float64       // Maximum cost per 1M tokens
	MaxLatency  time.Duration // Maximum acceptable latency
	Preferred   []string      // Preferred providers in priority order
	MinQuality  float64       // Minimum quality score (0-1)
}

// DefaultRules provides sensible defaults for different task types
var DefaultRules = []RoutingRule{
	{
		TaskType:   TaskChat,
		MaxCost:    0.10,
		MaxLatency: 2 * time.Second,
		Preferred:  []string{"openrouter/qwen-2.5-7b-instruct", "gemini/gemini-2.0-flash-exp"},
		MinQuality: 0.7,
	},
	{
		TaskType:   TaskRecon,
		MaxCost:    0.15,
		MaxLatency: 5 * time.Second,
		Preferred:  []string{"openrouter/llama-3.1-8b-instruct", "gemini/gemini-2.0-flash-exp"},
		MinQuality: 0.75,
	},
	{
		TaskType:   TaskPlanning,
		MaxCost:    3.0,
		MaxLatency: 10 * time.Second,
		Preferred:  []string{"openrouter/llama-3.1-70b-instruct", "openai/gpt-4o-mini"},
		MinQuality: 0.85,
	},
	{
		TaskType:   TaskExploitation,
		MaxCost:    15.0,
		MaxLatency: 30 * time.Second,
		Preferred:  []string{"openai/gpt-4o", "openrouter/anthropic/claude-3.5-sonnet"},
		MinQuality: 0.90,
	},
	{
		TaskType:   TaskReporting,
		MaxCost:    5.0,
		MaxLatency: 20 * time.Second,
		Preferred:  []string{"openrouter/anthropic/claude-3.5-sonnet", "openai/gpt-4o"},
		MinQuality: 0.88,
	},
	{
		TaskType:   TaskAnalysis,
		MaxCost:    10.0,
		MaxLatency: 30 * time.Second,
		Preferred:  []string{"openai/gpt-4o", "openrouter/anthropic/claude-3.5-sonnet"},
		MinQuality: 0.90,
	},
}

// GetRuleForTaskType returns the routing rule for a given task type
func GetRuleForTaskType(taskType TaskType) RoutingRule {
	for _, rule := range DefaultRules {
		if rule.TaskType == taskType {
			return rule
		}
	}
	// Fallback to medium-tier defaults
	return RoutingRule{
		TaskType:   taskType,
		MaxCost:    3.0,
		MaxLatency: 10 * time.Second,
		Preferred:  []string{"openrouter/llama-3.1-70b-instruct"},
		MinQuality: 0.8,
	}
}
