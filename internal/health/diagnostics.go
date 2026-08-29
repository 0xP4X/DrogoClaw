package health

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

// ── Colour palette (matches TUI styles) ─────────────────────────────────────

var (
	colAccent  = lipgloss.Color("#81D644") // brand green
	colSuccess = lipgloss.Color("#10B981") // emerald
	colDanger  = lipgloss.Color("#EF4444") // red
	colWarning = lipgloss.Color("#F59E0B") // amber
	colCyan    = lipgloss.Color("#06B6D4") // cyan
	colMuted   = lipgloss.Color("#6B7280") // gray
	colWhite   = lipgloss.Color("#F3F4F6") // near-white
	colGold    = lipgloss.Color("#FBBF24") // gold

	styleHeader  = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	styleSub     = lipgloss.NewStyle().Foreground(colCyan).Bold(true)
	styleOK      = lipgloss.NewStyle().Foreground(colSuccess).Bold(true)
	styleFail    = lipgloss.NewStyle().Foreground(colDanger).Bold(true)
	styleWarn    = lipgloss.NewStyle().Foreground(colWarning).Bold(true)
	styleLabel   = lipgloss.NewStyle().Foreground(colWhite)
	styleMuted   = lipgloss.NewStyle().Foreground(colMuted)
	styleGold    = lipgloss.NewStyle().Foreground(colGold).Bold(true)
	stylePct     = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	styleDivider = lipgloss.NewStyle().Foreground(colMuted)
)

// ToolDef describes a tool, its check command, and the capability domain it covers.
type ToolDef struct {
	Name     string
	Cmd      string // command to probe with (defaults to "which <Name>")
	Category string
	Critical bool // if true, absence degrades the section score heavily
}

type toolResult struct {
	name     string
	found    bool
	critical bool
}

// ── Tool registry by domain ──────────────────────────────────────────────────

