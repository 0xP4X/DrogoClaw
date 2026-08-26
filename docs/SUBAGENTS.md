# Parallel Subagents

DrogonClaw can execute multiple independent recon and exploitation tasks concurrently using the Subagent Manager. Inspired by Hermes Agent's subagent delegation system.

## The Problem

Sequential execution is slow. A typical recon workflow:
1. Nmap port scan — 3 minutes
2. Subdomain enumeration — 1 minute
3. Web probing — 1 minute
4. Directory bruteforce — 2 minutes
5. Vulnerability scan — 2 minutes

**Total: 9 minutes**

## The Solution

Run independent tasks in parallel:
1. Nmap + Subdomain enum + Web probe — 3 minutes (longest task)
2. Dir brute + Vuln scan (depend on port scan) — 2 minutes

**Total: 5 minutes (44% faster)**

## How It Works

```
┌─────────────────────────────────────────────────┐
│           Subagent Manager                       │
│                                                  │
│  Wave 1 (independent):                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐        │
│  │ Port Scan│ │ Subdomain│ │ Web Probe│        │
│  │ (nmap)   │ │ (subfind)│ │ (httpx)  │        │
│  └────┬─────┘ └──────────┘ └──────────┘        │
│       │                                          │
│  Wave 2 (depends on port scan):                 │
│  ┌──────────┐ ┌──────────┐                      │
│  │Dir Brute │ │ Vuln Scan│                      │
│  │(gobuster)│ │ (nuclei) │                      │
│  └──────────┘ └──────────┘                      │
│                                                  │
│  Merge Results → LLM Context                     │
└─────────────────────────────────────────────────┘
```

## Task Definition

```go
type SubagentTask struct {
    ID          string         // Unique identifier
    Name        string         // Human-readable name
    Tool        string         // Tool to execute
    Args        map[string]any // Tool arguments
    Context     string         // Additional context
    Priority    int            // Higher = runs first
    DependsOn   []string       // IDs of tasks that must complete first
}
```

## Preset Task Bundles

### Standard Recon (3 tasks, ~3 min)

```go
tasks := StandardReconTasks("10.0.0.1")
// → port_scan (nmap quick)
// → subdomain_enum (subfinder)
// → web_recon (httpx)
```

### Full Web Recon (5 tasks, ~5 min)

```go
tasks := FullWebReconTasks("example.com")
// → port_scan (nmap full)
// → subdomain_enum (subfinder)
// → web_probe (httpx) — depends on subdomain_enum
// → dir_bruteforce (gobuster) — depends on port_scan
// → vuln_scan (nuclei) — depends on port_scan
```

### Cloud Recon (3 tasks, ~2 min)

```go
tasks := CloudTasks("aws-target.com")
// → cert_transparency (osint_certs)
// → dns_enumeration (subfinder)
// → port_scan (nmap quick)
```

## Dependency Scheduling

Tasks are executed in waves based on dependencies:

1. **Wave 1**: All tasks with no dependencies run in parallel
2. **Wave 2**: Tasks whose dependencies are all completed
3. **Wave 3**: Tasks whose dependencies are all completed
4. ... and so on

If a circular dependency is detected, the first waiting task is forced to run.

## Concurrency Control

```go
sm := NewSubagentManager(provider, tools, 5) // max 5 concurrent
```

The semaphore pattern limits concurrent executions:
```go
sem := make(chan struct{}, sm.maxConc)
for _, task := range ready {
    wg.Add(1)
    sem <- struct{}{} // acquire
    go func(t SubagentTask) {
        defer wg.Done()
        defer func() { <-sem }() // release
        // execute task
    }(task)
}
```

## Result Merging

After all tasks complete, results are formatted for the LLM:

```
PARALLEL EXECUTION RESULTS (3 tasks completed):

─── port_scan [SUCCESS] (2m30s) ───
[NMAP — QUICK — 10.0.0.1]
80/tcp open  http
443/tcp open  https
22/tcp open  ssh

─── subdomain_enum [SUCCESS] (45s) ───
[SUBFINDER — example.com]
api.example.com
staging.example.com
dev.example.com

─── web_recon [SUCCESS] (30s) ───
[HTTPX — example.com]
Status: 200 | Tech: nginx, PHP
Status: 200 | Tech: Apache, WordPress
```

## Integration with Orchestrator

```go
// Execute parallel recon
tasks := StandardReconTasks(target)
result := o.ExecuteParallelTasks(ctx, tasks, events)

// Or use the subagent manager directly
sm := o.GetSubagents()
results := sm.ExecuteParallel(ctx, tasks, events)
formatted := FormatResultsForLLM(results)
```

## API Reference

### SubagentManager
- `NewSubagentManager(provider *Provider, tools *ToolRegistry, maxConcurrent int) *SubagentManager`
- `ExecuteParallel(ctx context.Context, tasks []SubagentTask, events chan<- Event) []SubagentResult`

### Preset Functions
- `StandardReconTasks(target string) []SubagentTask`
- `FullWebReconTasks(target string) []SubagentTask`
- `CloudTasks(target string) []SubagentTask`

### Result Functions
- `FormatResultsForLLM(results []SubagentResult) string`
- `MergeResults(results []SubagentResult) string`

### SubagentResult
```go
type SubagentResult struct {
    TaskID   string
    Tool     string
    Output   string
    Duration time.Duration
    Error    error
}
```
