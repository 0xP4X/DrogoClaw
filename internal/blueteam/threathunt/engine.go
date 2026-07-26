package threathunt

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

// ═══════════════════════════════════════════════════════════
// Threat Hunting Engine
// Proactive threat detection using IOC scanning, YARA rule
// matching, and Sigma rule processing.
// ═══════════════════════════════════════════════════════════

// IOCType classifies an indicator of compromise.
type IOCType string

const (
	IOCTypeIP       IOCType = "ip"
	IOCTypeDomain   IOCType = "domain"
	IOCTypeHash     IOCType = "hash_md5"
	IOCTypeHashSHA1 IOCType = "hash_sha1"
	IOCTypeHashSHA256 IOCType = "hash_sha256"
	IOCTypeURL      IOCType = "url"
	IOCTypeEmail    IOCType = "email"
	IOCTypeFilePath IOCType = "file_path"
	IOCTypeRegistry IOCType = "registry_key"
	IOCTypeMutex    IOCType = "mutex"
)

// IOC is an individual indicator of compromise.
type IOC struct {
	Type        IOCType   `json:"type"`
	Value       string    `json:"value"`
	Source      string    `json:"source"`      // threat intel source
	ThreatName  string    `json:"threatName"`  // associated malware/campaign
	Confidence  float64   `json:"confidence"`  // 0.0–1.0
	LastSeen    time.Time `json:"lastSeen"`
	Tags        []string  `json:"tags"`
}

// HuntResult is the outcome of a single threat hunting check.
type HuntResult struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`    // "ioc_match", "yara_match", "sigma_match", "anomaly"
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Evidence    string    `json:"evidence"`
	IOC         *IOC      `json:"ioc,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// HuntReport contains all threat hunting results.
type HuntReport struct {
	Target      string       `json:"target"`
	StartedAt   time.Time    `json:"startedAt"`
	CompletedAt time.Time    `json:"completedAt"`
	TotalHits   int          `json:"totalHits"`
	Results     []HuntResult `json:"results"`
}

// ScanForIOCs searches a target system for known indicators of compromise.
func ScanForIOCs(ctx context.Context, sb *sandbox.Docker, iocs []IOC) (*HuntReport, error) {
	report := &HuntReport{
		Target:    "localhost",
		StartedAt: time.Now(),
		Results:   make([]HuntResult, 0),
	}

	for i, ioc := range iocs {
		var cmd string
		switch ioc.Type {
		case IOCTypeIP:
			// Search network connections for suspicious IPs
			cmd = fmt.Sprintf("ss -tunp 2>/dev/null | grep '%s' || netstat -tunp 2>/dev/null | grep '%s'", ioc.Value, ioc.Value)
		case IOCTypeDomain:
			// Check DNS cache and /etc/hosts
			cmd = fmt.Sprintf("grep -r '%s' /etc/hosts /var/log/syslog /var/log/auth.log 2>/dev/null | head -5", ioc.Value)
		case IOCTypeHash:
			// Search for files matching the hash
			cmd = fmt.Sprintf("find / -xdev -type f -exec md5sum {} \\; 2>/dev/null | grep '%s' | head -3", ioc.Value)
		case IOCTypeHashSHA256:
			cmd = fmt.Sprintf("find / -xdev -type f -exec sha256sum {} \\; 2>/dev/null | grep '%s' | head -3", ioc.Value)
		case IOCTypeFilePath:
			cmd = fmt.Sprintf("ls -la '%s' 2>/dev/null && file '%s' 2>/dev/null", ioc.Value, ioc.Value)
		default:
			continue
		}

		output, err := sb.Execute(ctx, cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			continue
		}

		report.Results = append(report.Results, HuntResult{
			ID:          fmt.Sprintf("ioc-%d", i+1),
			Type:        "ioc_match",
			Severity:    "HIGH",
			Title:       fmt.Sprintf("IOC Match: %s (%s)", ioc.ThreatName, ioc.Type),
			Description: fmt.Sprintf("Indicator of compromise '%s' found on target system", ioc.Value),
			Evidence:    output,
			IOC:         &ioc,
			Timestamp:   time.Now(),
		})
		report.TotalHits++
	}

	report.CompletedAt = time.Now()
	return report, nil
}

// RunYARAScans runs YARA rules against a target directory.
func RunYARAScans(ctx context.Context, sb *sandbox.Docker, rulesPath, targetPath string) (*HuntReport, error) {
	report := &HuntReport{
		Target:    targetPath,
		StartedAt: time.Now(),
		Results:   make([]HuntResult, 0),
	}

	// Ensure YARA is installed
	installCmd := "which yara 2>/dev/null || (apt-get update -qq && apt-get install -y -qq yara 2>/dev/null)"
	if _, err := sb.Execute(ctx, installCmd); err != nil {
		return nil, fmt.Errorf("YARA installation failed: %v", err)
	}

	// Run YARA scan
	cmd := fmt.Sprintf("yara -r -s '%s' '%s' 2>/dev/null", rulesPath, targetPath)
	output, err := sb.Execute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("YARA scan failed: %v", err)
	}

	if strings.TrimSpace(output) == "" {
		report.CompletedAt = time.Now()
		return report, nil
	}

	// Parse YARA output (format: "rule_name file_path")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i, line := range lines {
		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		if len(parts) < 2 {
			continue
		}

		report.Results = append(report.Results, HuntResult{
			ID:          fmt.Sprintf("yara-%d", i+1),
			Type:        "yara_match",
			Severity:    "CRITICAL",
			Title:       fmt.Sprintf("YARA Match: %s", parts[0]),
			Description: fmt.Sprintf("YARA rule '%s' matched file: %s", parts[0], parts[1]),
			Evidence:    line,
			Timestamp:   time.Now(),
		})
		report.TotalHits++
	}

	report.CompletedAt = time.Now()
	return report, nil
}

