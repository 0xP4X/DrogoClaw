# Related Projects Research — DrogonClaw Improvement Analysis

**Date:** 2026-08-26
**Projects Analyzed:**
1. Hermes Agent (NousResearch) — 237k stars, Python AI agent
2. Scrapling (D4Vinci) — 76.6k stars, Python adaptive web scraper
3. OpenClaw — 388k stars, TypeScript personal AI assistant

---

## 1. Hermes Agent — NousResearch/hermes-agent

### What It Is
A self-improving AI agent built by Nous Research. The only agent with a **built-in learning loop** — creates skills from experience, improves them during use, searches its own past conversations, and builds a deepening user model across sessions.

### Key Architectural Patterns Relevant to DrogonClaw

#### A. Closed Learning Loop (CRITICAL — DrogonClaw lacks this)
```
Hermes: Execute → Evaluate → Create Skill → Improve Skill → Persist Memory → Recall Next Session
DrogonClaw: Execute → Log → Forget (no learning, no skill creation)
```
**Hermes implements:**
- Agent-curated memory with periodic self-nudges
- Autonomous skill creation after complex tasks
- Skills that self-improve during use (versioned, diff-based improvement)
- FTS5 session search with LLM summarization for cross-session recall
- Honcho dialectic user modeling (builds understanding of the operator over time)

**DrogonClaw Gap:** We have `AdaptiveSkillManager` and `SkillMastery` but no autonomous skill creation, no self-improvement loop, no cross-session memory search, no user modeling.

#### B. Trajectory Compression
Hermes has `trajectory_compressor.py` — compresses conversation trajectories for:
1. Training data generation (next-gen tool-calling models)
2. Context window optimization (fits more history in fewer tokens)
3. Cost reduction (fewer tokens = lower API costs)

**DrogonClaw Gap:** We truncate history at `MaxHistoryTurns` (5) with no semantic compression. A trajectory compressor could keep 50+ turns of context at the same token cost.

#### C. Cron Scheduler
Built-in cron with natural language scheduling:
```
"Run a nightly backup at 2am" → creates cron entry → delivers results to Telegram/Discord/CLI
```
**DrogonClaw Gap:** No scheduled tasks. Can't do "scan this target every 6 hours" or "daily vulnerability report".

#### D. Subagent Delegation
Spawns isolated subagents for parallel workstreams:
- Each subagent has its own tool context
- Results are aggregated back
- Zero context-cost turns via Python RPC scripts

**DrogonClaw Gap:** We have swarm topology but it's synchronous within a phase. No parallel subagent spawning for independent tasks.

#### E. Multi-Platform Gateway
Telegram, Discord, Slack, WhatsApp, Signal — all from a single process.
- Voice memo transcription
- Cross-platform conversation continuity
- Platform-specific slash commands

**DrogonClaw Gap:** CLI-only. No remote control from phone/Slack/etc. A pentester on the move can't check progress.

#### F. Seven Terminal Backends
Local, Docker, SSH, Singularity, Modal, Daytona, Vercel Sandbox.
- Serverless persistence (hibernate when idle, wake on demand)
- $5 VPS to GPU cluster

**DrogonClaw Gap:** Single terminal backend (local only). Can't run scans on remote machines.

### Key Files to Study
- `agent/` — Core agent loop
- `trajectory_compressor.py` — Trajectory compression
- `skills/` — Skill creation and improvement
- `hermes_state_search.py` — Cross-session memory search
- `cron/` — Scheduler implementation
- `providers/` — Provider abstraction (model switching)
- `tools/` — Tool execution framework
- `acp_adapter/` — Agent Communication Protocol adapter

---

## 2. Scrapling — D4Vinci/Scrapling

### What It Is
An adaptive web scraping framework that handles everything from a single request to a full-scale crawl. Bypasses anti-bot systems (Cloudflare Turnstile) out of the box.

### Key Features Relevant to DrogonClaw

#### A. Adaptive Scraping (CRITICAL — DrogonClaw's biggest weakness)
Scrapling's parser **learns from website changes** and automatically relocates elements when pages update:
```python
products = p.css('.product', auto_save=True)  # First visit: save selector
products = p.css('.product', adaptive=True)    # Later: auto-relocate if DOM changed
```
**DrogonClaw Gap:** Our `web_recon` and `http_recon` tools use static CSS selectors. When a target's web app updates mid-pentest, our tools break silently.

