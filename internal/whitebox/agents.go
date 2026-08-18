package whitebox

import (
	"regexp"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/agent"
)

// reFileLine matches "path:line" references emitted by SAST tooling.
var reFileLine = regexp.MustCompile(`([\w./\-]+\.\w+):(\d+)`)

// ─────────────────────────────────────────────────────────────────────────────
// Phase 1 — Pre-Recon (source analysis)
// ─────────────────────────────────────────────────────────────────────────────

func preRecon(ctx interface{ Done() <-chan struct{} }, tools *agent.ToolRegistry, repo string, calls *int) []Finding {
	// ctx is accepted as a minimal interface to avoid an unused-import lint while
	// keeping the signature parallel to the dynamic phases.
	_ = ctx
	var out []Finding
	n := 0
	id := func() string { n++; return "WB-SRC-" + itoa(n) }

	// Prefer semgrep-backed source review; the builtin falls back to a pattern
	// scanner when semgrep is unavailable.
	res := call(tools, "source_review", map[string]any{"target": repo}, calls)
	for _, m := range reFileLine.FindAllStringSubmatch(res, -1) {
		sev := severityFromText(m[0])
		out = append(out, Finding{
			ID:       id(),
			Title:    "Source sink candidate: " + sinkClass(res, m[0]),
			Class:    sinkClass(res, m[0]),
			Severity: sev,
			Target:   repo,
			Location: m[1] + ":" + m[2],
			Evidence: snippet(res, m[0]),
			PoC:      "Review " + m[1] + ":" + m[2] + " for the flagged sink.",
			Verified: false,
			Source:   "sast",
		})
	}
	return out
}

