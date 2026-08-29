# Telegram C2 — Operator Guide

DrogonClaw's headless control plane lives in Telegram. Every conversation is a
strictly whitelisted chat: the bot rejects messages from any chat that is not
the configured `TELEGRAM_CHAT_ID`. Configure it once with `./drogonclaw setup`
→ **Telegram C2 gateway**.

Start the bot (and nothing else) so telemetry lands here instead of the TUI:

```bash
./drogonclaw daemon
```

## How a conversation proceeds

1. You send a **mission** (any free text, e.g. `enumerate 10.10.10.5 and probe
   its web services`).
2. The bot opens a **single mission panel** — one message that is edited in
   place as the agent works, so the chat never floods:

   ```
   ⚡ MISSION a3f9c1
   enumerate 10.10.10.5 and probe its web services
   ─────────────────────
   ▰▰▰▱▱▱▱▱▱▱  3/6  · 00:42
   ✓ Enumerate target
   ✓ Probe open ports
   ☐ Fingerprint services
   ☐ Check for CVEs
   ─────────────────────
   🛠 Nmap target=10.10.10.5 · ports=22,80,443
   ✅ Subfinder found 12 subdomains
   · Cross-referencing results…
   ─────────────────────
   🔍 3 signals · 🛠 6 tools · ⏱ 05:12
   ```

   - Plan steps are ticked ✓ as tools complete, ☐ while pending.
   - The currently running tool is pinned with its two most relevant arguments.
   - The last four activity lines keep you oriented (`✅ done`, `· thinking`,
     `❌ error`).
   - The footer shows a signal/tool count and a live elapsed timer that ticks
     every few seconds.
3. While the agent is busy the bot shows the native **typing animation** so it
   feels alive — it stops on completion or as soon as an approval is requested.
4. When done, the panel turns into a completion summary and the full answer is
   delivered as a follow-up message (chunked at message-size limits). Every
   final message also carries a **`--- RAW TOOL RESULTS ---`** appendix with the
   actual tool output (e.g. the full 343-subdomain list), so the agent's
   summary is never the only copy of the data. Appendices are bounded
   (~6 KB, newest tool first) so huge scans can't flood the chat.

## Approvals (human-in-the-loop)

The agent can pause for a go/no-go in two situations:

- **Long-running low-risk tools** (e.g. `gobuster`, `ffuf`, large `nmap`) run
  in manual mode — the panel switches to:

  ```
  ⚠️ APPROVAL REQUIRED
  Awaiting operator acceptance to run gobuster (est. 5m)…
  ─────────────────────
  🔍 2 signals · 🛠 4 tools · ⏱ 03:11
  ```
  Tap **✓ Approve** to run it, **✗ Skip** to continue without it, or reply
  with free text. Skipping does not fail the mission — the agent moves on.

- **Dangerous actions** (docker/script execution, high-risk commands) suspend
  the agent and ask for your input. Buttons approve with `y`; if the agent is
  asking an open question, reply with plain text and your answer is fed back
  to the reasoning loop.

While an approval is pending, slash commands still work — send `/status` or
`/cancel` freely, they are never mistaken for an answer.

## Commands

| Command | What it does |
|---|---|
| `<free text>` | Run a mission |
| `/swarm <mission>` | Run parallel execution vectors (live panel + cancel) |
| `/findings` | Dump the loot ledger: ports, credentials, vulnerabilities |
| `/autopilot [on\|off]`, `/auto` | Toggle auto-accept of long-running low-risk tools (no arg = show current state). Persisted to config, so it survives daemon restarts |
| `/status` | Live snapshot of the running mission; when idle, the daemon dashboard (session, graph size, loot totals) |
| `/cancel` | Abort the running mission |
| `/report` | Generate and send the pentest report document |
| `/whoami` | Operator and agent identity |
| `/help` (`/start`) | This reference |

Every panel carries an inline **✕ Cancel mission** button, so you never have to
remember the command.

## Hints

- **Autopilot** is useful for unattended runs — long-running recon tools stop
  pausing. Toggle is per-daemon-process and resets on restart.
- **`/status` while idle** is a handy one-line dashboard of what the current
  session remembers (nodes/edges in the intelligence graph and total loot).
- **Credentials are shown on the `/findings` panel** — they are AES-encrypted at
  rest in `data/drogonclaw_loot.db` and only decrypted for the whitelisted chat.
- Secrets never appear in the panel: API keys, tokens and `Bearer` headers are
  scrubbed from tool output, streaming text and summaries before rendering.
- Missions are refused while one is already running — use `/status` and
  `/cancel` instead of stacking fresh instructions.