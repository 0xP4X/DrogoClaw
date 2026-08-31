# Intelligent Routing Guide

## 🧠 Overview

DrogonClaw's intelligent routing system automatically selects the optimal AI model for each task, balancing cost, performance, and quality. This can reduce costs by 18-25% while maintaining or improving response quality.

## 🎯 How It Works

### Task-Based Routing

Different penetration testing tasks have different requirements:

```
📊 Reconnaissance    → Fast, cheap models
🎯 Exploitation      → Premium, high-quality models  
📝 Report Writing    → Writing-focused models
🧠 Analysis          → High-quality reasoning models
📋 Planning          → Medium-quality models
💬 Simple Queries    → Ultra-cheap models
```

The router analyzes each request and routes it to the most appropriate model.

---

## 🚀 Quick Start

### 1. Enable Routing

```bash
# Using local rules (no API key needed)
./drogonclaw
> /router local

# Using 9router.ai service (requires API key)
> /router 9router

# Auto mode (9router.ai with local fallback)
> /router auto
```

### 2. Configure API Key (for 9router.ai)

```bash
# Edit config
vim ~/.drogonclaw/config.json

# Add:
{
  "NINEROUTER_API_KEY": "sk-9r-your-key-here",
  "ROUTER_MODE": "auto"
}
```

### 3. Verify Status

```bash
> /router status
```

---

## 📊 Routing Rules

### Default Task-Based Rules

| Task Type | Max Cost/1M | Latency | Preferred Models | Min Quality |
|-----------|-------------|---------|------------------|-------------|
| **Chat** | $0.10 | 2s | Qwen 7B, Gemini Flash | 70% |
| **Recon** | $0.15 | 5s | Llama 8B, Gemini Flash | 75% |
| **Planning** | $3.00 | 10s | Llama 70B, GPT-4o-mini | 85% |
| **Exploitation** | $15.00 | 30s | GPT-4o, Claude Sonnet | 90% |
| **Reporting** | $5.00 | 20s | Claude Sonnet, GPT-4o | 88% |
| **Analysis** | $10.00 | 30s | GPT-4o, Claude Sonnet | 90% |

### How Tasks Are Classified

```go
// Simple queries → Chat
"what is SQL injection?"

// Reconnaissance commands → Recon
"scan 192.168.1.0/24 for open ports"
"enumerate subdomains of example.com"

// Creating mission plans → Planning
"create attack plan for web application"

// Exploitation attempts → Exploitation
"exploit CVE-2024-1234 on target"
"attempt privilege escalation"

// Report generation → Reporting
"generate executive summary report"
"document findings in markdown"

// Deep analysis → Analysis
"analyze this binary for vulnerabilities"
"explain this exploit technique"
```

---

## 🎛️ Routing Modes

### 1. **OFF** (Default)
```bash
/router off
```
- No intelligent routing
- Uses configured provider/model
- Predictable behavior
- **Use when:** You want full control

### 2. **LOCAL**
```bash
/router local
```
- Uses built-in routing rules
- No external API needed
- Fast decisions (<1ms)
- **Use when:** You want cost optimization without external dependencies

**How it works:**
1. Analyzes task type from prompt
2. Checks prompt for urgency/complexity keywords
3. Selects best model from local rules
4. Falls back to default if all preferred providers unavailable

### 3. **9ROUTER**
```bash
/router 9router
```
- Uses 9router.ai service
- Advanced ML-based routing
- Real-time provider availability
- **Use when:** You want maximum optimization
- **Requires:** API key from 9router.ai

**How it works:**
1. Sends request to 9router.ai API with task context
2. 9router.ai analyzes and selects optimal model
3. Returns provider + model + reasoning
4. Falls back to local rules if API unavailable

### 4. **AUTO** (Recommended)
```bash
/router auto
```
- Intelligent fallback chain
- Best of both worlds
- Maximum reliability
- **Use when:** You want "set it and forget it"

