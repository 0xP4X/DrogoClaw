# DrogonClaw - Complete Deployment Guide

This guide walks you through deploying DrogonClaw from start to finish.

## 📋 Pre-Deployment Checklist

- [ ] Node.js 22.19.0+ installed
- [ ] npm 10.0.0+ installed
- [ ] Git installed
- [ ] Claude/OpenAI API key (or Ollama installed)
- [ ] Port 18789 available (or change GATEWAY_PORT)

## 🚀 Step 1: Clone and Initialize

### Option A: From GitHub

```bash
git clone https://github.com/yourusername/drogonclaw.git
cd drogonclaw
```

### Option B: From Local Directory

```bash
cd /path/to/drogonclaw
```

## 📦 Step 2: Install Dependencies

```bash
npm install
```

This will:
1. Run `preinstall` hook (creates directories)
2. Install npm packages
3. Run `postinstall` hook (generates source files)

**Expected output:**
```
added 180 packages in 45s
✓ Directory structure created!
✓ Generated source files
```

## 🔨 Step 3: Build Project

```bash
npm run build
```

This compiles TypeScript to JavaScript in the `dist/` directory.

**Expected output:**
```
✓ Successfully compiled TypeScript
✓ Output: dist/
```

## ✅ Step 4: Verify Build

```bash
npm run lint
npm test
```

Should see:
```
✓ All lint checks passed
✓ All tests passed
```

## 🔑 Step 5: Configure Environment

### Create .env file

```bash
cp .env.example .env
```

### Edit .env with your settings

Choose one AI provider:

**Option A: Claude (Recommended)**
```env
AI_PROVIDER=claude
ANTHROPIC_API_KEY=sk-ant-v4-xxx-xxx
```
Get API key: https://console.anthropic.com

**Option B: OpenAI**
```env
AI_PROVIDER=openai
OPENAI_API_KEY=sk-proj-xxx-xxx
```
Get API key: https://platform.openai.com/api-keys

**Option C: Ollama (Local)**
```env
AI_PROVIDER=ollama
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=mistral
```
Install Ollama: https://ollama.ai

### Complete .env file

```env
# AI Provider
AI_PROVIDER=claude
ANTHROPIC_API_KEY=sk-ant-v4-xxx

# Gateway
GATEWAY_PORT=18789
GATEWAY_HOST=localhost
ENABLE_WEBSOCKET=true

# Database
DATABASE_PATH=./data/drogonclaw.db

# Logging
LOG_LEVEL=info

# Tools
TOOL_WHITELIST=nmap,curl,dig,whois,traceroute,netstat

# Optional: Telegram
ENABLE_TELEGRAM=false
# TELEGRAM_TOKEN=xxx
# TELEGRAM_CHAT_ID=xxx
```

## 🎯 Step 6: Test Configuration

```bash
npm run build
```

Should complete without errors. If errors appear:
- Check your .env syntax
- Verify API keys are correct
- See CONFIGURATION.md for help

## 🖥️ Step 7: Run DrogonClaw

### Terminal 1: Start Gateway Server

```bash
npm run gateway
```

Expected output:
```
✓ Gateway listening on localhost:18789
✓ Database initialized
```

### Terminal 2: Use CLI

```bash
npm start
```

Follow the interactive prompts:
```
? Enter target (domain or IP): example.com
? Select strategy: (thorough)
```

### Test in Terminal 3

```bash
# Health check
curl http://localhost:18789/health

# List sessions
curl http://localhost:18789/api/sessions

# List findings
curl http://localhost:18789/api/findings
```

## 🐳 Step 8: Docker Deployment (Optional)

### Build Docker Image

```bash
docker build -t drogonclaw:latest .
```

### Run in Docker

```bash
docker run -p 18789:18789 \
  -e ANTHROPIC_API_KEY=sk-ant-xxx \
  -v $(pwd)/data:/app/data \
  drogonclaw:latest
```

### Docker Compose (Recommended)

```bash
# Update docker-compose.yml with your API key
nano docker-compose.yml

# Start services
docker-compose up -d

# View logs
docker-compose logs -f drogonclaw
```

