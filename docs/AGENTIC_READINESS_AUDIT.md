# DrogonClaw Agentic Readiness Audit

Date: 2026-07-15  
Scope: local code review of the Go implementation. This is an architecture review, not a claim that any target was tested.

## Executive assessment

DrogonClaw is currently a **tool-calling chat application with a large offensive-tool catalogue**, not yet a dependable autonomous problem solver. Its low actuation in a simple CTF is the expected outcome of the current design:

1. Plans and attack chains are prose supplied to the model; neither is a runtime state machine.
2. Commands are largely composed from strings and their results are passed back as unstructured text.
3. The system has no durable evidence ledger, success oracle, retry policy, or dependency resolver.
4. Many advertised capabilities expand the choice space without giving the model reliable, task-specific actions.

The priority is not adding more tools or a more theatrical prompt. It is to build a narrow, observable **local-CTF execution kernel**, then prove it against a benchmark suite.

## What works today

- A conventional ReAct loop invokes model-selected tools and feeds each text result back to the model.
- There are useful structured wrappers for a small set of common tools.
- The application has a Docker runtime option, cancellation context, memory graph, and some unit tests.
- Target classification makes a reasonable starting point for task routing.

These are good building blocks. They do not, by themselves, create dependable agency.

## Findings

| Priority | Finding | Why it prevents solving CTFs | Evidence |
|---|---|---|---|
| P0 | Plans are display-only | `MissionPlan` steps never become scheduled actions, acquire status, or gate the next action. The model may ignore the plan on its first call. | `internal/agent/orchestrator.go:88-99`, `internal/core/mission.go:73-111` |
| P0 | CTF modes are prompt injection, not workflows | A chain is appended to the system prompt. It has no bindings from placeholders such as `<binary>` to artifacts, no execution, and no postcondition validation. | `internal/tui/model.go:522-537`, `internal/intel/target_analyzer.go:105-122` |
| P0 | No success oracle | The loop ends whenever the model emits no tool call, even if no flag was found or the task is unfinished. The only other stop is a global 25-iteration cap. | `internal/agent/orchestrator.go:101-161` |
| P0 | Tool results have no stable schema | `ToolRegistry.Execute` returns free-form strings. There is no exit code, elapsed time, truncation marker, artifact list, parsed facts, or classification of failure. | `internal/agent/tools.go:536-558` |
| P0 | Output errors are frequently hidden | Native command execution deliberately returns `nil` error even when a command fails, losing the failure signal that a controller needs to select a recovery branch. | `internal/sandbox/docker.go:214-224` |
| P1 | Evidence validation is detached from evidence | Memory validation receives the string `Context implied by recent shell commands`, rather than the associated observation. It cannot establish provenance or reliably reject a false fact. | `internal/agent/tools.go:588-596` |
| P1 | The tool surface is too broad and ambiguous | At least 80 builtins plus manifest skills compete for attention; many manifest skills fall through to generic shell command construction. This increases tool-selection and argument errors. | `internal/agent/tools.go`, `internal/agent/toolwrappers.go`, `internal/agent/tools.go:547-552` |
| P1 | Runtime preflight is not task-aware | Tool installation is a single large best-effort command. There is no per-challenge check of needed tools, files, architecture, or working directory. | `internal/sandbox/docker.go:130-134` |
| P1 | “Swarm” has no isolation or merge protocol | Concurrent agents share one registry/sandbox/current directory and graph. Results are plain reports, not conflict-resolved observations. | `internal/agent/swarm.go:46-91` |
| P1 | HITL waiting busy-spins | Pending approval is polled in a tight loop. This wastes CPU and makes interactive suspension brittle. | `internal/agent/orchestrator.go:174-190` |
| P2 | Planning fallback masks outages and bad output | A planner failure is converted to a non-mission fallback, so the controller cannot distinguish provider failure, malformed JSON, and a genuine chat request. | `internal/core/mission.go:95-108` |
| P2 | Test signal is insufficient | Only five Go test files exist, and no end-to-end CTF benchmark proves a flag can be found. |

## Why the current CTF chains fail in practice

The CTF-PWN chain is a representative example. It says to run analysis tools, decompile, find gadgets, and write an exploit, but does not specify where the binary came from, how output becomes structured facts, which facts satisfy the next precondition, how a crash is reproduced, or how flag success is checked. A model must invent all of that while choosing from a large catalogue. One malformed tool call or an unrecognised command returns prose and the loop continues without a reliable recovery policy.

The same failure mode applies to web, crypto, and forensics. A prompt can suggest a process; it cannot enforce it.

## Target architecture: a small execution kernel

Build an explicit controller for local, authorised CTF challenges before expanding any other mode.

```text
Task intake → challenge classifier → typed workflow
                              ↓
                       Step scheduler
                              ↓
             tool adapter → observation normalizer
                              ↓
              evidence/artifact store + state reducer
                              ↓
              verifier ── success / recover / escalate
```

### Core contracts

1. **Task contract**: mode, challenge path or local URL, permitted scope, success predicate (for example a supplied flag regex), budget, and output directory.
2. **Workflow contract**: ordered DAG of steps with `preconditions`, typed inputs, a tool adapter, timeout, expected observation kinds, and recovery edges.
3. **Observation contract**: `command`, redacted arguments, exit status, stdout/stderr summaries, artifact IDs, parsed facts, duration, and error class.
4. **Evidence store**: append-only observations and content-addressed artifacts. Graph facts must reference their originating observation ID.
5. **Verifier**: deterministic checks for workflow completion and flag format; the LLM cannot declare success by prose alone.
6. **Controller policy**: a bounded retry and pivot budget per step, with explicit failure states (`tool_missing`, `bad_input`, `timeout`, `no_signal`, `contradiction`).

