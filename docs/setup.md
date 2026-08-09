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

DrogonClaw is configured through its **Setup Wizard**, not by exporting
environment variables. Run the interactive wizard:

```bash
./drogonclaw setup
```

The wizard prompts for a provider, API key (where required), and an optional
Telegram gateway, then writes credentials to `~/.drogonclaw/config.json`
(owner read/write only). Environment variables are only optional overrides
that take precedence over the saved config file — you do not need to export
anything.

### Provider reference

| Provider | Config Key | Environment Variable |
| --- | --- | --- |
| OpenRouter | `openrouter` | `OPENROUTER_API_KEY` |
| NVIDIA NIM | `nvidia` | `NVIDIA_API_KEY` |
| OpenAI | `openai` | `OPENAI_API_KEY` |
| Google Gemini | `gemini` | `GOOGLE_API_KEY` |
The table below lists the config key and the optional environment-variable
override for each provider. Prefer the wizard over exporting these.

| Provider | Config Key | Environment Variable |
| --- | --- | --- |
| OpenRouter | `openrouter` | `OPENROUTER_API_KEY` |
| NVIDIA NIM | `nvidia` | `NVIDIA_API_KEY` |
| OpenAI | `openai` | `OPENAI_API_KEY` |
| Google Gemini | `gemini` | `GOOGLE_API_KEY` |
| Ollama (local) | `ollama` | `OLLAMA_BASE_URL` |

If you prefer to drive configuration from the environment, these are the
supported overrides (the model is set with a single `AI_MODEL` key):

```bash
# OpenAI
export AI_PROVIDER=openai
export OPENAI_API_KEY=sk-your-key-here
export AI_MODEL=gpt-4o
./drogonclaw
```

```bash
# Local Ollama
export AI_PROVIDER=ollama
export OLLAMA_BASE_URL=http://localhost:11434
export AI_MODEL=llama3.1
./drogonclaw
```

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
3. Set both in the config (via `drogonclaw setup`, or directly in
   `~/.drogonclaw/config.json`):

   ```json
   {
     "TELEGRAM_TOKEN": "your-bot-token",
     "TELEGRAM_CHAT_ID": "your-chat-id"
   }
   ```

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
