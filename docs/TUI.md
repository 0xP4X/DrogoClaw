# DrogonClaw TUI Guide

## 🎨 Interface Overview

DrogonClaw features a professional terminal user interface (TUI) built with [Bubbletea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).

### Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│ DrogonClaw │ operator@agent │ gpt-4o │ ◉ EXECUTING │ 12 tools      │ Header
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Main Content Area                    │  Sidebar                   │
│  - Agent responses                    │  SESSION                   │
│  - Tool execution                     │  Session: abc123           │
│  - Tool results                       │  Runtime: 2m 34s           │
│  - Findings                           │                            │
│  - Command output                     │  MEMORY                    │
│                                       │  Entities: 45              │
│                                       │  Links: 78                 │
│                                       │                            │
│                                       │  ROUTING                   │
│                                       │  Mode: ● AUTO              │
│                                       │  Savings: $2.34            │
├─────────────────────────────────────────────────────────────────────┤
│ ● AUTO │ EXECUTING │ step 3/5 │ Ctrl+P cmds │ Ctrl+A auto │       │ Status
├─────────────────────────────────────────────────────────────────────┤
│ drogonclaw > scan target 192.168.1.1                               │ Input
└─────────────────────────────────────────────────────────────────────┘
```

---

## ⌨️ Keyboard Shortcuts

### Essential Shortcuts

| Shortcut | Action | Description |
|----------|--------|-------------|
| `Ctrl+P` | Command Palette | Shows all available commands with autocomplete |
| `Ctrl+A` | Toggle Autopilot | Enable/disable autonomous execution mode |
| `Ctrl+C` | Cancel/Quit | Cancel running task (double-press) or quit |
| `Enter` | Submit | Submit command or user input |
| `Tab` | Autocomplete | Accept command suggestion from palette |

### Navigation

| Shortcut | Action | Description |
|----------|--------|-------------|
| `↑` / `↓` | Scroll | Scroll output up/down (3 lines) |
| `PgUp` / `PgDn` | Page Scroll | Scroll half viewport up/down |
| `Home` | Top | Jump to top of output |
| `End` | Bottom | Jump to bottom of output |
| `Alt+↑` / `Alt+↓` | History | Browse command history |

### Panels & Views

| Shortcut | Action | Description |
|----------|--------|-------------|
| `Ctrl+B` | Sidebar | Toggle sidebar panel on/off |
| `Ctrl+T` | Tool Details | Toggle tool detail panel |
| `Ctrl+E` | Pager | Open output in external pager (less) |
| `Ctrl+Y` | Copy | Copy transcript to clipboard |

### Information

| Shortcut | Action | Description |
|----------|--------|-------------|
| `Ctrl+S` | Status | Show session status report |
| `Ctrl+D` | Cost | Show token usage and cost |

### Leader Key (Advanced)

Press `Ctrl+X` then one of these keys within 2 seconds:

| Key | Action | Description |
|-----|--------|-------------|
| `b` | Sidebar | Toggle sidebar |
| `n` | New Session | Clear memory, start fresh |
| `l` | List Sessions | Show available sessions |
| `m` | Models | Show available models |
| `t` | Themes | Show available themes |
| `e` | Editor | Open in external editor |
| `x` | Export | Export session |
| `q` | Quit | Exit DrogonClaw |

---

## 📋 Slash Commands

Commands are grouped the same way `/help` renders them — one registry, zero drift.

### Operations — Active Tasks

| Command | Arguments | Description |
|---------|-----------|-------------|
| `/workflow` `/mode` | `[name\|off]` | Select attack workflow methodology |
| `/analyze` | `<target>` | Classify target and determine attack path |
| `/skills` | `[query]` | List/search available execution modules |
| `/profile` | `<target>` | Build passive intelligence profile |
| `/ctf` | `<path>` | Run local CTF artifact triage |
| `/report` | - | Generate structured pentest report |
| `/swarm` | `<objective>` | Dispatch parallel sub-agent swarm |

### Controls — Settings

| Command | Arguments | Description |
|---------|-----------|-------------|
| `/config` | `[set KEY VALUE]` | View or modify settings (provider, model, routing, theme, execution) |
| `/set` | `<KEY> <VALUE>` | Quickly set a config value (alias: `/config set`) |
| `/router` | `[auto\|local\|9router\|off\|status]` | Configure intelligent routing |
| `/providers` | - | Provider health & routing status dashboard |
| `/theme` | `[name]` | Switch color theme (dark/light/dracula/nord/gruvbox) |

Runtime controls via `/set` / `/config set`:

| Setting | Values | Description |
|---------|--------|-------------|
| `EXECUTION_MODE` `AUTOPILOT` | `manual` / `autonomous` | Enable autonomous execution (alias: `AUTOPILOT on/off`) |
| `EVASION` `OPSEC` | `high` / `medium` / `low` | Rate-limiting & evasion level |
| `ISOLATION` `SANDBOX` | `on` / `off` | Docker sandbox isolation |

### Session — Metrics & Lifecycle

| Command | Arguments | Description |
|---------|-----------|-------------|
| `/help` | - | Show complete command reference |
| `/status` | - | Session metrics: runtime, tools run, findings, current phase |
| `/cost` | - | Token usage breakdown: input/output tokens, cost, routing savings |
| `/health` | - | Environment diagnostics: Docker, dependencies, sandbox readiness |
| `/timeline` | - | Execution log: when each tool ran, duration, results summary |
| `/findings` | - | Detection summary: vulnerabilities, credentials, flags |
| `/setup` | - | Run interactive configuration wizard |
| `/new` | - | Clear session memory (confirms with `yes/no`) |
| `/resume` | - | Resume interrupted execution from last checkpoint |
| `/copy` | - | Export transcript to clipboard & file |
| `/clear` | - | Clear visible terminal output |
| `/exit` `/quit` | - | Terminate session gracefully |

### UI — Display

| Command | Arguments | Description |
|---------|-----------|-------------|
| `/sidebar` | - | Toggle sidebar panel (Ctrl+B) |
| `/details` | - | Toggle tool detail panel (Ctrl+T) |

---

## 🎨 Customization

### Themes

DrogonClaw supports multiple color themes:

```bash
# Available themes
/theme dark       # Default GitHub-inspired dark theme
/theme light      # Light theme for daytime use
/theme dracula    # Dracula color scheme
/theme nord       # Nord color scheme
/theme gruvbox    # Gruvbox color scheme
```

### Configuration

Edit `~/.drogonclaw/config.json`:

```json
{
  "AI_PROVIDER": "openrouter",
  "AI_MODEL": "meta-llama/llama-3.1-70b-instruct",
  "ROUTER_MODE": "auto",
  "USE_SANDBOX": "true",
  "TELEGRAM_TOKEN": "",
  "WORKSPACE_ROOT": "/home/user"
}
```

### Environment Variables

```bash
# Provider configuration
export AI_PROVIDER=openrouter
export AI_MODEL=meta-llama/llama-3.1-70b-instruct
export OPENROUTER_API_KEY=sk-or-...

