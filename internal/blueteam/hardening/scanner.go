package hardening

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

// ═══════════════════════════════════════════════════════════
// CIS Benchmark Scanner
// Automated security hardening checks based on CIS benchmarks
// for Linux, Windows, and cloud environments.
// ═══════════════════════════════════════════════════════════

// Severity classifies the criticality of a finding.
type Severity string

const (
	SevCritical Severity = "CRITICAL"
	SevHigh     Severity = "HIGH"
	SevMedium   Severity = "MEDIUM"
	SevLow      Severity = "LOW"
	SevInfo     Severity = "INFO"
)

// CheckResult is the outcome of a single hardening check.
type CheckResult struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Category    string   `json:"category"`
	Severity    Severity `json:"severity"`
	Status      string   `json:"status"`      // PASS, FAIL, WARN, SKIP
	Description string   `json:"description"`
	Remediation string   `json:"remediation"`
	Evidence    string   `json:"evidence"`
	CISRef      string   `json:"cisRef"`       // CIS benchmark reference ID
}

// ScanReport contains all hardening check results.
type ScanReport struct {
	Target     string        `json:"target"`
	OS         string        `json:"os"`
	TotalPass  int           `json:"totalPass"`
	TotalFail  int           `json:"totalFail"`
	TotalWarn  int           `json:"totalWarn"`
	TotalSkip  int           `json:"totalSkip"`
	Score      float64       `json:"score"` // 0–100 compliance percentage
	Results    []CheckResult `json:"results"`
}

// Check defines a hardening check to execute.
type Check struct {
	ID          string
	Title       string
	Category    string
	Severity    Severity
	CISRef      string
	Command     string   // shell command to run
	PassIf      string   // substring that indicates PASS if found in output
	FailIf      string   // substring that indicates FAIL if found in output
	Invert      bool     // if true, PassIf/FailIf logic is inverted
	Remediation string
}

