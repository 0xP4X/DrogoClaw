package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// These tools implement the app/web validation parity track (c). They are for
// AUTHORIZED security testing only — never point them at systems you are not
// permitted to test. Outbound requests are timeout-bounded so they cannot hang
// a run.

var (
	// webClient never follows redirects (so reflected content is observed in the
	// immediate response) and has a hard 15s timeout.
	webClient = &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	sastMaxFiles = 2000
	sastMaxBytes int64 = 1 << 20 // 1 MiB

	sastExts = map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
		".php": true, ".java": true, ".rb": true, ".cs": true, ".jsp": true, ".html": true,
		".sql": true, ".c": true, ".cpp": true, ".lua": true, ".ps1": true,
	}

	// sastRules are conservative, high-signal patterns. Matches are CANDIDATES
	// requiring manual review, not confirmed vulnerabilities.
	sastRules = []struct {
		name     string
		severity string
		re       *regexp.Regexp
	}{
		{"SQL injection (string-built query)", "high", regexp.MustCompile(`(?i)(select|insert|update|delete)\b.{0,120}\+`)},
		{"Command injection (exec with concat)", "high", regexp.MustCompile(`(?i)(exec\.Command|os\.system|subprocess|popen|runtime\.exec)\b.{0,120}\+`)},
		{"Dynamic code evaluation (eval)", "high", regexp.MustCompile(`(?i)\beval\s*\(`)},
		{"XSS sink (innerHTML/document.write)", "high", regexp.MustCompile(`(?i)(innerHTML|outerHTML|document\.write|dangerouslySetInnerHTML)\s*=`)},
		{"XSS via echo of user input", "high", regexp.MustCompile(`(?i)echo\s+\$_(GET|POST|REQUEST|COOKIE)`)},
		{"Insecure deserialization", "medium", regexp.MustCompile(`(?i)(pickle\.loads|yaml\.load|unserialize|php\s+unserialize)`)},
		{"NoSQL injection ($where/$regex concat)", "high", regexp.MustCompile(`(?i)(\$where|\$regex)\b.{0,120}\+`)},
		{"Hardcoded secret/credential", "medium", regexp.MustCompile(`(?i)(api[_-]?key|secret|password|token|passwd)\s*[:=]\s*["'][^"']{8,}["']`)},
	}
)

// browserValidateScript drives a headless Chromium via Playwright to confirm a
// reflected-XSS candidate actually executes (not just reflects). Values are
// injected as JSON literals (valid Python) so there is no shell injection.
const browserValidateScript = `
import sys
try:
    from playwright.sync_api import sync_playwright
except Exception as e:
    print("BROWSER_UNAVAILABLE: " + str(e))
    sys.exit(2)

import urllib.parse
url = %s
payload = %s
marker = %s
param = %s

ok = False
detail = ""
try:
    with sync_playwright() as p:
        b = p.chromium.launch(headless=True, args=["--no-sandbox", "--disable-dev-shm-usage"])
        pg = b.new_page()
        dialog_fired = {"v": False}
        def on_dialog(d):
            dialog_fired["v"] = True
            try: d.dismiss()
            except Exception: pass
        pg.on("dialog", on_dialog)
        target = url
        if param:
            sep = "?" if "?" not in url else "&"
            target = url + sep + param + "=" + urllib.parse.quote(payload)
        pg.goto(target, wait_until="load", timeout=15000)
        pg.wait_for_timeout(1500)
        try:
            title = pg.title()
        except Exception:
            title = ""
        if marker in title:
            ok = True
            detail = "marker executed in document.title"
        elif dialog_fired["v"]:
            ok = True
            detail = "JS dialog fired (script executed)"
        b.close()
except Exception as e:
    detail = "error: " + str(e)
print("XSS_CONFIRMED" if ok else "XSS_NOT_CONFIRMED")
print("DETAIL: " + detail)
`

