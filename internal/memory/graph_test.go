package memory

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func newTestGraph(t *testing.T) *Graph {
	t.Helper()
	// Use temp directory to avoid polluting data/
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(origDir) })

	g := NewGraph("test-session")
	return g
}

func TestAddNode(t *testing.T) {
	g := newTestGraph(t)

	node := &Node{
		ID:    "target-1",
		Label: LabelTarget,
		Properties: map[string]any{
			"ip": "10.10.10.1",
		},
	}
	g.AddNode(node)

	if g.NodeCount() != 1 {
		t.Errorf("NodeCount() = %d, want 1", g.NodeCount())
	}

	nodes := g.GetNodes()
	if _, ok := nodes["target-1"]; !ok {
		t.Error("GetNodes() missing target-1")
	}
}

func TestAddNodeDedup(t *testing.T) {
	g := newTestGraph(t)

	g.AddNode(&Node{ID: "target-1", Label: LabelTarget})
	g.AddNode(&Node{ID: "target-1", Label: LabelTarget})

	if g.NodeCount() != 1 {
		t.Errorf("NodeCount() = %d, want 1 (dedup)", g.NodeCount())
	}
}

func TestAddEdge(t *testing.T) {
	g := newTestGraph(t)

	g.AddNode(&Node{ID: "host-1", Label: LabelAsset})
	g.AddNode(&Node{ID: "port-80", Label: LabelPort})

	g.AddEdge(&Edge{SourceID: "host-1", TargetID: "port-80", Relationship: "HAS_PORT"})

	if g.EdgeCount() != 1 {
		t.Errorf("EdgeCount() = %d, want 1", g.EdgeCount())
	}
}

func TestAddEdgeDedup(t *testing.T) {
	g := newTestGraph(t)

	g.AddNode(&Node{ID: "host-1", Label: LabelAsset})
	g.AddNode(&Node{ID: "port-80", Label: LabelPort})

	g.AddEdge(&Edge{SourceID: "host-1", TargetID: "port-80", Relationship: "HAS_PORT"})
	g.AddEdge(&Edge{SourceID: "host-1", TargetID: "port-80", Relationship: "HAS_PORT"})

	if g.EdgeCount() != 1 {
		t.Errorf("EdgeCount() = %d, want 1 (dedup)", g.EdgeCount())
	}
}

func TestLabelCounts(t *testing.T) {
	g := newTestGraph(t)

	g.AddNode(&Node{ID: "t1", Label: LabelTarget})
	g.AddNode(&Node{ID: "t2", Label: LabelTarget})
	g.AddNode(&Node{ID: "p1", Label: LabelPort})

	counts := g.LabelCounts()
	if counts[LabelTarget] != 2 {
		t.Errorf("LabelCounts()[Target] = %d, want 2", counts[LabelTarget])
	}
	if counts[LabelPort] != 1 {
		t.Errorf("LabelCounts()[Port] = %d, want 1", counts[LabelPort])
	}
}

func TestRelationshipCounts(t *testing.T) {
	g := newTestGraph(t)

	g.AddNode(&Node{ID: "h1", Label: LabelAsset})
	g.AddNode(&Node{ID: "p1", Label: LabelPort})
	g.AddNode(&Node{ID: "p2", Label: LabelPort})

	g.AddEdge(&Edge{SourceID: "h1", TargetID: "p1", Relationship: "HAS_PORT"})
	g.AddEdge(&Edge{SourceID: "h1", TargetID: "p2", Relationship: "HAS_PORT"})

	counts := g.RelationshipCounts()
	if counts["HAS_PORT"] != 2 {
		t.Errorf("RelationshipCounts()[HAS_PORT] = %d, want 2", counts["HAS_PORT"])
	}
}