// sinkClass derives an OWASP-ish class label from a SAST finding line.
func sinkClass(blob, line string) string {
	low := strings.ToLower(line + " " + blob)
	switch {
	case strings.Contains(low, "sql") || strings.Contains(low, "sqli"):
		return "injection"
	case strings.Contains(low, "xss") || strings.Contains(low, "innerhtml") || strings.Contains(low, "document.write"):
		return "xss"
	case strings.Contains(low, "ssrf") || strings.Contains(low, "gopher") || strings.Contains(low, "file://"):
		return "ssrf"
	case strings.Contains(low, "deserial") || strings.Contains(low, "pickle") || strings.Contains(low, "yaml.load"):
		return "injection"
	case strings.Contains(low, "secret") || strings.Contains(low, "api_key") || strings.Contains(low, "password"):
		return "authn"
	default:
		return "injection"
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Phase 2 — Recon (live attack surface)
// ─────────────────────────────────────────────────────────────────────────────

func recon(ctx interface{ Done() <-chan struct{} }, tools *agent.ToolRegistry, url string, calls *int) []Finding {
	_ = ctx
	var out []Finding
	n := 0
	id := func() string { n++; return "WB-REC-" + itoa(n) }

	// Lightweight DAST probe — discovers parameters, reflected-XSS candidates,
	// and missing CSRF tokens. Findings are candidates requiring validation.
	if r := call(tools, "web_probe", map[string]any{"url": url}, calls); strings.TrimSpace(r) != "" {
		if strings.Contains(strings.ToLower(r), "reflect") || strings.Contains(strings.ToLower(r), "csrf") {
			out = append(out, Finding{
				ID:       id(),
				Title:    "Reflected-input / missing CSRF candidate from web probe",
				Class:    "xss",
				Severity: SeverityLow,
				Target:   url,
				Location: url,
				Evidence: snippet(r, "web_probe"),
				PoC:      "Re-test the flagged parameter with a browser-driven payload (browser_validate).",
				Verified: false,
				Source:   "recon",
			})
		}
	}

	// Known-vulnerability + technology scan via Nuclei.
	if r := call(tools, "run_nuclei", map[string]any{"target": url, "severity": "critical,high,medium"}, calls); strings.TrimSpace(r) != "" {
		out = append(out, parseNuclei(r, url, id, "recon")...)
	}

	// Phase 3: DAST/fuzzing for unknown-vulnerability discovery. Reuses the
	// extended run_nuclei wrapper (dast=true) rather than a separate fuzzer.
	if r := call(tools, "run_nuclei", map[string]any{"target": url, "severity": "critical,high,medium", "dast": true}, calls); strings.TrimSpace(r) != "" {
		out = append(out, parseNuclei(r, url, id, "dast")...)
	}

	// Content/endpoint discovery for later fuzzing.
	_ = call(tools, "run_gobuster", map[string]any{"target": url}, calls)
	_ = call(tools, "run_httpx", map[string]any{"target": url}, calls)
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// Phase 3 — Vulnerability analysis (five parallel agents)
// ─────────────────────────────────────────────────────────────────────────────

func agentInjection(ctx interface{ Done() <-chan struct{} }, tools *agent.ToolRegistry, url string, calls *int) []Finding {
	_ = ctx
	var out []Finding
	n := 0
	id := func() string { n++; return "WB-INJ-" + itoa(n) }

	if r := call(tools, "run_sqlmap", map[string]any{"target": url, "level": 2, "risk": 2}, calls); strings.Contains(strings.ToLower(r), "is vulnerable") {
		out = append(out, Finding{
			ID:       id(),
			Title:    "SQL injection confirmed by sqlmap",
			Class:    "injection",
			Severity: SeverityHigh,
			Target:   url,
			Location: url,
			Evidence: snippet(r, "is vulnerable"),
			PoC:      "sqlmap -u " + url + " --batch --level 2 --risk 2",
			Verified: false,
			Source:   "dast",
		})
	}
	if r := call(tools, "run_nuclei", map[string]any{"target": url, "severity": "critical,high,medium", "tags": "sqli"}, calls); strings.TrimSpace(r) != "" {
		out = append(out, parseNuclei(r, url, id, "dast")...)
	}
	return out
}

func agentXSS(ctx interface{ Done() <-chan struct{} }, tools *agent.ToolRegistry, url string, calls *int) []Finding {
	_ = ctx
	var out []Finding
	n := 0
	id := func() string { n++; return "WB-XSS-" + itoa(n) }

	if r := call(tools, "run_nuclei", map[string]any{"target": url, "severity": "critical,high,medium", "tags": "xss"}, calls); strings.TrimSpace(r) != "" {
		out = append(out, parseNuclei(r, url, id, "dast")...)
	}
	// Browser-driven confirmation when a parameter is known.
	if r := call(tools, "browser_validate", map[string]any{"url": url, "autoinstall": true}, calls); strings.Contains(r, "CONFIRMED XSS") {
		out = append(out, Finding{
			ID:       id(),
			Title:    "Reflected XSS confirmed in headless browser",
			Class:    "xss",
			Severity: SeverityHigh,
			Target:   url,
			Location: url,
			Evidence: snippet(r, "CONFIRMED XSS"),
			PoC:      "browser_validate against " + url,
			Verified: false,
			Source:   "dast",
		})
	}
	return out
}

func agentSSRF(ctx interface{ Done() <-chan struct{} }, tools *agent.ToolRegistry, url string, calls *int) []Finding {
	_ = ctx
	var out []Finding
	n := 0
	id := func() string { n++; return "WB-SSRF-" + itoa(n) }

	if r := call(tools, "run_nuclei", map[string]any{"target": url, "severity": "critical,high,medium", "tags": "ssrf"}, calls); strings.TrimSpace(r) != "" {
		out = append(out, parseNuclei(r, url, id, "dast")...)
	}
	return out
}

func agentAuthN(ctx interface{ Done() <-chan struct{} }, tools *agent.ToolRegistry, url string, calls *int) []Finding {
	_ = ctx
	var out []Finding
	n := 0
	id := func() string { n++; return "WB-AUTHN-" + itoa(n) }

	if r := call(tools, "auth_bypass_scan", map[string]any{"target_url": url}, calls); strings.Contains(strings.ToLower(r), "bypass") || strings.Contains(strings.ToLower(r), "vulnerable") {
		out = append(out, Finding{
			ID:       id(),
			Title:    "Authentication weakness from auth_bypass_scan",
			Class:    "authn",
			Severity: severityFromText(r),
			Target:   url,
			Location: url,
			Evidence: snippet(r, "bypass"),
			PoC:      "auth_bypass_scan against " + url,
			Verified: false,
			Source:   "dast",
		})
	}
	if r := call(tools, "run_nuclei", map[string]any{"target": url, "severity": "critical,high,medium", "tags": "jwt,auth,login"}, calls); strings.TrimSpace(r) != "" {
		out = append(out, parseNuclei(r, url, id, "dast")...)
	}
	return out
}

func agentAuthZ(ctx interface{ Done() <-chan struct{} }, tools *agent.ToolRegistry, url string, calls *int) []Finding {
	_ = ctx
	var out []Finding
	n := 0
	id := func() string { n++; return "WB-AUTHZ-" + itoa(n) }

	// IDOR / broken-access-control detection. Full authz testing requires
	// authenticated sessions (low/high-privilege) to compare responses; the
	// scaffold runs the unauthenticated surface and flags candidates for the
	// operator to deepen with a logged-in session.
	if r := call(tools, "run_nuclei", map[string]any{"target": url, "severity": "critical,high,medium", "tags": "idor,auth"}, calls); strings.TrimSpace(r) != "" {
		out = append(out, parseNuclei(r, url, id, "dast")...)
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// Phase 4 — Exploitation / verification (Strix-style blind verification)
// ─────────────────────────────────────────────────────────────────────────────

func verifyFindings(ctx interface{ Done() <-chan struct{} }, tools *agent.ToolRegistry, in []Finding, calls *int) []Finding {
	_ = ctx
	for i := range in {
		f := &in[i]
		if f.Verified {
			continue
		}
		switch f.Class {
		case "xss":
			if r := call(tools, "browser_validate", map[string]any{"url": f.Target, "param": f.Location, "autoinstall": true}, calls); strings.Contains(r, "CONFIRMED XSS") {
				f.Verified = true
				f.PoC = "browser_validate confirmed execution at " + f.Target
				f.Evidence = snippet(r, "CONFIRMED XSS")
			}
		case "injection":
			if strings.Contains(f.Target, "http") {
				if r := call(tools, "run_sqlmap", map[string]any{"target": f.Target, "level": 2, "risk": 2}, calls); strings.Contains(strings.ToLower(r), "is vulnerable") {
					f.Verified = true
					f.PoC = "sqlmap -u " + f.Target + " --batch"
					f.Evidence = snippet(r, "is vulnerable")
				}
			}
		}
	}
	return in
}

// ─────────────────────────────────────────────────────────────────────────────
// Shared parsers
// ─────────────────────────────────────────────────────────────────────────────

// parseNuclei turns Nuclei output into findings. Nuclei lines look like:
//
//	[example.com] [info] ... or [CVE-2023-1234] [high] http://host/path
func parseNuclei(out, url string, id func() string, src string) []Finding {
	var out2 []Finding
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		low := strings.ToLower(line)
		if !strings.Contains(low, "http") && !strings.Contains(low, "cve") {
			continue
		}
		// Skip nuclei metadata / empty-scan lines.
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[NUCLEI —") {
			continue
		}
		if strings.Contains(low, "scan complete") ||
			strings.Contains(low, "no vulnerabilities found") ||
			strings.Contains(low, "no matches") ||
			strings.Contains(low, "no results") {
			continue
		}
		if strings.Contains(low, "[info]") && !strings.Contains(low, "cve") {
			continue // skip pure informational matchers
		}
		sev := SeverityLow
		switch {
		case strings.Contains(low, "[critical]"):
			sev = SeverityCritical
		case strings.Contains(low, "[high]"):
			sev = SeverityHigh
		case strings.Contains(low, "[medium]"):
			sev = SeverityMedium
		case strings.Contains(low, "[low]"):
			sev = SeverityLow
		default:
			sev = SeverityMedium
		}
		cve := cveRef(line)
		title := "Nuclei match"
		if cve != "" {
			title = cve
		}
		cls := classFromNuclei(line)
		if cls == "" {
			continue
		}
		out2 = append(out2, Finding{
			ID:       id(),
			Title:    title,
			Class:    cls,
			Severity: sev,
			Target:   url,
			Location: firstURL(line),
			Evidence: strings.TrimSpace(line),
			PoC:      "nuclei -u " + url + " -silent",
			Verified: cve != "", // a CVE match is treated as confirmed-by-reference
			Source:   src,
		})
	}
	return out2
}

func classFromNuclei(line string) string {
	low := strings.ToLower(line)
	switch {
	case strings.Contains(low, "sqli") || strings.Contains(low, "sql-injection"):
		return "injection"
	case strings.Contains(low, "xss"):
		return "xss"
	case strings.Contains(low, "ssrf"):
		return "ssrf"
	case strings.Contains(low, "jwt") || strings.Contains(low, "auth") || strings.Contains(low, "login"):
		return "authn"
	case strings.Contains(low, "idor") || strings.Contains(low, "access-control") || strings.Contains(low, "authz"):
		return "authz"
	default:
		return ""
	}
}

var reURL = regexp.MustCompile(`https?://[^\s\]]+`)

func firstURL(s string) string {
	return reURL.FindString(s)
}

// snippet returns a short, safe excerpt of blob around needle.
func snippet(blob, needle string) string {
	if needle == "" {
		if len(blob) > 400 {
			return blob[:400]
		}
		return blob
	}
	idx := strings.Index(blob, needle)
	if idx < 0 {
		if len(blob) > 400 {
			return blob[:400]
		}
		return blob
	}
	start := idx - 120
	if start < 0 {
		start = 0
	}
	end := idx + 200
	if end > len(blob) {
		end = len(blob)
	}
	return strings.TrimSpace(blob[start:end])
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
