# DrogonClaw Platform

DrogonClaw is an autonomous AI penetration testing platform designed to execute full-chain security assessments against authorized target infrastructure.

## Architecture

### Red Team (Offensive Operations)

- **ReAct Loop Engine**: Custom Go-based reasoning loop that plans, executes, and adapts to the environment
- **Mission Planner**: Breaks down objectives into executable steps with expected outcomes
- **Evidence Validator**: Verifies tool outputs to prevent LLM hallucination
- **Intelligence Graph**: JSON-backed memory storing targets, assets, ports, services, vulnerabilities, credentials, and flags
- **Docker Sandbox**: Isolated tool execution with host fallback
- **Session Management**: Persistent cookie jars per domain with adaptive rate limiting
- **Skill Learning**: Learns from successful attacks and reuses techniques on similar targets
- **Parallel Subagents**: Independent tasks run concurrently for faster recon

### Blue Team (Defensive Operations)

- **CIS Benchmark Scanner**: Shell-based OS hardening validation
- **Threat Hunting**: YARA rule matching and IOC scanning
- **Compliance Mapping**: Findings mapped to PCI-DSS, SOC 2, HIPAA
- **Incident Response**: Automated playbooks for active intrusions
- **Vulnerability Lifecycle Management**: Asset-based CVSS patch prioritization

## Getting Started

### Requirements

- Go 1.26+
- Docker (daemon running)

### Installation

```bash
git clone https://github.com/0xP4X/drogonclaw.git
cd drogonclaw
go mod tidy
go build -o drogonclaw ./cmd/drogonclaw/
```

### Configuration

Run the interactive setup wizard:

```bash
./drogonclaw setup
```

The wizard configures:
1. Authorization scope/compliance acknowledgement
2. Neural provider (OpenRouter, NVIDIA NIM, OpenAI, Gemini, Ollama)
3. Provider API credentials
4. Optional Telegram C2 gateway
5. Optional secondary API keys (Shodan, VirusTotal, GitHub, etc.)

Configuration is stored in `~/.drogonclaw/config.json` (owner read/write only).

### Local API

The REST API (`internal/api/server.go`) is an optional local control-plane component. It is not started by the CLI or Docker image by default.

- **Authentication**: Static Bearer token via `DROGONCLAW_API_KEY` environment variable
- **Transport**: No TLS by default — use `StartTLSServerAt` for non-loopback deployment
- **Authorization**: No RBAC — single-token authentication only
- **Binding**: Loopback only unless explicitly configured otherwise

### Telegram C2

Control DrogonClaw from your phone via the whitelisted Telegram bot — live
mission panel, approval buttons and operator commands. Full protocol, command
reference and tips in [telegram.md](telegram.md).

### TUI Commands

- `/health` — Run sandbox/toolkit diagnostics
- `/status` — Print runtime state and memory graph counts
- `/skills` — Show loaded modules by category
- `/skills <term>` — Search modules by name or description
- `/ctf <path>` — Offline CTF triage with flag detection
- `/auto` — Toggle autopilot mode
- `/help` — Full command reference

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+P` | Command palette |
| `Ctrl+A` | Toggle autopilot |
| `Ctrl+B` | Toggle sidebar |
| `Ctrl+S` | Show status |
| `Ctrl+D` | Show cost |
| `Ctrl+E` | Open pager |
| `Ctrl+Y` | Copy output |

## Legal Disclaimer

DrogonClaw is built for authorized security testing only. Never deploy offensive capabilities against networks without explicit written consent.
