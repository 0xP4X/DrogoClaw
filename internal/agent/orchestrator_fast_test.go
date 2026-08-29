package agent

import (
	"strings"
	"testing"
)

func TestIsFastPhaseTool(t *testing.T) {
	fast := []string{"run_nmap", "run_subfinder", "run_httpx", "run_gobuster", "run_ffuf", "run_nuclei", "osint_certs", "web_search"}
	for _, name := range fast {
		if !isFastPhaseTool(name) {
			t.Errorf("expected %q to be classified fast-phase", name)
		}
	}
	slow := []string{"shell_execute", "run_exploit", "run_msfvenom", "write_and_run_script", "ask_operator", "generate_fud_payload"}
	for _, name := range slow {
		if isFastPhaseTool(name) {
			t.Errorf("expected %q NOT to be classified fast-phase", name)
		}
	}
}

func TestAppendToolEvidence(t *testing.T) {
	o := &Orchestrator{}
	for i := 0; i < 10; i++ {
		o.appendToolEvidence("run_nmap", strings.Repeat("x", 8000))
	}
	if len(o.recentToolOutputs) != 6 {
		t.Fatalf("expected evidence window of 6, got %d", len(o.recentToolOutputs))
	}
	for _, rec := range o.recentToolOutputs {
		if len(rec.output) > 6000 {
			t.Fatalf("evidence entry not truncated: %d bytes", len(rec.output))
		}
	}

	o.appendToolEvidence("", "")
	if len(o.recentToolOutputs) != 6 {
		t.Fatalf("empty evidence should be skipped, got window %d", len(o.recentToolOutputs))
	}

	built := buildToolEvidence(o.recentToolOutputs)
	if !strings.Contains(built, "=== run_nmap ===") {
		t.Fatal("buildToolEvidence missing tool header")
	}
}

func TestProviderFastFallback(t *testing.T) {
	// With only a primary model configured, HasFast must be false and the fast
	// tier must fall back to the primary (no panic, no distinct model).
	p := &Provider{model: "primary", fastModel: ""}
	if p.HasFast() {
		t.Fatal("expected HasFast()==false when no fast model configured")
	}
	if p.fastModel != "" {
		t.Fatal("expected empty fast model")
	}

	// Distinct fast model => HasFast true; identical => false.
	p2 := &Provider{model: "primary", fastModel: "primary"}
	if p2.HasFast() {
		t.Fatal("expected HasFast()==false when fast model equals primary")
	}
	p3 := &Provider{model: "primary", fastModel: "fast-cheap"}
	if !p3.HasFast() {
		t.Fatal("expected HasFast()==true with distinct fast model")
	}
}

func TestTrimForVerification(t *testing.T) {
	long := strings.Repeat("a", 5000)
	if len(trimForVerification(long)) != 4000 {
		t.Fatal("expected trim to 4000 chars")
	}
	short := "ok"
	if trimForVerification(short) != short {
		t.Fatal("short strings must pass through unchanged")
	}
}

func TestBuildResultsAppendix(t *testing.T) {
	recs := []toolOutputEvidence{
		{tool: "old", output: strings.Repeat("o", 200)},
		{tool: "run_subfinder", output: "sub-a\nsub-b\nsub-c"},
	}
	app := buildResultsAppendix(recs)
	if !strings.Contains(app, "[run_subfinder]") {
		t.Fatal("expected most-recent tool output first in the appendix")
	}
	if idx := len(app) - 1; idx < 0 || strings.Contains(app[idx:], "[old]") {
		t.Fatal("expected the most-recent block to be listed before the older one")
	}
	if !strings.Contains(app, "sub-a\nsub-b\nsub-c") {
		t.Fatal("expected the subdomain list to be included verbatim")
	}

	big := []toolOutputEvidence{{tool: "huge", output: strings.Repeat("x", resultsAppendixMax*2)}}
	app = buildResultsAppendix(big)
	if len(app) > resultsAppendixMax {
		t.Fatalf("appendix exceeds budget: %d", len(app))
	}
}