**Fallback chain:**
```
9router.ai → Local Rules → Default Provider
```

---

## 💰 Cost Optimization Examples

### Without Routing
```
Task: "scan 192.168.1.0/24"
Model: GPT-4o ($15/1M tokens)
Cost: $0.045 (3000 tokens)
```

### With Routing
```
Task: "scan 192.168.1.0/24"
Model: Llama 3.1 8B ($0.10/1M tokens)
Cost: $0.0003 (3000 tokens)
Savings: $0.0447 (99.3% reduction!)
```

### Real-World Session Example

**Without Routing:**
```
10 recon tasks    @ $15/1M  = $0.45
5 planning tasks  @ $15/1M  = $0.23
2 exploit tasks   @ $15/1M  = $0.09
3 reporting tasks @ $15/1M  = $0.14
─────────────────────────────────
Total: $0.91
```

**With Intelligent Routing:**
```
10 recon tasks    @ $0.15/1M = $0.005
5 planning tasks  @ $3.00/1M = $0.045
2 exploit tasks   @ $15.0/1M = $0.090
3 reporting tasks @ $5.00/1M = $0.045
─────────────────────────────────
Total: $0.185
Savings: $0.725 (80% reduction!)
```

---

## 🔧 Configuration

### Config File

`~/.drogonclaw/config.json`:
```json
{
  "ROUTER_MODE": "auto",
  "NINEROUTER_API_KEY": "sk-9r-...",
  "AI_PROVIDER": "openrouter",
  "AI_MODEL": "meta-llama/llama-3.1-70b-instruct"
}
```

### Environment Variables

```bash
export ROUTER_MODE=auto
export NINEROUTER_API_KEY=sk-9r-...
export AI_PROVIDER=openrouter
export OPENROUTER_API_KEY=sk-or-...
```

### Runtime Commands

```bash
# Enable routing
/router auto

# Check status
/router status

# View cost savings
/cost

# Disable routing
/router off
```

---

## 📈 Monitoring

### Status Report

```bash
> /router status

INTELLIGENT ROUTING STATUS
──────────────────────────────────────────────

Mode             ● AUTO
Provider         openrouter
Model            meta-llama/llama-3.1-70b-instruct

ROUTING RULES
Chat/Simple      Fast models ($0.10/1M)
Recon            Fast models ($0.15/1M)
Planning         Medium models ($3.00/1M)
Exploitation     Premium models ($15.00/1M)
Reporting        Writing models ($5.00/1M)
Analysis         High-quality ($10.00/1M)

Commands: /router auto | local | 9router | off
```

### Cost Tracking

```bash
> /cost

TOKEN USAGE & COST
──────────────────────────────────────────────

Prompt Tokens      12,450
Completion Tokens  8,920
Total Tokens       21,370

Estimated Cost     $0.185
Baseline Cost      $0.910 (without routing)
Savings            $0.725 (80% reduction)
```

### Sidebar Display

The sidebar shows real-time routing status:

```
ROUTING
Mode        ● AUTO
Provider    9router.ai
Tasks       47 routed
Savings     $2.34 (18%)
```

---

## 🎯 Advanced Features

### Prompt Analysis

The local router analyzes prompts for context hints:

**Urgency Detection:**
```
"URGENT: exploit this NOW"  → Prefers fast models
"ASAP scan required"        → Prefers fast models
```

**Complexity Detection:**
```
"complex exploit chain"     → Prefers premium models
"sophisticated attack"      → Prefers premium models
```

**Simplicity Detection:**
```
"simple port scan"          → Prefers cheap models
"quick enumeration"         → Prefers cheap models
```

### Provider Health Tracking

The router monitors provider health:

```go
Provider: openrouter
Status:   ● UP
Latency:  245ms (average)
Success:  98.5%
Requests: 1,247
```

If a provider becomes unavailable, routing automatically switches to alternatives.

---

## 🔍 Troubleshooting

### Routing Not Working

**Check mode:**
```bash
> /router status
```