func jsonVal(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// registerWebTools adds the app/web validation capability (c): SAST, DAST
// reflected-XSS/CSRF probing, PoC request replay, and browser-driven XSS
// confirmation.
func (r *ToolRegistry) registerWebTools() {
	r.builtins["run_sast"] = func(ctx context.Context, args map[string]any) string {
		path, _ := args["path"].(string)
		if path == "" {
			path, _ = args["target"].(string)
		}
		if path == "" {
			return "[Error] path is required (local code directory or file)"
		}
		var findings []string
		count := 0
		_ = filepath.WalkDir(path, func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if d.IsDir() {
				return nil
			}
			if count >= sastMaxFiles {
				return nil
			}
			if !sastExts[strings.ToLower(filepath.Ext(p))] {
				return nil
			}
			info, err := d.Info()
			if err != nil || info.Size() > sastMaxBytes {
				return nil
			}
			content, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			count++
			for i, line := range strings.Split(string(content), "\n") {
				for _, rule := range sastRules {
					if rule.re.MatchString(line) {
						findings = append(findings, fmt.Sprintf("%s:%d: [%s] %s", p, i+1, rule.name, truncateToolEvidence(line, 120)))
						if r.lootDb != nil {
							_ = r.lootDb.InsertVulnerability(p, "", "SAST: "+rule.name, rule.severity)
						}
						break
					}
				}
			}
			return nil
		})
		r.lastTarget = path
		result := formatSAST(findings)
		r.lastResult = r.classifyOutcome("run_sast", result, nil)
		return result
	}

	r.builtins["web_probe"] = func(ctx context.Context, args map[string]any) string {
		u, _ := args["url"].(string)
		if u == "" {
			return "[Error] url is required (authorized target only)"
		}
		marker := "dcxss_" + randHex(6)
		base, err := getURL(ctx, u)
		if err != nil {
			return fmt.Sprintf("[web_probe Error] %v", err)
		}
		var findings []string
		inputs := extractInputNames(base)
		if len(inputs) == 0 {
			findings = append(findings, "no input parameters discovered in initial response")
		}
		tested := 0
		for _, name := range inputs {
			if tested >= 20 || ctx.Err() != nil {
				break
			}
			tested++
			probe := u
			sep := "?"
			if strings.Contains(probe, "?") {
				sep = "&"
			}
			resp, err := getURL(ctx, probe+sep+url.QueryEscape(name)+"="+marker)
			if err != nil {
				continue
			}
			if strings.Contains(resp, marker) {
				findings = append(findings, fmt.Sprintf("reflected parameter '%s' (possible reflected XSS) — validate in a browser / PoC", name))
			}
		}
		forms := extractFormActions(base)
		if len(forms) > 0 {
			low := strings.ToLower(base)
			if !strings.Contains(low, "csrf") && !strings.Contains(low, "_token") && !strings.Contains(low, "token") {
				findings = append(findings, fmt.Sprintf("%d form(s) without an apparent anti-CSRF token", len(forms)))
			}
		}
		r.lastTarget = u
		result := formatWeb(findings, "web_probe")
		r.lastResult = r.classifyOutcome("web_probe", result, nil)
		return result
	}

	r.builtins["replay_request"] = func(ctx context.Context, args map[string]any) string {
		u, _ := args["url"].(string)
		if u == "" {
			return "[Error] url is required (authorized target only)"
		}
		method, _ := args["method"].(string)
		if method == "" {
			method = "GET"
		}
		body, _ := args["body"].(string)
		headersRaw, _ := args["headers"].(string)
		var hdr map[string]string
		if headersRaw != "" {
			_ = json.Unmarshal([]byte(headersRaw), &hdr)
		}
		req, err := http.NewRequestWithContext(ctx, method, u, strings.NewReader(body))
		if err != nil {
			return fmt.Sprintf("[replay Error] %v", err)
		}
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		resp, err := webClient.Do(req)
		if err != nil {
			return fmt.Sprintf("[replay Error] %v", err)
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		r.lastTarget = u
		result := fmt.Sprintf("[replay %s] %s -> HTTP %d\n%s", method, u, resp.StatusCode, truncateToolEvidence(string(respBody), 800))
		r.lastResult = r.classifyOutcome("replay_request", result, nil)
		return result
	}

	r.builtins["browser_validate"] = func(ctx context.Context, args map[string]any) string {
		u, _ := args["url"].(string)
		if u == "" {
			return "[Error] url is required (authorized target only)"
		}
		param, _ := args["param"].(string)
		marker := "XSSOK_" + randHex(6)
		payload, _ := args["payload"].(string)
		if payload == "" {
			payload = "<img src=x onerror=\"document.title='" + marker + "'\">"
		}
		autoinstall, _ := args["autoinstall"].(bool)
		script := fmt.Sprintf(browserValidateScript, jsonVal(u), jsonVal(payload), jsonVal(marker), jsonVal(param))
		cmd := "python3 - <<'PYEOF'\n" + script + "\nPYEOF"
		if autoinstall {
			cmd = "pip install --quiet playwright >/dev/null 2>&1; playwright install --with-deps chromium >/dev/null 2>&1; " + cmd
		}
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[browser_validate Error] %v", err)
		}
		r.lastTarget = u
		var result string
		switch {
		case strings.Contains(out, "XSS_CONFIRMED"):
			result = fmt.Sprintf("[browser_validate] CONFIRMED XSS at %s (param=%q)\n%s", u, param, out)
			if r.lootDb != nil {
				_ = r.lootDb.InsertVulnerability(u, "", "Confirmed XSS (browser-executed): "+param, "high")
			}
		case strings.Contains(out, "BROWSER_UNAVAILABLE"):
			result = fmt.Sprintf("[browser_validate] Browser unavailable in sandbox. Install with: pip install playwright && playwright install chromium (or pass autoinstall=true).\n%s", out)
		default:
			result = fmt.Sprintf("[browser_validate] XSS NOT confirmed (candidate only — manual review).\n%s", out)
		}
		r.lastResult = r.classifyOutcome("browser_validate", result, nil)
		return result
	}
}

// --- helpers ---

func getURL(ctx context.Context, u string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}
	resp, err := webClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "deadbeef"
	}
	return hex.EncodeToString(b)
}

func extractInputNames(html string) []string {
	re := regexp.MustCompile(`(?i)<input[^>]+name=["']([^"']+)["']`)
	seen := map[string]bool{}
	var out []string
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		if m[1] != "" && !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

func extractFormActions(html string) []string {
	re := regexp.MustCompile(`(?i)<form[^>]+action=["']([^"']*)["']`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		out = append(out, m[1])
	}
	return out
}

func formatSAST(findings []string) string {
	if len(findings) == 0 {
		return "[SAST] No static-analysis pattern matches in the scanned tree. (Absence of matches is not proof of safety.)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[SAST] %d candidate pattern(s) found (manual review required):\n", len(findings))
	for _, f := range findings {
		b.WriteString("  - " + f + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatWeb(findings []string, tool string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %d observation(s) (candidates require manual/browser validation):\n", strings.ToUpper(tool), len(findings))
	for _, f := range findings {
		b.WriteString("  - " + f + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
