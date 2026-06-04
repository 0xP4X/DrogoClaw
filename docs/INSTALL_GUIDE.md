# DrogonClaw - Complete Installation Guide

## Prerequisites

- **Node.js** 22.19.0 or higher
- **npm** 10.0.0 or higher
- Windows, macOS, or Linux
- For pentesting tools: Install Kali Linux tools or equivalent security tools

## Installation Steps

### Step 1: Clone Repository

```bash
git clone https://github.com/yourusername/drogonclaw.git
cd drogonclaw
```

### Step 2: Install Dependencies

```bash
npm install
```

This will automatically:
- Create all directories
- Install npm packages
- Generate source files
- Configure the project

### Step 3: Build Project

```bash
npm run build
```

Compiles TypeScript to JavaScript in the `dist/` directory.

### Step 4: Configuration

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env` with your settings:

```env
# AI Provider Configuration
AI_PROVIDER=claude
ANTHROPIC_API_KEY=sk-ant-v4-xxx
OPENAI_API_KEY=sk-proj-xxx

# Gateway Configuration
GATEWAY_PORT=18789
GATEWAY_HOST=localhost

# Database Configuration
DATABASE_PATH=./data/drogonclaw.db

# Logging
LOG_LEVEL=info

# Optional: Telegram Bot
TELEGRAM_TOKEN=xxx
TELEGRAM_CHAT_ID=xxx
```

### Step 5: Verify Installation

```bash
# Run linter
npm run lint

# Run tests
npm test

# Start gateway
npm run gateway

# In another terminal, test health endpoint
curl http://localhost:18789/health
```

## Usage

### CLI Mode

```bash
npm start

# Or with tsx (watch mode)
npm run cli
```

### Gateway Server

```bash
npm run gateway
```

Starts HTTP server on `localhost:18789` with:
- Health check: `GET /health`
- Sessions: `GET /api/sessions`
- Findings: `GET /api/findings`

### Development Mode

```bash
npm run dev
```

Watches for changes and rebuilds automatically.

## Configuration Options

All options can be set via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `AI_PROVIDER` | claude | AI model provider: claude, openai, ollama |
| `ANTHROPIC_API_KEY` | - | Anthropic API key for Claude |
| `OPENAI_API_KEY` | - | OpenAI API key for GPT |
| `OLLAMA_URL` | localhost:11434 | Ollama server URL |
| `GATEWAY_PORT` | 18789 | Port for HTTP gateway |
| `GATEWAY_HOST` | localhost | Host for gateway |
| `DATABASE_PATH` | ./data/drogonclaw.db | SQLite database file |
| `LOG_LEVEL` | info | Logging level: debug, info, warn, error |
| `TELEGRAM_TOKEN` | - | Telegram bot token |
| `TELEGRAM_CHAT_ID` | - | Telegram chat ID |

## Troubleshooting

### Node.js Version Error
```bash
node --version  # Should be >= 22.19.0
npm install -g node@latest
```

### Port Already in Use
```bash
# Change port
GATEWAY_PORT=18790 npm run gateway
```

### Database Errors
```bash
# Reset database
rm -rf data/
npm run setup
```

### Build Errors
```bash
# Clean rebuild
rm -rf dist/
npm run build
```

## Next Steps

1. **Read Quick Start**: `docs/QUICKSTART.md`
2. **Configure Tools**: `docs/CONFIGURATION.md`
3. **Create Skills**: `docs/SKILLS.md`
4. **API Docs**: `docs/API.md`
5. **Contribute**: `docs/DEVELOPMENT.md`

## Support

- Documentation: https://drogonclaw.xyz
- Issues: https://github.com/yourusername/drogonclaw/issues
- Contributing: See `CONTRIBUTING.md`