// LinuxChecks contains CIS-aligned hardening checks for Linux systems.
var LinuxChecks = []Check{
	// ── Filesystem Configuration ────────────────────────────
	{
		ID:          "CIS-1.1.1",
		Title:       "Ensure mounting of cramfs filesystems is disabled",
		Category:    "Filesystem",
		Severity:    SevLow,
		CISRef:      "1.1.1.1",
		Command:     "modprobe -n -v cramfs 2>/dev/null; lsmod | grep cramfs",
		PassIf:      "install /bin/true",
		Remediation: "echo 'install cramfs /bin/true' >> /etc/modprobe.d/cramfs.conf",
	},
	{
		ID:          "CIS-1.1.2",
		Title:       "Ensure /tmp is a separate partition",
		Category:    "Filesystem",
		Severity:    SevMedium,
		CISRef:      "1.1.2",
		Command:     "findmnt -n /tmp",
		PassIf:      "/tmp",
		Remediation: "Configure /tmp as a separate partition in /etc/fstab",
	},

	// ── SSH Configuration ────────────────────────────────────
	{
		ID:          "CIS-5.2.1",
		Title:       "Ensure SSH root login is disabled",
		Category:    "SSH",
		Severity:    SevCritical,
		CISRef:      "5.2.8",
		Command:     "sshd -T 2>/dev/null | grep -i permitrootlogin || grep -i 'PermitRootLogin' /etc/ssh/sshd_config",
		PassIf:      "no",
		Remediation: "Set 'PermitRootLogin no' in /etc/ssh/sshd_config and restart sshd",
	},
	{
		ID:          "CIS-5.2.2",
		Title:       "Ensure SSH protocol 2 is used",
		Category:    "SSH",
		Severity:    SevHigh,
		CISRef:      "5.2.4",
		Command:     "sshd -T 2>/dev/null | grep -i protocol || grep -i 'Protocol' /etc/ssh/sshd_config",
		PassIf:      "2",
		Remediation: "Set 'Protocol 2' in /etc/ssh/sshd_config",
	},
	{
		ID:          "CIS-5.2.3",
		Title:       "Ensure SSH MaxAuthTries is set to 4 or less",
		Category:    "SSH",
		Severity:    SevMedium,
		CISRef:      "5.2.7",
		Command:     "sshd -T 2>/dev/null | grep -i maxauthtries || grep -i 'MaxAuthTries' /etc/ssh/sshd_config",
		PassIf:      "",  // requires numeric parse
		Remediation: "Set 'MaxAuthTries 4' in /etc/ssh/sshd_config",
	},
	{
		ID:          "CIS-5.2.4",
		Title:       "Ensure SSH PermitEmptyPasswords is disabled",
		Category:    "SSH",
		Severity:    SevCritical,
		CISRef:      "5.2.11",
		Command:     "sshd -T 2>/dev/null | grep -i permitemptypasswords || grep -i 'PermitEmptyPasswords' /etc/ssh/sshd_config",
		PassIf:      "no",
		Remediation: "Set 'PermitEmptyPasswords no' in /etc/ssh/sshd_config",
	},
	{
		ID:          "CIS-5.2.5",
		Title:       "Ensure SSH X11Forwarding is disabled",
		Category:    "SSH",
		Severity:    SevMedium,
		CISRef:      "5.2.6",
		Command:     "sshd -T 2>/dev/null | grep -i x11forwarding || grep -i 'X11Forwarding' /etc/ssh/sshd_config",
		PassIf:      "no",
		Remediation: "Set 'X11Forwarding no' in /etc/ssh/sshd_config",
	},

	// ── User Accounts ────────────────────────────────────────
	{
		ID:          "CIS-6.2.1",
		Title:       "Ensure password fields are not empty",
		Category:    "User Accounts",
		Severity:    SevCritical,
		CISRef:      "6.2.1",
		Command:     "awk -F: '($2 == \"\" ) { print $1 }' /etc/shadow 2>/dev/null",
		PassIf:      "",
		Invert:      true, // PASS if output is empty (no users without passwords)
		Remediation: "Lock or set passwords for all accounts with empty password fields",
	},
	{
		ID:          "CIS-6.2.2",
		Title:       "Ensure no duplicate UIDs exist",
		Category:    "User Accounts",
		Severity:    SevHigh,
		CISRef:      "6.2.15",
		Command:     "awk -F: '{print $3}' /etc/passwd | sort | uniq -d",
		PassIf:      "",
		Invert:      true,
		Remediation: "Resolve duplicate UIDs in /etc/passwd",
	},
	{
		ID:          "CIS-6.2.3",
		Title:       "Ensure root is the only UID 0 account",
		Category:    "User Accounts",
		Severity:    SevCritical,
		CISRef:      "6.2.6",
		Command:     "awk -F: '($3 == 0) { print $1 }' /etc/passwd",
		PassIf:      "root",
		Remediation: "Remove or change the UID of any non-root account with UID 0",
	},

	// ── Firewall ─────────────────────────────────────────────
	{
		ID:          "CIS-3.5.1",
		Title:       "Ensure iptables/nftables is installed",
		Category:    "Firewall",
		Severity:    SevHigh,
		CISRef:      "3.5.1.1",
		Command:     "which iptables 2>/dev/null || which nft 2>/dev/null",
		PassIf:      "/",
		Remediation: "Install iptables or nftables: apt-get install iptables",
	},
	{
		ID:          "CIS-3.5.2",
		Title:       "Ensure default deny firewall policy",
		Category:    "Firewall",
		Severity:    SevCritical,
		CISRef:      "3.5.2.1",
		Command:     "iptables -L INPUT -n 2>/dev/null | head -1",
		PassIf:      "DROP",
		Remediation: "Set default deny policy: iptables -P INPUT DROP",
	},

	// ── Kernel Parameters ────────────────────────────────────
	{
		ID:          "CIS-3.1.1",
		Title:       "Ensure IP forwarding is disabled",
		Category:    "Network",
		Severity:    SevMedium,
		CISRef:      "3.1.1",
		Command:     "sysctl net.ipv4.ip_forward 2>/dev/null",
		PassIf:      "= 0",
		Remediation: "sysctl -w net.ipv4.ip_forward=0",
	},
	{
		ID:          "CIS-3.1.2",
		Title:       "Ensure ICMP redirects are not accepted",
		Category:    "Network",
		Severity:    SevMedium,
		CISRef:      "3.2.2",
		Command:     "sysctl net.ipv4.conf.all.accept_redirects 2>/dev/null",
		PassIf:      "= 0",
		Remediation: "sysctl -w net.ipv4.conf.all.accept_redirects=0",
	},

	// ── Logging ──────────────────────────────────────────────
	{
		ID:          "CIS-4.1.1",
		Title:       "Ensure auditd is installed",
		Category:    "Logging",
		Severity:    SevHigh,
		CISRef:      "4.1.1.1",
		Command:     "dpkg -s auditd 2>/dev/null || rpm -q audit 2>/dev/null",
		PassIf:      "install ok installed",
		Remediation: "Install auditd: apt-get install auditd audispd-plugins",
	},
	{
		ID:          "CIS-4.1.2",
		Title:       "Ensure auditd service is running",
		Category:    "Logging",
		Severity:    SevHigh,
		CISRef:      "4.1.1.2",
		Command:     "systemctl is-active auditd 2>/dev/null",
		PassIf:      "active",
		Remediation: "systemctl enable --now auditd",
	},

	// ── Permissions ──────────────────────────────────────────
	{
		ID:          "CIS-6.1.1",
		Title:       "Ensure permissions on /etc/passwd are configured",
		Category:    "File Permissions",
		Severity:    SevHigh,
		CISRef:      "6.1.2",
		Command:     "stat -c '%a' /etc/passwd 2>/dev/null",
		PassIf:      "644",
		Remediation: "chmod 644 /etc/passwd",
	},
	{
		ID:          "CIS-6.1.2",
		Title:       "Ensure permissions on /etc/shadow are configured",
		Category:    "File Permissions",
		Severity:    SevCritical,
		CISRef:      "6.1.3",
		Command:     "stat -c '%a' /etc/shadow 2>/dev/null",
		PassIf:      "640",
		Remediation: "chmod 640 /etc/shadow",
	},
	{
		ID:          "CIS-6.1.3",
		Title:       "Ensure no world-writable files exist",
		Category:    "File Permissions",
		Severity:    SevHigh,
		CISRef:      "6.1.10",
		Command:     "find / -xdev -type f -perm -0002 2>/dev/null | head -5",
		PassIf:      "",
		Invert:      true,
		Remediation: "Remove world-writable permissions from identified files",
	},
	{
		ID:          "CIS-6.1.4",
		Title:       "Ensure no SUID/SGID executables exist in unusual locations",
		Category:    "File Permissions",
		Severity:    SevHigh,
		CISRef:      "6.1.13",
		Command:     "find /tmp /var/tmp /home -xdev -type f \\( -perm -4000 -o -perm -2000 \\) 2>/dev/null | head -5",
		PassIf:      "",
		Invert:      true,
		Remediation: "Remove SUID/SGID bits from files in non-standard locations",
	},
}

