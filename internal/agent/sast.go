package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// sastScanners lists the sink patterns the built-in scanner looks for when
// semgrep is unavailable. Each maps a regex to the vulnerability class it
// suggests, mimicking Strix's SAST+DAST source-review capability.
var sastScanners = []struct {
	pattern string
	class   string
}{
	{`eval\s*\(`, "Code injection via eval()"},
	{`exec\s*\(|os\.system\(|subprocess`, "OS command injection"},
	{`(?i)mysql_query\s*\([^)]*\+|execute\([^)]*\+|"[^"]*\+`, "SQL injection (string concatenation)"},
	{`(?i)\.format\(|%[sd]|f"|template\(`, "Template/format injection (SSTI)"},
	{`(?i)innerHTML\s*=|document\.write\(`, "DOM-based XSS sink"},
	{`(?i)deserialize|pickle\.load|yaml\.load\(`, "Insecure deserialization"},
	{`(?i)md5\(|sha1\(`, "Weak hash function"},
	{`(?i)verify\s*=\s*False|InsecureRequest`, "TLS verification disabled"},
}

// sastBuiltin is the tool the agent can call to review source code for
// vulnerability sinks. It prefers semgrep when installed, otherwise falls back
// to the built-in pattern scanner so source review works without extra deps.
func sastBuiltin(ctx context.Context, args map[string]any) string {
	target, _ := args["target"].(string)
	if target == "" {
		if p, ok := args["path"].(string); ok {
			target = p
		}
	}
	if target == "" {
		return "[Error] 'target' (path to file or directory) is required"
	}

	// Prefer semgrep if available on PATH.
	if path, lookErr := exec.LookPath("semgrep"); lookErr == nil {
		cmd := exec.CommandContext(ctx, path, "--config", "auto", "--json", "--quiet", target)
		out, runErr := cmd.CombinedOutput()
		if runErr == nil && len(out) > 0 {
			return fmt.Sprintf("[SAST:semgrep] %s\n%s", target, string(out))
		}
		// semgrep returned nothing (clean) or errored; fall through to built-in.
		if runErr == nil {
			return fmt.Sprintf("[SAST] %s: no findings from semgrep.", target)
		}
	}

	// Built-in fallback: walk files and match sink patterns.
	var findings []string
	walkErr := filepath.Walk(target, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		if !isSourceFile(path) {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		content := string(data)
		for _, s := range sastScanners {
			if matches := findLines(content, s.pattern); len(matches) > 0 {
				for _, m := range matches {
					findings = append(findings, fmt.Sprintf("  %s:%d  [%s] %s", path, m.line, s.class, m.snippet))
				}
			}
		}
		return nil
	})
	if walkErr != nil {
		return fmt.Sprintf("[Error] scanning %q: %v", target, walkErr)
	}

	if len(findings) == 0 {
		return fmt.Sprintf("[SAST] %s: no obvious sinks detected by built-in scanner.", target)
	}
	return fmt.Sprintf("[SAST] %s: %d potential sink(s) found:\n%s", target, len(findings), strings.Join(findings, "\n"))
}

func isSourceFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".py", ".js", ".ts", ".java", ".rb", ".php", ".c", ".cpp", ".cs", ".rs", ".jsp", ".html", ".txt":
		return true
	}
	return false
}

type lineMatch struct {
	line    int
	snippet string
}

// findLines returns the line numbers and trimmed snippets that match pattern.
func findLines(content, pattern string) []lineMatch {
	re := regexp.MustCompile(pattern)
	var out []lineMatch
	for i, line := range strings.Split(content, "\n") {
		if re.MatchString(line) {
			snippet := strings.TrimSpace(line)
			if len(snippet) > 120 {
				snippet = snippet[:117] + "..."
			}
			out = append(out, lineMatch{line: i + 1, snippet: snippet})
		}
	}
	return out
}
