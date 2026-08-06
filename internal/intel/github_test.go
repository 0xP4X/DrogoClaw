package intel

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGitHubDorkAuthenticated(t *testing.T) {
	orig := httpClient
	defer func() { httpClient = orig }()
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body string
		switch {
		case strings.Contains(r.URL.Path, "/search/code"):
			body = `{"items":[{"repository":{"full_name":"acme/secrets","html_url":"https://github.com/acme/secrets"},"path":".env","html_url":"https://github.com/acme/secrets/blob/main/.env"}]}`
		case strings.Contains(r.URL.Path, "/search/repositories"):
			body = `{"items":[{"full_name":"acme/api","html_url":"https://github.com/acme/api"}]}`
		default:
			body = `{"items":[]}`
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}

	res, err := GitHubDork("acme", "fake-token")
	if err != nil {
		t.Fatalf("GitHubDork error: %v", err)
	}
	if res.Source != "GitHubRecon" {
		t.Errorf("Source = %q, want GitHubRecon", res.Source)
	}
	if !res.Data["authenticated"].(bool) {
		t.Errorf("authenticated flag should be true")
	}
	if !strings.Contains(res.Summary, "acme/secrets") {
		t.Errorf("summary missing code hit: %q", res.Summary)
	}
	if !strings.Contains(res.Summary, "Related repositories") {
		t.Errorf("summary missing repo section: %q", res.Summary)
	}
}

func TestGitHubDorkNoTokenUsesPassive(t *testing.T) {
	// With no token the function must delegate to the passive path without
	// attempting the authenticated API. It returns either a passive result
	// (when the search engine is reachable) or a clean error — never a panic.
	res, err := GitHubDork("acme", "")
	if res != nil {
		if res.Source != "GitHubDorkPassive" {
			t.Errorf("no-token Source = %q, want GitHubDorkPassive", res.Source)
		}
	} else if err == nil {
		t.Fatalf("GitHubDork returned nil result and nil error")
	}
}
