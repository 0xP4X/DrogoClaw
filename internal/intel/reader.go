package intel

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

// ReadResult is a structured, source-accountable extraction of a web page.
// It is the native-Go analogue of Agent Reach's per-channel read(), but typed
// and backend-attributed so the orchestrator always knows how the content was
// obtained and can feed links straight into the intelligence graph.
type ReadResult struct {
	URL        string   // canonical URL actually fetched
	Title      string   // <title> or og:title
	SiteName   string   // og:site_name
	Byline     string   // article:author / meta author
	Excerpt    string   // meta description / og:description
	Lang       string   // html lang attribute
	CleanText  string   // best-effort plain-text body (whitespace collapsed)
	Markdown   string   // structured markdown body when available
	Links      []string // absolute http(s) links discovered on the page
	FetchedVia string   // backend that produced the result: jina | native | strip
	Truncated  bool     // true when content exceeded the size budget
}

// PageBackend is one strategy for turning a URL into a ReadResult.
type PageBackend func(ctx context.Context, rawURL string) (*ReadResult, error)

// pageBackends is the ordered fallback chain (preferred first). This mirrors
// Agent Reach's "preferred + fallback" channel routing, but every backend is a
// real, dependency-free Go implementation that is probed at call time — there is
// no shelling out to external CLIs (the fragile part Agent Reach relies on).
var pageBackends = []struct {
	name string
	fn   PageBackend
}{
	{"jina", readJina},
	{"native", readNative},
	{"strip", readStrip},
}

// pageBackendStatus records the last probe outcome per backend so /health can
// report which readers are actually reachable (Agent Reach's doctor concept).
var (
	pageBackendMu      sync.Mutex
	pageBackendStatus  = map[string]string{}
	pageBackendDetail  = map[string]string{}
)

func recordBackend(name, status, detail string) {
	pageBackendMu.Lock()
	pageBackendStatus[name] = status
	pageBackendDetail[name] = detail
	pageBackendMu.Unlock()
}

// PageBackendHealth returns the most recent probe status for each reader
// backend, suitable for surfacing in /health.
func PageBackendHealth() map[string]string {
	pageBackendMu.Lock()
	defer pageBackendMu.Unlock()
	out := make(map[string]string, len(pageBackendStatus))
	for k, v := range pageBackendStatus {
		out[k] = v
	}
	return out
}

// FetchPage downloads and extracts a URL using the ordered backend chain.
// The first backend that returns a non-empty result wins; failures fall through
// to the next. FetchedVia records which backend succeeded.
func FetchPage(rawURL string) (*ReadResult, error) {
	return FetchPageContext(context.Background(), rawURL)
}

// FetchPageContext is FetchPage with an explicit context.
func FetchPageContext(ctx context.Context, rawURL string) (*ReadResult, error) {
	target, err := normalizeReadURL(rawURL)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, b := range pageBackends {
		res, berr := b.fn(ctx, target)
		if berr != nil {
			recordBackend(b.name, "off", berr.Error())
			lastErr = berr
			continue
		}
		if res == nil || strings.TrimSpace(res.CleanText) == "" {
			recordBackend(b.name, "warn", "empty result")
			lastErr = fmt.Errorf("%s returned empty content", b.name)
			continue
		}
		res.FetchedVia = b.name
		res.URL = target
		recordBackend(b.name, "ok", "available")
		return res, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("all readers returned empty content")
	}
	return nil, fmt.Errorf("failed to read %s: %w", target, lastErr)
}

// normalizeReadURL validates that the target is an absolute http(s) URL.
func normalizeReadURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("URL is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid URL: %q", rawURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q (only http/https)", u.Scheme)
	}
	return u.String(), nil
}

