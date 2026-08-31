package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// NineRouterClient implements the Router interface using the 9router.ai service
type NineRouterClient struct {
	apiKey    string
	baseURL   string
	client    *http.Client
	stats     *RoutingStats
	mu        sync.RWMutex
	available bool
}

// RouteRequest represents a request to the 9router.ai API
type RouteRequest struct {
	TaskType         string   `json:"task_type"`
	Prompt           string   `json:"prompt,omitempty"`
	MaxCost          float64  `json:"max_cost,omitempty"`
	MaxLatencyMs     int      `json:"max_latency_ms,omitempty"`
	AllowedProviders []string `json:"allowed_providers,omitempty"`
	RequireQuality   float64  `json:"require_quality,omitempty"`
}

// RouteResponse represents a response from the 9router.ai API
type RouteResponse struct {
	Provider      string  `json:"provider"`
	Model         string  `json:"model"`
	EstimatedCost float64 `json:"estimated_cost"`
	Reasoning     string  `json:"reasoning"`
	Confidence    float64 `json:"confidence,omitempty"`
}

// NewNineRouterClient creates a new 9router.ai client
func NewNineRouterClient(apiKey string) *NineRouterClient {
	return &NineRouterClient{
		apiKey:  apiKey,
		baseURL: "https://api.9router.ai/v1",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		stats: &RoutingStats{
			ProviderCounts: make(map[string]int),
			TaskTypeCounts: make(map[TaskType]int),
		},
		available: true,
	}
}

// Route implements the Router interface
func (c *NineRouterClient) Route(ctx context.Context, taskType TaskType, prompt string) (*RouteDecision, error) {
	c.mu.Lock()
	c.stats.TotalRequests++
	c.stats.TaskTypeCounts[taskType]++
	c.mu.Unlock()

	// Get routing rule for constraints
	rule := GetRuleForTaskType(taskType)

	// Build request
	req := RouteRequest{
		TaskType:       taskType.String(),
		Prompt:         truncatePrompt(prompt, 500), // Send first 500 chars for context
		MaxCost:        rule.MaxCost,
		MaxLatencyMs:   int(rule.MaxLatency.Milliseconds()),
		RequireQuality: rule.MinQuality,
	}

	// Make API request
	resp, err := c.makeRequest(ctx, req)
	if err != nil {
		c.mu.Lock()
		c.stats.FailedRoutes++
		c.available = false
		c.mu.Unlock()

		// Fallback to local routing
		return c.fallbackRoute(taskType, rule), nil
	}

	c.mu.Lock()
	c.stats.SuccessfulRoutes++
	c.stats.ProviderCounts[resp.Provider]++
	c.available = true
	c.mu.Unlock()

	return &RouteDecision{
		Provider:      resp.Provider,
		Model:         resp.Model,
		EstimatedCost: resp.EstimatedCost,
		Reasoning:     resp.Reasoning,
		Timestamp:     time.Now(),
	}, nil
}

// GetStats returns routing statistics
func (c *NineRouterClient) GetStats() *RoutingStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to avoid race conditions
	statsCopy := *c.stats
	statsCopy.ProviderCounts = make(map[string]int)
	statsCopy.TaskTypeCounts = make(map[TaskType]int)

	for k, v := range c.stats.ProviderCounts {
		statsCopy.ProviderCounts[k] = v
	}
	for k, v := range c.stats.TaskTypeCounts {
		statsCopy.TaskTypeCounts[k] = v
	}

	return &statsCopy
}

// IsAvailable checks if the 9router.ai service is available
func (c *NineRouterClient) IsAvailable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.available
}

// makeRequest sends a request to the 9router.ai API
func (c *NineRouterClient) makeRequest(ctx context.Context, req RouteRequest) (*RouteResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/route", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "DrogonClaw/1.0")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("routing failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var result RouteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// fallbackRoute provides a local fallback when 9router.ai is unavailable
func (c *NineRouterClient) fallbackRoute(taskType TaskType, rule RoutingRule) *RouteDecision {
	// Use first preferred provider from rule
	provider := "openrouter"
	model := "meta-llama/llama-3.1-70b-instruct"

	if len(rule.Preferred) > 0 {
		parts := splitProviderModel(rule.Preferred[0])
		if len(parts) == 2 {
			provider = parts[0]
			model = parts[1]
		}
	}

	return &RouteDecision{
		Provider:      provider,
		Model:         model,
		EstimatedCost: rule.MaxCost * 0.5, // Estimate at 50% of max
		Reasoning:     fmt.Sprintf("9router.ai unavailable, using fallback for %s task", taskType.String()),
		Timestamp:     time.Now(),
	}
}

// truncatePrompt truncates prompt to maxLen runes for API efficiency (rune-safe).
func truncatePrompt(prompt string, maxLen int) string {
	runes := []rune(prompt)
	if len(runes) <= maxLen {
		return prompt
	}
	return string(runes[:maxLen]) + "..."
}

// splitProviderModel splits "provider/model" into [provider, model].
// Uses SplitN so model names containing "/" are preserved (e.g. "openrouter/anthropic/claude-3.5-sonnet").
func splitProviderModel(s string) []string {
	parts := bytes.SplitN([]byte(s), []byte("/"), 2)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		result = append(result, string(part))
	}
	return result
}

// Health checks the health of the 9router.ai service
func (c *NineRouterClient) Health(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		c.mu.Lock()
		c.available = false
		c.mu.Unlock()
		return err
	}
	defer resp.Body.Close()

	c.mu.Lock()
	c.available = (resp.StatusCode == http.StatusOK)
	c.mu.Unlock()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: %d", resp.StatusCode)
	}

	return nil
}
