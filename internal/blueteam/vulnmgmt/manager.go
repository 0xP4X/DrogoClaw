package vulnmgmt

import (
	"fmt"
	"strings"
	"time"
)

// Vulnerability represents a managed vulnerability lifecycle.
type Vulnerability struct {
	ID           string       `json:"id"`
	CVE          string       `json:"cve"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Severity     string       `json:"severity"` // Low, Medium, High, Critical
	CVSSScore    float64      `json:"cvssScore"`
	Status       string       `json:"status"` // Open, In Progress, Mitigated, False Positive, Accepted Risk
	Target       string       `json:"target"`
	DiscoveredAt time.Time    `json:"discoveredAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	Remediation  string       `json:"remediation"`
	Tags         []string     `json:"tags"`
}

// Manager handles the lifecycle of vulnerabilities.
type Manager struct {
	vulns map[string]*Vulnerability
}

func NewManager() *Manager {
	return &Manager{
		vulns: make(map[string]*Vulnerability),
	}
}

// AddVulnerability adds a new finding to the vulnerability management system.
func (m *Manager) AddVulnerability(v *Vulnerability) {
	if v.ID == "" {
		v.ID = fmt.Sprintf("VULN-%d", time.Now().UnixNano())
	}
	if v.Status == "" {
		v.Status = "Open"
	}
	v.DiscoveredAt = time.Now()
	v.UpdatedAt = time.Now()
	
	m.vulns[v.ID] = v
}

// UpdateStatus changes the state of a vulnerability (e.g., Mitigated).
func (m *Manager) UpdateStatus(id, status string) error {
	v, exists := m.vulns[id]
	if !exists {
		return fmt.Errorf("vulnerability %s not found", id)
	}
	v.Status = status
	v.UpdatedAt = time.Now()
	return nil
}

// GeneratePatchPlan creates a prioritized list of patches based on CVSS severity.
func (m *Manager) GeneratePatchPlan() string {
	var critical, high, medium, low []string
	
	for _, v := range m.vulns {
		if v.Status == "Mitigated" || v.Status == "False Positive" {
			continue
		}
		
		line := fmt.Sprintf("[%s] %s on %s (Score: %.1f)", v.CVE, v.Title, v.Target, v.CVSSScore)
		switch strings.ToLower(v.Severity) {
		case "critical":
			critical = append(critical, line)
		case "high":
			high = append(high, line)
		case "medium":
			medium = append(medium, line)
		default:
			low = append(low, line)
		}
	}
	
	var sb strings.Builder
	sb.WriteString("═══════════════════════════════════════════\n")
	sb.WriteString(" PRIORITIZED PATCH PLAN\n")
	sb.WriteString("═══════════════════════════════════════════\n\n")
	
	if len(critical) > 0 {
		sb.WriteString("🔴 CRITICAL (SLA: 24 Hours)\n")
		for _, l := range critical {
			sb.WriteString("  - " + l + "\n")
		}
		sb.WriteString("\n")
	}
	
	if len(high) > 0 {
		sb.WriteString("🟠 HIGH (SLA: 7 Days)\n")
		for _, l := range high {
			sb.WriteString("  - " + l + "\n")
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}
