# DrogonClaw

<div align="center">
  <img src="assets/logo.png" alt="DrogonClaw Logo" width="300" />
</div>

<div align="center">

[![CI](https://github.com/0xP4X/drogonclaw/actions/workflows/CI-CD.yml/badge.svg)](https://github.com/0xP4X/drogonclaw/actions/workflows/CI-CD.yml)
[![License](https://img.shields.io/badge/license-AGPL%20v3-000000?logo=gnu&logoColor=white)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26.1-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Platform](https://img.shields.io/badge/platform-Linux%20only-important)](docs/INDEX.md)
[![Docker](https://img.shields.io/badge/sandbox-Docker-2496ED?logo=docker&logoColor=white)](docker-compose.yml)
[![Security](https://img.shields.io/badge/security-policy-0f766e)](SECURITY.md)

</div>

> **Autonomous AI-Powered Penetration Testing Platform**

DrogonClaw is an autonomous AI agent that plans, executes, and adapts penetration tests against target infrastructure. It operates as a **ReAct-based intelligence core** with persistent memory, parallel subagents, learned attack patterns, and sandboxed tool execution.

[Quick Start](#quick-start) | [Capabilities](#capabilities) | [Architecture](#architecture) | [Tools](docs/TOOLS.md) | [Commands](#commands) | [Development](#development) | [Setup Guide](docs/setup.md)

---

## Warning

> [!WARNING]
> **Linux Only — Authorized Testing Only**
> DrogonClaw is designed for **Linux-based operating systems** (Kali, Ubuntu, Debian). It relies on Linux-specific networking APIs, filesystem permissions, and process management. **Never** use it against systems without explicit written authorization.

---

## Capabilities

### Core Engine
- **ReAct Orchestration** — Hand-rolled Go reasoning loop (no Node.js/LangChain overhead) with mission planning, evidence verification, and self-correction.
- **Persistent Intelligence Graph** — JSON-backed memory storing targets, assets, ports, services, vulnerabilities, credentials, and flags with typed relationships.
- **Sandboxed Execution** — Docker-isolated tool execution with host fallback that fails closed.
- **Structured Tool Wrappers** — Typed wrappers for 10+ pentest tools with best-practice flag defaults.

### Intelligence & Exploitation
- **7-State Exploit Parser** — Every exploit attempt classified into actionable states: `SUCCESS_SHELL`, `SUCCESS_SILENT`, `PATCHED`, `FILTERED`, `WRONG_ARCH`, `AUTH_REQUIRED`, `CRASHED`.
- **Verified Exploit Templates** — Pre-compiled chains for critical CVEs (EternalBlue, Log4Shell, PrintNightmare, MS08-067, Spring4Shell).
- **Active Directory Arsenal** — Impacket/BloodHound wrappers (Kerberoasting, Pass-the-Hash, DCSync).
- **Dual-Layer CVE Intelligence** — Rolling 120-day NVD cache + offline database of 100 classic CTF/pentest CVEs.
- **Binary Triage** — Automated `checksec`, `strings`, `nm`, `gdb` fed into LLM context.

### New Capabilities (Scrapling + Hermes Inspired)
- **Session Persistence** — Persistent cookie jars per domain. Login once, enumerate everything across tool calls. ([docs/SESSION_MANAGEMENT.md](docs/SESSION_MANAGEMENT.md))
- **Adaptive Rate Limiting (AutoThrottle)** — Per-domain rate control that speeds up when targets allow and backs off on 429/503. Respects `Retry-After` headers.
- **Response Caching** — Tool responses cached to disk for retry without re-hitting targets.
- **WAF Detection & Bypass Hints** — Detects Cloudflare, Akamai, AWS WAF, ModSecurity, Imperva, Sucuri with actionable bypass guidance.
- **Skill Learning** — After verified successes, techniques are saved as reusable skills. Future targets get automatically matched against learned patterns. ([docs/SKILL_LEARNING.md](docs/SKILL_LEARNING.md))
- **Parallel Subagents** — Independent recon/exploitation tasks run concurrently with dependency-aware scheduling. 3-5x faster recon. ([docs/SUBAGENTS.md](docs/SUBAGENTS.md))

### Safety & Control
- **Human-in-the-Loop (HitL)** — Dangerous actions halt for operator approval with tiered risk levels.
- **Autonomous Code Gates** — Script execution, ghost tools, and reverse shells always require approval.
- **Prompt Injection Defense** — External tool outputs sanitized before LLM context injection.
- **Dynamic Skill Denylist** — 21 patterns blocked (reverse shells, `curl|sh`, `chmod 777`, etc.).
- **Memory Failure Loops** — Failed command syntax tracked globally; agent cannot repeat the same mistake.

### Interfaces
- **TUI** — Professional terminal interface with sidebar, status bar, command palette (Ctrl+P), leader key system (Ctrl+X).
- **Telegram C2 Gateway** — Remote mobile control via Telegram bot.
- **Headless Daemon** — Runs without terminal for automated scanning.

---

## Quick Start

### Requirements

- Go `1.26+`
- Docker (daemon running, for sandbox execution)

### Install (From Source)

```bash
git clone https://github.com/0xP4X/drogonclaw.git
cd drogonclaw
go mod tidy
go build -o drogonclaw ./cmd/drogonclaw/
```

### Install (From npm)

```bash
npm install -g drogonclaw
drogonclaw setup
drogonclaw
```

### Configure

Run the interactive setup wizard:

```bash
./drogonclaw setup
```

The wizard configures:
1. **Authorization** — Scope/compliance acknowledgement
2. **Neural Provider** — OpenRouter, NVIDIA NIM, OpenAI, Google Gemini, or Ollama
3. **Credentials** — Provider API key
4. **Remote C2** (optional) — Telegram bot token + chat ID
5. **Secondary API Keys** (optional) — GitHub, Shodan, VirusTotal, Brave Search, Hunter.io, Exa

Configuration stored in `~/.drogonclaw/config.json` (owner read/write only).

### Start

```bash
./drogonclaw
```

### Daemon Mode (Headless)

```bash
./drogonclaw daemon
```

---

## Supported Providers

| Provider | Config Key | Notes |
| --- | --- | --- |
| OpenRouter | `openrouter` | Multi-model gateway; live model list with curated fallback |
| NVIDIA NIM | `nvidia` | High-performance inference (Nemotron, Qwen, DeepSeek, Llama) |
| OpenAI | `openai` | Direct API (`gpt-4o`, `gpt-4o-mini`) |
| Google Gemini | `gemini` | Enterprise reasoning (`gemini-2.5-pro`, `gemini-2.5-flash`) |
| Ollama | `ollama` | Offline runtime; set `OLLAMA_BASE_URL` |

---

## Architecture

```mermaid
flowchart TD
    subgraph UI ["User Interfaces"]
        CLI["CLI / TUI"]
        TG["Telegram C2"]
        DAEMON["Headless Daemon"]
    end

    subgraph Core ["DrogonClaw Engine (Go)"]
        Orchestrator["ReAct Orchestrator"]
        Planner["Mission Planner"]
        Evidence["Evidence Validator"]
        Graph[("Intelligence Graph")]
        LootDB[("LootDB (SQLite)")]
    end

    subgraph Intelligence ["Intelligence Layer"]
        Skills["Skill Learner"]
        Sessions["Session Manager"]
        Throttle["AutoThrottle"]
        Cache["Response Cache"]
        WAF["WAF Detector"]
    end

    subgraph Execution ["Execution Layer"]
        Swarm["Swarm Commander"]
        Subagents["Parallel Subagents"]
        Sandbox["Docker Sandbox"]
        Tools["Tool Wrappers"]
    end

    Target((("Target Network")))

    UI --> Orchestrator
    Orchestrator --> Planner
    Orchestrator --> Evidence
    Orchestrator <--> Graph
    Orchestrator --> Skills
    Orchestrator --> Subagents
    Subagents --> Swarm
    Swarm --> Sandbox
    Sandbox --> Tools
    Tools --> Sessions
    Sessions --> Throttle
    Sessions --> Cache
    Tools --> WAF
    Tools --> LootDB
    Tools --> Target
```

For detailed architecture documentation, see [ARCHITECTURE.md](ARCHITECTURE.md).

---

## Commands

### Keyboard Shortcuts

| Shortcut | Action |
| --- | --- |
| `Ctrl+P` | Command palette |
| `Ctrl+A` | Toggle autopilot |
| `Ctrl+B` | Toggle sidebar |
| `Ctrl+S` | Show status |
| `Ctrl+D` | Show cost |
| `Ctrl+E` | Open pager |
| `Ctrl+Y` | Copy output |
| `Ctrl+T` | New session |
| `Ctrl+C` | Abort execution |
| `Ctrl+X` then `b/n/l/m/t/e/x/q` | Leader key commands |

### Slash Commands

| Command | Description |
| --- | --- |
| `/setup` | Launch configuration wizard |
| `/skills` | Show loaded modules |
| `/skills <term>` | Search modules |
| `/status` | Runtime state + memory graph |
| `/health` | Sandbox/toolkit diagnostics |
| `/ctf <path>` | Offline CTF triage |
| `/profile <target>` | Passive target profiling |
| `/auto` | Toggle autopilot |
| `/persona` | Switch agent persona |
| `/stealth` | Adjust OPSEC posture |
| `/mode` | Switch operational mode |
| `/report` | Generate engagement report |
| `/swarm` | Manage agent swarm |
| `/analyze <path>` | Analyze artifact |
| `/sandbox` | Inspect sandbox runtime |
| `/queue` | Show pending tasks |
| `/sidebar` | Toggle sidebar |
| `/details` | Show last tool details |
| `/copy` | Copy transcript to clipboard |
| `/help` | Full command reference |

---

## Development

```bash
make build           # Build binary
make test            # Run tests with race detection
make lint            # Run golangci-lint
make run             # Build and launch TUI
make daemon          # Build and run headless
make docker-compose  # Start Docker Compose services
make test-cover      # Tests with coverage
make format          # Format code
make vet             # Run go vet
make skills          # Regenerate skill manifest
make doctor          # System diagnostics
make clean           # Clean build artifacts
```

### Repository Structure

```
cmd/drogonclaw/          CLI entrypoint
internal/
  agent/                 ReAct orchestrator, tools, subagents, skill learning
  httputil/              Session management, AutoThrottle, response cache, WAF detection
  memory/                Intelligence graph, loot DB, action journal
  sandbox/               Docker sandbox execution
  shell/                 Shell manager
  shellutil/             Shared shell utilities
  skills/                Skill manifest loader
  tui/                   Terminal UI (Bubbletea + Lipgloss)
  cvss/                  CVSS v3.1 scoring
  core/                  HitL, mission planning, loot types
  redteam/               Exploitation, evasion, lateral movement, post-exploitation
  ghost/                 Anti-forensics
  c2/                    C2 infrastructure
  cloud/                 Cloud recon (AWS, Azure, GCP)
  intel/                 OSINT feeds, Shodan, VirusTotal
  mitre/                 MITRE ATT&CK mapping
  adapt/                 Adaptive skill management
  whitebox/              White-box assessment
  benchmark/             Performance benchmarks
  billing/               Cost tracking
docs/                    Documentation
skills/                  Executable module definitions
scripts/                 Build and asset scripts
assets/                  Logos and charts
```

---

## Documentation

| Document | Description |
| --- | --- |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Detailed architecture with all subsystems |
| [docs/TOOLS.md](docs/TOOLS.md) | Complete tool reference catalog |
| [docs/SESSION_MANAGEMENT.md](docs/SESSION_MANAGEMENT.md) | Sessions, throttle, cache, WAF detection |
| [docs/SKILL_LEARNING.md](docs/SKILL_LEARNING.md) | Learned attack patterns system |
| [docs/SUBAGENTS.md](docs/SUBAGENTS.md) | Parallel execution framework |
| [docs/setup.md](docs/setup.md) | Detailed setup guide |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contribution guidelines |
| [SECURITY.md](SECURITY.md) | Vulnerability reporting |
| [CHANGELOG.md](CHANGELOG.md) | Version history |

---

## Security

DrogonClaw is built for authorized security testing only. Report vulnerabilities via [SECURITY.md](SECURITY.md). Never deploy against networks without explicit written consent.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Ensure `make build`, `make lint`, and `make test` pass before submitting PRs.

## Disclaimer

DrogonClaw is designed for **authorized security testing only**. Always ensure you have explicit permission before testing any system. Unauthorized access to computer systems is illegal.

## License

GNU AGPL v3

---

## Star History

![Star History](./assets/star-history.svg)