# Routing configuration
export ROUTER_MODE=auto
export NINEROUTER_API_KEY=sk-9r-...

# Execution environment
export USE_SANDBOX=true
export OLLAMA_BASE_URL=http://localhost:11434
```

---

## 💡 Tips & Tricks

### 1. Command Palette Workflow
Press `Ctrl+P` and start typing any command. The palette shows all matching commands with descriptions. Use `↑`/`↓` to select and `Tab` to autocomplete.

### 2. Queue Multiple Tasks
When a task is running, type another command and press Enter. It will be queued and run automatically when the current task finishes.

### 3. Copy Output
Press `Ctrl+Y` to copy the entire transcript. It's saved to `~/.drogonclaw/loot/drogonclaw_transcript.txt` and copied to clipboard if available.

### 4. External Pager
For long output, press `Ctrl+E` to open in your system pager (usually `less`). Supports searching and full scrollback.

### 5. History Navigation
Use `Alt+↑` and `Alt+↓` to browse previous commands. Much faster than retyping.

### 6. Autopilot Mode
Press `Ctrl+A` to enable autopilot. The agent will continue executing without asking for approval (use with caution on authorized targets only).

### 7. Cost Tracking
Press `Ctrl+D` anytime to see current token usage and estimated cost. Helps stay within budget.

### 8. Sidebar Information
The sidebar shows live session info:
- Session ID and runtime
- Memory graph statistics
- Routing mode and savings
- Provider status
- Queue status

Toggle with `Ctrl+B` if you need more screen space.

---

## 🐛 Troubleshooting

### TUI Not Rendering Correctly

**Problem:** Garbled characters, broken boxes, or wrong colors

**Solutions:**
```bash
# Set correct TERM variable
export TERM=xterm-256color

