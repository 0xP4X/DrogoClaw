# Provider Configuration Guide

## 🌐 Supported Providers

DrogonClaw supports 6 AI providers with easy switching between them.

| Provider | Best For | Cost | Speed | Setup Difficulty |
|----------|----------|------|-------|------------------|
| **OpenRouter** | Multi-model access | Low-High | Fast | ⭐ Easy |
| **OpenAI** | GPT-4o, GPT-4o-mini | Medium-High | Fast | ⭐ Easy |
| **NVIDIA NIM** | High-performance inference | Low-Medium | Very Fast | ⭐⭐ Moderate |
| **Google Gemini** | Gemini 2.5, Flash | Low-Medium | Fast | ⭐ Easy |
| **Ollama** | Local/offline models | Free | Fast | ⭐⭐⭐ Advanced |
| **9Router.ai** | Intelligent routing | Varies | Fast | ⭐ Easy |

---

## 🚀 Quick Setup

### Interactive Setup Wizard

The easiest way to configure any provider:

```bash
./drogonclaw setup
```

This wizard will:
1. Ask which provider you want to use
2. Request your API key
3. Let you test the connection
4. Save configuration securely

### Manual Configuration

Edit `~/.drogonclaw/config.json`:

```json
{
  "AI_PROVIDER": "openrouter",
  "AI_MODEL": "meta-llama/llama-3.1-70b-instruct",
  "OPENROUTER_API_KEY": "sk-or-v1-...",
  "WORKSPACE_ROOT": "/home/user"
}
```

---

## 📋 Provider Details

### 1. OpenRouter

**Best for:** Access to 100+ models from one API

**Setup:**
```bash
# 1. Get API key from https://openrouter.ai
# 2. Configure
vim ~/.drogonclaw/config.json
```

```json
{
  "AI_PROVIDER": "openrouter",
  "AI_MODEL": "meta-llama/llama-3.1-70b-instruct",
  "OPENROUTER_API_KEY": "sk-or-v1-..."
}
```

**Recommended Models:**
```json
// Cost-effective (default)
"AI_MODEL": "meta-llama/llama-3.1-70b-instruct"  // $0.30/1M

// High quality
"AI_MODEL": "anthropic/claude-3.5-sonnet"        // $3.00/1M

// Budget option
"AI_MODEL": "qwen/qwen-2.5-7b-instruct"          // $0.05/1M

// Ultra-fast
"AI_MODEL": "google/gemini-2.0-flash-exp"        // $0.075/1M
```

**Environment Variables:**
```bash
export AI_PROVIDER=openrouter
export AI_MODEL=meta-llama/llama-3.1-70b-instruct
export OPENROUTER_API_KEY=sk-or-v1-...
```

---

### 2. OpenAI

**Best for:** GPT-4o and GPT-4o-mini access

**Setup:**
```json
{
  "AI_PROVIDER": "openai",
  "AI_MODEL": "gpt-4o",
  "OPENAI_API_KEY": "sk-..."
}
```

**Available Models:**
- `gpt-4o` - Latest, most capable ($5.00/1M input, $15.00/1M output)
- `gpt-4o-mini` - Fast and cheap ($0.15/1M input, $0.60/1M output)
- `gpt-4-turbo` - Previous generation ($10.00/1M input, $30.00/1M output)

**Get API Key:**
1. Visit https://platform.openai.com/api-keys
2. Create new secret key
3. Copy and save securely

**Environment Variables:**
```bash
export AI_PROVIDER=openai
export AI_MODEL=gpt-4o
export OPENAI_API_KEY=sk-...
```

---

### 3. NVIDIA NIM

**Best for:** High-performance inference with enterprise models

**Setup:**
```json
{
  "AI_PROVIDER": "nvidia",
  "AI_MODEL": "meta/llama-3.1-70b-instruct",
  "NVIDIA_API_KEY": "nvapi-..."
}
```

