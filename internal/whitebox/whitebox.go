// Package whitebox implements DrogonClaw's autonomous white-box web/API
// penetration-testing pipeline — the capability that previously required a
// separate tool (Shannon) and several point solutions.
//
// It elevates the existing agent/* primitives (source_review, web_probe,
// browser_validate, run_nuclei, run_sqlmap, auth_bypass_scan, …) into a
// five-phase proof-by-exploitation pipeline:
//
//  1. Pre-Recon  — source analysis (SAST) to build the architectural baseline
//  2. Recon      — attack-surface mapping of the live application
//  3. Analysis   — five parallel vulnerability agents (injection, xss,
//     ssrf, authn, authz)
//  4. Exploit    — re-execute/re-verify every candidate to prove exploitability
//  5. Report     — PoC-backed Markdown + SARIF, recorded to the loot DB
//
// AUTHORIZED USE ONLY. Every tool invoked here already enforces the same
// boundary; never point this at systems you do not own or are not authorized
// to test.
package whitebox

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/agent"
	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/0xP4X/drogonclaw-go/internal/planner"
)

// Severity mirrors the reporting tiers used elsewhere in DrogonClaw.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Config controls a single white-box engagement.
type Config struct {
	// TargetURL is the live application/API base URL (required for dynamic phases).
	TargetURL string
	// RepoPath is an optional local source tree for the Pre-Recon SAST phase.
	RepoPath string
	// SessionID ties findings to the memory graph / loot DB session.
	SessionID string
	// OutDir is where the Markdown + SARIF reports are written.
	OutDir string
	// Verify triggers the exploitation phase (re-execution to prove impact).
	// When false, the pipeline only reports candidates from static/dynamic analysis.
	Verify bool
}

// Finding is a single, evidence-backed result.
type Finding struct {
	ID       string   // stable identifier, e.g. WB-001
	Title    string   // human-readable summary
	Class    string   // owasp class: injection | xss | ssrf | authn | authz
	Severity Severity // risk rating
	Target   string   // URL or file:line
	Location string   // parameter, route, or source location
	Evidence string   // verbatim proof (tool output snippet, request/response)
	PoC      string   // how to reproduce / what was executed
	Verified bool     // true only after the exploit phase confirmed impact
	Source   string   // "sast" | "dast" | "source" | "recon"
}

// Report is the aggregated result of a run.
type Report struct {
	TargetURL string
	RepoPath  string
	SessionID string
	Started   time.Time
	Finished  time.Time
	ToolCalls int
	Findings  []Finding
}

// Run executes the full pipeline. The provided *agent.ToolRegistry must already
// be initialized (sandbox + builtins registered). loot may be nil.
func Run(ctx context.Context, cfg Config, tools *agent.ToolRegistry, graph *memory.Graph, loot *memory.LootDB) (*Report, error) {
	if cfg.TargetURL == "" && cfg.RepoPath == "" {
		return nil, fmt.Errorf("whitebox: provide at least one of -u <url> or -r <repo>")
	}
	rep := &Report{
		TargetURL: cfg.TargetURL,
		RepoPath:  cfg.RepoPath,
		SessionID: cfg.SessionID,
		Started:   time.Now(),
	}

	// Planning: build (or, on resume, reload) the task tree on the intelligence
	// graph. Resume is automatic — Graph reloads prior task statuses by session
	// ID, so already-completed phases are skipped on a re-run.
	pl := planner.New(graph)
	pl.BuildWhiteboxPlan(cfg.TargetURL, cfg.RepoPath)

	// phase runs a step, skipping it when the planner already marks it done
	// (resume) and recording completion afterwards.
	phase := func(class string, fn func() []Finding) {
		if t := pl.TaskByClass(class); t != nil && t.Status == planner.StatusDone {
			return
		}
		got := fn()
		rep.Findings = append(rep.Findings, got...)
		if t := pl.TaskByClass(class); t != nil {
			pl.MarkDone(t.ID, fmt.Sprintf("%d findings", len(got)))
		}
	}

	// Phase 1 — Pre-Recon (source)
	if cfg.RepoPath != "" {
		phase("sast", func() []Finding { return preRecon(ctx, tools, cfg.RepoPath, &rep.ToolCalls) })
	}

	// Phase 2 — Recon (live surface)
	if cfg.TargetURL != "" {
		phase("recon", func() []Finding { return recon(ctx, tools, cfg.TargetURL, &rep.ToolCalls) })
	}

	// Phase 3 — Vulnerability analysis (five agents)
	if cfg.TargetURL != "" {
		phase("injection", func() []Finding { return agentInjection(ctx, tools, cfg.TargetURL, &rep.ToolCalls) })
		phase("xss", func() []Finding { return agentXSS(ctx, tools, cfg.TargetURL, &rep.ToolCalls) })
		phase("ssrf", func() []Finding { return agentSSRF(ctx, tools, cfg.TargetURL, &rep.ToolCalls) })
		phase("authn", func() []Finding { return agentAuthN(ctx, tools, cfg.TargetURL, &rep.ToolCalls) })
		phase("authz", func() []Finding { return agentAuthZ(ctx, tools, cfg.TargetURL, &rep.ToolCalls) })
	}

	// Phase 4 — Exploitation / verification (Strix-style blind verification)
	if cfg.Verify {
		if t := pl.TaskByClass("verify"); t == nil || t.Status != planner.StatusDone {
			rep.Findings = verifyFindings(ctx, tools, rep.Findings, &rep.ToolCalls)
			if t := pl.TaskByClass("verify"); t != nil {
				pl.MarkDone(t.ID, "verified findings")
			}
		}
	}

	// Phase 5 — Record to loot DB + report
	for i := range rep.Findings {
		if loot != nil {
			_ = loot.InsertVulnerability(
				rep.Findings[i].Target,
				"",
				fmt.Sprintf("[%s] %s", rep.Findings[i].Class, rep.Findings[i].Title),
				string(rep.Findings[i].Severity),
			)
		}
	}

	rep.Finished = time.Now()
	return rep, nil
}

// call invokes a registered builtin through the existing tool dispatcher and
// returns its raw output. It is the single integration point with agent/*, so
// the white-box pipeline reuses the same sandboxed, evidenced tooling rather
// than re-implementing scanners.
func call(tools *agent.ToolRegistry, name string, args map[string]any, calls *int) string {
	*calls++
	buf, _ := json.Marshal(args)
	return tools.Execute(context.Background(), name, string(buf))
}

// severityFromText infers a severity from tool output keywords. Conservative by
// default — unknown matches stay at medium rather than being inflated.
func severityFromText(s string) Severity {
	low := strings.ToLower(s)
	switch {
	case strings.Contains(low, "critical") || strings.Contains(low, "rce") ||
		strings.Contains(low, "remote code execution") || strings.Contains(low, "command execution"):
		return SeverityCritical
	case strings.Contains(low, "high") || strings.Contains(low, "sqli") ||
		strings.Contains(low, "sql injection") || strings.Contains(low, "authentication bypass"):
		return SeverityHigh
	case strings.Contains(low, "medium") || strings.Contains(low, "xss") ||
		strings.Contains(low, "ssrf") || strings.Contains(low, "csrf") ||
		strings.Contains(low, "idor"):
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// cveRef extracts the first CVE reference from a blob of text, if any.
var reCVE = regexp.MustCompile(`CVE-\d{4}-\d{3,7}`)

func cveRef(s string) string {
	return reCVE.FindString(s)
}
