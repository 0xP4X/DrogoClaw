# DrogonClaw - Quick Start Guide

Get DrogonClaw running in 5 minutes.

## 1. Install (2 minutes)

```bash
npm install
npm run build
npm run lint
```

## 2. Configure (1 minute)

```bash
cp .env.example .env
# Edit .env with your ANTHROPIC_API_KEY or OPENAI_API_KEY
```

## 3. Run (2 minutes)

### Start the Gateway Server

```bash
npm run gateway
```

You'll see:
```
✓ Gateway listening on localhost:18789
```

### In Another Terminal, Run the CLI

```bash
npm start
```

Follow the prompts:
1. Enter target (e.g., `example.com` or `192.168.1.1`)
2. Select strategy: `light`, `thorough`, `deep`, or `aggressive`
3. Watch findings appear in real-time

## Testing the API

Once gateway is running:

```bash
# Health check
curl http://localhost:18789/health

# List sessions
curl http://localhost:18789/api/sessions

# List findings
curl http://localhost:18789/api/findings
```

## What Happens Next

DrogonClaw will:
1. Load your target and strategy
2. Call the AI model to plan reconnaissance
3. Execute security tools (nmap, curl, dig, etc.)
4. Parse output and discover findings
5. Continue until target is fully assessed

Findings are saved to SQLite database at `./data/drogonclaw.db`

## View Results

```bash
# Results are displayed in real-time
# Also saved to findings table in SQLite

sqlite3 ./data/drogonclaw.db
> SELECT * FROM findings;
```

## Generate Report

```bash
# TODO: Implement report generation
npm run report session-id
```

## Next Steps

- **Telegram Bot**: Set `TELEGRAM_TOKEN` in `.env` and `npm run telegram`
- **Custom Skills**: Create `.yaml` files in `skills/` directory
- **Advanced Config**: See `CONFIGURATION.md`
- **API Reference**: See `API.md`

## Troubleshooting

| Problem | Solution |
|---------|----------|
| "API key not set" | Edit `.env` with your key |
| "Port already in use" | Change `GATEWAY_PORT` in `.env` |
| "Database error" | Run `rm -rf data/` and retry |
| "Build failed" | Run `npm install` again |

## Common Commands

```bash
npm start           # Run CLI
npm run gateway     # Start server
npm run dev         # Watch mode
npm test            # Run tests
npm run lint        # Check code
npm run build       # Compile TypeScript
```

That's it! You're ready to pentesting with DrogonClaw.

