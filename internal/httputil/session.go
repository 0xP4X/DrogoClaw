// Package httputil provides HTTP session management, adaptive rate limiting,
// and response caching for DrogonClaw's web-based tools.
//
// Inspired by Scrapling's session management, AutoThrottle, and pause/resume
// patterns. These utilities allow DrogonClaw to:
// - Maintain persistent cookie jars across tool calls (login once, enumerate all)
// - Adaptively rate-limit per domain to avoid triggering WAF/IDS
// - Cache responses for retry without re-hitting the target
package httputil

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ── Session Manager ──────────────────────────────────────────────────────────

// SessionManager maintains persistent HTTP sessions with cookie jars per domain.
// This allows DrogonClaw to maintain login state across multiple tool calls.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session // keyed by domain
	dataDir  string
}

// Session holds state for a single domain.
type Session struct {
	Domain    string            `json:"domain"`
	Cookies   []*http.Cookie    `json:"cookies,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	LastUsed  time.Time         `json:"last_used"`
	RequestN  int               `json:"request_count"`
}

// NewSessionManager creates a session manager that persists state to disk.
func NewSessionManager(dataDir string) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		dataDir:  filepath.Join(dataDir, "sessions"),
	}
	os.MkdirAll(sm.dataDir, 0700)
	sm.loadAll()
	return sm
}

// GetClient returns an http.Client with the persistent cookie jar for the domain.
func (sm *SessionManager) GetClient(domain string) *http.Client {
	jar, _ := cookiejar.New(nil)

	sm.mu.RLock()
	sess, ok := sm.sessions[domain]
	sm.mu.RUnlock()

	if ok {
		u := &url.URL{Scheme: "https", Host: domain}
		jar.SetCookies(u, sess.Cookies)
	}

	return &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		// Follow redirects but cap at 10
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// GetClientWithThrottle returns an http.Client with both persistent cookies
// and adaptive rate limiting for the domain.
func (sm *SessionManager) GetClientWithThrottle(domain string, throttle *AutoThrottle) *http.Client {
	client := sm.GetClient(domain)
	transport := client.Transport.(*http.Transport).Clone()
	client.Transport = &ThrottledTransport{
		Transport: transport,
		Throttle:  throttle,
		Domain:    domain,
	}
	return client
}

// RecordRequest updates session state after a request.
func (sm *SessionManager) RecordRequest(domain string, resp *http.Response) {
	if resp == nil {
		return
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sess, ok := sm.sessions[domain]
	if !ok {
		sess = &Session{
			Domain:    domain,
			Headers:   make(map[string]string),
			CreatedAt: time.Now(),
		}
		sm.sessions[domain] = sess
	}

	// Merge cookies from response
	if resp.Cookies() != nil {
		existing := make(map[string]*http.Cookie)
		for _, c := range sess.Cookies {
			existing[c.Name] = c
		}
		for _, c := range resp.Cookies() {
			existing[c.Name] = c
		}
		sess.Cookies = sess.Cookies[:0]
		for _, c := range existing {
			sess.Cookies = append(sess.Cookies, c)
		}
	}

	sess.LastUsed = time.Now()
	sess.RequestN++
	sm.save(domain, sess)
}

// SetHeader sets a custom header for all requests to a domain.
func (sm *SessionManager) SetHeader(domain, key, value string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sess, ok := sm.sessions[domain]
	if !ok {
		sess = &Session{
			Domain:    domain,
			Headers:   make(map[string]string),
			CreatedAt: time.Now(),
		}
		sm.sessions[domain] = sess
	}
	if sess.Headers == nil {
		sess.Headers = make(map[string]string)
	}
	sess.Headers[key] = value
	sm.save(domain, sess)
}

// GetHeaders returns custom headers for a domain.
func (sm *SessionManager) GetHeaders(domain string) map[string]string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sess, ok := sm.sessions[domain]; ok {
		return sess.Headers
	}
	return nil
}

// GetSession returns session info for a domain.
func (sm *SessionManager) GetSession(domain string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sess, ok := sm.sessions[domain]; ok {
		cp := *sess
		return &cp
	}
	return nil
}

// ListDomains returns all domains with active sessions.
func (sm *SessionManager) ListDomains() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	domains := make([]string, 0, len(sm.sessions))
	for d := range sm.sessions {
		domains = append(domains, d)
	}
	return domains
}

// ClearDomain removes the session for a specific domain.
func (sm *SessionManager) ClearDomain(domain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, domain)
	os.Remove(filepath.Join(sm.dataDir, domain+".json"))
}

func (sm *SessionManager) save(domain string, sess *Session) {
	b, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(filepath.Join(sm.dataDir, domain+".json"), b, 0600)
}

func (sm *SessionManager) loadAll() {
	entries, err := os.ReadDir(sm.dataDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(sm.dataDir, e.Name()))
		if err != nil {
			continue
		}
		var sess Session
		if json.Unmarshal(b, &sess) == nil && sess.Domain != "" {
			sm.sessions[sess.Domain] = &sess
		}
	}
}

// ── AutoThrottle (Adaptive Rate Limiter) ─────────────────────────────────────

// AutoThrottle adaptively limits request rate per domain based on response
// times and server signals (429, Retry-After). Inspired by Scrapling's
// AutoThrottle which "stops guessing delays" and lets the spider tune itself.
type AutoThrottle struct {
	mu       sync.RWMutex
	domains  map[string]*domainThrottle
	minDelay time.Duration
	maxDelay time.Duration
}

type domainThrottle struct {
	delay       time.Duration
	lastRequest time.Time
	blocked     bool
	blockedAt   time.Time
	retryAfter  time.Duration
	consecutive int // consecutive successful requests
}

// NewAutoThrottle creates a rate limiter with configurable bounds.
// minDelay is the floor (fastest allowed), maxDelay is the ceiling.
func NewAutoThrottle(minDelay, maxDelay time.Duration) *AutoThrottle {
	if minDelay == 0 {
		minDelay = 100 * time.Millisecond
	}
	if maxDelay == 0 {
		maxDelay = 30 * time.Second
	}
	return &AutoThrottle{
		domains:  make(map[string]*domainThrottle),
		minDelay: minDelay,
		maxDelay: maxDelay,
	}
}

// Wait blocks until it's safe to make a request to the domain.
func (at *AutoThrottle) Wait(domain string) {
	at.mu.Lock()
	dt, ok := at.domains[domain]
	if !ok {
		dt = &domainThrottle{delay: at.minDelay}
		at.domains[domain] = dt
	}
	at.mu.Unlock()

	// If we were blocked, wait for retry-after period
	if dt.blocked && !dt.blockedAt.IsZero() {
		elapsed := time.Since(dt.blockedAt)
		if elapsed < dt.retryAfter {
			time.Sleep(dt.retryAfter - elapsed)
		}
		at.mu.Lock()
		dt.blocked = false
		at.mu.Unlock()
	}

	// Wait for the adaptive delay since last request
	at.mu.RLock()
	delay := dt.delay
	lastReq := dt.lastRequest
	at.mu.RUnlock()

	if !lastReq.IsZero() {
		elapsed := time.Since(lastReq)
		if elapsed < delay {
			time.Sleep(delay - elapsed)
		}
	}

	at.mu.Lock()
	dt.lastRequest = time.Now()
	at.mu.Unlock()
}

// RecordSuccess records a successful response and speeds up.
func (at *AutoThrottle) RecordSuccess(domain string, responseTime time.Duration) {
	at.mu.Lock()
	defer at.mu.Unlock()

	dt, ok := at.domains[domain]
	if !ok {
		dt = &domainThrottle{delay: at.minDelay}
		at.domains[domain] = dt
	}

	dt.consecutive++
	// Speed up: reduce delay based on response time
	// Target: 2x the response time as the delay
	target := responseTime * 2
	if target < at.minDelay {
		target = at.minDelay
	}
	// Gradually move toward target
	dt.delay = time.Duration(float64(dt.delay)*0.7 + float64(target)*0.3)
	if dt.delay < at.minDelay {
		dt.delay = at.minDelay
	}
}

// RecordBlocked records that the server pushed back (429, timeout, etc).
func (at *AutoThrottle) RecordBlocked(domain string, retryAfter time.Duration) {
	at.mu.Lock()
	defer at.mu.Unlock()

	dt, ok := at.domains[domain]
	if !ok {
		dt = &domainThrottle{delay: at.minDelay}
		at.domains[domain] = dt
	}

	// Double the delay (exponential backoff)
	dt.delay = dt.delay * 2
	if dt.delay > at.maxDelay {
		dt.delay = at.maxDelay
	}

	dt.blocked = true
	dt.blockedAt = time.Now()
	dt.consecutive = 0

	if retryAfter > 0 {
		dt.retryAfter = retryAfter
	} else {
		// Default: wait 2x current delay
		dt.delay = dt.delay * 2
		if dt.delay > at.maxDelay {
			dt.delay = at.maxDelay
		}
		dt.retryAfter = dt.delay
	}
}

// GetDelay returns the current delay for a domain.
func (at *AutoThrottle) GetDelay(domain string) time.Duration {
	at.mu.RLock()
	defer at.mu.RUnlock()
	if dt, ok := at.domains[domain]; ok {
		return dt.delay
	}
	return at.minDelay
}

// ThrottledTransport wraps http.Transport with adaptive rate limiting.
type ThrottledTransport struct {
	Transport http.RoundTripper
	Throttle  *AutoThrottle
	Domain    string
}

func (t *ThrottledTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.Throttle.Wait(t.Domain)
	start := time.Now()

	resp, err := t.Transport.RoundTrip(req)
	elapsed := time.Since(start)

	if err != nil {
		t.Throttle.RecordBlocked(t.Domain, 0)
		return resp, err
	}

	switch resp.StatusCode {
	case 429:
		// Parse Retry-After header
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		t.Throttle.RecordBlocked(t.Domain, retryAfter)
	case 503:
		t.Throttle.RecordBlocked(t.Domain, 10*time.Second)
	default:
		if resp.StatusCode < 400 {
			t.Throttle.RecordSuccess(t.Domain, elapsed)
		}
	}

	return resp, err
}

func parseRetryAfter(s string) time.Duration {
	if s == "" {
		return 0
	}
	// Try parsing as seconds
	if n, err := fmt.Sscanf(s, "%d"); err == nil && n > 0 {
		var secs int
		fmt.Sscanf(s, "%d", &secs)
		return time.Duration(secs) * time.Second
	}
	// Try parsing as HTTP date
	if t, err := time.Parse(time.RFC1123, s); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

// ── Response Cache ───────────────────────────────────────────────────────────

// ResponseCache caches tool responses to disk for retry without re-hitting
// the target. Inspired by Scrapling's development mode which "caches responses
// to disk on the first run and replays them on subsequent runs."
type ResponseCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	dataDir string
	maxAge  time.Duration
}

type CacheEntry struct {
	Key       string    `json:"key"`
	Response  string    `json:"response"`
	Timestamp time.Time `json:"timestamp"`
	HitCount  int       `json:"hit_count"`
}

// NewResponseCache creates a cache with configurable TTL.
func NewResponseCache(dataDir string, maxAge time.Duration) *ResponseCache {
	if maxAge == 0 {
		maxAge = 1 * time.Hour
	}
	rc := &ResponseCache{
		entries: make(map[string]*CacheEntry),
		dataDir: filepath.Join(dataDir, "cache"),
		maxAge:  maxAge,
	}
	os.MkdirAll(rc.dataDir, 0700)
	rc.loadAll()
	return rc
}

// MakeKey creates a cache key from tool name and arguments.
func MakeKey(tool string, args map[string]any) string {
	// Sort keys for deterministic ordering
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	parts = append(parts, tool)
	for _, k := range keys {
		v := args[k]
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, ":")
}

// Get returns a cached response if available and not expired.
func (rc *ResponseCache) Get(key string) (string, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, ok := rc.entries[key]
	if !ok {
		return "", false
	}
	if time.Since(entry.Timestamp) > rc.maxAge {
		return "", false
	}
	entry.HitCount++
	return entry.Response, true
}

// Set stores a response in the cache.
func (rc *ResponseCache) Set(key, response string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.entries[key] = &CacheEntry{
		Key:       key,
		Response:  response,
		Timestamp: time.Now(),
	}
	rc.save(key, rc.entries[key])
}

// Invalidate removes a specific cache entry.
func (rc *ResponseCache) Invalidate(key string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	delete(rc.entries, key)
	os.Remove(filepath.Join(rc.dataDir, key+".json"))
}

// Clear removes all cached entries.
func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.entries = make(map[string]*CacheEntry)
	os.RemoveAll(rc.dataDir)
	os.MkdirAll(rc.dataDir, 0700)
}

// Stats returns cache statistics.
func (rc *ResponseCache) Stats() (entries int, totalHits int) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	for _, e := range rc.entries {
		entries++
		totalHits += e.HitCount
	}
	return
}

func (rc *ResponseCache) save(key string, entry *CacheEntry) {
	b, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return
	}
	// Sanitize key for filesystem
	safeKey := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, key)
	os.WriteFile(filepath.Join(rc.dataDir, safeKey+".json"), b, 0600)
}

func (rc *ResponseCache) loadAll() {
	entries, err := os.ReadDir(rc.dataDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(rc.dataDir, e.Name()))
		if err != nil {
			continue
		}
		var entry CacheEntry
		if json.Unmarshal(b, &entry) == nil && entry.Key != "" {
			rc.entries[entry.Key] = &entry
		}
	}
}

// ── Anti-Bot / WAF Detection ─────────────────────────────────────────────────

// WAFSignature represents a known WAF/CDN fingerprint.
type WAFSignature struct {
	Name    string
	Headers []string            // Response headers that indicate this WAF
	Body    []string            // Body patterns
	Cookies []string            // Set-Cookie patterns
	Status  []int               // Unusual status codes
	Bypass  string              // Hint for bypassing
}

// KnownWAFs lists common WAF signatures and bypass hints.
var KnownWAFs = []WAFSignature{
	{
		Name:    "Cloudflare",
		Headers: []string{"cf-ray", "cf-cache-status", "server: cloudflare"},
		Cookies: []string{"__cfduid", "cf_clearance"},
		Status:  []int{403, 503},
		Bypass:  "Use headless browser with Cloudflare bypass, or try alternative subdomains (direct IP, staging, dev). Check for origin IP leaks via DNS history.",
	},
	{
		Name:    "Akamai",
		Headers: []string{"x-akamai-transformed", "server: akamaighost"},
		Cookies: []string{"akamai_cookies"},
		Status:  []int{403},
		Bypass:  "Try IP-based access via Host header manipulation. Check for staging/debug endpoints. Use HTTP/1.1 instead of HTTP/2.",
	},
	{
		Name:    "AWS WAF",
		Headers: []string{"x-amzn-waf", "x-amzn-requestid"},
		Cookies: []string{"aws-waf-token"},
		Status:  []int{403},
		Bypass:  "Rate limit aggressively. Try different User-Agent strings. Check for API endpoints that bypass WAF.",
	},
	{
		Name:    "ModSecurity",
		Headers: []string{"server: mod_security", "server: nobytes"},
		Body:    []string{"mod_security", "ModSecurity"},
		Status:  []int{403},
		Bypass:  "Try encoding tricks (URL encode, double encode, Unicode). Use chunked transfer encoding. Try HTTP parameter pollution.",
	},
	{
		Name:    "Imperva/Incapsula",
		Headers: []string{"x-iinfo", "x-cdn: incapsula"},
		Cookies: []string{"incap_ses", "visid_incap"},
		Status:  []int{403},
		Bypass:  "Use residential proxy rotation. Try mobile User-Agent strings. Check for API endpoints not protected by WAF.",
	},
	{
		Name:    "Sucuri",
		Headers: []string{"server: sucuri", "x-sucuri-id"},
		Status:  []int{403},
		Bypass:  "Try direct IP access. Check for origin server IP in DNS records. Try different HTTP methods.",
	},
}

// DetectWAF inspects an HTTP response and returns any detected WAF signatures.
func DetectWAF(resp *http.Response) []WAFSignature {
	if resp == nil {
		return nil
	}

	var detected []WAFSignature
	body := ""
	if resp.Body != nil {
		// Read limited body for detection
		limitedBody, err := io.ReadAll(io.LimitReader(resp.Body, 10240))
		if err == nil {
			body = string(limitedBody)
		}
		// Reconstruct the body for further use
		resp.Body = io.NopCloser(strings.NewReader(body))
	}

	for _, waf := range KnownWAFs {
		matched := false

		// Check headers
		for _, h := range waf.Headers {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				if strings.EqualFold(strings.TrimSpace(parts[0]), "server") {
					if strings.EqualFold(resp.Header.Get("Server"), strings.TrimSpace(parts[1])) {
						matched = true
					}
				} else if strings.EqualFold(resp.Header.Get(parts[0]), parts[1]) {
					matched = true
				}
			} else {
				if resp.Header.Get(h) != "" {
					matched = true
				}
			}
		}

		// Check body patterns
		for _, pattern := range waf.Body {
			if strings.Contains(body, pattern) {
				matched = true
			}
		}

		// Check cookies
		for _, cookiePattern := range waf.Cookies {
			for _, c := range resp.Cookies() {
				if strings.Contains(c.Name, cookiePattern) {
					matched = true
				}
			}
		}

		// Check status
		for _, code := range waf.Status {
			if resp.StatusCode == code {
				matched = true
			}
		}

		if matched {
			detected = append(detected, waf)
		}
	}

	return detected
}

// FormatWAFDetection formats detected WAFs into a human-readable string
// with bypass hints for the LLM.
func FormatWAFDetection(wafs []WAFSignature) string {
	if len(wafs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("[WAF DETECTED] ")
	names := make([]string, len(wafs))
	for i, w := range wafs {
		names[i] = w.Name
	}
	sb.WriteString(strings.Join(names, ", "))
	sb.WriteString("\n")
	for _, w := range wafs {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", w.Name, w.Bypass))
	}
	return sb.String()
}