var toolsByCategory = []struct {
	Label string
	Tools []ToolDef
}{
	{
		Label: "OSINT / Reconnaissance",
		Tools: []ToolDef{
			{Name: "nmap", Category: "recon", Critical: true},
			{Name: "subfinder", Category: "recon", Critical: true},
			{Name: "amass", Category: "recon"},
			{Name: "whatweb", Category: "recon"},
			{Name: "whois", Category: "recon"},
			{Name: "curl", Category: "recon", Critical: true},
			{Name: "dig", Category: "recon"},
			{Name: "host", Category: "recon"},
			{Name: "theHarvester", Category: "recon"},
			{Name: "shodan", Category: "recon"},
		},
	},
	{
		Label: "Web Enumeration",
		Tools: []ToolDef{
			{Name: "gobuster", Category: "web", Critical: true},
			{Name: "ffuf", Category: "web", Critical: true},
			{Name: "sqlmap", Category: "web", Critical: true},
			{Name: "wpscan", Category: "web"},
			{Name: "nikto", Category: "web"},
			{Name: "dirb", Category: "web"},
			{Name: "wfuzz", Category: "web"},
			{Name: "httpx", Category: "web", Cmd: `{ httpx -version 2>&1; } | grep -qi projectdiscovery; { httpx-toolkit -version 2>&1; } | grep -qi projectdiscovery`},
			{Name: "feroxbuster", Category: "web"},
		},
	},
	{
		Label: "Vulnerability Scanning",
		Tools: []ToolDef{
			{Name: "nuclei", Category: "vuln", Critical: true},
			{Name: "nmap", Category: "vuln", Critical: true}, // script engine reuse
			{Name: "openvas", Category: "vuln"},
			{Name: "nikto", Category: "vuln"},
		},
	},
	{
		Label: "Exploitation",
		Tools: []ToolDef{
			{Name: "msfconsole", Category: "exploit", Critical: true},
			{Name: "python3", Category: "exploit", Critical: true},
			{Name: "searchsploit", Category: "exploit"},
			{Name: "exploitdb", Category: "exploit", Cmd: "test -d /usr/share/exploitdb/exploits && echo ok"},
		},
	},
	{
		Label: "Post-Exploitation",
		Tools: []ToolDef{
			{Name: "nxc", Category: "post", Critical: true, Cmd: `command -v nxc || command -v netexec`},
			{Name: "impacket-secretsdump", Category: "post"},
			{Name: "evil-winrm", Category: "post"},
			{Name: "mimikatz", Category: "post"},
			{Name: "chisel", Category: "post"},
			{Name: "ligolo-ng", Category: "post", Cmd: "command -v ligolo-agent || command -v ligolo-ng-agent || command -v ligolo-ng-proxy"},
		},
	},
	{
		Label: "Password / Hashing",
		Tools: []ToolDef{
			{Name: "john", Category: "pass", Critical: true},
			{Name: "hashcat", Category: "pass", Critical: true},
			{Name: "hydra", Category: "pass"},
			{Name: "medusa", Category: "pass"},
			{Name: "crunch", Category: "pass"},
		},
	},
	{
		Label: "Binary / Reverse Engineering",
		Tools: []ToolDef{
			{Name: "gdb", Category: "binary", Critical: true},
			{Name: "radare2", Category: "binary"},
			{Name: "objdump", Category: "binary"},
			{Name: "strings", Category: "binary", Critical: true},
			{Name: "file", Category: "binary", Critical: true},
			{Name: "pwntools", Cmd: "python3 -c \"import pwn\"", Category: "binary"},
			{Name: "angr", Cmd: "python3 -c \"import angr\"", Category: "binary"},
			{Name: "ropper", Category: "binary"},
		},
	},
	{
		Label: "Forensics / DFIR",
		Tools: []ToolDef{
			{Name: "volatility3", Cmd: "python3 -c \"import volatility3\"", Category: "forensics"},
			{Name: "binwalk", Category: "forensics"},
			{Name: "foremost", Category: "forensics"},
			{Name: "autopsy", Category: "forensics"},
			{Name: "exiftool", Category: "forensics"},
			{Name: "steghide", Category: "forensics"},
			{Name: "stegseek", Category: "forensics"},
		},
	},
	{
		Label: "Wireless / SDR",
		Tools: []ToolDef{
			{Name: "aircrack-ng", Category: "wireless"},
			{Name: "aireplay-ng", Category: "wireless"},
			{Name: "airodump-ng", Category: "wireless"},
			{Name: "bettercap", Category: "wireless"},
		},
	},
	{
		Label: "Infrastructure / Runtime",
		Tools: []ToolDef{
			{Name: "docker", Category: "infra", Critical: true},
			{Name: "git", Category: "infra", Critical: true},
			{Name: "python3", Category: "infra", Critical: true},
			{Name: "go", Category: "infra"},
			{Name: "node", Category: "infra"},
			{Name: "ssh", Category: "infra"},
			{Name: "wget", Category: "infra"},
		},
	},
	{
		Label: "Documentation / Reporting",
		Tools: []ToolDef{
{Name: "pandoc", Category: "docs"},
			{Name: "weasyprint", Category: "docs", Cmd: "python3 -c \"import weasyprint\""},
			{Name: "pdflatex", Category: "docs"},
			{Name: "jq", Category: "docs"},
			{Name: "yq", Category: "docs"},
		},
	},
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func probeLocal(name, cmd string) bool {
	if cmd != "" {
		c := exec.Command("bash", "-c", cmd)
		c.Env = os.Environ()
		return c.Run() == nil
	}
	_, err := exec.LookPath(name)
	return err == nil
}

func probeSandbox(ctx context.Context, sb *sandbox.Docker, name, cmd string) bool {
	checkCmd := "which " + name
	if cmd != "" {
		checkCmd = cmd + " 2>/dev/null && echo ok"
	}
	out, err := sb.Execute(ctx, checkCmd)
	if err != nil {
		return false
	}
	out = strings.TrimSpace(out)
	if out == "" || strings.Contains(out, "not found") || strings.Contains(out, "command produced no output") {
		return false
	}
	return true
}

func renderBar(pct int, width int) string {
	filled := (pct * width) / 100
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	var col lipgloss.Color
	switch {
	case pct >= 80:
		col = colSuccess
	case pct >= 50:
		col = colWarning
	default:
		col = colDanger
	}
	return lipgloss.NewStyle().Foreground(col).Render(bar)
}

func pctColor(pct int) string {
	var s lipgloss.Style
	switch {
	case pct >= 80:
		s = styleOK
	case pct >= 50:
		s = styleWarn
	default:
		s = styleFail
	}
	return s.Render(fmt.Sprintf("%3d%%", pct))
}

func divider(width int) string {
	return styleDivider.Render("  " + strings.Repeat("─", width))
}

// ── Main entry point ─────────────────────────────────────────────────────────

// RunDiagnostics executes a real, colour-coded health report for DrogonClaw.
// It checks every tool category from OSINT through to documentation, calculates
// per-section and overall percentages, and renders an ASCII progress bar for each.
func RunDiagnostics(ctx context.Context, sb *sandbox.Docker) string {
	return RunDiagnosticsWithWidth(ctx, sb, 88)
}

// RunDiagnosticsWithWidth renders diagnostics for the available terminal width.
func RunDiagnosticsWithWidth(ctx context.Context, sb *sandbox.Docker, width int) string {
	var out strings.Builder
	start := time.Now()
	width = clamp(width, 48, 140)
	contentWidth := maxInt(44, width-4)

	// ── Header ───────────────────────────────────────────────────────────────

	out.WriteString("\n")
	out.WriteString(styleHeader.Render("  "+boxTop(contentWidth)) + "\n")
	out.WriteString(styleHeader.Render("  "+boxTitle("DROGONCLAW - SYSTEM HEALTH", contentWidth)) + "\n")
	out.WriteString(styleHeader.Render("  "+boxBottom(contentWidth)) + "\n\n")

	// ── Runtime check ────────────────────────────────────────────────────────

	out.WriteString(styleSub.Render("  ◈ RUNTIME ENVIRONMENT") + "\n")
	out.WriteString(divider(minInt(60, contentWidth)) + "\n")

	sandboxReady := sb != nil && sb.IsReady()
	if !sandboxReady {
		out.WriteString(styleWarn.Render("  [!] Sandbox unavailable — probing host tools directly") + "\n")
	} else if sb.IsNativeMode() {
		out.WriteString(styleOK.Render("  [✓] Runtime: ") + styleLabel.Render("Host native (WSL / Linux)") + "\n")
	} else {
		out.WriteString(styleOK.Render("  [✓] Runtime: ") + styleLabel.Render("Docker sandbox (kalilinux/kali-rolling)") + "\n")
	}

	// Node + skills
	nodeOK := probeLocal("node", "")
	if nodeOK {
		out.WriteString(styleOK.Render("  [✓] Node.js:  ") + styleMuted.Render("skill manifest generation available") + "\n")
	} else {
		out.WriteString(styleFail.Render("  [✗] Node.js:  ") + styleMuted.Render("not found — run: apt install nodejs") + "\n")
	}

	goOK := probeLocal("go", "")
	if goOK {
		out.WriteString(styleOK.Render("  [✓] Go:       ") + styleMuted.Render("build toolchain available") + "\n")
	} else {
		out.WriteString(styleWarn.Render("  [~] Go:       ") + styleMuted.Render("not in PATH (binary may still run)") + "\n")
	}

	out.WriteString("\n")

	// ── Per-category checks ───────────────────────────────────────────────────

	type catResult struct {
		label string
		total int
		pct   int
		tools []toolResult
	}

	var results []catResult
	overallTotal := 0
	overallPresent := 0

	for _, cat := range toolsByCategory {
		present := 0
		total := len(cat.Tools)
		toolResults := make([]toolResult, 0, total)

		for _, tool := range cat.Tools {
			var found bool
			if sandboxReady {
				found = probeSandbox(ctx, sb, tool.Name, tool.Cmd)
			} else {
				found = probeLocal(tool.Name, tool.Cmd)
			}

			if found {
				present++
			}
			toolResults = append(toolResults, toolResult{
				name:     tool.Name,
				found:    found,
				critical: tool.Critical,
			})
		}

		pct := 0
		if total > 0 {
			pct = (present * 100) / total
		}

		results = append(results, catResult{
			label: cat.Label,
			total: total,
			pct:   pct,
			tools: toolResults,
		})
		overallTotal += total
		overallPresent += present
	}

	// ── Render each category ─────────────────────────────────────────────────

	columns := 1
	if contentWidth >= 104 {
		columns = 2
	}
	gap := 2
	blockWidth := contentWidth
	if columns == 2 {
		blockWidth = (contentWidth - gap) / 2
	}

	var blocks []string
	for _, r := range results {
		blocks = append(blocks, renderCategoryBlock(r.label, r.pct, r.total, r.tools, blockWidth))
	}

	for i := 0; i < len(blocks); i += columns {
		if columns == 1 || i+1 >= len(blocks) {
			out.WriteString(blocks[i])
		} else {
			out.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, blocks[i], strings.Repeat(" ", gap), blocks[i+1]))
		}
		out.WriteString("\n")
	}

	// ── Overall score ─────────────────────────────────────────────────────────

	overallPct := 0
	if overallTotal > 0 {
		overallPct = (overallPresent * 100) / overallTotal
	}

	out.WriteString(divider(minInt(60, contentWidth)) + "\n")
	out.WriteString(fmt.Sprintf("  %s  %s  %s  %s/%d tools\n",
		styleGold.Render(padVisible("◈ OVERALL READINESS", minInt(26, contentWidth/2))),
		renderBar(overallPct, barWidth(contentWidth)),
		stylePct.Render(fmt.Sprintf("%3d%%", overallPct)),
		styleLabel.Render(fmt.Sprintf("%d", overallPresent)),
		overallTotal,
	))

	var verdict string
	switch {
	case overallPct >= 85:
		verdict = styleOK.Render("  ✔  FULL OPERATIONAL READINESS — DrogonClaw is mission-ready.")
	case overallPct >= 60:
		verdict = styleWarn.Render("  ⚠  PARTIAL READINESS — Some critical tools are missing.")
	case overallPct >= 35:
		verdict = styleFail.Render("  ✗  DEGRADED READINESS — Install missing tools before operation.")
	default:
		verdict = styleFail.Render("  ✗  MINIMAL INSTALL — Run: apt-get install -y kali-linux-everything")
	}
	out.WriteString(verdict + "\n")

	elapsed := time.Since(start).Round(time.Millisecond)
	out.WriteString("\n" + styleMuted.Render(fmt.Sprintf("  Diagnostic completed in %s", elapsed)) + "\n\n")

	return out.String()
}

