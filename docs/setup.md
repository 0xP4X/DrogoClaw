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

The wizard walks you through everything and writes it to
`~/.drogonclaw/config.json` (owner read/write only):

1. **Authorisation** — scope/compliance acknowledgement.
2. **Neural Provider** — OpenRouter (default), NVIDIA NIM, OpenAI, Google Gemini, or local Ollama, plus model selection.
3. **Credentials** — the provider API key (or Ollama endpoint).
4. **Remote C2 Gateway** (optional) — Telegram bot token + chat ID.
5. **Secondary API Keys** (optional) — choose any of GitHub, Shodan, VirusTotal, Brave Search, Hunter.io, or Exa to enable OSINT/recon features. You can skip the whole set or pick only the ones you want.

You do **not** need to export anything — re-run `./drogonclaw setup` (or `/setup`
inside the terminal) any time to change settings.

## Sandbox Execution

DrogonClaw runs tools in an isolated, ephemeral Docker sandbox. Native host
execution is disabled unless you explicitly set `USE_SANDBOX=false`; the engine
fails closed rather than silently falling back to the host.

Ensure Docker is running:

```bash
docker info
```

## Telegram C2 Gateway

To control DrogonClaw from your phone:

1. Create a bot with [@BotFather](https://t.me/BotFather) and copy the token.
2. Get your numeric chat ID.
3. Enable it in the wizard — when `drogonclaw setup` asks about the Remote C2
   Gateway, choose yes and paste the token and chat ID. They are saved to
   `~/.drogonclaw/config.json`.

The `TELEGRAM_CHAT_ID` is a strict whitelist — any command from another chat is
rejected.

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
