package agent

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/config"
	"github.com/0xP4X/drogonclaw-go/internal/core"
	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/0xP4X/drogonclaw-go/internal/opsec"
	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
	"github.com/0xP4X/drogonclaw-go/internal/skills"
)

func TestProbeTelegramWhois(t *testing.T) {
	if os.Getenv("DC_PROBE") != "1" {
		t.Skip("set DC_PROBE=1 to run the live mission probe")
	}

	cfg := config.Get()

	manifest, err := skills.Load("skills_manifest.json")
	if err != nil {
		manifest = &skills.Manifest{}
	}

	sb, sErr := sandbox.New()
	if sErr != nil {
		t.Fatalf("sandbox: %v", sErr)
	}
	if err := sb.Initialize(context.Background(), true); err != nil {
		t.Logf("sandbox native init: %v (whois does not need it)", err)
	}

	provider := NewProvider(cfg)
	pingCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := provider.Ping(pingCtx); err != nil {
		t.Fatalf("provider ping failed: %v", err)
	}

	sessionID := "probe-0000000001"
	graph := memory.NewGraph(sessionID)
	graph.UpdateOperatorProfile(&memory.OperatorProfile{Name: "jiggon", SkillLevel: "advanced"})

	lootDb, err := memory.NewLootDB()
	if err == nil {
		defer lootDb.Close()
	}

	validator := NewEvidenceValidator(provider)
	tools := NewToolRegistry(manifest, sb, validator, lootDb, cfg, graph, provider)

	opsecMgr := opsec.NewManager()
	sysPrompt := BuildSystemPrompt(graph, opsecMgr, "", sb.RuntimeLabel())
	orch := NewOrchestratorWithJournal(provider, tools, sysPrompt, sessionID, graph, memory.NewActionJournal(sessionID), cfg.GetMaxIterations())

	planner := core.NewMissionPlanner(provider, graph)
	plan, perr := planner.GeneratePlan(context.Background(), "whois example.com")
	valid := plan != nil && plan.IsValidMission
	steps := 0
	objective := ""
	if plan != nil {
		steps = len(plan.Steps)
		objective = plan.Objective
	}
	t.Logf("PLANNER: err=%v valid=%v objective=%q steps=%d", perr != nil, valid, objective, steps)
	if plan != nil {
		for _, s := range plan.Steps {
			t.Logf("  step[%s] %s -> %s", s.ID, s.Action, s.TargetAssetID)
		}
	}

	orch.Autopilot = true
	t.Log("AUTOPILOT ON")

	events := make(chan Event)
	runCtx, runCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer runCancel()
	go orch.Execute(runCtx, "whois example.com", events)

	for ev := range events {
		t.Logf("EVENT %s: %s", ev.Type, truncProbe(ev.Content, 300))
	}

	t.Logf("last tool result: tool=%s success=%v failclass=%s", tools.lastResult.ToolName, tools.lastResult.Success, tools.lastResult.FailureClass)
}

func truncProbe(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
