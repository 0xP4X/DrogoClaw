# System Cleanup & Reorientation Audit

## Issues to Fix for Excellence

### 1. CONFIGURATION DISCOVERY (Critical)
**Problem:** First-time users don't know they need to run `/setup`
- No startup prompt
- No welcome message
- Config is optional → leads to silent failures

**Solution:**
- ✅ Detect missing config at startup
- ✅ Show banner: "Welcome! Run `/setup` to configure provider"
- ✅ Auto-launch `/setup` on first run option
- ✅ Validate config after setup (health check)

**Files:** cmd/drogonclaw/main.go, internal/tui/model.go

---

### 2. STATUS & INTELLIGENCE (Conflicting)
**Problem:** Too many status commands showing overlapping data
- `/status` = session stats (but what stats?)
- `/cost` = token tracking (buried in SESSION category)
- `/timeline` = execution history (same info as status?)
- `/health` = diagnostics (should be silent unless there's a problem)

**Solution:**
- `/status` → **Only** shows: session ID, elapsed time, tools run count, findings count, tokens used
- `/cost` → **Only** shows: cost breakdown (recon $0.05, exploitation $15, savings 80%)
- `/timeline` → **Only** shows: chronological tool execution (when + what + duration)
- `/health` → Remove from main help, only appear if errors detected (proactive error reporting)

**Reorient:**
```
/status              → "What have I done this session?" (metrics)
/cost                → "How much did this cost vs baseline?" (financial)
/timeline            → "Show me the execution sequence" (debugging)
/health              → Auto-runs on startup, silent if OK, loud if errors
```

---

### 3. HIDDEN CRUFT (Dead Code)
**Problem:** Code exists but isn't in help or discoverable

Currently in codebase but not exposed:
- Session sections (unused memory)
- Benchmarks command (incomplete)
- Queue command (shows pending but doesn't manage them)
- Pager integration (/details opens external pager - why?)
- Sidebar + details toggles (nice but undiscovered)

**Solution:**
- ✅ Remove `/sections`, `/section` entirely (clean audit shows unused)
- ✅ Remove `/benchmarks` (merge data into `/report` or `/timeline`)
- ✅ Remove `/queue` (users don't need a "pending" view, just execute)
- ✅ Remove external pager (`/details` should show inline, not external)
- ✅ Move `/sidebar`, `/details` to keyboard shortcuts (Ctrl+B, Ctrl+T) - don't need commands

**Files to clean:** internal/tui/view.go, internal/tui/events.go, internal/tui/commands.go

---

### 4. CONFIRMATION HELL (UX Friction)
**Problem:** Too many confirmations for things that shouldn't need them

Current bad patterns:
- `/new` requires "CLEAR SESSION" typed out (why? just ask yes/no)
- `/sandbox` requires "TOGGLE SANDBOX" confirmation (paranoid)
- `/resume` silently might fail (should show what it's restoring)

**Solution:**
- `/new` → Show: "Clear session? (10 findings will be lost)" [y/n]
- `/sandbox` → Just do it: "Sandbox: OFF → ON"
- `/resume` → Show: "Resuming 'scan 192.168.1.0/24' from 45% complete"

**Files:** internal/tui/commands.go, internal/tui/events.go

---

### 5. PERSONA & DIRECTIVES (Unused Feature Bloat)
**Problem:** `/persona` exists but users don't know about it or need it

Current:
- `/persona "be more cautious"` → injects into system prompt
- Rarely discovered
- Conflicts with `/set` (should be `/set PERSONA`)
- Too magical (users don't understand system prompt injection)

**Solution:**
- ✅ Remove `/persona` command entirely
- ✅ Add `/config show PERSONA` to view current
- ✅ Add `/set PERSONA "directive"` if really needed
- **Better:** Remove this feature—let system prompt be managed by team, not users

**Rationale:** Feature creep. If users need to inject personas, they should use the API or config file, not a command.

**Files:** internal/tui/commands.go (delete persona handler)

---

### 6. EXECUTION MODES (Unclear Concepts)
**Problem:** Too many overlapping execution modes

Current:
- `/auto` = autopilot (autonomous agent)
- `/stealth` = OPSEC (evasion timing)
- `/sandbox` = Docker sandbox (isolation)
- `/mode` = attack workflow (recon/exploit/ctf)

Users don't understand the relationships. Is autopilot + stealth on at once? Does mode affect sandbox?

**Solution - Clarify hierarchy:**
```
EXECUTION_MODE (how to execute)
  ├─ autopilot: true/false (autonomous vs manual approval)
  ├─ opsec: high/medium/low (rate limiting + evasion)
  └─ sandbox: true/false (isolated vs host)

WORKFLOW_MODE (what to execute)
  ├─ recon, exploit, web, api, ctf, mail, ...
  └─ independent of execution mode
```

**Rename for clarity:**
- `/mode` → `/workflow` (select what to attack)
- `/set AUTOPILOT` → `/set EXECUTION_MODE [manual|autonomous]`
- `/set OPSEC` → `/set EVASION [high|medium|low]`
- `/set SANDBOX` → `/set ISOLATION [on|off]`

**Result:** Clear mental model for users.

**Files:** internal/tui/commands.go, internal/intel/modes.go

---

### 7. PROVIDER HEALTH (Noise)
**Problem:** `/providers` shows all 6 providers even if only 1 is configured

Current output:
```
PROVIDER HEALTH DASHBOARD
openrouter    ● set    ○ idle       https://openrouter.ai/api/v1
openai        ○ none   ○ idle       https://api.openai.com/v1
nvidia        ○ none   ○ idle       https://integrate.api.nvidia.com/v1
gemini        ○ none   ○ idle       https://generativelanguage.googleapis.com
ollama        ○ none   ○ idle       http://localhost:11434
9router       ● set    ○ ready      https://api.9router.ai/v1
```

This is noise. Users only care about:
1. What's active now?
2. What's available?

**Solution:**
```
ACTIVE PROVIDER
  openrouter / meta-llama/llama-3.1-70b  ($0.30/1M)

AVAILABLE BACKUPS
  ✓ openai (gpt-4o)
  ✓ 9router (auto)

ROUTING
  Mode: AUTO
  Savings: 80% ($2.45 saved this session)
```

Much cleaner. Only show what matters.

**Files:** internal/tui/providers.go

---

### 8. KEYBOARD SHORTCUTS (Undiscovered)
**Problem:** Power features buried in help, users don't know they exist

Current shortcuts:
- Ctrl+P = command palette (good)
- Ctrl+B = toggle sidebar (hidden)
- Ctrl+S = status (hidden)
- Ctrl+D = show cost (hidden)
- Ctrl+E = open pager (hidden)
- Ctrl+Y = copy output (hidden)

**Solution:**
- Show shortcuts in header or footer bar: `[?] for help | Ctrl+P palette | Ctrl+B sidebar`
- Add visual hint when user types `/help`: "Tip: Ctrl+B toggles sidebar, Ctrl+P opens command palette"
- Make footer show: `[Ctrl+P] [Ctrl+B] [Ctrl+S] [?]`

**Files:** internal/tui/view.go (header/footer rendering)

---

### 9. ERROR HANDLING (Silent Failures)
**Problem:** Commands fail silently or with cryptic messages

Examples:
- Provider not configured → "Connection error" (not helpful)
- Invalid routing mode → "[✗] Unknown routing mode: foo" (no hint to available modes)
- Missing API key → "401 Unauthorized" (what key? where to get it?)

**Solution - Helpful errors:**
```
❌ BEFORE: [✗] Unknown routing mode: foo

✅ AFTER: [✗] Unknown routing mode: 'foo'
          Available: auto (recommended), local, 9router, off
          Example: /router auto
```

Implement `HelpfulError()` helper that:
1. Explains what went wrong
2. Shows valid options
3. Gives example command

**Files:** internal/tui/commands.go (all command handlers)

---

### 10. SESSION PERSISTENCE (Confusing)
**Problem:** Session/memory management is unclear

Current:
- `/new` clears memory
- `/resume` restores from checkpoint
- `/sections` lists sections (unused)
- But what actually persists? What's lost?

**Solution:**
- Remove `/sections` (dead feature)
- Make `/new` show: "Clear memory? Session findings, credentials, flags will be lost. (y/n)"
- Auto-save session state every 30 seconds to `~/.drogonclaw/session.json`
- On startup: "Resume previous session? (3 findings, 15 tools executed)"
- Remove complexity—just one session per run, auto-recovery on crash

**Files:** internal/tui/model.go, internal/memory/graph.go

---

## Priority Cleanup

### 🔴 HIGH PRIORITY (Do First)
1. **Add startup config detection** - Users need guidance on first run
2. **Remove dead commands** - `/sections`, `/section`, `/benchmarks`, `/queue`, `/persona`
3. **Clarify execution modes** - Rename to `WORKFLOW`, `EXECUTION_MODE`, `EVASION`, `ISOLATION`
4. **Fix error messages** - All commands should give helpful, actionable errors

### 🟡 MEDIUM PRIORITY (Nice to Have)
5. Simplify provider display (show only what matters)
6. Show keyboard shortcuts in header/footer
7. Reorient status/cost/timeline (no overlap)
8. Remove external pager (show inline)

### 🟢 LOW PRIORITY (Polish)
9. Simplify session persistence
10. Add `/workflow` as alias for `/mode`

---

## Result: Excellent System

✅ **No confusion** - Clear command hierarchy, no overlaps
✅ **Discoverable** - Startup guidance, visible shortcuts, helpful errors
✅ **No cruft** - Dead features removed, unused commands deleted
✅ **High UX** - Confirmations make sense, modes are clear, errors are helpful
✅ **Claude Code parity** - Clean, intuitive, modern

---

## Estimated Work

- **HIGH priority:** 4-6 hours (biggest impact)
- **MEDIUM priority:** 2-3 hours (polish)
- **LOW priority:** 1-2 hours (refinement)

**Total:** 7-11 hours for truly excellent system.