#### B. Anti-Bot Bypass (CRITICAL for web pentesting)
```
StealthyFetcher.fetch(url, headless=True, solve_cloudflare=True)
```
Bypasses:
- Cloudflare Turnstile/Interstitial
- Fingerprint spoofing (TLS, headers, browser fingerprint)
- JavaScript challenge pages
- CAPTCHA solving integration

**DrogonClaw Gap:** Our `httpx` and `gobuster` tools can't handle Cloudflare-protected sites. A huge portion of modern targets are behind Cloudflare.

#### C. Session Management
Persistent sessions with cookie/state management:
```python
with StealthySession(headless=True, solve_cloudflare=True) as session:
    page1 = session.fetch('https://target.com/login')
    page2 = session.fetch('https://target.com/dashboard')  # Same session
```
**DrogonClaw Gap:** No persistent HTTP sessions. Each request is independent. Can't maintain login state across tool calls.

#### D. Proxy Rotation
Built-in `ProxyRotator` with:
- Cyclic or custom rotation strategies
- Per-request proxy overrides
- Domain-based proxy assignment
- DNS-over-HTTPS to prevent leaks

**DrogonClaw Gap:** No proxy support. Real engagements need proxy rotation to avoid IP blocking.

#### E. AutoThrottle
Automatically adapts request rate:
- Measures response time per domain
- Doubles delay when rate-limited
- Respects `Retry-After` headers
- Speeds back up when blocking stops

**DrogonClaw Gap:** No rate limiting. Tools fire requests as fast as possible, triggering IDS/WAF.

#### F. Full Crawling Framework
- Concurrent requests with configurable limits
- Pause/resume with checkpoints
- Streaming mode with real-time stats
- Robots.txt compliance
- Development mode (cache responses to disk)

**DrogonClaw Gap:** No full web crawler. Only individual tool calls. Can't do systematic web app enumeration.

#### G. MCP Server
Scrapling can serve as an MCP server, letting AI agents scrape through it:
- One-shot or session-based tools
- Pages narrowed with CSS selectors before AI sees them
- Stripped of prompt-injection content
- Screenshots, remote browsers over CDP

**DrogonClaw Gap:** No MCP integration. Could expose our tools as MCP servers for other agents.

### Key Files to Study
- `scrapling/fetchers/` — StealthyFetcher, DynamicFetcher, Fetcher
- `scrapling/spiders/` — Spider framework, AutoThrottle, pause/resume
- `scrapling/core/translator.py` — CSS/XPath translation
- `agent-skill/` — How Scrapling teaches AI agents its API

---

## 3. OpenClaw — openclaw/openclaw

### What It Is
A personal AI assistant that runs on your devices and meets you in channels you already use. Built by the OpenClaw Foundation (non-profit).

### Key Architectural Patterns Relevant to DrogonClaw

#### A. Gateway Pattern (CRITICAL architecture improvement)
```
Channels (WhatsApp/Telegram/Slack) → Gateway → Tools/Skills/Plugins → Model Providers
                                   ↑
                              Control UI / CLI / TUI
```
**The Gateway is the central control plane** that:
- Routes messages between channels and the agent
- Manages sessions, tools, events
- Handles security (DM pairing, approval)
- Supports companion apps and device nodes

**DrogonClaw Gap:** Monolithic architecture. Agent, TUI, and tools are tightly coupled. A Gateway would allow:
- Multiple frontends (TUI, web, mobile, Slack bot)
- Remote pentest monitoring
- Team collaboration on engagements

#### B. Plugin System
OpenClaw has a full plugin SDK:
```typescript
// plugins are first-class citizens
// shared through ClawHub (plugin registry)
```
**DrogonClaw Gap:** Our `AdaptiveSkillManager` is limited. A proper plugin system would allow:
- Community-contributed pentest modules
- Custom WAF bypass plugins
- Industry-specific attack chains

#### C. Channels
WhatsApp, Telegram, Slack, Discord, Google Chat, Signal, iMessage.
- Each channel is a separate adapter
- Shared conversation state

**DrogonClaw Gap:** CLI-only. A pentester can't get real-time alerts on their phone when a critical vulnerability is found.

#### D. Security Model
- DM-capable channels pair unknown senders by default
- Approve pairing requests with code
- Sandboxing guide for tools
- Exposure runbook

**DrogonClaw Hit:** Our HitL gates are good, but OpenClaw's pairing model is more sophisticated.

