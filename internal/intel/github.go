package intel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// GitHubSearchItem is a single hit from the GitHub Search API.
type GitHubSearchItem struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	FullName  string `json:"full_name"`
	HTMLURL   string `json:"html_url"`
	Repository struct {
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
	Score float64 `json:"score"`
}

// GitHubDork hunts GitHub for leaked credentials, API keys, .env files,
// connection strings, and private keys related to a target org or domain.
//
// When a GITHUB_TOKEN is supplied it performs authenticated Search API queries
// (code + repository search) — the path Agent Reach only reaches via its `gh`
// CLI channel — and falls back to passive dorking if the API yields nothing or
// errors. With no token it degrades to the existing passive dork.
func GitHubDork(target, token string) (*OSINTResult, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("target is required")
	}
	if token == "" {
		return githubDorkPassive(target)
	}

	res, err := githubReconAuthenticated(target, token)
	if err != nil {
		// API failure: keep the operator moving with the passive path.
		recordGitHub("code-search", "off", err.Error())
		return githubDorkPassive(target)
	}
	// Enrich authenticated results with the passive view when code search is thin.
	if hits, ok := res.Data["code_hits"].([]map[string]interface{}); !ok || len(hits) == 0 {
		if passive, perr := githubDorkPassive(target); perr == nil && passive != nil {
			res.Summary += "\n\n" + passive.Summary
		}
	}
	return res, nil
}

// githubReconAuthenticated runs the GitHub Search API across the standard
// secret-leak dork categories plus a repository search for org/related repos.
func githubReconAuthenticated(target, token string) (*OSINTResult, error) {
	ctx := context.Background()

	r := NewReport(fmt.Sprintf("GitHub Authenticated Recon: %s", target))

	codeQueries := []struct {
		label string
		query string
	}{
		{"API Keys / Secrets", fmt.Sprintf("%s (password OR secret OR api_key OR token) in:file", target)},
		{"Config Files", fmt.Sprintf("%s filename:.env OR filename:config.yml OR filename:secrets.json", target)},
		{"Connection Strings", fmt.Sprintf("%s (connectionstring OR database_url OR DB_PASSWORD) in:file", target)},
		{"AWS Credentials", fmt.Sprintf("%s (AKIA OR aws_access_key_id) in:file", target)},
		{"Private Keys", fmt.Sprintf("%s (\"BEGIN RSA PRIVATE KEY\" OR \"BEGIN OPENSSH PRIVATE KEY\") in:file", target)},
	}

	var allHits []map[string]interface{}
	total := 0
	codeSearchesOK := 0
	for _, d := range codeQueries {
		items, err := githubCodeSearch(ctx, token, d.query)
		if err != nil {
			recordGitHub("code-search", "warn", err.Error())
			r.Section("Secret-leak code search").
				Line(fmt.Sprintf("- ⚠️ %s: code search failed: %s", d.label, err))
			continue
		}
		codeSearchesOK++
		if len(items) == 0 {
			continue
		}
		r.Section(fmt.Sprintf("%s — %d hits", d.label, len(items)))
		for _, it := range items {
			r.Bullet(fmt.Sprintf("%s — %s", mdCode(it.Repository.FullName+"/"+it.Path), it.HTMLURL))
			allHits = append(allHits, map[string]interface{}{
				"label": d.label,
				"repo":  it.Repository.FullName,
				"path":  it.Path,
				"url":   it.HTMLURL,
			})
		}
		total += len(items)
	}

	repos, err := githubRepoSearch(ctx, token, target)
	if err != nil {
		recordGitHub("repo-search", "warn", err.Error())
	} else if len(repos) > 0 {
		r.Section(fmt.Sprintf("Related repositories — %d", len(repos)))
		for _, rp := range repos {
			r.Bullet(fmt.Sprintf("%s — %s", mdCode(rp.FullName), rp.HTMLURL))
		}
	}

	if codeSearchesOK > 0 {
		recordGitHub("code-search", "ok", "available")
	} else {
		recordGitHub("code-search", "warn", "all code searches failed")
	}

	if total == 0 && len(repos) == 0 {
		r.Note("No authenticated code/repo hits. Broaden the query or try passive dorking.")
	} else {
		r.Section("Summary")
		r.KV("Code hits", fmt.Sprintf("%d", total))
		r.KV("Repositories", fmt.Sprintf("%d", len(repos)))
		r.KV("Source", "GitHub Search API (authenticated)")
	}

	return &OSINTResult{
		Source:  "GitHubRecon",
		Target:  target,
		Data:    map[string]interface{}{"code_hits": allHits, "target": target, "authenticated": true},
		Summary: r.String(),
	}, nil
}

// githubCodeSearch runs an authenticated GitHub code search.
func githubCodeSearch(ctx context.Context, token, query string) ([]GitHubSearchItem, error) {
	endpoint := "https://api.github.com/search/code?q=" + url.QueryEscape(query) + "&per_page=20"
	return githubSearch(ctx, token, endpoint)
}

// githubRepoSearch finds repositories related to the target (org repos, forks).
func githubRepoSearch(ctx context.Context, token, query string) ([]GitHubSearchItem, error) {
	endpoint := "https://api.github.com/search/repositories?q=" + url.QueryEscape(query) + "&per_page=10"
	return githubSearch(ctx, token, endpoint)
}

// githubSearch performs a GET against the GitHub Search API with a bearer token.
func githubSearch(ctx context.Context, token, endpoint string) ([]GitHubSearchItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "DrogonClaw/2.0 (security-assessment-tool)")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("github: unauthorized (check GITHUB_TOKEN)")
	case http.StatusForbidden:
		return nil, fmt.Errorf("github: forbidden / rate-limited (HTTP 403)")
	case http.StatusOK:
		// fall through
	default:
		return nil, fmt.Errorf("github error %d: %s", resp.StatusCode, string(body))
	}

	var out struct {
		Items []GitHubSearchItem `json:"items"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("github decode error: %w", err)
	}
	return out.Items, nil
}

// ── Health reporting (mirrors reader.go / search.go / github.go doctor) ───────

var (
	githubMu     sync.Mutex
	githubStatus = map[string]string{}
	githubDetail = map[string]string{}
)

func recordGitHub(name, status, detail string) {
	githubMu.Lock()
	githubStatus[name] = status
	githubDetail[name] = detail
	githubMu.Unlock()
}

// GitHubBackendHealth returns the most recent probe status for the GitHub recon
// backend, suitable for surfacing in /health.
func GitHubBackendHealth() map[string]string {
	githubMu.Lock()
	defer githubMu.Unlock()
	out := make(map[string]string, len(githubStatus))
	for k, v := range githubStatus {
		out[k] = v
	}
	return out
}
