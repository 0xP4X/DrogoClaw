package httputil

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestSessionManagerCookies(t *testing.T) {
	// Start a test server that sets cookies
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc123"})
		http.SetCookie(w, &http.Cookie{Name: "user", Value: "admin"})
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sm := NewSessionManager(t.TempDir())
	client := sm.GetClient("example.com")

	// Make a request to get cookies
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	// Record the response
	sm.RecordRequest("example.com", resp)

	// Verify session was saved
	sess := sm.GetSession("example.com")
	if sess == nil {
		t.Fatal("GetSession() returned nil")
	}
	if len(sess.Cookies) < 2 {
		t.Errorf("Expected 2 cookies, got %d", len(sess.Cookies))
	}
	if sess.RequestN != 1 {
		t.Errorf("RequestN = %d, want 1", sess.RequestN)
	}
}

func TestSessionManagerPersistence(t *testing.T) {
	dir := t.TempDir()

	// Create session
	sm1 := NewSessionManager(dir)
	sm1.SetHeader("example.com", "Authorization", "Bearer token123")

	// Create new manager with same dir — should load
	sm2 := NewSessionManager(dir)
	headers := sm2.GetHeaders("example.com")
	if headers["Authorization"] != "Bearer token123" {
		t.Errorf("Headers not persisted: got %v", headers)
	}
}

func TestSessionManagerListDomains(t *testing.T) {
	sm := NewSessionManager(t.TempDir())
	sm.SetHeader("a.com", "X-Test", "1")
	sm.SetHeader("b.com", "X-Test", "2")

	domains := sm.ListDomains()
	if len(domains) != 2 {
		t.Errorf("ListDomains() = %d domains, want 2", len(domains))
	}
}

func TestAutoThrottleSuccess(t *testing.T) {
	at := NewAutoThrottle(100*time.Millisecond, 30*time.Second)

	// Record successes — delay should decrease
	for i := 0; i < 10; i++ {
		at.RecordSuccess("example.com", 50*time.Millisecond)
	}

	delay := at.GetDelay("example.com")
	if delay > 500*time.Millisecond {
		t.Errorf("After 10 successes, delay = %v, expected < 500ms", delay)
	}
}

func TestAutoThrottleBlocked(t *testing.T) {
	at := NewAutoThrottle(100*time.Millisecond, 30*time.Second)

	// Record a block — delay should increase
	at.RecordBlocked("example.com", 5*time.Second)

	delay := at.GetDelay("example.com")
	if delay < 200*time.Millisecond {
		t.Errorf("After block, delay = %v, expected > 200ms", delay)
	}
}

func TestAutoThrottleMaxCap(t *testing.T) {
	at := NewAutoThrottle(100*time.Millisecond, 5*time.Second)

	// Record many blocks — delay should not exceed max
	for i := 0; i < 20; i++ {
		at.RecordBlocked("example.com", 0)
	}

	delay := at.GetDelay("example.com")
	if delay > 5*time.Second {
		t.Errorf("Delay %v exceeds max 5s", delay)
	}
}

func TestAutoThrottleConcurrent(t *testing.T) {
	at := NewAutoThrottle(10*time.Millisecond, 1*time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			at.Wait("example.com")
			at.RecordSuccess("example.com", 20*time.Millisecond)
		}()
	}
	wg.Wait()
}

func TestResponseCache(t *testing.T) {
	rc := NewResponseCache(t.TempDir(), 1*time.Hour)

	// Cache miss
	_, ok := rc.Get("test:key")
	if ok {
		t.Error("Expected cache miss")
	}

	// Set
	rc.Set("test:key", "cached response")

	// Cache hit
	val, ok := rc.Get("test:key")
	if !ok {
		t.Error("Expected cache hit")
	}
	if val != "cached response" {
		t.Errorf("Got %q, want %q", val, "cached response")
	}

	// Stats
	entries, hits := rc.Stats()
	if entries != 1 {
		t.Errorf("Entries = %d, want 1", entries)
	}
	if hits < 1 {
		t.Errorf("Hits = %d, want >= 1", hits)
	}
}

func TestResponseCacheExpiry(t *testing.T) {
	rc := NewResponseCache(t.TempDir(), 1*time.Millisecond) // 1ms TTL

	rc.Set("test:key", "value")

	// Wait for expiry
	time.Sleep(5 * time.Millisecond)

	_, ok := rc.Get("test:key")
	if ok {
		t.Error("Expected cache miss after expiry")
	}
}

func TestResponseCacheClear(t *testing.T) {
	rc := NewResponseCache(t.TempDir(), 1*time.Hour)
	rc.Set("key1", "val1")
	rc.Set("key2", "val2")

	rc.Clear()

	entries, _ := rc.Stats()
	if entries != 0 {
		t.Errorf("After Clear(), entries = %d, want 0", entries)
	}
}

func TestMakeKey(t *testing.T) {
	key1 := MakeKey("run_nmap", map[string]any{"target": "10.0.0.1", "mode": "quick"})
	key2 := MakeKey("run_nmap", map[string]any{"target": "10.0.0.1", "mode": "quick"})
	key3 := MakeKey("run_nmap", map[string]any{"target": "10.0.0.2", "mode": "quick"})

	if key1 != key2 {
		t.Errorf("Same args produced different keys: %s vs %s", key1, key2)
	}
	if key1 == key3 {
		t.Errorf("Different args produced same key: %s", key1)
	}
}

func TestDetectWAFCloudflare(t *testing.T) {
	resp := &http.Response{
		StatusCode: 403,
		Header:     http.Header{"Server": []string{"cloudflare"}, "Cf-Ray": []string{"abc123"}},
		Body:       http.NoBody,
	}
	wafs := DetectWAF(resp)
	if len(wafs) == 0 {
		t.Error("Expected Cloudflare detection")
	}
	if wafs[0].Name != "Cloudflare" {
		t.Errorf("Got %s, want Cloudflare", wafs[0].Name)
	}
}

func TestDetectWAFNone(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Server": []string{"nginx"}},
		Body:       http.NoBody,
	}
	wafs := DetectWAF(resp)
	if len(wafs) != 0 {
		t.Errorf("Expected no WAF detection, got %d", len(wafs))
	}
}

func TestFormatWAFDetection(t *testing.T) {
	wafs := []WAFSignature{{Name: "Cloudflare", Bypass: "Use headless browser"}}
	result := FormatWAFDetection(wafs)
	if result == "" {
		t.Error("Expected non-empty format")
	}
}

func TestFormatWAFDetectionEmpty(t *testing.T) {
	result := FormatWAFDetection(nil)
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}
