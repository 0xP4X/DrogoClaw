# DrogonClaw System Readiness Report

**Date:** 2026-07-16  
**Scope:** Local codebase architecture, execution reliability, operational controls, documentation, build, and test readiness.  
**Boundary:** This is a source review only. No targets were tested and no offensive actions were performed.

## Executive assessment

DrogonClaw has useful building blocks, but is not ready for scaled autonomous pentesting or CTF execution. It is currently a prompt-driven tool runner rather than a dependable orchestration platform.

The immediate priority is to stabilize a small, authorized, local-CTF execution kernel with typed results, durable state, deterministic verification, and reproducible tests. Adding more skills or agents before that will make execution less reliable.

## Immediate release blockers

1. **Repository integrity is at risk.** `git ls-files` returned zero tracked files; the project files are all untracked. Establish an initial commit and protected branches before further engineering work.
2. **The remaining orchestration and evidence gaps are release-blocking for autonomous use.** A passing build is necessary but does not supply workflow state, verified completion, or enforced scope.

### Remediated during the initial hardening pass

- The TUI compile failure at `internal/tui/model.go:1473` was fixed.
- `go build ./cmd/drogonclaw` and `go test ./...` now pass with the Go build cache redirected to `/tmp` because the default cache path is read-only in this environment.
- Native and Docker command adapters now return non-zero command failures to callers.
- HitL waiting is event-driven instead of a busy-spin loop.
- The simulated red-team lifecycle now waits on an explicit approval event rather than fixed sleeps, and its phase transitions are synchronized.
- Sandbox startup now fails closed rather than automatically switching to host command execution.
- The API no longer accepts a built-in default token and defaults to a loopback listener.

## Findings

| Priority | Finding | Evidence and impact |
|---|---|---|
| P0 | Plans are display-only | `internal/agent/orchestrator.go` generates and prints an LLM plan, then proceeds with an unconstrained ReAct loop. Steps have no runtime status, dependencies, preconditions, retries, or completion gates. |
| P0 | No deterministic success oracle | The loop treats a model response with no tool call as a final answer. A model can claim a solved CTF or completed engagement without verified evidence. |
| P0 | Tool outcomes are untyped | `ToolRegistry.Execute` returns prose strings only. Exit status, stdout/stderr, artifacts, duration, truncation, parsed facts, and error class are unavailable to the controller. |
| Resolved | Native command failure was hidden | `internal/sandbox/docker.go` now returns the partial command output together with a non-nil error for native or Docker non-zero exits. |
| P0 | API engagement execution is simulated | `internal/redteam/orchestrator/orchestrator.go` adds a simulated vulnerability instead of executing the advertised workflow. The API server is also not invoked by the CLI entrypoint. |
| P0 | No enforceable target scope | Model-controlled values are interpolated into shell commands. The dangerous-command check is a bypassable regex denylist, not a policy engine. There is no authorization record, target allowlist, rate policy, or engagement scope enforcement. |
| Resolved | Sandbox failure previously fell back to the host | `cmd/drogonclaw/main.go` now fails closed. Native execution requires an explicit `USE_SANDBOX=false` configuration choice. |
| P1 | Docker runtime is not strong isolation | The persistent Kali container has network capabilities and a writable host mount. It is not an ephemeral, per-task workspace. |
| P1 | Swarm workers share mutable state | `internal/agent/swarm.go` shares one tool registry, sandbox, working directory, and graph among concurrent workers. There is no branch isolation, quota, dependency graph, or conflict-aware merge. |
| P1 | HITL remains global | Waiting is now event-driven, but `internal/core/hitl.go` still has one global approval state. Concurrent runs can interfere until approval state is made run-scoped. |
| P1 | CTF workflow is only file triage | `internal/ctf/triage.go` scans supplied regular files for a flag-shaped regex. It does not perform archive extraction, metadata analysis, decoding, binary analysis, or challenge-specific validation. |
| P1 | Documentation conflicts with implementation | `docs/INDEX.md` claims TLS, JWT authentication, and RBAC. `internal/api/server.go` implements static Bearer authentication and no RBAC; it now requires an explicitly configured token and defaults to a loopback listener. |
| P2 | Tool surface is too broad | Generic manifest fallback turns tool names and arguments into shell syntax. This creates invalid calls and expands model choice without improving reliability. |

## Why orchestration is weak today

The current architecture delegates too much control to prompt text. A plan, attack chain, and tool result are all prose. The runtime does not know:

- which step is active;
- what evidence satisfies the next step;
- whether a command succeeded;
- which artifacts were produced;
- when to retry, pivot, stop, or request operator input; or
- whether a claimed flag or finding is genuine.

At scale, this produces repeated commands, incorrect pivots, conflicting swarm results, and convincing but unverified summaries.

## Target execution model