**Available Models:**
- `meta/llama-3.1-70b-instruct` - Llama 3.1 70B ($0.35/1M)
- `nvidia/nemotron-4-340b-instruct` - NVIDIA flagship ($5.00/1M)
- `mistralai/mistral-large` - Mistral Large ($2.00/1M)
- `qwen/qwen2.5-72b-instruct` - Qwen 2.5 72B ($0.40/1M)

**Get API Key:**
1. Visit https://build.nvidia.com
2. Sign in with NVIDIA account
3. Generate API key
4. Copy key starting with `nvapi-`

**Environment Variables:**
```bash
export AI_PROVIDER=nvidia
export AI_MODEL=meta/llama-3.1-70b-instruct
export NVIDIA_API_KEY=nvapi-...
```

---

### 4. Google Gemini

**Best for:** Google's Gemini models with competitive pricing

**Setup:**
```json
{
  "AI_PROVIDER": "gemini",
  "AI_MODEL": "gemini-2.5-pro",
  "GOOGLE_API_KEY": "AIza..."
}
```

**Available Models:**
- `gemini-2.5-pro` - Latest flagship ($1.25/1M input, $5.00/1M output)
- `gemini-2.5-flash` - Fast and cheap ($0.075/1M input, $0.30/1M output)
- `gemini-1.5-pro` - Previous gen ($1.25/1M input, $5.00/1M output)

**Get API Key:**
1. Visit https://makersuite.google.com/app/apikey
2. Create API key
3. Enable Generative Language API
4. Copy key starting with `AIza`

**Environment Variables:**
```bash
export AI_PROVIDER=gemini
export AI_MODEL=gemini-2.5-flash
export GOOGLE_API_KEY=AIza...
```

---

### 5. Ollama (Local/Offline)

**Best for:** Running models locally without internet or API costs

**Setup:**

1. **Install Ollama:**
```bash
# Linux/macOS
curl -fsSL https://ollama.com/install.sh | sh

# Or download from https://ollama.com/download
```

2. **Pull a model:**
```bash
ollama pull llama3.1:70b
# Or other models:
# ollama pull qwen2.5:7b
# ollama pull mistral:7b
# ollama pull codellama:13b
```

3. **Start Ollama server:**
```bash
ollama serve
# Runs on http://localhost:11434
```

4. **Configure DrogonClaw:**
```json
{
  "AI_PROVIDER": "ollama",
  "AI_MODEL": "llama3.1:70b",
  "OLLAMA_BASE_URL": "http://localhost:11434"
}
```

**Available Models:**
- `llama3.1:70b` - Llama 3.1 70B (40GB VRAM)
- `llama3.1:8b` - Llama 3.1 8B (4.7GB VRAM)
- `qwen2.5:7b` - Qwen 2.5 7B (4.7GB VRAM)
- `mistral:7b` - Mistral 7B (4.1GB VRAM)
- `codellama:13b` - Code Llama 13B (7.4GB VRAM)

**Hardware Requirements:**
- 8B models: 8GB+ RAM
- 13B models: 16GB+ RAM
- 70B models: 48GB+ RAM

**Environment Variables:**
```bash
export AI_PROVIDER=ollama
export AI_MODEL=llama3.1:8b
export OLLAMA_BASE_URL=http://localhost:11434
```

---

### 6. 9Router.ai (Intelligent Routing)

**Best for:** Automatic cost optimization across providers

**Setup:**
```json
{
  "AI_PROVIDER": "9router",
  "NINEROUTER_API_KEY": "sk-9r-...",
  "ROUTER_MODE": "auto"
}
```

**How it works:**
- Analyzes each request
- Routes to optimal model automatically
- Falls back to local rules if unavailable
- Tracks cost savings

**Get API Key:**
1. Visit https://9router.ai
2. Sign up for account
3. Generate API key
4. Copy key starting with `sk-9r-`

See [ROUTING.md](ROUTING.md) for complete routing guide.

---

## 🔄 Switching Providers

### Runtime Switching

You can switch providers without restarting:

