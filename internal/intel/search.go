package intel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var httpClient = &http.Client{Timeout: 20 * time.Second}

// SearchResult represents a single web search result. Source records which
// backend produced it so the orchestrator can attribute findings (evidence-led,
// like the rest of the intel package).
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
	Source  string
}

// searchBackend is one strategy for answering a query. Backends are tried in
// the order returned by activeSearchBackends; the first to return results wins
// and failures fall through to the next.
type searchBackend struct {
	name     string
	needsKey bool
	fn       func(ctx context.Context, query string, n int, key string) ([]SearchResult, error)
}

// activeSearchBackends builds the ordered fallback chain based on which API
// keys are actually present. Brave and Exa require keys; DuckDuckGo HTML is the
// always-available no-key fallback. This mirrors Agent Reach's "preferred +
// fallback" routing but is a real probe chain, not a static if/else.
func activeSearchBackends(apiKey string) []searchBackend {
	backends := []searchBackend{}
	if apiKey != "" {
		backends = append(backends, searchBackend{"brave", true, braveSearchCtx})
	}
	if exaKey := os.Getenv("EXA_API_KEY"); exaKey != "" {
		backends = append(backends, searchBackend{"exa", true, exaSearchCtx})
	}
	backends = append(backends, searchBackend{"duckduckgo", false, ddgSearchCtx})
	return backends
}

// Search performs a web search using the ordered backend chain (Brave → Exa →
// DuckDuckGo). It preserves the original signature used by GitHub dorking and
// email harvesting.
func Search(query, apiKey string, numResults int) ([]SearchResult, error) {
	return SearchContext(context.Background(), query, apiKey, numResults)
}

// SearchContext is Search with an explicit context.
func SearchContext(ctx context.Context, query, apiKey string, numResults int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if numResults <= 0 {
		numResults = 5
	}

	var lastErr error
	for _, b := range activeSearchBackends(apiKey) {
		key := apiKey
		if b.name == "exa" {
			key = os.Getenv("EXA_API_KEY")
		}
		results, berr := b.fn(ctx, query, numResults, key)
		if berr != nil {
			recordSearchBackend(b.name, "off", berr.Error())
			lastErr = berr
			continue
		}
		if len(results) == 0 {
			recordSearchBackend(b.name, "warn", "no results")
			lastErr = fmt.Errorf("%s returned no results", b.name)
			continue
		}
		for i := range results {
			if results[i].Source == "" {
				results[i].Source = b.name
			}
		}
		recordSearchBackend(b.name, "ok", "available")
		return results, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("all search backends returned no results")
	}
	return nil, fmt.Errorf("search failed for %q: %w", query, lastErr)
}

// ── Brave ─────────────────────────────────────────────────────────────────────

func braveSearchCtx(ctx context.Context, query string, numResults int, apiKey string) ([]SearchResult, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("brave requires an API key")
	}
	if numResults <= 0 {
		numResults = 5
	}
	endpoint := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query), numResults)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("brave search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("brave search error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	var out []SearchResult
	for _, r := range result.Web.Results {
		out = append(out, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Description,
			Source:  "brave",
		})
	}
	return out, nil
}

// ── Exa (semantic search) ────────────────────────────────────────────────────

// exaSearchCtx queries Exa's REST API for neural/semantic search results. It
// requires EXA_API_KEY and returns result text as the snippet, which is far more
// useful for the orchestrator than a keyword snippet.
func exaSearchCtx(ctx context.Context, query string, numResults int, apiKey string) ([]SearchResult, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("exa requires EXA_API_KEY")
	}
	if numResults <= 0 {
		numResults = 5
	}
	payload, err := json.Marshal(map[string]interface{}{
		"query":      query,
		"numResults": numResults,
		"contents":   map[string]interface{}{"text": true, "highlights": true},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.exa.ai/search", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exa request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("exa error %d: %s", resp.StatusCode, string(body))
	}

	var out struct {
		Results []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
			Text  string `json:"text"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("exa decode error: %w", err)
	}

	results := make([]SearchResult, 0, len(out.Results))
	for _, r := range out.Results {
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: truncate(r.Text, 400),
			Source:  "exa",
		})
	}
	return results, nil
}

// ── DuckDuckGo (no-key fallback) ─────────────────────────────────────────────

func ddgSearchCtx(ctx context.Context, query string, numResults int, _ string) ([]SearchResult, error) {
	if numResults <= 0 {
		numResults = 5
	}
	endpoint := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("duckduckgo request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	titleRe := regexp.MustCompile(`class="result__a"[^>]*href="([^"]+)"[^>]*>([^<]+)<`)
	snippetRe := regexp.MustCompile(`class="result__snippet">([^<]+)<`)

	titleMatches := titleRe.FindAllStringSubmatch(string(body), numResults)
	snippetMatches := snippetRe.FindAllStringSubmatch(string(body), numResults)

	var results []SearchResult
	for i, m := range titleMatches {
		if i >= numResults {
			break
		}
		snippet := ""
		if i < len(snippetMatches) {
			snippet = snippetMatches[i][1]
		}
		results = append(results, SearchResult{
			Title:   strings.TrimSpace(m[2]),
			URL:     m[1],
			Snippet: strings.TrimSpace(snippet),
			Source:  "duckduckgo",
		})
	}
	return results, nil
}

// ── Health reporting (mirrors reader.go's doctor concept) ────────────────────

var (
	searchBackendMu     sync.Mutex
	searchBackendStatus = map[string]string{}
	searchBackendDetail = map[string]string{}
)

func recordSearchBackend(name, status, detail string) {
	searchBackendMu.Lock()
	searchBackendStatus[name] = status
	searchBackendDetail[name] = detail
	searchBackendMu.Unlock()
}

// SearchBackendHealth returns the most recent probe status for each search
// backend, suitable for surfacing in /health.
func SearchBackendHealth() map[string]string {
	searchBackendMu.Lock()
	defer searchBackendMu.Unlock()
	out := make(map[string]string, len(searchBackendStatus))
	for k, v := range searchBackendStatus {
		out[k] = v
	}
	return out
}

// stripHTML removes all HTML tags from a string. It is the final fallback used
// by reader.go's readStrip backend.
func stripHTML(s string) string {
	tagRe := regexp.MustCompile(`<[^>]+>`)
	s = tagRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return s
}
