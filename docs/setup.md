# DrogonClaw Setup Guide

This guide covers installing DrogonClaw, configuring an AI provider, and
enabling the optional sandbox, Telegram C2 gateway, and headless daemon.

> **Authorized use only.** DrogonClaw performs offensive security operations.
> Only run it against systems you are explicitly permitted to test.

## Requirements

- Go `1.26+`
- Docker (daemon running) for sandbox execution
- Linux (Kali, Ubuntu, or Debian recommended)

## Install

### From source

```bash
git clone https://github.com/0xP4X/drogonclaw.git
cd drogonclaw
go mod tidy
go build -o drogonclaw ./cmd/drogonclaw/
```

### From npm (prebuilt Linux x64)

```bash
npm install -g drogonclaw
drogonclaw setup
drogonclaw
```

## Configure a Provider

DrogonClaw is configured **entirely through its Setup Wizard** — there is no
environment-variable or export-based configuration. Run the interactive wizard:

```bash
./drogonclaw setup
```

The wizard is menu-driven rather than a single linear pass. On every run it
first shows a **Current Configuration** summary — provider, model, API-key
status, Telegram gateway, OSINT keys and identity, with all secrets masked to
their last four characters — then offers a menu of sections:

1. **Change AI provider or model** — OpenRouter (default), NVIDIA NIM, OpenAI,
   Google Gemini, or local Ollama. Any step you enter is prefilled with the
   stored value; leaving the API key blank keeps the existing key.
2. **Telegram C2 gateway** — see the current bot status, then keep, replace, or
   disable it. Even if you already configured Telegram earlier, re-running the
   wizard shows you what is stored (token masked, chat ID visible).
3. **Secondary API keys** — GitHub, Shodan, VirusTotal, Brave Search, Hunter.io,
   or Exa. Pick any to add or update, then optionally clear stored ones. Each is
   marked `(set)` / `(not set)`.
4. **Operator & agent identity** — display names used in headers and reports.
5. **Reset everything to defaults** — clears every credential and re-enables the
   authorisation gate.

Every section returns to the menu, so you can make several changes in one run,
and any step can be cancelled without aborting the process. Everything is saved
the moment you confirm a value, to `~/.drogonclaw/config.json` (owner read/write
only).

The scope **authorisation** acknowledgement is asked on your first run and
persisted, so returning users are not re-prompted.

> Tip: you do **not** need to open the wizard to check your settings. Inside the
> terminal run `/config` for the same masked configuration summary.

You do **not** need to export anything — re-run `./drogonclaw setup` (or `/setup`
inside the terminal) any time to change settings.

## Sandbox Execution

DrogonClaw can run tools in two modes:

- **Native mode** (default): commands run directly on your host OS. This is the recommended mode for most use cases.
- **Sandbox mode**: commands run inside an isolated Docker container. Use this if you want complete isolation from your host.

You can switch modes with the `/sandbox` command inside the terminal, or launch
with `./drogonclaw sandbox` / `./drogonclaw native`.

If Docker is not available, DrogonClaw will fall back to native mode automatically.

## Telegram C2 Gateway

To control DrogonClaw from your phone:

1. Create a bot with [@BotFather](https://t.me/BotFather) and copy the token.
2. Get your numeric chat ID.
3. In the wizard, pick **Telegram C2 gateway** from the menu (or run `/config`
   first to confirm the current state), then choose *Replace* and paste the token
   and chat ID. They are saved to `~/.drogonclaw/config.json`.

To inspect or revoke a gateway that is already configured, re-run `drogonclaw
setup` and open the same section — it shows the stored token (masked) and chat
ID and offers keep / replace / disable.

The `TELEGRAM_CHAT_ID` is a strict whitelist — any command from another chat is
rejected.

Once running, the bot presents a live mission panel, inline approve/skip/cancel
buttons and a small command set (`/help`, `/status`, `/findings`, `/autopilot`,
`/cancel`, `/report`, `/whoami`). See the full protocol in
[telegram.md](telegram.md).

## Headless Daemon

Run the agent without the terminal UI (useful with the Telegram gateway):

```bash
./drogonclaw daemon
```

## Docker Compose

To start the supporting services:

```bash
make docker-compose   # docker compose up -d
```

## Next Steps

- Inside the terminal, run `/help` for the full command reference.
- See [README.md](../README.md) for architecture and capabilities.
- Report vulnerabilities per [SECURITY.md](../SECURITY.md).