```bash
# In TUI
> /config

# Edit config file
vim ~/.drogonclaw/config.json

# Reload config
> /setup
# Or restart DrogonClaw
```

### Quick Switch Script

```bash
#!/bin/bash
# switch-provider.sh

PROVIDER=$1
case $PROVIDER in
  openrouter)
    export AI_PROVIDER=openrouter
    export AI_MODEL=meta-llama/llama-3.1-70b-instruct
    ;;
  openai)
    export AI_PROVIDER=openai
    export AI_MODEL=gpt-4o-mini
    ;;
  nvidia)
    export AI_PROVIDER=nvidia
    export AI_MODEL=meta/llama-3.1-70b-instruct
    ;;
  gemini)
    export AI_PROVIDER=gemini
    export AI_MODEL=gemini-2.5-flash
    ;;
  ollama)
    export AI_PROVIDER=ollama
    export AI_MODEL=llama3.1:8b
    ;;
esac

./drogonclaw
```

Usage:
```bash
chmod +x switch-provider.sh
./switch-provider.sh openai
```

---

## 💰 Cost Comparison

### Cost per 1M Tokens (Input/Output Combined)

| Provider | Budget Models | Mid-Range | Premium |
|----------|---------------|-----------|---------|
| **OpenRouter** | $0.05-0.15 | $0.30-1.00 | $3.00-15.00 |
| **OpenAI** | $0.15 (mini) | - | $5.00-15.00 |
| **NVIDIA** | $0.35-0.50 | $2.00-3.00 | $5.00+ |
| **Gemini** | $0.075 (flash) | - | $1.25-5.00 |
| **Ollama** | Free | Free | Free |
| **9Router** | Varies (optimized) | - | - |

### Example Session Costs

**Typical pentest session (20 requests, ~100K tokens total):**

| Provider | Model | Cost |
|----------|-------|------|
| OpenRouter | Llama 3.1 70B | $0.03 |
| OpenAI | GPT-4o-mini | $0.015 |
| NVIDIA | Llama 3.1 70B | $0.035 |
| Gemini | Gemini 2.5 Flash | $0.0075 |
| Ollama | Llama 3.1 8B | $0.00 |
| 9Router | Auto-routed | $0.006 (80% savings) |

---

## 🔧 Advanced Configuration

### Multiple API Keys

Keep backup keys for failover:

```json
{
  "OPENROUTER_API_KEY": "sk-or-v1-primary",
  "OPENROUTER_API_KEY_BACKUP": "sk-or-v1-backup",
  "OPENAI_API_KEY": "sk-openai-primary"
}
```

### Custom Base URLs

For proxies or custom endpoints:

```json
{
  "AI_PROVIDER": "openai",
  "OPENAI_BASE_URL": "https://my-proxy.example.com/v1"
}
```

### Rate Limiting

Configure rate limits per provider:

```bash
> /stealth

# Enables adaptive rate limiting
# Respects 429/503 responses
# Adds delay between requests
```

---

## 🔍 Troubleshooting

### Provider Connection Failed

```bash
> /health
```

This checks:
- Provider connectivity
- API key validity
- Model availability
- Sandbox status

### API Key Invalid

**Symptoms:**
- 401 Unauthorized errors
- "Invalid API key" messages

**Solutions:**
1. Verify key in config: `/config`
2. Check key hasn't expired
3. Regenerate key from provider
4. Ensure no extra spaces/quotes

### Model Not Available

**Symptoms:**
- 404 Model not found
- "Model does not exist"

**Solutions:**
1. Check model ID spelling
2. Verify model is available from provider
3. Try alternative model
4. Check provider's model list

### Ollama Connection Refused

**Symptoms:**
- "Connection refused localhost:11434"

**Solutions:**
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Start Ollama if not running
ollama serve

