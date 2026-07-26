package agent

import (
	"testing"

	"github.com/0xP4X/drogonclaw-go/internal/memory"
)

func TestGateToolExecutionBlocksActiveToolWithoutEvidence(t *testing.T) {
	r := &ToolRegistry{graph: memory.NewGraph("test_gate_block")}

	msg := r.gateToolExecution("run_exploit", map[string]any{})
	if msg == "" {
		t.Fatal("expected active tool to be blocked without evidence")
	}
}

func TestGateToolExecutionAllowsActiveToolWithExplicitTarget(t *testing.T) {
	r := &ToolRegistry{graph: memory.NewGraph("test_gate_target")}

	msg := r.gateToolExecution("run_exploit", map[string]any{"target": "10.10.10.10"})
	if msg != "" {
		t.Fatalf("expected active tool to be allowed with explicit target, got %q", msg)
	}
}

func TestGateToolExecutionBlocksPostAccessToolWithoutSession(t *testing.T) {
	r := &ToolRegistry{graph: memory.NewGraph("test_gate_post")}

	msg := r.gateToolExecution("download_loot", map[string]any{})
	if msg == "" {
		t.Fatal("expected post-access tool to be blocked without session evidence")
	}
}

func TestGateToolExecutionAllowsObservationShellCommand(t *testing.T) {
	r := &ToolRegistry{graph: memory.NewGraph("test_gate_shell")}

	msg := r.gateToolExecution("shell_execute", map[string]any{"command": "ls -la"})
	if msg != "" {
		t.Fatalf("expected observation shell command to be allowed, got %q", msg)
	}
}
