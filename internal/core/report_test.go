package core

import (
	"context"
	"strings"
	"testing"

	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/openai/openai-go"
)

// MockLLMProvider is a simple mock for testing report generation.
type MockLLMProvider struct {
	Response string
	Err      error
}

func (m *MockLLMProvider) CompleteText(ctx context.Context, msgs []openai.ChatCompletionMessageParamUnion) (string, error) {
	return m.Response, m.Err
}

func TestReportGenerator_EmptyGraph(t *testing.T) {
	mockProvider := &MockLLMProvider{Response: "Fake report"}
	graph := memory.NewGraph("test")
	
	generator := NewReportGenerator(mockProvider, graph)
	
	_, err := generator.GenerateMarkdownReport(context.Background())
	if err == nil {
		t.Errorf("Expected error for empty graph, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("Expected error to mention empty graph, got %v", err)
	}
}
