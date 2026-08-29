# Changelog

All notable changes to DrogonClaw are documented here.

## [Unreleased] — 2026-08-29

### Added

#### Grounding Guard (anti-hallucination)
- **Deterministic grounding guard** (`internal/agent/grounding.go`) — before the
  final answer is surfaced, a rules-only check cross-references every claim
  against the recorded tool evidence. Fabricated interface names (`eth0`, `lo`),
  invented IPs, and any claim that denies WiFi/wireless hardware while tools
  observed a wireless interface get corrected inline with an `[AUTO-GROUNDING]`
  note and a status event. No LLM call: it cannot be skipped by prompt drift.
- **Prompt hardening** (`internal/agent/prompt.go`) — RUNTIME ENVIRONMENT and
  GROUND TRUTH RULE now instruct the model to only assert interfaces/IPs it has
  directly observed and never to deny the existence of wireless hardware.
- **Regression pins** — `TestGroundingCatchesLiveTranscript` replays the exact
  `wlan0 → P4X` WiFi recon transcript that previously produced a fabricated
  `eth0 172.17.0.2/16` answer and asserts the correction fires.

#### Command Registry (TUI)
- **Single slash-command registry** (`internal/tui/commands.go`) — the 28
  in-TUI commands (was a 60-complexity `switch`) are now one declarative table
  with canonical name, aliases, category, description, argument hint and handler.
  Built from per-category builders (OPERATIONS / CONTROLS / SESSION / UI).
- `/help` and `/commands` now render **straight from the registry**, so an
  undocumented or orphaned command is impossible by construction.
- **`/config`** — show the stored provider, model, Telegram and OSINT-key
  configuration summary (secrets masked) directly in the terminal.
- Async-task boilerplate (`/profile`, `/ctf`, `/report`, `/swarm`, `/resume`)
  extracted into `beginTask`, which returns a cancellable `context.Context` that
  every long-running handler now honors.

#### CLI Dispatch & Versioning
- **Command table + `help`** (`cmd/drogonclaw/cli.go`) — `./drogonclaw help`
  renders a graphical sub-command reference; unknown subcommands now fail with
  exit 2 instead of silently dropping into the TUI.
- **`version` command** — prints build version, build time and Go runtime.
- **Build metadata injection** — Makefile now links `main.version` and
  `main.buildTime` via `-X` ldflags (`drogonclaw version` shows the git
  describe tag).

#### Setup Wizard Redesign
- **Menu-driven controller** (`internal/tui/setup.go`) — instead of one linear
  pass, the wizard now shows a **Current Configuration summary** (provider,
  model, API-key status, Telegram state, OSINT keys, identity — all secrets
  masked) and a menu: change provider/model, manage Telegram, manage secondary
  keys, edit identity, or reset everything.
- **Returning-user friendly** — the authorisation gate is asked once and
  persisted; stored values are prefilled so an API key can be viewed/changed
  without being re-entered.
- **Telegram is now reviewable** — re-running setup shows the current token
  (masked) and chat ID and offers keep / replace / disable, answering the
  previous gap where already-configured users could not inspect the gateway.
- **Graceful cancellation** — every section returns to the menu; the wizard no
  longer calls `os.Exit(1)` mid-flow (which would previously abort the whole
  process).
- Credential display uses a fixed-width mask revealing only the last 4
  characters; the renderer reads config through a `configReader` interface so it
  never depends on the operator's live config in tests.

### Changed
- TUI help: stale leader keys (`l`/`m`/`t`/`e`/`x`) and dead keyboard hints
  removed; `/help` now groups commands by category and lists only live bindings.
- Palette and inline hint bar derive from the registry (order = category order).
- `main.go` dispatch reduced from a flat if-chain to `parseCLI` +
  `runStandaloneCommand` (`main` gocyclo 28 → 18).
- Makefile `doctor` target now runs the real `health` subcommand (previously
  invoked a nonexistent `doctor` subcommand).
- **Setup wizard modernization** — the flat text headings are replaced by the
  TUI's own dark-chrome visual language: a rounded-border brand banner, filled
  accent "pill" section headers (`AUTH` / `CONFIG` / `ACTION` / `PROVIDER` /
  `C2` / `KEYS` / `PROFILE` / `RESET`), a boxed `CURRENT CONFIGURATION` panel
  with accent label columns and green/⇢ status markers, and a bordered
  completion panel. Purely presentational — flows and prompts unchanged.
- **CLI help/version modernization** — `drogonclaw help` and `drogonclaw
  version` now present their title + usage/build lines inside a rounded
  bordered banner matching the wizard (GitHub-dark palette).

### Documentation
- `docs/setup.md` rewritten for the menu-driven wizard and `/config` command.
- This changelog — entries cover the grounding guard, command registry, CLI
  dispatch/versioning and setup redesign above.

### Tests
- `internal/tui/setup_test.go` — `maskSecret`, `providerLabel`, `storedKeyStatus`,
  `telegramStatus`, `renderConfigSummary` (cloud + ollama paths), secondary-key
  table sync, provider→config-key mapping, curated model options, reset-list
  coverage.
- `internal/tui/commands_test.go` — registry consistency (unique aliases,
  canonical names, valid categories, `/config` present), bare `/` help,
  unknown-command warning, alias-to-entry resolution.
- `cmd/drogonclaw/cli_test.go` — `parseCLI` for every subcommand, run modes,
  flag forwarding, unknown rejection, help/version rendering.

---

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