# Or try
export TERM=screen-256color

# Verify terminal supports 256 colors
tput colors  # Should output 256
```

### Sidebar Too Wide/Narrow

The sidebar automatically adjusts based on terminal width:
- Minimum terminal width: 44 characters
- Sidebar shows when width ≥ 44 characters
- Sidebar width: 24-36 characters (adaptive)

Resize your terminal or toggle sidebar with `Ctrl+B`.

### Command Palette Not Showing

If `Ctrl+P` doesn't work:
1. Check your terminal supports Ctrl key combinations
2. Try the `/` prefix instead: type `/` then command name
3. Use `/help` to see all commands

### Output Scrolling Issues

If output doesn't auto-scroll:
- Press `End` to jump to bottom
- The TUI auto-scrolls only when you're at the bottom
- Use `↑`/`↓` to scroll, then press `End` to resume auto-scroll

### Colors Look Wrong

Different terminals render colors differently:
- Try a different theme: `/theme light` or `/theme dracula`
- Check terminal's color scheme settings
- Some terminals (Windows CMD) have limited color support

### Can't Copy Output

`Ctrl+Y` requires clipboard tools:
- **Linux:** Install `xclip`, `xsel`, or `wl-clipboard`
- **macOS:** Built-in `pbcopy` should work
- **Windows:** Built-in `clip` should work

Alternatively, output is always saved to file even if clipboard fails.

---

## 🎓 Learning Resources

### Getting Started
1. Run `/setup` to configure DrogonClaw
2. Try `/health` to verify everything works
3. Run `/help` to see all commands
4. Press `Ctrl+P` to explore the command palette

### Example Workflows

**Reconnaissance:**
```bash
# Start with passive profiling
/profile example.com

# Activate recon workflow
/mode recon

# Let agent plan and execute
scan example.com for services and vulnerabilities
```

**CTF Challenge:**
```bash
# Activate CTF mode
/mode ctf

# Analyze binary
/ctf /path/to/challenge

# Let agent solve
solve this CTF challenge: [description]
```

**Web Application Testing:**
```bash
# Activate web mode
/mode web

# Let agent enumerate and test
test https://example.com for vulnerabilities
```

### Advanced Features

**Parallel Execution:**
```bash
# Use swarm for concurrent tasks
/swarm scan these 10 targets and identify exploits
```

**Skill Learning:**
After successful exploits, DrogonClaw automatically saves techniques as reusable skills. View them with `/skills`.

**Custom Execution Controls:**
```bash
# Runtime controls are now unified under /set and /config set
/set EXECUTION_MODE autonomous   # enable autonomous execution
/set EVASION high                # high/medium/low evasion
/set ISOLATION on                # Docker sandbox on/off
```

---

## 📚 See Also

- [ROUTING.md](ROUTING.md) - Intelligent routing guide
- [PROVIDERS.md](PROVIDERS.md) - Provider configuration
- [TOOLS.md](TOOLS.md) - Available tools reference
- [DEVELOPMENT.md](DEVELOPMENT.md) - Contributing guide

---

**Built with:** [Bubbletea](https://github.com/charmbracelet/bubbletea) | [Lipgloss](https://github.com/charmbracelet/lipgloss) | [Bubbles](https://github.com/charmbracelet/bubbles)
