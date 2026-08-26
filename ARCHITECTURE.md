# DrogonClaw Architecture

This document describes the current architecture of DrogonClaw, including all subsystems implemented as of August 2026.

## Overview

DrogonClaw is an autonomous AI penetration testing agent built in Go. It uses a ReAct (Reasoning & Acting) loop to plan, execute, and adapt security assessments against target infrastructure.

```
┌─────────────────────────────────────────────────────────────────┐
│                    User Interfaces                              │
│  ┌──────────┐  ┌───────────┐  ┌──────────┐                    │
│  │   TUI    │  │ Telegram  │  │  Daemon  │                    │
│  │(Bubbletea)│  │   C2 GW   │  │(headless)│                    │
│  └────┬─────┘  └─────┬─────┘  └────┬─────┘                    │
│       └───────────────┼─────────────┘                          │
└───────────────────────┼─────────────────────────────────────────┘
                        │
┌───────────────────────┼─────────────────────────────────────────┐
│              ReAct Orchestrator                                 │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────────┐     │
│  │   Mission   │  │   Evidence   │  │    Skill          │     │
│  │   Planner   │  │   Validator  │  │    Learner        │     │
│  └──────┬──────┘  └──────┬───────┘  └────────┬──────────┘     │
│         └────────────────┼───────────────────┘                 │
│                          │                                      │
│  ┌───────────────────────┼──────────────────────────────┐      │
│  │              Tool Registry                           │      │
│  │  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐ │      │
│  │  │Nmap     │ │Nuclei    │ │Gobuster  │ │FFUF     │ │      │
│  │  │Wrapper  │ │Wrapper   │ │Wrapper   │ │Wrapper  │ │      │
│  │  └─────────┘ └──────────┘ └──────────┘ └─────────┘ │      │
│  │  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐ │      │
│  │  │SQLMap   │ │Subfinder │ │HTTPX     │ │Hydra    │ │      │
│  │  │Wrapper  │ │Wrapper   │ │Wrapper   │ │Wrapper  │ │      │
│  │  └─────────┘ └──────────┘ └──────────┘ └─────────┘ │      │
│  └──────────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────────┘
                        │
┌───────────────────────┼─────────────────────────────────────────┐
│              Intelligence Layer                                 │
│  ┌────────────┐ ┌───────────┐ ┌──────────┐ ┌──────────────┐  │
│  │  Session   │ │ Auto      │ │ Response │ │   WAF        │  │
│  │  Manager   │ │ Throttle  │ │ Cache    │ │   Detector   │  │
│  │(per-domain)│ │(adaptive) │ │(disk)    │ │(6 WAFs)      │  │
│  └────────────┘ └───────────┘ └──────────┘ └──────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                        │
┌───────────────────────┼─────────────────────────────────────────┐
│              Execution Layer                                    │
│  ┌──────────────┐ ┌───────────────┐ ┌────────────────────┐    │
│  │  Subagent    │ │  Swarm        │ │  Docker Sandbox    │    │
│  │  Manager     │ │  Commander    │ │  (ephemeral)       │    │
│  │  (parallel)  │ │  (goroutines) │ │                    │    │
│  └──────────────┘ └───────────────┘ └────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                        │
┌───────────────────────┼─────────────────────────────────────────┐
│              Storage Layer                                      │
│  ┌────────────┐ ┌───────────┐ ┌──────────┐ ┌──────────────┐  │
│  │Intelligence│ │  LootDB   │ │  Action  │ │   Learned    │  │
│  │  Graph     │ │  (SQLite) │ │  Journal │ │   Skills     │  │
│  │  (JSON)    │ │           │ │          │ │   (JSON)     │  │
│  └────────────┘ └───────────┘ └──────────┘ └──────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. ReAct Orchestrator (`internal/agent/orchestrator.go`)

The brain of DrogonClaw. A hand-rolled Go ReAct loop that:

1. Receives a user objective
2. Generates a mission plan via the Mission Planner
3. Injects learned attack patterns from the Skill Learner
4. Builds LLM messages with memory graph context
5. Loops: LLM reasons → calls tool → evaluates evidence → repeats
6. Emits events to the TUI/channel for real-time display

**Key features:**
- Bounded execution (30-minute run budget, 20-iteration cap)
- Mission plan injection into LLM context with step tracking
- Evidence verification footer on every tool result
- History management (retains last 24 messages)
- Parallel task execution via SubagentManager

### 2. Mission Planner (`internal/core/mission.go`)

Breaks down objectives into executable steps:
- Uses the LLM to generate a structured `MissionPlan`
- Each step has: Action, Target, ExpectedOutcome, Status
- Plan is injected into the LLM context for step-by-step execution
- Greedy regex parser extracts plan from LLM output

### 3. Evidence Validator (`internal/agent/validator.go`)

Prevents LLM hallucination by verifying tool outputs:
- Checks that tool calls are well-formed
- Validates response choices exist (bounds check)
- Returns structured validation results

### 4. Success Oracle (`internal/agent/tools.go`)

Deterministic verification of claimed success:
- Regex-based flag pattern matching (`flag{...}`, `CTF{...}`)
- Only verified evidence counts as success
- Prevents the model from declaring success by prose alone

## Intelligence Layer

### 5. Session Manager (`internal/httputil/session.go`)

Maintains persistent HTTP sessions per domain:
- Cookie jars that survive across tool calls
- Login once, enumerate everything
- Custom headers per domain (Authorization, API keys)
- Session state persisted to disk (`data/sessions/`)

### 6. AutoThrottle (`internal/httputil/session.go`)

Adaptive rate limiting per domain (inspired by Scrapling):
- Measures response time per domain
- Speeds up when targets respond quickly
- Exponential backoff on 429/503 responses
- Respects `Retry-After` headers
- Configurable min/max delay bounds

### 7. Response Cache (`internal/httputil/session.go`)

Caches tool responses to disk:
- Prevents re-hitting targets for identical queries
- Configurable TTL (default: 1 hour)
- Deterministic key generation from tool + args
- Hit count tracking for analytics

### 8. WAF Detector (`internal/httputil/session.go`)

Identifies Web Application Firewalls and provides bypass guidance:
- Cloudflare (Turnstile, Interstitial)
- Akamai
- AWS WAF
- ModSecurity
- Imperva/Incapsula
- Sucuri
- Returns actionable bypass hints for each

### 9. Skill Learner (`internal/agent/skilllearn.go`)

Learns from successful attacks (inspired by Hermes Agent):
- After verified success, saves technique as a reusable skill
- Target classification (wordpress, apache, ssh, mysql, etc.)
- Success rate tracking per skill
- Injects learned patterns into LLM context per target
- Persists to `data/learned_skills/`

## Execution Layer

### 10. Subagent Manager (`internal/agent/subagent.go`)

Enables parallel task execution:
- Dependency-aware scheduling (`DependsOn`)
- Concurrency limiting (default: 5)
- Preset task bundles: `StandardReconTasks`, `FullWebReconTasks`, `CloudTasks`
- Results merged and formatted for LLM context

### 11. Swarm Commander (`internal/agent/swarm.go`)

Goroutine-based parallel execution:
- Topology-based concurrent attack vectors
- Isolated contexts per swarm agent
- Race condition safe (unique temp file paths)

### 12. Docker Sandbox (`internal/sandbox/docker.go`)

Isolated tool execution:
- Ephemeral Docker containers
- Host fallback with fail-closed behavior
- Persistent shell state within sessions
- Human-in-the-loop gates for dangerous operations

## Storage Layer

### 13. Intelligence Graph (`internal/memory/graph.go`)

JSON-backed knowledge graph:
- Typed nodes: Target, Asset, Port, Service, Vulnerability, Credential, Flag, Task
- Typed edges with relationships
- Snapshot method for LLM context injection
- Thread-safe (sync.RWMutex)
- Atomic writes (temp file + rename)

### 14. LootDB (`internal/memory/loot.go`)

SQLite database for structured findings:
- Vulnerabilities with CVE, severity, description
- Credentials with type, username, password
- Flags with value, challenge, platform
- Query by target, type, severity

### 15. Action Journal (`internal/memory/journal.go`)

Durable execution state:
- Tracks in-flight tool calls
- Enables crash recovery
- Records completed steps
- Supports resume after interruption

## Tool Wrappers (`internal/agent/toolwrappers.go`)

Typed wrappers for common pentest tools:

| Tool | Wrapper | Key Modes |
|------|---------|-----------|
| Nmap | `run_nmap` | quick, udp, vuln, stealth, full |
| Nuclei | `run_nuclei` | severity filtering, DAST mode |
| Gobuster | `run_gobuster` | dir, vhost, dns |
| FFUF | `run_ffuf` | directory fuzzing, parameter fuzzing |
| SQLMap | `run_sqlmap` | automatic SQL injection testing |
| Subfinder | `run_subfinder` | subdomain enumeration |
| HTTPX | `run_httpx` | web probing, tech detection |
| Hydra | `run_hydra` | credential brute-forcing |

All wrappers use `shellutil.Quote()` for safe argument escaping.

## Safety Systems

### Human-in-the-Loop (`internal/core/hitl.go`)

Tiered approval system:
- **Long-running tools** — Operator approval required in manual mode
- **Autonomous code tools** — Always require approval
- **Ghost tools** — Always require approval
- **Autopilot mode** — Skips all approvals

### Prompt Injection Defense (`internal/agent/orchestrator.go`)

Sanitizes external tool outputs:
- Strips XML-like tags
- Detects common injection patterns
- Flags suspicious output without stripping

### Dynamic Skill Denylist (`internal/adapt/skills.go`)

21 blocked patterns including:
- Reverse shells (`bash -i`, `nc -e`, `mkfifo`)
- Dangerous commands (`curl|sh`, `chmod 777`, `rm -rf /`)
- Network manipulation (`iptables`, `crontab`)
- Privilege escalation attempts

## Configuration

All configuration stored in `~/.drogonclaw/config.json`:
- Provider settings (API keys, model selection)
- Telegram C2 credentials
- Sandbox mode (Docker vs native)
- OPSEC stealth level
- Secondary API keys (Shodan, VirusTotal, etc.)

No environment variable overrides — the setup wizard is the single source of truth.
