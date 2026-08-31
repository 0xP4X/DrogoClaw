# Command System Modernization Plan

## Current State (40+ commands)
**BLOATED:** scattered categories, overlapping functionality, confusing hierarchy

## Target State (20 core commands)
**CLEAN:** 5 focused categories, zero duplication, obvious purpose

---

## NEW COMMAND STRUCTURE

### 1. OPERATIONS (Active Tasks) — 7 commands
```
/mode <name>              Select attack workflow (recon/exploit/ctf/web/api)
/analyze <target>         Classify target and suggest attack path
/profile <target>         Build passive intelligence profile
/ctf <path>               Solve local CTF challenge
/report                   Generate penetration test report
/swarm <objective>        Dispatch parallel sub-agent swarm
/skills [query]           List/search available execution modules
```

### 2. INTELLIGENCE (Status & Metrics) — 5 commands
```
/status                   Session stats: runtime, memory, execution count
/cost                     Token usage & estimated cost (with routing savings)
/health                   Verify runtime environment & dependencies
/timeline                 Execution timeline (tools, findings, duration)
/findings                 Summarize all detected findings (vuln/cred/flag/info)
```
**REMOVED:** /benchmarks (infrequent—in /report), /queue (use task list)

### 3. CONFIG (Settings) — 5 commands
```
/config                   View all settings (provider, model, routing, theme)
/set <KEY> <VALUE>        Change setting instantly (no restart needed)
/router [auto|local|9router|off]  Switch routing mode + status
/providers                Provider health dashboard
/theme [name]             Switch color theme (dark/light/dracula/nord/gruvbox)
```
**REMOVED:** /sandbox, /stealth, /auto (move to /set or /config set)

### 4. EXECUTION (Runtime Control) — 3 commands
```
/config set AUTOPILOT [on|off]     Enable/disable autonomous mode
/config set OPSEC [high|medium|low] Adjust rate limiting + evasion
/config set SANDBOX [on|off]        Toggle Docker sandbox
```
**REMOVED:** /stealth, /auto, /sandbox (consolidated into /config set)

### 5. SESSION (Lifecycle) — 5 commands
```
/setup                    Interactive configuration wizard
/new                      Clear memory, start fresh session
/resume                   Resume interrupted execution
/copy [format]            Export transcript (txt/json/markdown)
/clear                    Clear terminal output
/exit                     Graceful shutdown
/help                     Command reference
```
**REMOVED:** /sections, /section (unused cruft), /persona (rarely used—move to /config), /commands (alias for /help)

---

## Changes (30 → 20 Commands)

### ✅ CONSOLIDATE
- /stealth, /auto, /sandbox → `/config set`
- /sections, /section → removed (unused)
- /benchmarks → `/report` (already generates)
- /queue → task display (built-in)
- /commands → alias of /help
- /persona → `/config set PERSONA "<directive>"`

### ✅ ADD
- `/set KEY VALUE` — instant config changes
- `/findings` — summary of detections (new)

### ✅ RENAME/CLARIFY
- keep `/status` but make it session-focused (not config)
- keep `/config` but make it view-only + support `/config set`
- keep `/cost` but add routing savings display

### ✅ REORGANIZE
- No more "UI" category (merge into CONFIG)
- No more scattered toggles (all → CONFIG SET)
- Operations = tasks, Intelligence = metrics, Config = settings, Execution = control, Session = lifecycle

---

## Why This Works

| Before | After | Benefit |
|--------|-------|---------|
| `/auto` ⊕ `/sandbox` ⊕ `/stealth` ⊕ `/persona` | `/config set` | Single command, consistent UX |
| `/config` (view only) | `/config` + `/set` | Easy changes without JSON |
| `/sections` + `/section` | removed | Dead code removal |
| 40+ scattered commands | 20 organized commands | Clear hierarchy, zero confusion |
| `/status` + `/config` + `/cost` + `/providers` | 5 intelligence commands | Metrics are separated from settings |
| `/help` ⊕ `/commands` | `/help` | No aliases |

---

## Implementation Steps

1. ✅ Rename/remove duplication in commands.go
2. ✅ Update /help output to reflect new structure
3. ✅ Add `/set KEY VALUE` command
4. ✅ Add `/findings` command (aggregate findings)
5. ✅ Update README Commands table
6. ✅ Update docs/TUI.md
7. ✅ Test all commands work post-consolidation

---

## Command Reference (New)

```
OPERATIONS (Active Tasks)
  /mode <name>              Select attack workflow
  /analyze <target>         Classify target
  /profile <target>         Passive intelligence
  /ctf <path>               Solve CTF challenge
  /report                   Generate pentest report
  /swarm <objective>        Parallel subagents
  /skills [query]           List modules

INTELLIGENCE (Metrics)
  /status                   Session statistics
  /cost                     Token usage + savings
  /health                   Environment check
  /timeline                 Execution timeline
  /findings                 Detected findings summary

CONFIG (Settings)
  /config                   View all settings
  /set <KEY> <VALUE>        Change setting instantly
  /router [mode]            Routing mode + status
  /providers                Provider dashboard
  /theme [name]             Switch theme

EXECUTION (Control)
  /config set AUTOPILOT [on|off]
  /config set OPSEC [high|medium|low]
  /config set SANDBOX [on|off]

SESSION (Lifecycle)
  /setup                    Configuration wizard
  /new                      Start fresh
  /resume                   Resume interrupted
  /copy [fmt]               Export transcript
  /clear                    Clear output
  /exit                     Shutdown
  /help                     This reference
```