func TestSnapshot(t *testing.T) {
	g := newTestGraph(t)

	g.AddNode(&Node{ID: "target-1", Label: LabelTarget, Properties: map[string]any{"ip": "10.10.10.1"}})
	g.AddNode(&Node{ID: "port-80", Label: LabelPort, Properties: map[string]any{"port": 80}})
	g.AddEdge(&Edge{SourceID: "target-1", TargetID: "port-80", Relationship: "HAS_PORT"})

	snap := g.Snapshot()
	if snap == "" {
		t.Error("Snapshot() returned empty string")
	}
	if !containsStr(snap, "Entities: 2") {
		t.Errorf("Snapshot() missing entity count, got: %s", snap)
	}
	if !containsStr(snap, "Relationships: 1") {
		t.Errorf("Snapshot() missing relationship count, got: %s", snap)
	}
}

func TestSnapshotEmpty(t *testing.T) {
	g := newTestGraph(t)

	snap := g.Snapshot()
	if !containsStr(snap, "Entities: none recorded yet") {
		t.Errorf("Snapshot() for empty graph = %s", snap)
	}
}

func TestOperatorProfile(t *testing.T) {
	g := newTestGraph(t)

	g.UpdateOperatorProfile(&OperatorProfile{Name: "Alice", SkillLevel: "advanced"})

	profile := g.GetOperatorProfile()
	if profile == nil {
		t.Fatal("GetOperatorProfile() returned nil")
	}
	if profile.Name != "Alice" {
		t.Errorf("GetOperatorProfile().Name = %s, want Alice", profile.Name)
	}
	if profile.SkillLevel != "advanced" {
		t.Errorf("GetOperatorProfile().SkillLevel = %s, want advanced", profile.SkillLevel)
	}
}

func TestAgentProfile(t *testing.T) {
	g := newTestGraph(t)

	g.UpdateAgentProfile(&AgentProfile{Name: "DrogonClaw"})

	profile := g.GetAgentProfile()
	if profile == nil {
		t.Fatal("GetAgentProfile() returned nil")
	}
	if profile.Name != "DrogonClaw" {
		t.Errorf("GetAgentProfile().Name = %s, want DrogonClaw", profile.Name)
	}
}

func TestReset(t *testing.T) {
	g := newTestGraph(t)

	g.AddNode(&Node{ID: "target-1", Label: LabelTarget})
	g.AddNode(&Node{ID: "port-80", Label: LabelPort})
	g.AddEdge(&Edge{SourceID: "target-1", TargetID: "port-80", Relationship: "HAS_PORT"})

	g.Reset()

	if g.NodeCount() != 0 {
		t.Errorf("NodeCount() after Reset() = %d, want 0", g.NodeCount())
	}
	if g.EdgeCount() != 0 {
		t.Errorf("EdgeCount() after Reset() = %d, want 0", g.EdgeCount())
	}
}

func TestGetFullJSON(t *testing.T) {
	g := newTestGraph(t)

	g.AddNode(&Node{ID: "target-1", Label: LabelTarget})
	json := g.GetFullJSON()
	if json == "" {
		t.Error("GetFullJSON() returned empty string")
	}
}

func TestConcurrentAccess(t *testing.T) {
	g := newTestGraph(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			g.AddNode(&Node{
				ID:    filepath.Base(t.TempDir()),
				Label: LabelAsset,
			})
			g.AddEdge(&Edge{
				SourceID: "nonexistent",
				TargetID: "nonexistent",
				Relationship: "TEST",
			})
			_ = g.Snapshot()
			_ = g.NodeCount()
			_ = g.EdgeCount()
		}(i)
	}
	wg.Wait()
}

func TestPersistence(t *testing.T) {
	g := newTestGraph(t)

	g.AddNode(&Node{ID: "target-1", Label: LabelTarget, Properties: map[string]any{"ip": "10.10.10.1"}})
	g.AddEdge(&Edge{SourceID: "target-1", TargetID: "target-1", Relationship: "SELF"})

	// Create new graph with same session ID to test load
	g2 := NewGraph("test-session")

	if g2.NodeCount() != 1 {
		t.Errorf("NodeCount() after reload = %d, want 1", g2.NodeCount())
	}
	if g2.EdgeCount() != 1 {
		t.Errorf("EdgeCount() after reload = %d, want 1", g2.EdgeCount())
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