# Check custom port
export OLLAMA_BASE_URL=http://localhost:11434
```

### Slow Responses

**Possible causes:**
1. Provider rate limiting
2. Network latency
3. Model size (large models slower)
4. Provider load

**Solutions:**
- Try different provider
- Use faster model (8B instead of 70B)
- Enable routing for automatic optimization
- Check network connectivity

---

## 🎯 Best Practices

### 1. Start with OpenRouter
- Easiest setup
- Access to 100+ models
- Competitive pricing
- Good for testing

### 2. Use Fast Models for Recon
```json
// For reconnaissance tasks
"AI_MODEL": "qwen/qwen-2.5-7b-instruct"  // $0.05/1M
```

### 3. Use Premium for Exploitation
```json
// For complex exploits
"AI_MODEL": "gpt-4o"  // $5.00/1M
```

### 4. Enable Routing
```bash
> /router auto
```
Let DrogonClaw optimize automatically.

### 5. Keep Backup Provider
```json
{
  "AI_PROVIDER": "openrouter",
  "OPENROUTER_API_KEY": "sk-or-...",
  "OPENAI_API_KEY": "sk-..."  // Backup
}
```

### 6. Monitor Costs
```bash
> /cost
```
Check regularly to avoid surprises.

### 7. Use Ollama for Offline
Keep Ollama configured for:
- Air-gapped environments
- Sensitive operations
- Development/testing

---

## 📊 Provider Comparison Matrix

| Feature | OpenRouter | OpenAI | NVIDIA | Gemini | Ollama | 9Router |
|---------|-----------|--------|--------|--------|--------|---------|
| **Setup** | ⭐ Easy | ⭐ Easy | ⭐⭐ Moderate | ⭐ Easy | ⭐⭐⭐ Hard | ⭐ Easy |
| **Cost** | $-$$$ | $$-$$$ | $-$$$ | $-$$ | Free | $ (optimized) |
| **Speed** | Fast | Fast | Very Fast | Fast | Depends | Fast |
| **Models** | 100+ | 3 | 10+ | 3 | 50+ | All |
| **Offline** | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ |
| **Privacy** | Medium | Medium | Medium | Medium | High | Medium |
| **Latency** | 200-500ms | 150-400ms | 100-300ms | 200-450ms | 50-2000ms | 200-500ms |

---

## 🔐 Security Best Practices

### API Key Storage

**✅ DO:**
- Store in `~/.drogonclaw/config.json` (chmod 600)
- Use environment variables
- Rotate keys regularly

**❌ DON'T:**
- Commit keys to git
- Share keys in logs
- Use keys across teams

### Key Rotation

```bash
# 1. Generate new key from provider
# 2. Update config
vim ~/.drogonclaw/config.json

# 3. Test new key
./drogonclaw health

# 4. Revoke old key from provider dashboard
```

### Environment Isolation

```bash
# Production
export OPENROUTER_API_KEY=$PROD_KEY

# Development  
export OPENROUTER_API_KEY=$DEV_KEY

# Testing
export OPENROUTER_API_KEY=$TEST_KEY
```

---

## 📚 Additional Resources

- [OpenRouter Documentation](https://openrouter.ai/docs)
- [OpenAI API Reference](https://platform.openai.com/docs/api-reference)
- [NVIDIA NIM Documentation](https://docs.nvidia.com/nim)
- [Google AI Studio](https://makersuite.google.com)
- [Ollama Documentation](https://github.com/ollama/ollama/tree/main/docs)
- [9Router.ai Docs](https://docs.9router.ai)

---

## 💡 Quick Tips

1. **Test providers:** Use `/health` after setup
2. **Compare costs:** Track with `/cost`
3. **Enable routing:** Use `/router auto` for optimization
4. **Keep backup:** Configure multiple providers
5. **Monitor usage:** Check provider dashboards
6. **Rotate keys:** Update keys every 90 days
7. **Use fast models:** For simple tasks
8. **Go premium:** For critical exploits

---

**Need help?** Join our [Discord](https://discord.gg/drogonclaw) or open an [issue](https://github.com/0xP4X/drogonclaw/issues).
