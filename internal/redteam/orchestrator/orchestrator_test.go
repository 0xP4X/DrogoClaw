package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

func TestOrchestrator_StartEngagement(t *testing.T) {
	orch := New()
	eng := orch.StartEngagement("10.0.0.1")

	if eng.ID == "" {
		t.Errorf("Expected engagement ID to be generated")
	}
	if eng.Target != "10.0.0.1" {
		t.Errorf("Expected target 10.0.0.1, got %s", eng.Target)
	}
	if eng.Status != PhaseRecon {
		t.Errorf("Expected initial status to be RECONNAISSANCE, got %s", eng.Status)
	}
	
	if _, ok := orch.ActiveEngagements[eng.ID]; !ok {
		t.Errorf("Expected engagement to be stored in orchestrator")
	}
}

func TestOrchestrator_ExecuteLifecycle(t *testing.T) {
	orch := New()
	eng := orch.StartEngagement("10.0.0.1")

	// Create a context with a short timeout to prevent hanging if the loop fails to exit
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Approve the engagement after it has entered its explicit approval state.
	go func() {
		time.Sleep(500 * time.Millisecond)
		err := orch.ApproveEngagement(eng.ID)
		if err != nil {
			t.Errorf("ApproveEngagement failed: %v", err)
		}
	}()

	err := orch.Execute(ctx, eng.ID)
	if err != nil {
		t.Fatalf("Orchestrator Execute failed: %v", err)
	}

	if eng.Status != PhaseNeedsHuman {
		t.Errorf("Expected final status to be NEEDS_HUMAN, got %s", eng.Status)
	}
	if eng.Reason == "" {
		t.Error("Expected an explicit reason for the unsupported live workflow")
	}
	if counts := eng.Graph.LabelCounts(); counts[memory.LabelVulnerability] != 0 {
		t.Errorf("Expected no fabricated vulnerabilities, got %d", counts[memory.LabelVulnerability])
	}
	if eng.EndTime.IsZero() {
		t.Errorf("Expected EndTime to be set")
	}
}

func TestOrchestrator_ExecuteLifecycle_WithSandbox(t *testing.T) {
	sb, err := sandbox.New()
	if err != nil {
		t.Skipf("Sandbox unavailable, skipping: %v", err)
	}
	err = sb.Initialize(context.Background(), false)
	if err != nil {
		t.Skipf("Sandbox initialization failed, skipping: %v", err)
	}
	defer sb.Stop(context.Background())

	orch := New()
	orch.SetSandbox(sb)
	eng := orch.StartEngagement("10.0.0.1")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		time.Sleep(500 * time.Millisecond)
		err := orch.ApproveEngagement(eng.ID)
		if err != nil {
			t.Errorf("ApproveEngagement failed: %v", err)
		}
	}()

	err = orch.Execute(ctx, eng.ID)
	if err != nil {
		t.Fatalf("Orchestrator Execute failed: %v", err)
	}

	if eng.Status != PhaseComplete {
		t.Errorf("Expected final status to be COMPLETE with sandbox, got %s", eng.Status)
	}
	if eng.EndTime.IsZero() {
		t.Errorf("Expected EndTime to be set")
	}
}