// RunLinuxHardening executes all Linux CIS benchmark checks against a target.
func RunLinuxHardening(ctx context.Context, sb *sandbox.Docker) (*ScanReport, error) {
	report := &ScanReport{
		Target:  "localhost",
		OS:      "Linux",
		Results: make([]CheckResult, 0, len(LinuxChecks)),
	}

	for _, check := range LinuxChecks {
		result := executeCheck(ctx, sb, check)
		report.Results = append(report.Results, result)

		switch result.Status {
		case "PASS":
			report.TotalPass++
		case "FAIL":
			report.TotalFail++
		case "WARN":
			report.TotalWarn++
		case "SKIP":
			report.TotalSkip++
		}
	}

	total := report.TotalPass + report.TotalFail + report.TotalWarn
	if total > 0 {
		report.Score = float64(report.TotalPass) / float64(total) * 100
	}

	return report, nil
}

func executeCheck(ctx context.Context, sb *sandbox.Docker, check Check) CheckResult {
	result := CheckResult{
		ID:          check.ID,
		Title:       check.Title,
		Category:    check.Category,
		Severity:    check.Severity,
		CISRef:      check.CISRef,
		Remediation: check.Remediation,
	}

	output, err := sb.Execute(ctx, check.Command)
	if err != nil {
		result.Status = "SKIP"
		result.Description = fmt.Sprintf("Check could not be executed: %v", err)
		return result
	}

	output = strings.TrimSpace(output)
	result.Evidence = output

	passed := evaluateResult(output, check)
	if passed {
		result.Status = "PASS"
		result.Description = "Configuration meets CIS benchmark requirements"
	} else {
		result.Status = "FAIL"
		result.Description = fmt.Sprintf("Configuration does not meet CIS benchmark requirements (CIS %s)", check.CISRef)
	}

	return result
}

func evaluateResult(output string, check Check) bool {
	if check.Invert {
		// For inverted checks, PASS means the output is empty (nothing bad found)
		return strings.TrimSpace(output) == ""
	}

	if check.PassIf != "" {
		return strings.Contains(strings.ToLower(output), strings.ToLower(check.PassIf))
	}

	return false
}

// FormatReport returns a human-readable text report of the hardening scan.
func FormatReport(report *ScanReport) string {
	var sb strings.Builder

	sb.WriteString("═══════════════════════════════════════════\n")
	sb.WriteString(" CIS BENCHMARK HARDENING REPORT\n")
	sb.WriteString("═══════════════════════════════════════════\n\n")
	sb.WriteString(fmt.Sprintf(" Target: %s (%s)\n", report.Target, report.OS))
	sb.WriteString(fmt.Sprintf(" Score:  %.1f%% compliance\n", report.Score))
	sb.WriteString(fmt.Sprintf(" Results: %d PASS | %d FAIL | %d WARN | %d SKIP\n\n",
		report.TotalPass, report.TotalFail, report.TotalWarn, report.TotalSkip))

	// Group by category
	categories := make(map[string][]CheckResult)
	for _, r := range report.Results {
		categories[r.Category] = append(categories[r.Category], r)
	}

	for category, results := range categories {
		sb.WriteString(fmt.Sprintf("┌─ %s\n", category))
		for _, r := range results {
			icon := "✓"
			switch r.Status {
			case "FAIL":
				icon = "✗"
			case "WARN":
				icon = "⚠"
			case "SKIP":
				icon = "○"
			}
			sb.WriteString(fmt.Sprintf("│  %s [%s] %s (%s)\n", icon, r.ID, r.Title, r.Severity))
			if r.Status == "FAIL" && r.Remediation != "" {
				sb.WriteString(fmt.Sprintf("│    → Fix: %s\n", r.Remediation))
			}
		}
		sb.WriteString("│\n")
	}

	sb.WriteString("═══════════════════════════════════════════\n")
	return sb.String()
}
