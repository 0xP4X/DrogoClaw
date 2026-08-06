package intel

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// FeedItem is a single entry from an RSS/Atom/advisory feed.
type FeedItem struct {
	Title       string
	Link        string
	Description string
	Published   time.Time
	Source      string
	Categories  []string
}

// Feed is a parsed syndication feed.
type Feed struct {
	Title  string
	Source string
	Items  []FeedItem
}

// AdvisoryFeed describes a known security-advisory source to poll.
type AdvisoryFeed struct {
	Name string
	URL  string
	Kind string // "rss" | "atom" | "kev-json"
}

// DefaultAdvisoryFeeds are public, authorized-security sources. CISA KEV is the
// highest-value: it lists vulnerabilities exploited in the wild, which the
// orchestrator should prioritise over the 120-day NVD cache alone.
var DefaultAdvisoryFeeds = []AdvisoryFeed{
	{"CISA KEV", "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json", "kev-json"},
	{"CISA Advisories", "https://www.cisa.gov/cybersecurity-advisories/all.xml", "rss"},
	{"The Hacker News", "https://feeds.feedburner.com/TheHackersNews", "rss"},
	{"Dark Reading", "https://www.darkreading.com/rss.xml", "rss"},
	{"Krebs on Security", "https://krebsonsecurity.com/feed/", "rss"},
}

var cveIDRe = regexp.MustCompile(`CVE-\d{4}-\d{4,7}`)

// ── Generic feed reader (Agent Reach's RSS channel, native Go) ────────────────

// FetchFeed fetches and parses any RSS 2.0 or Atom feed by URL. It is the native
// equivalent of Agent Reach's RSS channel but returns typed, source-attributed
// items instead of raw text.
func FetchFeed(ctx context.Context, rawURL string) (*Feed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DrogonClaw/2.0 (security-assessment-tool)")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, */*")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("feed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed error %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, err
	}
	return parseFeedBytes(rawURL, body)
}

func parseFeedBytes(source string, body []byte) (*Feed, error) {
	var rss rssDoc
	if err := xml.Unmarshal(body, &rss); err == nil && len(rss.Channel.Items) > 0 {
		return rssToFeed(source, rss), nil
	}
	var atom atomDoc
	if err := xml.Unmarshal(body, &atom); err == nil && len(atom.Entries) > 0 {
		return atomToFeed(source, atom), nil
	}
	return nil, fmt.Errorf("unrecognized feed format for %s", source)
}