## 🔒 Step 9: Production Configuration

### For Production Deployment

```env
GATEWAY_HOST=0.0.0.0           # Listen on all interfaces
LOG_LEVEL=info                 # Don't expose debug info
ENABLE_WEBSOCKET=true          # Enable for monitoring
DATABASE_PATH=/data/drogonclaw.db  # Use persistent volume
```

### Security Hardening

1. **Firewall**: Restrict access to port 18789
2. **HTTPS**: Use reverse proxy (nginx, traefik)
3. **Authentication**: Add API key authentication
4. **Secrets**: Use secret management (Vault, K8s secrets)
5. **Logging**: Send logs to centralized system

### Nginx Reverse Proxy

```nginx
server {
    listen 443 ssl http2;
    server_name pentesting.example.com;

    ssl_certificate /etc/letsencrypt/live/pentesting.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/pentesting.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:18789;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 📊 Step 10: Monitoring

### View Application Logs

**Development:**
```bash
npm run dev
```

**Production:**
```bash
tail -f logs/drogonclaw.log
```

### Monitor Database

```bash
# View sessions
sqlite3 data/drogonclaw.db "SELECT * FROM sessions;"

# View findings
sqlite3 data/drogonclaw.db "SELECT * FROM findings;"
```

### Health Checks

```bash
# Monitor endpoint
watch -n 5 'curl http://localhost:18789/health'

# Or with jq
curl http://localhost:18789/health | jq .
```

## 🚨 Troubleshooting

### Port Already in Use

```bash
# Find process using port 18789
lsof -i :18789

# Kill process
kill -9 <PID>

# Or change port
GATEWAY_PORT=9000 npm run gateway
```

### API Key Issues

```bash
# Verify API key format
grep ANTHROPIC_API_KEY .env

# Test API key
curl -X POST https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model":"claude-3-sonnet-20240229","max_tokens":1024,"messages":[{"role":"user","content":"hello"}]}'
```

### Database Errors

```bash
# Reset database
rm -rf data/
npm run setup

# Or check integrity
sqlite3 data/drogonclaw.db ".check"
```

### Build Errors

```bash
# Clean rebuild
rm -rf dist node_modules package-lock.json
npm install
npm run build
```

## 📝 Common Tasks

### Start Development Server

```bash
npm run dev
```

### Run Tests

```bash
npm test
npm test:watch
```

### Check Code Quality

```bash
npm run lint
npm run format
```

### View API Documentation

Open browser to: http://localhost:18789/api/docs

(Requires additional setup - see API.md)

## 🔄 Updating DrogonClaw

```bash
git pull origin main
npm install
npm run build
npm test
npm run gateway
```

## 📚 Next Steps

1. **Read Documentation**
   - [API Reference](API.md) - Learn the API
   - [Skills Guide](SKILLS.md) - Create custom skills
   - [Configuration](CONFIGURATION.md) - Advanced setup

2. **Create Skills**
   - Add pentesting workflows in `skills/` directory
   - See example files: `example-skill-dns.yaml`

3. **Integrate with Tools**
   - Add custom tools to `TOOL_WHITELIST`
   - Extend agent with new capabilities

4. **Deploy to Production**
   - Set up Docker
   - Configure HTTPS/TLS
   - Enable monitoring
   - Set up logging

## 🆘 Support

- **Documentation**: https://drogonclaw.xyz
- **Issues**: https://github.com/yourusername/drogonclaw/issues
- **Discussions**: https://github.com/yourusername/drogonclaw/discussions
- **Email**: support@drogonclaw.dev

## ✅ Final Checklist

- [ ] Node.js installed (22.19.0+)
- [ ] Repository cloned
- [ ] npm install completed
- [ ] npm run build succeeded
- [ ] .env configured with API key
- [ ] npm run gateway started successfully
- [ ] API endpoint responding (/health)
- [ ] Tests passing (npm test)
- [ ] Lint passing (npm run lint)
- [ ] Ready for pentesting!

## 🎉 You're Done!

DrogonClaw is now running and ready for autonomous pentesting.

**Start pentesting:**
```bash
npm start
```

**Happy hacking! 🐉**