// readJina uses the Jina Reader endpoint (https://r.jina.ai/<url>) for clean,
// markdown-formatted extraction without any local HTML parsing. It is the
// preferred backend because it handles paywalls, lazy-loaded content, and
// boilerplate removal server-side.
func readJina(ctx context.Context, rawURL string) (*ReadResult, error) {
	endpoint := "https://r.jina.ai/" + rawURL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DrogonClaw/2.0 (security-assessment-tool)")
	req.Header.Set("Accept", "text/markdown, text/plain, */*")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jina request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("jina error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, readMaxBytes))
	if err != nil {
		return nil, fmt.Errorf("jina read failed: %w", err)
	}

	md := strings.TrimSpace(string(body))
	res := &ReadResult{Markdown: md, CleanText: md}
	// Jina usually emits a leading "# Title" line.
	if i := strings.Index(md, "\n"); i > 0 {
		first := strings.TrimSpace(md[:i])
		if strings.HasPrefix(first, "# ") {
			res.Title = strings.TrimSpace(first[2:])
			res.CleanText = strings.TrimSpace(md[i+1:])
		}
	}
	return res, nil
}

// readNative parses the raw HTML with golang.org/x/net/html and produces both
// a structured markdown view and a plain-text view, plus discovered links and
// metadata. This is the dependency-free fallback that does not rely on any
// external service — and is more robust than a regex strip because it walks the
// actual DOM tree (correctly skipping <script>/<style>/<noscript>).
func readNative(ctx context.Context, rawURL string) (*ReadResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, rawURL)
	}

	doc, err := html.Parse(io.LimitReader(resp.Body, readMaxBytes))
	if err != nil {
		return nil, fmt.Errorf("html parse failed: %w", err)
	}

	base, _ := url.Parse(rawURL)
	res := &ReadResult{Links: []string{}}
	var (
		mdBuilder  strings.Builder
		txtBuilder strings.Builder
		linkSet    = map[string]bool{}
		truncated  bool
	)

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				txtBuilder.WriteString(t + " ")
				mdBuilder.WriteString(t + " ")
			}
			return
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript", "template", "svg", "head":
				// Skip non-content nodes entirely (but still capture <title>/<meta>).
				if n.Data == "head" {
					extractHead(n, res)
				}
				return
			case "html":
				if res.Lang == "" {
					res.Lang = attr(n, "lang")
				}
			case "title":
				if res.Title == "" {
					res.Title = renderText(n)
				}
				return
			case "a":
				href := attr(n, "href")
				if href != "" {
					if abs := absURL(base, href); abs != "" && !linkSet[abs] {
						linkSet[abs] = true
						res.Links = append(res.Links, abs)
						if text := strings.TrimSpace(renderText(n)); text != "" {
							mdBuilder.WriteString("[" + text + "](" + abs + ")")
						}
					}
				}
			case "h1", "h2", "h3", "h4", "h5", "h6":
				level := int(n.Data[1] - '0')
				text := strings.TrimSpace(renderText(n))
				if text == "" {
					return
				}
				line := strings.Repeat("#", level) + " " + text
				mdBuilder.WriteString(line + "\n\n")
				txtBuilder.WriteString(text + "\n\n")
				return
			case "li":
				text := strings.TrimSpace(renderText(n))
				if text == "" {
					return
				}
				mdBuilder.WriteString("- " + text + "\n")
				txtBuilder.WriteString(text + "\n")
				return
			case "blockquote":
				text := strings.TrimSpace(renderText(n))
				if text != "" {
					mdBuilder.WriteString("> " + text + "\n\n")
					txtBuilder.WriteString(text + "\n\n")
				}
				return
			case "br":
				txtBuilder.WriteString("\n")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode && (n.Data == "p" || n.Data == "div") {
			// Treat block boundaries as paragraph breaks.
			txtBuilder.WriteString("\n")
		}
	}
	walk(doc)

	res.Markdown = collapseSpace(mdBuilder.String())
	res.CleanText = collapseSpace(txtBuilder.String())
	res.Truncated = truncated || len(res.CleanText) > readMaxChars
	if len(res.CleanText) > readMaxChars {
		res.CleanText = res.CleanText[:readMaxChars] + "\n\n[Content truncated]"
	}
	if res.Excerpt == "" {
		res.Excerpt = firstSentence(res.CleanText)
	}
	return res, nil
}

// readStrip is the last-resort fallback using the existing regex-based HTML
// stripper. It keeps the previous FetchURL behavior available when both the
// Jina service and the structured parser fail.
func readStrip(ctx context.Context, rawURL string) (*ReadResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, rawURL)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, err
	}
	text := stripHTML(string(body))
	if len(text) > 8000 {
		text = text[:8000] + "\n\n[Content truncated at 8000 chars]"
	}
	return &ReadResult{CleanText: strings.TrimSpace(text)}, nil
}

// extractHead pulls metadata (description, og:*, lang, author) from a <head> node.
func extractHead(head *html.Node, res *ReadResult) {
	var crawl func(*html.Node)
	crawl = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if res.Title == "" {
					res.Title = renderText(n)
				}
			case "meta":
				property := strings.ToLower(attr(n, "property"))
				name := strings.ToLower(attr(n, "name"))
				content := attr(n, "content")
				switch {
				case property == "og:title" && res.Title == "":
					res.Title = content
				case property == "og:site_name":
					res.SiteName = content
				case property == "og:description" && res.Excerpt == "":
					res.Excerpt = content
				case name == "description" && res.Excerpt == "":
					res.Excerpt = content
				case property == "article:author":
					res.Byline = content
				case name == "author":
					res.Byline = content
				}
			case "html":
				if res.Lang == "" {
					res.Lang = attr(n, "lang")
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			crawl(c)
		}
	}
	crawl(head)
}

// ── helpers ─────────────────────────────────────────────────────────────────

const (
	readMaxBytes = 8 * 1024 * 1024 // 8MB hard cap on downloaded bytes
	readMaxChars = 40000           // soft cap on returned plain text
)

// renderText returns the concatenated text content of a node subtree.
func renderText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(c *html.Node) {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
		for k := c.FirstChild; k != nil; k = k.NextSibling {
			walk(k)
		}
	}
	walk(n)
	return sb.String()
}

// attr returns the value of attribute key on n (case-insensitive match on key).
func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

// absURL resolves a possibly-relative href against the page base URL.
func absURL(base *url.URL, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "#") || strings.HasPrefix(ref, "mailto:") ||
		strings.HasPrefix(ref, "javascript:") || strings.HasPrefix(ref, "tel:") {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(u)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	return resolved.String()
}

// collapseSpace normalizes runs of whitespace into single spaces/line breaks.
func collapseSpace(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	re := strings.NewReplacer("\t", " ")
	s = re.Replace(s)
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(s)
}

// firstSentence returns the opening sentence of a text block for use as an excerpt.
func firstSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if i := strings.IndexAny(text, ".!?"); i > 0 {
		return strings.TrimSpace(text[:i+1])
	}
	if len(text) > 200 {
		return text[:200]
	}
	return text
}

// FetchURL downloads a URL and returns its content as clean text, preserving the
// original signature used by the researcher and agent tools. It now routes
// through the ordered reader chain (Jina → native DOM → regex strip).
func FetchURL(rawURL string) (string, error) {
	res, err := FetchPage(rawURL)
	if err != nil {
		return "", err
	}
	return res.CleanText, nil
}
