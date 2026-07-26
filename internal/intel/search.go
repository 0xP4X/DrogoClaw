package intel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 20 * time.Second}

// SearchResult represents a single web search result.
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// Search performs a web search using Brave Search API, falling back to DuckDuckGo HTML scraping.
func Search(query, apiKey string, numResults int) ([]SearchResult, error) {
	if apiKey != "" {
		return braveSearch(query, apiKey, numResults)
	}
	return duckduckgoSearch(query, numResults)
}

func braveSearch(query, apiKey string, numResults int) ([]SearchResult, error) {
	if numResults <= 0 {
		numResults = 5
	}
	endpoint := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query), numResults)

	req, err := http.NewRequest("GET", endpoint, nil)
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
		})
	}
	return out, nil
}

// duckduckgoSearch scrapes DuckDuckGo HTML as a no-API fallback.
func duckduckgoSearch(query string, numResults int) ([]SearchResult, error) {
	endpoint := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))
	req, err := http.NewRequest("GET", endpoint, nil)
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

	// Extract result links and snippets via regex (DDG HTML is stable enough)
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
		})
	}
	return results, nil
}

// FetchURL downloads a URL and returns its content as clean text (HTML stripped).
func FetchURL(rawURL string) (string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, rawURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024)) // 512KB limit
	if err != nil {
		return "", err
	}

	text := stripHTML(string(body))
	// Collapse whitespace
	wsRe := regexp.MustCompile(`\s{3,}`)
	text = wsRe.ReplaceAllString(text, "\n\n")
	if len(text) > 8000 {
		text = text[:8000] + "\n\n[Content truncated at 8000 chars]"
	}
	return strings.TrimSpace(text), nil
}

// stripHTML removes all HTML tags from a string.
func stripHTML(s string) string {
	tagRe := regexp.MustCompile(`<[^>]+>`)
	s = tagRe.ReplaceAllString(s, " ")
	// Decode common HTML entities
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return s
}