The model should choose among a small number of valid next steps and explain ambiguous observations. It should not be responsible for enforcing the workflow.

## Delivery roadmap

### Phase 0 — Stop widening the product (1–2 days)

- Freeze new high-risk/offensive tool additions.
- Make `ctf-local` an explicit mode limited to user-supplied local artifacts or an allowlisted local container/network.
- Create a baseline benchmark: 10 small, legal local fixtures across forensics, crypto, web, and binary categories. Record completion, correct flag, tool calls, elapsed time, and intervention count.
- Add a runtime preflight that reports exactly what is missing. Do not silently fall back from a failed workflow.

Exit criterion: every fixture can be launched and reset reproducibly; baseline results are committed as machine-readable JSON.

### Phase 1 — Make one path dependable (1 week)

- Implement `Task`, `Step`, `Observation`, `Artifact`, `Fact`, and `RunState` Go types.
- Implement a sequential workflow runner with durable step transitions and per-step contexts.
- Convert **CTF forensics** first: inspect file → metadata/strings → embedded-content extraction → deterministic flag scan → verifier.
- Replace raw string returns in its adapters with structured `ToolResult` values; preserve raw logs as artifacts.
- Add fixture-driven integration tests, including missing tool, timeout, no-output, and incorrect-flag cases.

Exit criterion: at least 90% correct completion on the frozen forensics fixtures over three clean runs, with every claimed fact linked to an observation.

### Phase 2 — Add stateful reasoning (1–2 weeks)

- Add a fact extractor/normalizer for each tool and a contradiction check before facts are promoted.
- Give the model a compact state view: current step, accepted facts, artifacts, remaining budget, and permitted next actions.
- Add deterministic recovery policies before LLM escalation (retry only transient failures; install only approved missing dependencies; branch on known error classes).
- Make “ask operator” an event/channel wait with a question and a resumable run ID, not a spin loop.

Exit criterion: interrupted runs resume without losing state, and failure classification is correct in at least 95% of injected failure tests.

### Phase 3 — Expand by vertical slice (2–4 weeks)

- Add CTF crypto, then web, then binary workflows. Each must have its own fixtures, typed tools, postconditions, and verifier.
- Keep tool definitions mode-scoped: a crypto task should not receive a huge network/payload tool catalogue.
- Retire prompt-only attack chains or convert each into a tested workflow DAG.

Exit criterion: 80%+ correct flag capture per category on a held-out local benchmark; no flag claim without verifier evidence.

### Phase 4 — Controlled parallelism (after deterministic runs work)

- Give each worker an isolated workspace and an immutable snapshot of state.
- Allow parallelism only for independent, declared branches.
- Merge observations through one reducer; reject conflicting facts rather than racing on shared maps/files.

Exit criterion: parallel runs are reproducible and do not change the result of the same fixture compared with serial execution.

## Implementation order

1. Add types and a JSONL run ledger under `internal/agent` or a new `internal/runtime` package.
2. Change sandbox execution to return an exit code and typed error class; never turn non-zero status into a successful nil error.
3. Add `WorkflowRunner` and implement only `ctf-local/forensics` adapters.
4. Change the TUI from prompt injection to submitting a `Task` to the runner.
5. Add a deterministic `FlagVerifier` and integration fixtures.
6. Refactor memory updates to require an observation ID and reject ungrounded claims.
7. Reduce the tool set exposed to each mode; preserve advanced tools behind explicit, separate modes.
8. Only then add LLM planning as a constrained selection mechanism.

## Metrics that reveal real agency

Track these per fixture and in CI:

- verified solve rate;
- median tool calls and wall-clock time to verification;
- recovery rate after injected tool failure;
- invalid-tool-call rate;
- fraction of facts with observation provenance;
- unverified-success rate (target: zero);
- human interventions per solved task;
- reproducibility across three fresh runs.

Do not use tool count, model eloquence, or number of listed skills as capability metrics.

## Immediate acceptance suite

Before claiming improvement, the project should pass:

1. A file-forensics fixture whose flag is in metadata.
2. A file-forensics fixture requiring extraction of an embedded archive.
3. A crypto fixture with a deterministic decoding path.
4. A small local web fixture with an explicitly permitted route and a known flag predicate.
5. A binary fixture where the runner records architecture/protections and correctly reports `needs-human` if it lacks a supported exploit path.
6. Negative cases: missing binary, missing dependency, non-zero command, timeout, malformed model action, and a fake flag.

Each test must assert the final run state, verifier result, and provenance—not merely that the agent printed a convincing answer.

## Validation performed for this audit

- Source review completed for orchestration, planning, target routing, tool dispatch, swarm handling, sandbox execution, and tests.
- `go test ./...` was attempted with caches redirected to `/tmp`. It could not download dependencies because this environment denies network/DNS access, so the suite was not executable here. This is an environment limitation, not a passing result.

## Bottom line

Make DrogonClaw smaller before making it broader. A single verified local-CTF workflow with typed observations, recovery logic, and a success oracle will feel far more alive than dozens of uncoordinated tools. Once that kernel solves a benchmark repeatably, extend it one category at a time.