```text
Task intake and authorization
          |
          v
Challenge / engagement classifier
          |
          v
Typed workflow DAG and step scheduler
          |
          v
Tool adapter -> normalized observation -> artifact store
          |                  |
          |                  v
          |            fact reducer
          v
Deterministic verifier -> success / recover / needs-human / failed
```

### Required contracts

1. **Task:** mode, allowed scope, local artifact or allowlisted target, success predicate, budget, and output directory.
2. **Step:** typed inputs, preconditions, adapter, timeout, expected observations, retries, and recovery branches.
3. **Observation:** command identifier, redacted arguments, exit status, stdout/stderr summaries, duration, artifact IDs, parsed facts, and failure class.
4. **Evidence store:** append-only JSONL or database ledger; facts must reference their originating observation ID.
5. **Verifier:** deterministic completion and flag/finding checks. The LLM must not be able to declare success by prose alone.
6. **Policy engine:** run-scoped approvals, target restrictions, capability limits, rate limits, and audit logging.

## Stabilization roadmap

### Phase 0 — Restore engineering baseline

- Fix the TUI compile failure.
- Create and commit a clean repository baseline.
- Make build, unit tests, race tests, linting, and dependency/security scanning blocking CI checks.
- Remove `continue-on-error` from security and lint gates once initially remediated.
- Correct documentation that claims unsupported API and security functionality.

**Exit criterion:** a clean checkout builds and passes CI deterministically.

### Phase 1 — Build a safe local-CTF kernel

- Add `Task`, `Step`, `RunState`, `ToolResult`, `Observation`, `Artifact`, and `Fact` types.
- Change sandbox adapters to return structured results and real non-zero exit errors.
- Add a durable run ledger and task-scoped workspace.
- Implement one workflow first: local forensics CTF.
- Add a deterministic flag verifier and fixtures for success, no flag, timeout, missing tool, and malformed action.

**Exit criterion:** repeated runs solve at least 90% of the frozen local-forensics fixtures with evidence provenance for every result.

### Phase 2 — Controlled reasoning and recovery

- Expose the LLM only to the current task state, verified facts, artifacts, budget, and permitted next actions.
- Add deterministic recovery policies for known failures before LLM escalation.
- Replace global HitL polling with a run-scoped blocking approval event and resumable run ID.
- Require observation provenance before memory facts can be promoted.

**Exit criterion:** interrupted tasks resume correctly and injected failures are classified correctly in at least 95% of test cases.

### Phase 3 — Add workflows one vertical slice at a time

- Add crypto, then local web, then binary CTF workflows.
- Give each mode a small, mode-specific set of typed tools.
- Convert prompt-only chains into tested workflow DAGs or retire them.
- Keep any network-active capability behind explicit scope, approval, and rate-control policies.

**Exit criterion:** each category reaches 80% or better verified completion on a held-out legal benchmark, with zero unverified success claims.

### Phase 4 — Parallelism after serial reliability

- Assign each worker an isolated workspace and immutable state snapshot.
- Parallelize only declared independent workflow branches.
- Merge observations through one reducer and reject contradictions.
- Enforce worker, tool-call, runtime, and network budgets.

**Exit criterion:** parallel and serial runs give reproducible results on the same fixture set.

## Documentation corrections needed

Document the product as it exists today, including:

- actual execution modes and host-execution fallback behavior;
- API authentication and binding behavior;
- scope/authorization requirements;
- sandbox limitations and mounted data paths;
- data retention, log redaction, and secrets handling;
- supported CTF categories and explicit unsupported cases;
- benchmark methodology and known reliability limits.

Do not claim JWT, RBAC, TLS-by-default, full autonomous exploit execution, verified payloads, or swarm isolation until they are implemented and tested.

## Metrics for real agentic capability

Track these per fixture and enforce them in CI:

- verified completion rate;
- median tool calls and wall-clock time to verification;
- recovery rate after injected command failure;
- invalid tool-call rate;
- fact provenance coverage;
- unverified-success rate, with a target of zero;
- human interventions per solved task; and
- reproducibility across three clean runs.

## Validation performed

- Reviewed orchestration, planner, tool dispatch, sandbox execution, swarm, HitL, CTF triage, API, configuration, Docker, CI, and project documentation.
- Ran `go build ./cmd/drogonclaw` successfully with `GOCACHE=/tmp/drogonclaw-go-build`.
- Ran `go test ./...` successfully with the same cache setting.
- The initial hardening pass changed build, command-result, HitL, sandbox-startup, and API-authentication behavior as described above.

## Bottom line

Make DrogonClaw smaller, observable, and verifiable before making it broader. One authorized local-CTF workflow with typed observations, durable state, deterministic verification, and recovery logic will provide more real capability than a large catalogue of unconstrained tools and agents.