#### E. Companion Apps
Voice, Canvas, camera, screen, device-local actions on supported platforms.

**DrogonClaw Gap:** No voice input. Can't say "scan this IP" while on the move.

#### F. Custodian Skills
Specialized skills for sensitive operations with extra approval gates.

**DrogonClaw Gap:** Our `longRunningTools` map is a flat list. OpenClaw's custodian model is more nuanced — different approval levels for different operation types.

### Key Files to Study
- `src/` — Core agent implementation
- `packages/` — Shared libraries
- `extensions/` — Extension system
- `skills/` — Skill framework
- `ui/` — Control UI
- `config/` — Configuration system
- `security/` — Security policies

---

## Consolidated Improvement Recommendations for DrogonClaw

### Phase 1: Web Intelligence Revolution (from Scrapling)
**Impact: CRITICAL | Effort: 2-3 weeks**

| Feature | Source | What to Build |
|---------|--------|---------------|
| Stealth HTTP Fetcher | Scrapling `StealthyFetcher` | Go module using `chromedp` with TLS fingerprint spoofing, Cloudflare bypass |
| Session Manager | Scrapling `FetcherSession` | Persistent cookie jar, login state, request chaining |
| Proxy Rotator | Scrapling `ProxyRotator` | Configurable proxy pool with rotation strategies, DNS leak prevention |
| AutoThrottle | Scrapling AutoThrottle | Rate limiter per domain, adaptive delay based on response time |
| Adaptive Selectors | Scrapling adaptive scraping | Element relocation after page changes using similarity matching |
| Full Crawler | Scrapling Spider framework | Concurrent web crawler with pause/resume, robots.txt compliance |

**Why Critical:** Right now DrogonClaw can't handle Cloudflare, can't maintain sessions, can't rotate proxies, and can't rate-limit. This means it fails against 60%+ of modern web targets.

### Phase 2: Learning Loop (from Hermes)
**Impact: CRITICAL | Effort: 2-3 weeks**

| Feature | Source | What to Build |
|---------|--------|---------------|
| Autonomous Skill Creation | Hermes skills system | After successful attack chain, auto-generate reusable skill |
| Skill Self-Improvement | Hermes skill versioning | Skills track success rate, auto-refine failing patterns |
| Cross-Session Search | Hermes FTS5 search | Full-text search over past engagements with LLM summarization |
| User Modeling | Hermes Honcho integration | Build operator preference model (risk tolerance, report style, skill level) |
| Trajectory Compression | Hermes trajectory_compressor | Semantic compression of conversation history for cost/context optimization |
| Memory Nudges | Hermes memory system | Periodic self-prompts: "What did I learn? What should I remember?" |

**Why Critical:** DrogonClaw is stateless across engagements. Every new pentest starts from zero. A learning loop would make it progressively more effective.

### Phase 3: Remote Operations (from Hermes + OpenClaw)
**Impact: HIGH | Effort: 1-2 weeks**

| Feature | Source | What to Build |
|---------|--------|---------------|
| Multi-Platform Gateway | Hermes gateway + OpenClaw channels | WebSocket gateway for remote CLI, Telegram alerts, Slack bot |
| Scheduled Scans | Hermes cron | Natural language scheduling: "scan this every 6 hours" |
| Remote Terminal | Hermes SSH backend | Run scans on remote machines via SSH |
| Mobile Monitoring | OpenClaw companion apps | Check scan progress, approve HitL gates from phone |

**Why Critical:** Pentesters work remotely. Can't be tied to a terminal.

### Phase 4: Plugin Ecosystem (from OpenClaw)
**Impact: MEDIUM | Effort: 2-3 weeks**

| Feature | Source | What to Build |
|---------|--------|---------------|
| Plugin SDK | OpenClaw plugin system | Go plugin system with hot-reload |
| ClawHub Integration | OpenClaw ClawHub | Community marketplace for pentest modules |
| MCP Server | Scrapling MCP + OpenClaw MCP | Expose DrogonClaw tools as MCP servers for other agents |
| Custodian Skills | OpenClaw custodian-skills | Tiered approval system (low/medium/high/critical operations) |

### Phase 5: Advanced Agent Patterns (from Hermes)
**Impact: MEDIUM | Effort: 1-2 weeks**