type rssDoc struct {
	Channel struct {
		Title string     `xml:"title"`
		Items []rssItem  `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	PubDate     string   `xml:"pubDate"`
	Categories  []string `xml:"category"`
}

type atomDoc struct {
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string      `xml:"title"`
	Link    atomLink    `xml:"link"`
	Summary string      `xml:"summary"`
	Updated string      `xml:"updated"`
	Categories []atomCategory `xml:"category"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
}

type atomCategory struct {
	Term string `xml:"term,attr"`
}

func rssToFeed(source string, d rssDoc) *Feed {
	f := &Feed{Title: d.Channel.Title, Source: source}
	for _, it := range d.Channel.Items {
		f.Items = append(f.Items, FeedItem{
			Title:       strings.TrimSpace(it.Title),
			Link:        it.Link,
			Description: stripHTML(it.Description),
			Published:   parseFeedDate(it.PubDate),
			Source:      source,
			Categories:  it.Categories,
		})
	}
	return f
}

func atomToFeed(source string, d atomDoc) *Feed {
	f := &Feed{Title: strings.TrimSpace(d.Title), Source: source}
	for _, it := range d.Entries {
		cats := make([]string, 0, len(it.Categories))
		for _, c := range it.Categories {
			if c.Term != "" {
				cats = append(cats, c.Term)
			}
		}
		f.Items = append(f.Items, FeedItem{
			Title:       strings.TrimSpace(it.Title),
			Link:        it.Link.Href,
			Description: stripHTML(it.Summary),
			Published:   parseFeedDate(it.Updated),
			Source:      source,
			Categories:  cats,
		})
	}
	return f
}

func parseFeedDate(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, time.RFC1123Z, time.RFC1123, "2006-01-02T15:04:05Z07:00", "Mon, 2 Jan 2006 15:04:05 -0700"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ── CISA KEV (known-exploited vulnerabilities) ───────────────────────────────

type kevDoc struct {
	Title          string `json:"title"`
	Vulnerabilities []struct {
		CVEID                     string `json:"cveID"`
		VendorProject             string `json:"vendorProject"`
		Product                   string `json:"product"`
		VulnerabilityName         string `json:"vulnerabilityName"`
		DateAdded                 string `json:"dateAdded"`
		ShortDescription          string `json:"shortDescription"`
		KnownRansomwareCampaignUse string `json:"knownRansomwareCampaignUse"`
	} `json:"vulnerabilities"`
}

// FetchKEV downloads the CISA KEV catalog and returns it as CVE entries flagged
// as exploited in the wild. This is the single highest-signal source for
// prioritising an engagement.
func FetchKEV(ctx context.Context) ([]CVEEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DrogonClaw/2.0 (security-assessment-tool)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("KEV request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KEV error %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024))
	if err != nil {
		return nil, err
	}

	var doc kevDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("KEV decode error: %w", err)
	}

	entries := make([]CVEEntry, 0, len(doc.Vulnerabilities))
	for _, v := range doc.Vulnerabilities {
		if v.CVEID == "" {
			continue
		}
		desc := v.ShortDescription
		if desc == "" {
			desc = v.VulnerabilityName
		}
		if v.Product != "" {
			desc = fmt.Sprintf("[%s/%s] %s", v.VendorProject, v.Product, desc)
		}
		entries = append(entries, CVEEntry{
			ID:              v.CVEID,
			Description:     desc,
			CVSSSeverity:    "KEV",
			Published:       v.DateAdded,
			ExploitAvailable: true,
		})
	}
	return entries, nil
}

// ── Ingestion into the CVE intelligence layer ────────────────────────────────

// IngestCVEEntries merges entries into the shared global CVE database,
// de-duplicating by ID and upgrading ExploitAvailable when a source confirms
// in-the-wild exploitation. Returns the number of newly added CVEs.
func IngestCVEEntries(entries []CVEEntry) int {
	db := globalCVEDB
	if db == nil {
		db = &CVEDatabase{}
		globalCVEDB = db
	}
	byID := make(map[string]int, len(db.entries))
	for i, e := range db.entries {
		byID[e.ID] = i
	}
	added := 0
	for _, e := range entries {
		if e.ID == "" {
			continue
		}
		if idx, ok := byID[e.ID]; ok {
			if e.ExploitAvailable {
				db.entries[idx].ExploitAvailable = true
			}
			continue
		}
		byID[e.ID] = len(db.entries)
		db.entries = append(db.entries, e)
		added++
	}
	return added
}

// FeedIngestReport summarises a RefreshAdvisoryFeeds run.
type FeedIngestReport struct {
	Checked int
	OK      int
	NewCVEs int
	PerFeed []FeedStatus
}

// FeedStatus is the per-feed outcome of an ingest run.
type FeedStatus struct {
	Name   string
	Status string // ok | warn | off
	Detail string
	Items  int
	CVEs   int
}

// RefreshAdvisoryFeeds polls DefaultAdvisoryFeeds, extracts CVE references, and
// ingests them into the global CVE database so LookupCVE can surface freshly
// published / exploited vulnerabilities. It is safe to call repeatedly.
func RefreshAdvisoryFeeds(ctx context.Context) (*FeedIngestReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	report := &FeedIngestReport{}

	// Ensure the NVD cache is loaded first so freshly ingested KEV/feed CVEs
	// merge onto the 120-day NVD cache rather than replacing it.
	if globalCVEDB == nil {
		if _, err := LoadCVEDatabase(); err != nil {
			recordFeed("cve-db", "warn", "NVD cache unavailable: "+err.Error())
		}
	}

	// KEV first — highest signal.
	kev, err := FetchKEV(ctx)
	if err != nil {
		recordFeed("CISA KEV", "off", err.Error())
		report.PerFeed = append(report.PerFeed, FeedStatus{Name: "CISA KEV", Status: "off", Detail: err.Error()})
	} else {
		added := IngestCVEEntries(kev)
		recordFeed("CISA KEV", "ok", fmt.Sprintf("%d CVEs", len(kev)))
		report.PerFeed = append(report.PerFeed, FeedStatus{Name: "CISA KEV", Status: "ok", Detail: "kev catalog", Items: len(kev), CVEs: added})
		report.NewCVEs += added
	}
	report.Checked++
	report.OK++

	// Remaining RSS/Atom advisory feeds.
	for _, src := range DefaultAdvisoryFeeds {
		if src.Kind == "kev-json" {
			continue // handled above
		}
		report.Checked++
		feed, ferr := FetchFeed(ctx, src.URL)
		if ferr != nil {
			recordFeed(src.Name, "off", ferr.Error())
			report.PerFeed = append(report.PerFeed, FeedStatus{Name: src.Name, Status: "off", Detail: ferr.Error()})
			continue
		}
		var cves []CVEEntry
		for _, it := range feed.Items {
			for _, id := range extractCVEs(it.Title + " " + it.Description) {
				cves = append(cves, CVEEntry{
					ID:          id,
					Description: it.Title,
					Published:   it.Published.Format("2006-01-02T15:04:05.000"),
					ExploitAvailable: strings.Contains(strings.ToLower(it.Title+it.Description), "exploit"),
				})
			}
		}
		added := IngestCVEEntries(cves)
		recordFeed(src.Name, "ok", fmt.Sprintf("%d items", len(feed.Items)))
		report.PerFeed = append(report.PerFeed, FeedStatus{Name: src.Name, Status: "ok", Detail: src.URL, Items: len(feed.Items), CVEs: added})
		report.NewCVEs += added
		report.OK++
	}

	return report, nil
}

// extractCVEs returns the unique, normalised CVE IDs found in text.
func extractCVEs(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range cveIDRe.FindAllString(strings.ToUpper(text), -1) {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// ── Health reporting (mirrors reader.go / search.go / github.go doctor) ───────

var (
	feedMu     sync.Mutex
	feedStatus = map[string]string{}
	feedDetail = map[string]string{}
)

func recordFeed(name, status, detail string) {
	feedMu.Lock()
	feedStatus[name] = status
	feedDetail[name] = detail
	feedMu.Unlock()
}

// FeedBackendHealth returns the most recent probe status per advisory feed, for
// surfacing in /health.
func FeedBackendHealth() map[string]string {
	feedMu.Lock()
	defer feedMu.Unlock()
	out := make(map[string]string, len(feedStatus))
	for k, v := range feedStatus {
		out[k] = v
	}
	return out
}
