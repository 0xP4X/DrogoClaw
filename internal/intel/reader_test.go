package intel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchPageNative(t *testing.T) {
	page := `<!doctype html><html lang="en"><head>
<title>Test Page</title>
<meta property="og:site_name" content="Example Org">
<meta name="description" content="A sample description.">
<meta property="article:author" content="Jane">
</head><body>
<h1>Heading One</h1>
<p>First paragraph with <a href="/page2">a link</a> inside.</p>
<script>var x = 1;</script>
<ul><li>item one</li><li>item two</li></ul>
</body></html>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()

	res, err := FetchPageContext(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchPage error: %v", err)
	}
	if res.Title != "Test Page" {
		t.Errorf("Title = %q, want %q", res.Title, "Test Page")
	}
	if res.SiteName != "Example Org" {
		t.Errorf("SiteName = %q", res.SiteName)
	}
	if res.Excerpt != "A sample description." {
		t.Errorf("Excerpt = %q", res.Excerpt)
	}
	if res.Byline != "Jane" {
		t.Errorf("Byline = %q", res.Byline)
	}
	if res.Lang != "en" {
		t.Errorf("Lang = %q", res.Lang)
	}
	linked := false
	for _, l := range res.Links {
		if l == srv.URL+"/page2" {
			linked = true
		}
	}
	if !linked {
		t.Errorf("link not extracted: %v", res.Links)
	}
	if !strings.Contains(res.CleanText, "First paragraph") {
		t.Errorf("CleanText missing body: %q", res.CleanText)
	}
	if strings.Contains(res.CleanText, "var x") {
		t.Errorf("script content leaked: %q", res.CleanText)
	}
	if res.FetchedVia != "native" {
		t.Errorf("FetchedVia = %q, want native", res.FetchedVia)
	}
}

func TestActiveSearchBackendsOrdering(t *testing.T) {
	t.Setenv("EXA_API_KEY", "")

	if bs := activeSearchBackends(""); len(bs) != 1 || bs[0].name != "duckduckgo" {
		t.Fatalf("no-key chain = %v, want [duckduckgo]", bs)
	}
	if bs := activeSearchBackends("brave-token"); len(bs) != 2 || bs[0].name != "brave" || bs[1].name != "duckduckgo" {
		t.Fatalf("brave chain = %v", bs)
	}
	t.Setenv("EXA_API_KEY", "exa-token")
	if bs := activeSearchBackends("brave-token"); len(bs) != 3 || bs[0].name != "brave" || bs[1].name != "exa" || bs[2].name != "duckduckgo" {
		t.Fatalf("brave+exa chain = %v", bs)
	}
}