If mode is "OFF", enable it:
```bash
> /router auto
```

### 9Router.ai API Errors

**Check API key:**
```bash
> /config
```

Look for `NINEROUTER_API_KEY`. If missing:
```bash
vim ~/.drogonclaw/config.json
# Add: "NINEROUTER_API_KEY": "sk-9r-..."
```

**Verify connectivity:**
```bash
curl -H "Authorization: Bearer sk-9r-..." \
     https://api.9router.ai/v1/health
```

### High Costs Despite Routing

**Check which tasks are expensive:**
```bash
> /timeline
```

Look for exploitation or analysis tasks. These intentionally use premium models for quality.

**Adjust rules** (advanced):
Edit `internal/router/router.go` to change cost limits.

### Models Not Available

**Check provider status:**
```bash
> /providers status
```

If preferred providers are down, routing falls back to alternatives.

---

## 🚦 Best Practices

### 1. Start with Auto Mode
```bash
/router auto
```
This gives you intelligent routing with automatic fallbacks.

### 2. Monitor Costs
Check costs regularly:
```bash
/cost
```

### 3. Use Local Mode for Privacy
If you don't want to send prompts to external APIs:
```bash
/router local
```

### 4. Trust the Premium Models
Don't over-optimize exploitation tasks. Premium models ($15/1M) are worth it for complex exploits where failure is expensive.

### 5. Review Savings
After each session:
```bash
> /cost

Savings: $2.34 (18%)
```

---

## 📊 Performance Impact

### Routing Overhead

| Mode | Decision Time | Network | Total Overhead |
|------|---------------|---------|----------------|
| OFF | 0ms | None | **0ms** |
| LOCAL | <1ms | None | **<1ms** |
| 9ROUTER | 50-150ms | Yes | **50-150ms** |
| AUTO | <1ms (fallback) | Optional | **<1-150ms** |

The overhead is negligible compared to LLM inference time (1-30 seconds).

---

## 🔐 Privacy & Security

### Local Mode
- ✅ No external API calls
- ✅ Prompts never leave your machine
- ✅ Works offline
- ✅ Fast decisions

### 9Router.ai Mode
- ⚠️ Sends first 500 chars of prompt for context
- ⚠️ Sends task type and constraints
- ✅ Does NOT send full prompt
- ✅ API key encrypted in transit (HTTPS)
- ✅ No prompt storage (per 9router.ai policy)

**For sensitive operations:** Use local mode.

---

## 📚 API Reference

### Router Interface

```go
type Router interface {
    Route(ctx context.Context, taskType TaskType, prompt string) (*RouteDecision, error)
    GetStats() *RoutingStats
    IsAvailable() bool
}

type RouteDecision struct {
    Provider      string
    Model         string
    EstimatedCost float64
    Reasoning     string
    Timestamp     time.Time
}
```

### Task Types

```go
const (
    TaskChat          TaskType = iota  // Simple queries
    TaskRecon                           // Reconnaissance
    TaskPlanning                        // Mission planning
    TaskExploitation                    // Exploitation
    TaskReporting                       // Report generation
    TaskAnalysis                        // Deep analysis
)
```

---

## 🎓 Learn More

- [9router.ai Documentation](https://docs.9router.ai) - Official API docs
- [PROVIDERS.md](PROVIDERS.md) - Provider configuration guide
- [Cost Optimization Blog Post](docs/blog/cost-optimization.md) - Deep dive
- [TUI.md](TUI.md) - Complete TUI guide

---

## 💡 Tips

1. **Start with AUTO mode** - Best balance of cost and reliability
2. **Monitor your costs** - Use `/cost` regularly
3. **Trust premium models** - Don't over-optimize critical tasks
4. **Use local mode for sensitive data** - No external API calls
5. **Check provider health** - Use `/providers status` if routing seems slow

---

**Questions?** Open an issue on [GitHub](https://github.com/0xP4X/drogonclaw/issues) or check `/help` in the TUI.
