package agent

import "testing"

func TestBuildMemoryEdgesInfersTargetRelationship(t *testing.T) {
	props := map[string]any{"target_id": "target:example.com"}

	edges := buildMemoryEdges("port:example.com:443", "Port", props, "", "", "")

	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].SourceID != "target:example.com" {
		t.Fatalf("unexpected source id: %s", edges[0].SourceID)
	}
	if edges[0].TargetID != "port:example.com:443" {
		t.Fatalf("unexpected target id: %s", edges[0].TargetID)
	}
	if edges[0].Relationship != "HAS_PORT" {
		t.Fatalf("unexpected relationship: %s", edges[0].Relationship)
	}
}

func TestBuildMemoryEdgesUsesExplicitRelationship(t *testing.T) {
	edges := buildMemoryEdges("vuln:log4shell", "Vulnerability", nil, "service:ldap", "", "HAS_VULNERABILITY")

	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].SourceID != "service:ldap" || edges[0].TargetID != "vuln:log4shell" {
		t.Fatalf("unexpected edge: %+v", edges[0])
	}
	if edges[0].Relationship != "HAS_VULNERABILITY" {
		t.Fatalf("unexpected relationship: %s", edges[0].Relationship)
	}
}