| Feature | Source | What to Build |
|---------|--------|---------------|
| Parallel Subagents | Hermes subagent delegation | Spawn isolated agents for parallel recon/exploitation |
| Context Optimization | Hermes trajectory_compressor | Keep 50+ turns of context at same token cost |
| Provider Abstraction | Hermes providers/ | Model-agnostic provider switching with fallback chains |
| ACP Protocol | Hermes acp_adapter | Agent Communication Protocol for multi-agent coordination |

### Phase 6: Speed & Accuracy (from all three)
**Impact: HIGH | Effort: 1-2 weeks**

| Feature | Source | What to Build |
|---------|--------|---------------|
| Streaming Tool Output | Scrapling streaming mode | Real-time tool output streaming to TUI |
| Checkpoint/Resume | Scrapling pause/resume | Save attack state, resume after crash |
| Response Caching | Scrapling dev mode | Cache tool responses to disk, replay on retry |
| Batch Operations | Hermes batch_runner | Run multiple scans in parallel, aggregate results |
| Result Deduplication | Scrapling similarity | Deduplicate findings across tools using fuzzy matching |

---

## Architecture Comparison

```
DrogonClaw (Current):
  TUI ←→ Agent Loop ←→ Tools ←→ Target
                    ↕
              Memory (graph)

Hermes:
  Telegram/Discord/CLI ←→ Gateway ←→ Agent ←→ Tools ←→ Target
                    ↕              ↕         ↕
              FTS5 Search    Skills Hub   Terminal Backends
                    ↕              ↕
              User Model    Trajectory Compressor

OpenClaw:
  WhatsApp/Telegram/Slack ←→ Gateway ←→ Agent ←→ Tools ←→ Target
                    ↕              ↕         ↕
              Plugin System   Skills    Companion Apps
                    ↕              ↕
              ClawHub        Custodian Skills

Scrapling:
  Spider Framework ←→ Sessions ←→ Stealth Bypass ←→ Adaptive Parser
                    ↕              ↕                ↕
              AutoThrottle   Proxy Rotation   Element Relocation
```

**Target Architecture for DrogonClaw:**
```
Channels (CLI/Telegram/Slack/Web) ←→ Gateway ←→ Agent Loop ←→ Tools ←→ Target
                    ↕                    ↕         ↕            ↕
              Session Store        Skills Hub   Swarm     Stealth Fetcher
                    ↕                    ↕         ↕            ↕
              Cross-Session       Learning    Subagents    Proxy Pool
              FTS5 Search         Loop                    AutoThrottle
                    ↕                    ↕
              User Model         Trajectory Compressor
```

---

## Priority Matrix

| Priority | Feature | Impact | Effort | Source |
|----------|---------|--------|--------|--------|
| P0 | Stealth HTTP Fetcher + Cloudflare bypass | CRITICAL | 2 weeks | Scrapling |
| P0 | Session Management (cookie persistence) | CRITICAL | 1 week | Scrapling |
| P0 | Autonomous Skill Creation | CRITICAL | 2 weeks | Hermes |
| P1 | Proxy Rotation + AutoThrottle | HIGH | 1 week | Scrapling |
| P1 | Cross-Session Memory Search | HIGH | 1 week | Hermes |
| P1 | Multi-Platform Gateway | HIGH | 2 weeks | OpenClaw + Hermes |
| P1 | Scheduled Scans (Cron) | HIGH | 1 week | Hermes |
| P2 | Adaptive Web Crawler | MEDIUM | 2 weeks | Scrapling |
| P2 | Trajectory Compression | MEDIUM | 1 week | Hermes |
| P2 | Plugin SDK | MEDIUM | 3 weeks | OpenClaw |
| P2 | Parallel Subagents | MEDIUM | 1 week | Hermes |
| P3 | MCP Server (expose tools) | LOW | 1 week | Scrapling + OpenClaw |
| P3 | Companion Apps | LOW | 2 weeks | OpenClaw |
| P3 | User Modeling | LOW | 2 weeks | Hermes |

---

## Quick Wins (can implement this week)

1. **Session Cookies** — Add persistent cookie jar to HTTP tools (1 day)
2. **Rate Limiting** — Add per-domain delay to HTTP tools (1 day)
3. **Response Caching** — Cache tool responses to disk for retry (1 day)
4. **Streaming Output** — Stream tool results to TUI in real-time (2 days)
5. **Checkpoint/Resume** — Save attack state to disk periodically (2 days)

These alone would dramatically improve reliability and speed without major architectural changes.
