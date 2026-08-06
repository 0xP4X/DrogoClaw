package intel

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

const rssXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
<title>Test RSS</title>
<item><title>CVE-2024-1234 found in Widget</title><link>https://example.com/1</link>
<description>Vuln &lt;b&gt;found&lt;/b&gt; in widget</description>
<pubDate>Mon, 02 Jan 2024 15:04:05 -0700</pubDate></item>
</channel></rss>`

const atomXML = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
<title>Test Atom</title>
<entry><title>CVE-2023-9999 affects Foo</title><link href="https://example.com/2"/>
<summary>remote code execution</summary><updated>2023-05-01T00:00:00Z</updated></entry>
</feed>`

const kevJSON = `{"title":"KEV","vulnerabilities":[
{"cveID":"CVE-2021-44228","vendorProject":"Apache","product":"Log4j","vulnerabilityName":"Log4Shell","dateAdded":"2021-12-10","shortDescription":"JNDI RCE"},
{"cveID":"CVE-2011-2523","vendorProject":"vsftpd","product":"vsftpd","vulnerabilityName":"Backdoor","dateAdded":"2022-01-01","shortDescription":"Backdoor"}
]}`

func TestFeedParsingAndKEV(t *testing.T) {
	orig := httpClient
	defer func() { httpClient = orig }()
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body string
		switch {
		case strings.Contains(r.URL.String(), "atom"):
			body = atomXML
		case strings.Contains(r.URL.String(), "known_exploited"):
			body = kevJSON
		default:
			body = rssXML
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	rss, err := FetchFeed(context.Background(), "https://example.com/rss")
	if err != nil {
		t.Fatalf("RSS parse error: %v", err)
	}
	if len(rss.Items) != 1 || !strings.Contains(rss.Items[0].Title, "CVE-2024-1234") {
		t.Fatalf("RSS item wrong: %+v", rss.Items)
	}
	if !strings.Contains(rss.Items[0].Description, "found") {
		t.Errorf("RSS description not stripped: %q", rss.Items[0].Description)
	}

	atom, err := FetchFeed(context.Background(), "https://example.com/atom")
	if err != nil {
		t.Fatalf("Atom parse error: %v", err)
	}
	if len(atom.Items) != 1 || atom.Items[0].Link != "https://example.com/2" {
		t.Fatalf("Atom item wrong: %+v", atom.Items)
	}

	kev, err := FetchKEV(context.Background())
	if err != nil {
		t.Fatalf("KEV error: %v", err)
	}
	if len(kev) != 2 {
		t.Fatalf("KEV count = %d, want 2", len(kev))
	}
	if !kev[0].ExploitAvailable {
		t.Errorf("KEV entry should be flagged ExploitAvailable")
	}
	if kev[0].CVSSSeverity != "KEV" {
		t.Errorf("KEV severity = %q", kev[0].CVSSSeverity)
	}
}

func TestIngestCVEEntries(t *testing.T) {
	globalCVEDB = &CVEDatabase{}
	defer func() { globalCVEDB = nil }()

	added := IngestCVEEntries([]CVEEntry{
		{ID: "CVE-2024-1234", Description: "widget rce"},
		{ID: "CVE-2023-9999", Description: "foo rce", ExploitAvailable: true},
	})
	if added != 2 {
		t.Fatalf("initial ingest added %d, want 2", added)
	}

	// Re-ingest with a duplicate + an upgrade (exploit confirmed).
	added = IngestCVEEntries([]CVEEntry{
		{ID: "CVE-2024-1234", Description: "widget rce updated", ExploitAvailable: true},
		{ID: "CVE-2099-0001", Description: "future"},
	})
	if added != 1 {
		t.Fatalf("second ingest added %d, want 1", added)
	}

	// CVE-2024-1234 must now be flagged exploited (upgraded, not duplicated).
	var found *CVEEntry
	for i := range globalCVEDB.entries {
		if globalCVEDB.entries[i].ID == "CVE-2024-1234" {
			found = &globalCVEDB.entries[i]
		}
	}
	if found == nil {
		t.Fatal("CVE-2024-1234 missing after ingest")
	}
	if !found.ExploitAvailable {
		t.Errorf("CVE-2024-1234 should be upgraded to ExploitAvailable")
	}
	if len(globalCVEDB.entries) != 3 {
		t.Errorf("total entries = %d, want 3 (no duplicate)", len(globalCVEDB.entries))
	}
}

func TestExtractCVEs(t *testing.T) {
	got := extractCVEs("mentions CVE-2024-1234 and cve-2023-9999 and CVE-2024-1234 again")
	if len(got) != 2 {
		t.Fatalf("extractCVEs = %v, want 2 unique", got)
	}
}
