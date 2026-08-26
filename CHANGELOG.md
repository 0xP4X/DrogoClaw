# Changelog

All notable changes to DrogonClaw are documented here.

## [Unreleased] — 2026-08-26

### Added

#### Scrapling-Inspired Capabilities
- **Session Persistence** (`internal/httputil/session.go`) — Persistent cookie jars per domain. Login once, enumerate everything across tool calls. Sessions saved to disk.
- **AutoThrottle** (`internal/httputil/session.go`) — Adaptive rate limiting per domain. Measures response time, speeds up when targets allow, backs off exponentially on 429/503. Respects `Retry-After` headers.
- **Response Cache** (`internal/httputil/session.go`) — Tool responses cached to disk with configurable TTL. Prevents re-hitting targets for identical queries.
- **WAF Detection** (`internal/httputil/session.go`) — Detects Cloudflare, Akamai, AWS WAF, ModSecurity, Imperva/Incapsula, Sucuri. Returns actionable bypass hints for each.
- **ThrottledTransport** (`internal/httputil/session.go`) — `http.Transport` wrapper with automatic rate limiting per domain.

#### Hermes-Inspired Capabilities
- **Skill Learner** (`internal/agent/skilllearn.go`) — After verified successes, saves techniques as reusable skills. Target classification (wordpress, apache, ssh, mysql, etc.). Success rate tracking. Injects learned patterns into LLM context per target.
- **Parallel Subagents** (`internal/agent/subagent.go`) — Independent recon/exploitation tasks run concurrently with dependency-aware scheduling. Concurrency limiting. Preset task bundles: `StandardReconTasks`, `FullWebReconTasks`, `CloudTasks`.
- **Trajectory Compression Placeholder** (`internal/agent/skilllearn.go:SaveHistory`) — Foundation for future trajectory compression.

#### Bug Fixes
- Fix duplicate history append in orchestrator (final assistant message was appended twice)
- Fix evidence evaluation discard (evidence footer now always appended to tool results)
- Fix validator panic on empty response choices (added bounds check)
- Fix deprecated `Temporary()` usage in provider (removed, kept `Timeout()`)
- Fix swarm graph race condition (unique temp file paths using `UnixNano()`)
- Fix mission plan not injected into LLM context (plan now injected as system message)
- Fix greedy regex in mission parser (`\{.*\}` → `\{.*?\}`)
- Fix prompt directive numbering (added directives 8 and 9, renumbered 10)
- Fix Docker/native contradiction in prompt (replaced hardcoded "NOT in container" with runtime-accurate instructions)
- Fix `format string %!s(MISSING)` errors in prompt.go

#### Security Fixes
- Add `shellutil.Quote()` for safe shell argument escaping across all tool wrappers (nmap, nuclei, gobuster, ffuf, sqlmap, subfinder, httpx, hydra)
- Add human-in-the-loop gates for autonomous code tools (`write_and_run_script`, `autonomous_fuzzing_engine`, `dynamic_payload_compiler`, `advanced_web_exploiter`, `zero_click_exploiter`, `async_race_condition_engine`) and ghost tools
- Sanitize external tool outputs before LLM context injection (strip XML tags, detect injection patterns)
- Expand dynamic skill denylist from 6 to 21 patterns (reverse shells, `curl|sh`, `chmod 777`, `rm -rf /`, iptables, crontab, etc.)

#### Performance
- Cache `Definitions()` result in ToolRegistry (computed once, reused on every LLM call)

#### Cleanup
- Consolidate dual LootDB implementations (removed `internal/core/loot.go`, `memory.LootDB` is the active implementation)

#### Documentation
- Complete README.md rewrite with all new capabilities
- Complete ARCHITECTURE.md rewrite with all subsystems documented
- New `docs/TOOLS.md` — Complete tool reference catalog with parameters
- New `docs/SESSION_MANAGEMENT.md` — Session persistence, AutoThrottle, response cache, WAF detection
- New `docs/SKILL_LEARNING.md` — Learned attack patterns system
- New `docs/SUBAGENTS.md` — Parallel execution framework
- New `CHANGELOG.md` — This file
- New `AGENTS.md` — AI agent instructions
- Fix `docs/INDEX.md` — Remove false JWT/RBAC/TLS claims
- New `RELATED_PROJECTS_RESEARCH.md` — Research on Hermes Agent, Scrapling, OpenClaw
- New `CAPABILITY_IMPROVEMENTS.md` — 6-phase improvement plan

#### Tests
- Add CVSS scorer tests (`internal/cvss/scoring_test.go`)
- Add exploit parser tests (`internal/redteam/exploitation/parser_test.go`)
- Add memory graph tests (`internal/memory/graph_test.go`)
- Add HTTP session/throttle/cache/WAF tests (`internal/httputil/session_test.go`)
- Add skill learner tests (`internal/agent/skilllearn_test.go`)
- Add subagent tests (`internal/agent/subagent_test.go`)

### Changed
- TUI: Add keybind hints for recognition over recall (Ctrl+P, Ctrl+A, Ctrl+B in status bar)
- TUI: Add KEYBINDS section to sidebar
- TUI: Update help text with all keyboard shortcuts
- TUI: Fix sidebar vertical spacing (removed forced height, added top padding)
- TUI: Fix status bar separators (dot notation)
- Orchestrator: RecordVerifiedFinding now teaches the Skill Learner
- Orchestrator: Learned attack patterns injected into LLM context per target
- ToolRegistry: Added `skillLearner` field and `GetSkillLearner()`/`GetLearnedContext()` methods
- Orchestrator: Added `subagents` field and `GetSubagents()`/`ExecuteParallelTasks()` methods

### Removed
- `internal/core/loot.go` — Consolidated to `internal/memory/loot.go`

---

## [0.1.0] — 2026-08-11

### Added
- Initial release
- ReAct orchestration core
- Intelligence graph (JSON-backed)
- LootDB (SQLite)
- Docker sandbox execution
- Structured tool wrappers (Nmap, Nuclei, Gobuster, FFUF, SQLMap, Subfinder, HTTPX, Hydra)
- Verified exploit templates (EternalBlue, Log4Shell, PrintNightmare, MS08-067, Spring4Shell)
- Active Directory arsenal (Impacket/BloodHound)
- 7-state exploit parser
- Dual-layer CVE intelligence
- Human-in-the-Loop safety
- Telegram C2 gateway
- TUI with Bubbletea/Lipgloss
- Evidence validation pipeline
- OPSEC stealth modes
- Mission planning
- Action journal for crash recovery