// RunPersistenceHunt checks for common persistence mechanisms on Linux.
func RunPersistenceHunt(ctx context.Context, sb *sandbox.Docker) (*HuntReport, error) {
	report := &HuntReport{
		Target:    "localhost",
		StartedAt: time.Now(),
		Results:   make([]HuntResult, 0),
	}

	checks := []struct {
		title    string
		severity string
		cmd      string
	}{
		{
			title:    "Suspicious crontab entries",
			severity: "HIGH",
			cmd:      "for user in $(cut -f1 -d: /etc/passwd); do crontab -l -u $user 2>/dev/null | grep -v '^#' | grep -v '^$'; done",
		},
		{
			title:    "Unusual systemd services",
			severity: "HIGH",
			cmd:      "systemctl list-unit-files --type=service --state=enabled 2>/dev/null | grep -v 'enabled-runtime' | tail -20",
		},
		{
			title:    "Files modified in last 24h in /etc",
			severity: "MEDIUM",
			cmd:      "find /etc -mtime -1 -type f 2>/dev/null | head -20",
		},
		{
			title:    "Unusual SUID binaries",
			severity: "HIGH",
			cmd:      "find / -perm -4000 -type f 2>/dev/null | head -20",
		},
		{
			title:    "SSH authorized_keys modifications",
			severity: "CRITICAL",
			cmd:      "find /home -name authorized_keys -exec ls -la {} \\; 2>/dev/null; ls -la /root/.ssh/authorized_keys 2>/dev/null",
		},
		{
			title:    "Processes running from /tmp or /dev/shm",
			severity: "CRITICAL",
			cmd:      "ls -la /proc/*/exe 2>/dev/null | grep -E '(/tmp/|/dev/shm/)' | head -10",
		},
		{
			title:    "Unusual listening ports",
			severity: "MEDIUM",
			cmd:      "ss -tlnp 2>/dev/null | grep -v '127.0.0.1' | tail -20",
		},
		{
			title:    "LD_PRELOAD persistence",
			severity: "CRITICAL",
			cmd:      "cat /etc/ld.so.preload 2>/dev/null; grep -r LD_PRELOAD /etc/environment /etc/profile.d/ 2>/dev/null",
		},
		{
			title:    "Suspicious .bashrc/.profile additions",
			severity: "HIGH",
			cmd:      "grep -rn 'curl\\|wget\\|nc\\|ncat\\|base64\\|eval\\|python' /home/*/.bashrc /root/.bashrc 2>/dev/null | head -10",
		},
		{
			title:    "Kernel modules (potential rootkit)",
			severity: "CRITICAL",
			cmd:      "lsmod | grep -v -E '^(Module|ip_|nf_|xt_|x_|tcp_|udp_|ext4|xfs|dm_|sd_|sr_|sg_|scsi_|loop|fuse)' | head -10",
		},
	}

	for i, check := range checks {
		output, err := sb.Execute(ctx, check.cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			continue
		}

		report.Results = append(report.Results, HuntResult{
			ID:          fmt.Sprintf("persist-%d", i+1),
			Type:        "anomaly",
			Severity:    check.severity,
			Title:       check.title,
			Description: fmt.Sprintf("Persistence hunt detected potential indicators"),
			Evidence:    strings.TrimSpace(output),
			Timestamp:   time.Now(),
		})
		report.TotalHits++
	}

	report.CompletedAt = time.Now()
	return report, nil
}

// FormatHuntReport renders a human-readable threat hunting report.
func FormatHuntReport(report *HuntReport) string {
	var sb strings.Builder

	sb.WriteString("═══════════════════════════════════════════\n")
	sb.WriteString(" THREAT HUNTING REPORT\n")
	sb.WriteString("═══════════════════════════════════════════\n\n")
	sb.WriteString(fmt.Sprintf(" Target:   %s\n", report.Target))
	sb.WriteString(fmt.Sprintf(" Duration: %s\n", report.CompletedAt.Sub(report.StartedAt).Round(time.Millisecond)))
	sb.WriteString(fmt.Sprintf(" Hits:     %d findings\n\n", report.TotalHits))

	if report.TotalHits == 0 {
		sb.WriteString(" ✓ No threats detected.\n")
	} else {
		for _, r := range report.Results {
			icon := "⚠"
			if r.Severity == "CRITICAL" {
				icon = "🔴"
			}
			sb.WriteString(fmt.Sprintf(" %s [%s] %s\n", icon, r.Severity, r.Title))
			sb.WriteString(fmt.Sprintf("   %s\n", r.Description))
			if r.Evidence != "" {
				evidenceLines := strings.Split(r.Evidence, "\n")
				for _, line := range evidenceLines {
					if strings.TrimSpace(line) != "" {
						sb.WriteString(fmt.Sprintf("   │ %s\n", line))
					}
				}
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("═══════════════════════════════════════════\n")
	return sb.String()
}
