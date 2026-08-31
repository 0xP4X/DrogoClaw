package router

import (
	"context"
	"testing"
	"time"
)

func TestLocalRouter_Route(t *testing.T) {
	router := NewLocalRouter(DefaultRules)

	tests := []struct {
		name         string
		taskType     TaskType
		prompt       string
		wantProvider string
		wantErr      bool
	}{
		{
			name:         "recon task routes to fast model",
			taskType:     TaskRecon,
			prompt:       "scan ports on target 192.168.1.1",
			wantProvider: "openrouter",
			wantErr:      false,
		},
		{
			name:         "exploitation task routes to premium model",
			taskType:     TaskExploitation,
			prompt:       "exploit CVE-2024-1234 on Windows Server",
			wantProvider: "openai",
			wantErr:      false,
		},
		{
			name:         "planning task routes to medium model",
			taskType:     TaskPlanning,
			prompt:       "create mission plan for web app pentest",
			wantProvider: "openrouter",
			wantErr:      false,
		},
		{
			name:         "reporting task routes to writing model",
			taskType:     TaskReporting,
			prompt:       "generate executive summary report",
			wantProvider: "openrouter",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			decision, err := router.Route(ctx, tt.taskType, tt.prompt)

			if (err != nil) != tt.wantErr {
				t.Errorf("Route() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if decision == nil {
				t.Fatal("Route() returned nil decision")
			}

			if decision.Provider != tt.wantProvider {
				t.Errorf("Route() provider = %v, want %v", decision.Provider, tt.wantProvider)
			}

			if decision.Model == "" {
				t.Error("Route() returned empty model")
			}

			if decision.EstimatedCost < 0 {
				t.Error("Route() returned negative cost")
			}

			if decision.Reasoning == "" {
				t.Error("Route() returned empty reasoning")
			}
		})
	}
}

func TestLocalRouter_GetStats(t *testing.T) {
	router := NewLocalRouter(DefaultRules)

	// Perform some routes
	ctx := context.Background()
	_, _ = router.Route(ctx, TaskRecon, "test prompt 1")
	_, _ = router.Route(ctx, TaskExploitation, "test prompt 2")
	_, _ = router.Route(ctx, TaskRecon, "test prompt 3")

	stats := router.GetStats()

	if stats.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", stats.TotalRequests)
	}

	if stats.TaskTypeCounts[TaskRecon] != 2 {
		t.Errorf("Expected 2 recon tasks, got %d", stats.TaskTypeCounts[TaskRecon])
	}

	if stats.TaskTypeCounts[TaskExploitation] != 1 {
		t.Errorf("Expected 1 exploitation task, got %d", stats.TaskTypeCounts[TaskExploitation])
	}
}

func TestLocalRouter_PromptAnalysis(t *testing.T) {
	router := NewLocalRouter(DefaultRules)

	tests := []struct {
		name         string
		prompt       string
		expectUrgent bool
		expectSimple bool
	}{
		{
			name:         "urgent prompt detected",
			prompt:       "URGENT: scan this target ASAP",
			expectUrgent: true,
			expectSimple: false,
		},
		{
			name:         "simple task detected",
			prompt:       "simple port scan on localhost",
			expectUrgent: false,
			expectSimple: true,
		},
		{
			name:         "normal prompt",
			prompt:       "perform reconnaissance on target network",
			expectUrgent: false,
			expectSimple: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := router.analyzePrompt(tt.prompt)

			if analysis["urgent"] != tt.expectUrgent {
				t.Errorf("Expected urgent=%v, got %v", tt.expectUrgent, analysis["urgent"])
			}

			if analysis["simple"] != tt.expectSimple {
				t.Errorf("Expected simple=%v, got %v", tt.expectSimple, analysis["simple"])
			}
		})
	}
}

func TestLocalRouter_ProviderStatus(t *testing.T) {
	router := NewLocalRouter(DefaultRules)

	// Update provider status
	router.UpdateProviderStatus("openrouter", true, 100*time.Millisecond)
	router.UpdateProviderStatus("openai", false, 0)

	// Verify availability
	if !router.IsAvailable() {
		t.Error("Router should be available when at least one provider is available")
	}

	// Mark all providers unavailable
	for provider := range router.providers {
		router.UpdateProviderStatus(provider, false, 0)
	}

	if router.IsAvailable() {
		t.Error("Router should not be available when all providers are unavailable")
	}
}

func TestLocalRouter_RecordMetrics(t *testing.T) {
	router := NewLocalRouter(DefaultRules)

	// Record some successes and failures
	router.RecordSuccess("openrouter")
	router.RecordSuccess("openrouter")
	router.RecordFailure("openrouter")

	info := router.providers["openrouter"]
	if info.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", info.TotalRequests)
	}

	expectedSuccessRate := 2.0 / 3.0
	if info.SuccessRate < expectedSuccessRate-0.01 || info.SuccessRate > expectedSuccessRate+0.01 {
		t.Errorf("Expected success rate ~%.2f, got %.2f", expectedSuccessRate, info.SuccessRate)
	}
}

func TestGetRuleForTaskType(t *testing.T) {
	tests := []struct {
		taskType    TaskType
		expectCost  float64
		expectQual  float64
	}{
		{TaskRecon, 0.15, 0.75},
		{TaskExploitation, 15.0, 0.90},
		{TaskReporting, 5.0, 0.88},
		{TaskPlanning, 3.0, 0.85},
	}

	for _, tt := range tests {
		t.Run(tt.taskType.String(), func(t *testing.T) {
			rule := GetRuleForTaskType(tt.taskType)

			if rule.MaxCost != tt.expectCost {
				t.Errorf("Expected max cost %.2f, got %.2f", tt.expectCost, rule.MaxCost)
			}

			if rule.MinQuality != tt.expectQual {
				t.Errorf("Expected min quality %.2f, got %.2f", tt.expectQual, rule.MinQuality)
			}
		})
	}
}

func BenchmarkLocalRouter_Route(b *testing.B) {
	router := NewLocalRouter(DefaultRules)
	ctx := context.Background()
	prompt := "scan ports on target 192.168.1.1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = router.Route(ctx, TaskRecon, prompt)
	}
}

func TestTaskType_String(t *testing.T) {
	tests := []struct {
		taskType TaskType
		want     string
	}{
		{TaskChat, "chat"},
		{TaskRecon, "recon"},
		{TaskPlanning, "planning"},
		{TaskExploitation, "exploitation"},
		{TaskReporting, "reporting"},
		{TaskAnalysis, "analysis"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.taskType.String(); got != tt.want {
				t.Errorf("TaskType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