func renderCategoryBlock(label string, pct, total int, tools []toolResult, width int) string {
	width = maxInt(44, width)
	barW := barWidth(width)
	nameW := clamp(width-23, 12, 24)
	present := 0
	for _, tool := range tools {
		if tool.found {
			present++
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s\n", styleSub.Render("◈ "+truncateVisible(label, width-4))))
	b.WriteString(fmt.Sprintf("    %s  %s  %s/%d\n",
		renderBar(pct, barW),
		pctColor(pct),
		styleLabel.Render(fmt.Sprintf("%d", present)),
		total,
	))

	for _, tool := range tools {
		name := padVisible(truncateVisible(tool.name, nameW), nameW)
		switch {
		case tool.found:
			b.WriteString(fmt.Sprintf("    %s %s %s\n",
				styleOK.Render("[✓]"),
				styleLabel.Render(name),
				styleMuted.Render("installed"),
			))
		case tool.critical:
			b.WriteString(fmt.Sprintf("    %s %s %s\n",
				styleFail.Render("[✗]"),
				styleFail.Render(name),
				styleWarn.Render("critical missing"),
			))
		default:
			b.WriteString(fmt.Sprintf("    %s %s %s\n",
				styleWarn.Render("[~]"),
				styleMuted.Render(name),
				styleMuted.Render("not installed"),
			))
		}
	}

	return lipgloss.NewStyle().Width(width).Render(strings.TrimRight(b.String(), "\n"))
}

func barWidth(width int) int {
	switch {
	case width >= 78:
		return 20
	case width >= 58:
		return 16
	default:
		return 10
	}
}

func boxTop(width int) string {
	return "╔" + strings.Repeat("═", maxInt(0, width-2)) + "╗"
}

func boxBottom(width int) string {
	return "╚" + strings.Repeat("═", maxInt(0, width-2)) + "╝"
}

func boxTitle(title string, width int) string {
	inner := maxInt(0, width-2)
	return "║" + lipgloss.PlaceHorizontal(inner, lipgloss.Center, title) + "║"
}

func padVisible(s string, width int) string {
	if lipgloss.Width(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-lipgloss.Width(s))
}

func truncateVisible(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	if width <= 3 {
		runes := []rune(s)
		return string(runes[:minInt(len(runes), width)])
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func clamp(v, low, high int) int {
	return minInt(maxInt(v, low), high)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
