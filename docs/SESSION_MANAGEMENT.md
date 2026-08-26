# Session Management

DrogonClaw maintains persistent HTTP sessions, adaptive rate limiting, response caching, and WAF detection. Inspired by Scrapling's session management and AutoThrottle patterns.

## Overview

```
┌─────────────────────────────────────────────────────┐
│               Session Manager                        │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────┐ │
│  │ Cookie Jar  │  │ Custom       │  │ Disk      │ │
│  │ per Domain  │  │ Headers      │  │ Persistence│ │
│  └─────────────┘  └──────────────┘  └───────────┘ │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────┼──────────────────────────────┐
│               AutoThrottle                           │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐ │
│  │ Response Time│  │ Exponential  │  │ Retry-After│ │
│  │ Measurement  │  │ Backoff      │  │ Parsing   │ │
│  └──────────────┘  └──────────────┘  └───────────┘ │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────┼──────────────────────────────┐
│               Response Cache                         │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐ │
│  │ Disk Cache   │  │ TTL-Based    │  │ Hit Count │ │
│  │              │  │ Expiry       │  │ Tracking  │ │
│  └──────────────┘  └──────────────┘  └───────────┘ │
└─────────────────────────────────────────────────────┘
```

## Session Persistence

### Problem
Without session persistence, every HTTP tool call is independent. If you login to a target web application, the next tool call forgets the session cookie and you're logged out.

### Solution
The `SessionManager` maintains persistent cookie jars per domain:

```go
sm := httputil.NewSessionManager("data")

// Get a client with persistent cookies for the domain
client := sm.GetClient("target.example.com")

// Login — cookies are automatically saved
resp, _ := client.Post("https://target.example.com/login", ...)
sm.RecordRequest("target.example.com", resp)

// Later — same session, same cookies
resp, _ = client.Get("https://target.example.com/dashboard")
```

### Features
- **Per-domain cookie jars** — Each domain gets its own isolated cookie store
- **Disk persistence** — Sessions survive restarts (stored in `data/sessions/`)
- **Custom headers** — Set Authorization, API keys, or any header per domain
- **Request counting** — Track how many requests made to each domain
- **Thread-safe** — Concurrent access via `sync.RWMutex`

### Storage
Sessions are stored as JSON files in `data/sessions/<domain>.json`:
```json
{
  "domain": "target.example.com",
  "cookies": [
    {"name": "session", "value": "abc123", "domain": ".example.com"}
  ],
  "headers": {
    "Authorization": "Bearer token123"
  },
  "created_at": "2026-08-26T10:00:00Z",
  "last_used": "2026-08-26T10:05:00Z",
  "request_count": 15
}
```

## AutoThrottle (Adaptive Rate Limiting)

### Problem
Fixed-rate request patterns either waste time (too slow) or trigger WAF/IDS (too fast). Real targets have varying response times and rate limits.

### Solution
The `AutoThrottle` adaptively adjusts request rate per domain based on server behavior:

```go
throttle := httputil.NewAutoThrottle(100*time.Millisecond, 30*time.Second)

// Get a throttled client
client := sm.GetClientWithThrottle("target.example.com", throttle)

// Requests are automatically rate-limited
resp, err := client.Get("https://target.example.com/page")
```

### Behavior
| Server Response | Throttle Action |
|----------------|-----------------|
| 2xx Success | Speed up (reduce delay) |
| 429 Too Many Requests | Double delay, respect Retry-After |
| 503 Service Unavailable | Wait 10 seconds |
| Timeout | Exponential backoff |

### Configuration
```go
// Min delay: 100ms (fastest allowed)
// Max delay: 30s (slowest allowed)
throttle := httputil.NewAutoThrottle(100*time.Millisecond, 30*time.Second)
```

### How It Works
1. **Measure response time** for each request
2. **Calculate target delay** = 2x response time
3. **Gradually adjust** current delay toward target (30% weight)
4. **On block**: double delay (exponential backoff)
5. **On success**: move toward target (speed up)

## Response Cache

### Problem
Running the same scan twice wastes time and risks triggering rate limits.

### Solution
The `ResponseCache` stores tool responses on disk for retry:

