# AGENTS.md — DrogonClaw Agent Instructions

This file provides instructions for AI agents working on the DrogonClaw codebase.

## Project Overview

DrogonClaw is an autonomous AI penetration testing agent written in Go. It uses a ReAct loop to plan, execute, and adapt security assessments.

## Build & Test Commands

```bash
make build           # Build binary
make test            # Run all tests with race detection
make lint            # Run golangci-lint
make vet             # Run go vet
make format          # Format code
```

Run specific package tests:
```bash
go test ./internal/agent/... -v      # Agent tests
go test ./internal/httputil/... -v   # HTTP utilities tests
go test ./internal/memory/... -v     # Memory/graph tests
go test ./internal/cvss/... -v       # CVSS scoring tests
```

## Code Conventions

- **Go version**: 1.26+
- **No comments in code** unless explicitly requested
- **Follow existing patterns** — check neighboring files before writing new code
- **Use `shellutil.Quote()`** for all shell argument escaping
- **Evidence pipeline**: All tool executions go through classifyOutcome → extractFindings → EvaluateTool → RecordVerifiedFinding
- **Human-in-the-Loop**: Dangerous tools must be in the `longRunningTools` map or explicitly gated

## Architecture Key Points

- `internal/agent/orchestrator.go` — ReAct loop, event emission, tool dispatch
- `internal/agent/tools.go` — ToolRegistry, EvidenceValidator, SuccessOracle
- `internal/agent/toolwrappers.go` — Typed wrappers for Nmap, Nuclei, etc.
- `internal/agent/skilllearn.go` — Learns from successful attacks
- `internal/agent/subagent.go` — Parallel task execution
- `internal/httputil/session.go` — Sessions, AutoThrottle, cache, WAF detection
- `internal/memory/graph.go` — Intelligence graph (JSON-backed)
- `internal/memory/loot.go` — LootDB (SQLite)
- `internal/sandbox/docker.go` — Docker sandbox execution
- `internal/tui/view.go` — TUI rendering

## Important Patterns

### Tool Registration
Tools are registered in `registerBuiltins()` in `tools.go`. Each tool is a `BuiltinFn` with signature `func(ctx context.Context, args map[string]any) string`.

### Evidence Verification
After tool execution, the orchestrator calls:
```go
verified, estatus, reason := o.tools.EvaluateTool(tc.Function.Name, result)
if verified {
    o.tools.RecordVerifiedFinding()
}
```
This automatically teaches the Skill Learner.

### Session Management
HTTP tools should use the SessionManager for persistent cookies:
```go
client := sm.GetClientWithThrottle(domain, throttle)
resp, err := client.Do(req)
sm.RecordRequest(domain, resp)
```

### Parallel Execution
Use the SubagentManager for independent tasks:
```go
tasks := []SubagentTask{
    {ID: "scan1", Tool: "run_nmap", Args: map[string]any{"target": ip}},
    {ID: "enum1", Tool: "run_subfinder", Args: map[string]any{"target": domain}},
}
results := sm.ExecuteParallel(ctx, tasks, events)
```

## Security Checklist

- [ ] All shell arguments use `shellutil.Quote()`
- [ ] External outputs sanitized before LLM injection
- [ ] Dangerous tools gated behind HitL
- [ ] No secrets committed to repository
- [ ] Skill denylist patterns not bypassed
