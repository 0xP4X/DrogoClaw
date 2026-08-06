package intel

import (
	"fmt"
	"strings"
)

// Report is a small, chainable builder for structured, markdown-formatted
// intel output. Centralising output here keeps every tool's results consistent
// and scannable (and renders cleanly in the TUI, which uses glamour). It is the
// scalable primitive other recon commands should adopt instead of hand-rolled
// "===" / "[+]" ASCII art.
type Report struct {
	buf strings.Builder
}

// NewReport starts a report with a top-level heading.
func NewReport(title string) *Report {
	r := &Report{}
	r.buf.WriteString("# " + title + "\n\n")
	return r
}

// Section adds a level-2 heading.
func (r *Report) Section(title string) *Report {
	r.buf.WriteString("## " + title + "\n\n")
	return r
}

// KV adds a bold key/value line.
func (r *Report) KV(key, value string) *Report {
	r.buf.WriteString(fmt.Sprintf("- **%s:** %s\n", key, value))
	return r
}

// Bullet adds a bullet line.
func (r *Report) Bullet(line string) *Report {
	r.buf.WriteString("- " + line + "\n")
	return r
}

// Line adds a raw line (already formatted).
func (r *Report) Line(line string) *Report {
	r.buf.WriteString(line + "\n")
	return r
}

// Note adds an italicised note line.
func (r *Report) Note(line string) *Report {
	r.buf.WriteString("_" + line + "_\n")
	return r
}

// String returns the rendered markdown (trailing whitespace trimmed).
func (r *Report) String() string {
	return strings.TrimRight(r.buf.String(), "\n")
}

// mdCode wraps a value in inline code backticks — useful for repo paths/URLs.
func mdCode(s string) string {
	return "`" + s + "`"
}
