# DrogonClaw - Configuration Guide

Complete reference for all configuration options.

## Environment Variables

All configuration is done through `.env` file or environment variables.

### AI Provider Configuration

#### Claude (Anthropic)

```env
AI_PROVIDER=claude
ANTHROPIC_API_KEY=sk-ant-v4-xxx-xxx
```

**Get API Key**: https://console.anthropic.com

#### OpenAI (GPT)

```env
AI_PROVIDER=openai
OPENAI_API_KEY=sk-proj-xxx-xxx
```

**Get API Key**: https://platform.openai.com/api-keys

#### Local Ollama

```env
AI_PROVIDER=ollama
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL_NAME=llama3.1:latest
```

Install Ollama: https://ollama.ai

### Gateway Configuration

```env
GATEWAY_PORT=18789              # HTTP server port
GATEWAY_HOST=localhost          # Bind address
ENABLE_WEBSOCKET=true          # Enable WebSocket support
```

### Database Configuration

```env
DATABASE_PATH=./data/drogonclaw.db
```

SQLite database location. Will be created automatically.

### Security Tools Configuration

```env
TOOL_WHITELIST=nmap,curl,dig,whois,traceroute,netstat
```

Comma-separated list of allowed tools. Only whitelisted tools can be executed.

### Telegram Bot Configuration

```env
ENABLE_TELEGRAM=false          # Enable Telegram bot
TELEGRAM_TOKEN=xxx             # Telegram bot token
TELEGRAM_CHAT_ID=xxx           # Telegram chat ID
```

**Get Telegram Token**: Talk to @BotFather on Telegram

### Logging Configuration

```env
LOG_LEVEL=info                 # debug, info, warn, error
```

### Timeout Configuration

```env
SESSION_TIMEOUT=3600000        # Session timeout (ms)
MAX_TOOL_TIMEOUT=300000        # Tool execution timeout (5 min)
```

## Example Configurations

### Development Setup

```env
AI_PROVIDER=ollama
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL_NAME=mistral
LOG_LEVEL=debug
GATEWAY_PORT=18789
ENABLE_WEBSOCKET=true
ENABLE_TELEGRAM=false
```

### Production Setup (Claude)

```env
AI_PROVIDER=claude
ANTHROPIC_API_KEY=sk-ant-v4-xxx
LOG_LEVEL=warn
GATEWAY_PORT=18789
GATEWAY_HOST=0.0.0.0
ENABLE_WEBSOCKET=true
ENABLE_TELEGRAM=true
TELEGRAM_TOKEN=xxx
TELEGRAM_CHAT_ID=xxx
DATABASE_PATH=/data/drogonclaw.db
```

### Production Setup (OpenAI)

```env
AI_PROVIDER=openai
OPENAI_API_KEY=sk-proj-xxx
LOG_LEVEL=info
GATEWAY_PORT=18789
GATEWAY_HOST=0.0.0.0
ENABLE_WEBSOCKET=true
DATABASE_PATH=/data/drogonclaw.db
```

## Tool Whitelisting

Control which system commands can be executed:

```env
TOOL_WHITELIST=nmap,curl,dig,whois,traceroute,netstat,grep,awk,sed
```

**Built-in Safe Tools:**
- nmap - Network scanning
- curl - HTTP requests
- dig - DNS queries
- whois - Domain information
- traceroute - Network path tracing
- netstat - Network statistics

**Add Custom Tools:**
```env
TOOL_WHITELIST=nmap,curl,dig,whois,my-custom-script
```

## Skill Configuration

Skills are YAML files in `skills/` directory.

### Creating a Skill

See `SKILLS.md` for complete guide.

## Advanced Configuration

### Host and Port Binding

```env
GATEWAY_HOST=0.0.0.0           # Listen on all interfaces
GATEWAY_PORT=8080              # Custom port
```

### Session Management

```env
SESSION_TIMEOUT=3600000        # 1 hour default
```

### Logging Levels

- `debug` - Very verbose, includes function calls
- `info` - Normal operation logs
- `warn` - Warning messages only
- `error` - Errors only

### Tool Timeouts

```env
MAX_TOOL_TIMEOUT=300000        # 5 minutes per tool
SESSION_TIMEOUT=3600000        # 1 hour total session
```

## Configuration Priority

Configuration is loaded in this order (highest to lowest priority):

1. Environment variables
2. `.env` file
3. Default values in code

Example:
```bash
# Environment variable overrides .env
GATEWAY_PORT=9000 npm start
```

## Validation

Configuration is validated on startup using Zod schema.

Invalid configuration will show:
```
Configuration validation failed
Errors: [...]
```

Fix the errors and restart.

## Default Configuration

If `.env` doesn't exist, DrogonClaw uses:

```env
AI_PROVIDER=claude
GATEWAY_PORT=18789
GATEWAY_HOST=localhost
DATABASE_PATH=./data/drogonclaw.db
LOG_LEVEL=info
SESSION_TIMEOUT=3600000
MAX_TOOL_TIMEOUT=300000
ENABLE_WEBSOCKET=true
ENABLE_TELEGRAM=false
```

## Security Best Practices

1. **Never commit `.env` to git**
   ```bash
   # .gitignore should have:
   .env
   .env.local
   .env.*.local
   ```

2. **Use strong API keys**
   - Rotate keys regularly
   - Use environment-specific keys
   - Don't share keys across projects

3. **Restrict tool whitelist**
   - Only allow necessary tools
   - Avoid shell=true execution
   - Use full paths to executables

4. **Enable logging for audit trail**
   ```env
   LOG_LEVEL=info
   ```

5. **Set proper timeouts**
   - Prevent hanging tools
   - Limit session duration
   - Configure resource limits

## Next Steps

- Create your `.env` file with your API keys
- Read `QUICKSTART.md` to get running
- See `SKILLS.md` to create custom skills
- Check `API.md` for API reference