```go
cache := httputil.NewResponseCache("data", 1*time.Hour)

// Check cache before executing
key := httputil.MakeKey("run_nmap", map[string]any{"target": "10.0.0.1", "mode": "quick"})
if cached, ok := cache.Get(key); ok {
    // Use cached response
    return cached
}

// Execute and cache
result := executeNmap(target)
cache.Set(key, result)
```

### Features
- **Deterministic keys** — Same tool + args = same key (sorted by key name)
- **Configurable TTL** — Default 1 hour, adjustable
- **Hit count tracking** — Know which caches are actually useful
- **Disk persistence** — Stored in `data/cache/`
- **Clear/Invalidate** — Manual cache invalidation available

## WAF Detection

### Problem
Many modern targets are behind WAFs (Cloudflare, Akamai, etc.) that block automated scanning. Without detection, tools silently fail.

### Solution
The `DetectWAF` function inspects HTTP responses for WAF signatures:

```go
resp, err := client.Get("https://target.example.com")
wafs := httputil.DetectWAF(resp)
if len(wafs) > 0 {
    fmt.Print(httputil.FormatWAFDetection(wafs))
}
```

### Detected WAFs

| WAF | Detection Signals | Bypass Hints |
|-----|-------------------|--------------|
| **Cloudflare** | `cf-ray`, `cf-cache-status`, `__cfduid` cookie | Use headless browser, try direct IP, check origin leaks |
| **Akamai** | `x-akamai-transformed`, `server: akamaighost` | Try IP-based access, Host header manipulation |
| **AWS WAF** | `x-amzn-waf`, `aws-waf-token` cookie | Rate limit, different User-Agent, check API endpoints |
| **ModSecurity** | `server: mod_security` | Encoding tricks, chunked transfer, HTTP parameter pollution |
| **Imperva** | `x-iinfo`, `incap_ses` cookie | Residential proxies, mobile User-Agent, API endpoints |
| **Sucuri** | `server: sucuri`, `x-sucuri-id` | Direct IP access, DNS origin lookup |

## Integration with Tool Wrappers

The session/throttle/cache system integrates with tool wrappers:

```go
// In a tool wrapper function:
func (r *ToolRegistry) httpTool(ctx context.Context, args map[string]any) string {
    target := args["target"].(string)
    domain := extractDomain(target)
    
    // Get persistent session with rate limiting
    client := r.sessions.GetClientWithThrottle(domain, r.throttle)
    
    // Add custom headers
    headers := r.sessions.GetHeaders(domain)
    for k, v := range headers {
        req.Header.Set(k, v)
    }
    
    // Execute request
    resp, err := client.Do(req)
    
    // Record for session tracking
    r.sessions.RecordRequest(domain, resp)
    
    // Check WAF
    if wafs := httputil.DetectWAF(resp); len(wafs) > 0 {
        return httputil.FormatWAFDetection(wafs) + "\n" + body
    }
    
    return body
}
```

## API Reference

### SessionManager
- `NewSessionManager(dataDir string) *SessionManager`
- `GetClient(domain string) *http.Client`
- `GetClientWithThrottle(domain string, throttle *AutoThrottle) *http.Client`
- `RecordRequest(domain string, resp *http.Response)`
- `SetHeader(domain, key, value string)`
- `GetHeaders(domain string) map[string]string`
- `GetSession(domain string) *Session`
- `ListDomains() []string`
- `ClearDomain(domain string)`

### AutoThrottle
- `NewAutoThrottle(minDelay, maxDelay time.Duration) *AutoThrottle`
- `Wait(domain string)`
- `RecordSuccess(domain string, responseTime time.Duration)`
- `RecordBlocked(domain string, retryAfter time.Duration)`
- `GetDelay(domain string) time.Duration`

### ResponseCache
- `NewResponseCache(dataDir string, maxAge time.Duration) *ResponseCache`
- `MakeKey(tool string, args map[string]any) string`
- `Get(key string) (string, bool)`
- `Set(key, response string)`
- `Invalidate(key string)`
- `Clear()`
- `Stats() (entries int, totalHits int)`

### WAF Detection
- `DetectWAF(resp *http.Response) []WAFSignature`
- `FormatWAFDetection(wafs []WAFSignature) string`
- `KnownWAFs` — List of 6 WAF signatures with bypass hints
