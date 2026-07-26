package compliance

import (
	"fmt"
	"strings"
	"time"
)

// ═══════════════════════════════════════════════════════════
// Compliance Framework Engine
// Maps security controls to PCI-DSS 4.0, SOC 2 Type II,
// HIPAA, and ISO 27001 requirements.
// ═══════════════════════════════════════════════════════════

// Framework identifies a compliance standard.
type Framework string

const (
	FrameworkPCIDSS  Framework = "PCI-DSS-4.0"
	FrameworkSOC2    Framework = "SOC2-TypeII"
	FrameworkHIPAA   Framework = "HIPAA"
	FrameworkISO27001 Framework = "ISO-27001"
	FrameworkNIST    Framework = "NIST-CSF"
)

// ControlStatus is the compliance state of a control.
type ControlStatus string

const (
	StatusCompliant    ControlStatus = "COMPLIANT"
	StatusNonCompliant ControlStatus = "NON_COMPLIANT"
	StatusPartial      ControlStatus = "PARTIAL"
	StatusNotAssessed  ControlStatus = "NOT_ASSESSED"
)

// Control represents a single compliance control/requirement.
type Control struct {
	ID            string        `json:"id"`            // e.g. "PCI-DSS-6.2.4"
	Framework     Framework     `json:"framework"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	Category      string        `json:"category"`
	Status        ControlStatus `json:"status"`
	Evidence      string        `json:"evidence"`       // supporting evidence
	LastAssessed  time.Time     `json:"lastAssessed"`
	LinkedFindings []string     `json:"linkedFindings"` // DrogonClaw finding IDs
	Remediation   string        `json:"remediation"`
}

// ComplianceReport aggregates control assessments for a framework.
type ComplianceReport struct {
	Framework       Framework     `json:"framework"`
	GeneratedAt     time.Time     `json:"generatedAt"`
	TotalControls   int           `json:"totalControls"`
	Compliant       int           `json:"compliant"`
	NonCompliant    int           `json:"nonCompliant"`
	Partial         int           `json:"partial"`
	NotAssessed     int           `json:"notAssessed"`
	ComplianceScore float64       `json:"complianceScore"` // 0–100
	Controls        []Control     `json:"controls"`
}

// ControlMapping maps a DrogonClaw finding category to compliance controls.
type ControlMapping struct {
	FindingCategory string
	Controls        map[Framework][]string // Framework → control IDs
}

// controlMappings maps DrogonClaw security findings to compliance control IDs.
var controlMappings = []ControlMapping{
	{
		FindingCategory: "vulnerability_scan",
		Controls: map[Framework][]string{
			FrameworkPCIDSS:  {"6.2.4", "6.3.1", "6.3.2", "11.3.1", "11.3.2"},
			FrameworkSOC2:    {"CC7.1", "CC7.2"},
			FrameworkHIPAA:   {"164.312(a)(1)", "164.308(a)(1)(ii)(A)"},
			FrameworkISO27001: {"A.12.6.1", "A.18.2.3"},
			FrameworkNIST:    {"ID.RA-1", "ID.RA-5", "DE.CM-8"},
		},
	},
	{
		FindingCategory: "penetration_test",
		Controls: map[Framework][]string{
			FrameworkPCIDSS:  {"11.4.1", "11.4.2", "11.4.3", "11.4.4"},
			FrameworkSOC2:    {"CC4.1", "CC7.1"},
			FrameworkHIPAA:   {"164.308(a)(8)"},
			FrameworkISO27001: {"A.18.2.1", "A.14.2.8"},
			FrameworkNIST:    {"PR.IP-12", "DE.DP-3"},
		},
	},
	{
		FindingCategory: "access_control",
		Controls: map[Framework][]string{
			FrameworkPCIDSS:  {"7.1.1", "7.2.1", "7.2.2", "8.2.1", "8.3.1"},
			FrameworkSOC2:    {"CC6.1", "CC6.2", "CC6.3"},
			FrameworkHIPAA:   {"164.312(a)(1)", "164.312(d)"},
			FrameworkISO27001: {"A.9.1.1", "A.9.2.1", "A.9.2.3"},
			FrameworkNIST:    {"PR.AC-1", "PR.AC-4", "PR.AC-7"},
		},
	},
	{
		FindingCategory: "encryption",
		Controls: map[Framework][]string{
			FrameworkPCIDSS:  {"3.5.1", "4.2.1", "4.2.2"},
			FrameworkSOC2:    {"CC6.1", "CC6.7"},
			FrameworkHIPAA:   {"164.312(a)(2)(iv)", "164.312(e)(1)", "164.312(e)(2)(ii)"},
			FrameworkISO27001: {"A.10.1.1", "A.10.1.2"},
			FrameworkNIST:    {"PR.DS-1", "PR.DS-2"},
		},
	},
	{
		FindingCategory: "logging_monitoring",
		Controls: map[Framework][]string{
			FrameworkPCIDSS:  {"10.2.1", "10.2.2", "10.3.1", "10.4.1"},
			FrameworkSOC2:    {"CC7.2", "CC7.3"},
			FrameworkHIPAA:   {"164.312(b)", "164.308(a)(1)(ii)(D)"},
			FrameworkISO27001: {"A.12.4.1", "A.12.4.2", "A.12.4.3"},
			FrameworkNIST:    {"DE.AE-3", "DE.CM-1", "DE.CM-7", "PR.PT-1"},
		},
	},
	{
		FindingCategory: "incident_response",
		Controls: map[Framework][]string{
			FrameworkPCIDSS:  {"12.10.1", "12.10.2", "12.10.4"},
			FrameworkSOC2:    {"CC7.3", "CC7.4", "CC7.5"},
			FrameworkHIPAA:   {"164.308(a)(6)(i)", "164.308(a)(6)(ii)"},
			FrameworkISO27001: {"A.16.1.1", "A.16.1.2", "A.16.1.4", "A.16.1.5"},
			FrameworkNIST:    {"RS.RP-1", "RS.AN-1", "RS.CO-1"},
		},
	},
	{
		FindingCategory: "network_security",
		Controls: map[Framework][]string{
			FrameworkPCIDSS:  {"1.2.1", "1.3.1", "1.4.1", "1.4.2"},
			FrameworkSOC2:    {"CC6.1", "CC6.6"},
			FrameworkHIPAA:   {"164.312(e)(1)"},
			FrameworkISO27001: {"A.13.1.1", "A.13.1.2", "A.13.1.3"},
			FrameworkNIST:    {"PR.AC-5", "PR.DS-5", "PR.PT-4"},
		},
	},
	{
		FindingCategory: "patch_management",
		Controls: map[Framework][]string{
			FrameworkPCIDSS:  {"6.3.3"},
			FrameworkSOC2:    {"CC7.1"},
			FrameworkHIPAA:   {"164.308(a)(5)(ii)(B)"},
			FrameworkISO27001: {"A.12.6.1"},
			FrameworkNIST:    {"ID.RA-1", "PR.IP-12"},
		},
	},
}

// GetControlsForFinding returns the compliance controls impacted by a given finding category.
func GetControlsForFinding(findingCategory string, framework Framework) []string {
	for _, mapping := range controlMappings {
		if mapping.FindingCategory == findingCategory {
			if controls, ok := mapping.Controls[framework]; ok {
				return controls
			}
		}
	}
	return nil
}

// GetAllFrameworks returns all supported compliance frameworks.
func GetAllFrameworks() []Framework {
	return []Framework{
		FrameworkPCIDSS,
		FrameworkSOC2,
		FrameworkHIPAA,
		FrameworkISO27001,
		FrameworkNIST,
	}
}

// GenerateComplianceReport generates a compliance report for a framework based on findings.
func GenerateComplianceReport(framework Framework, assessedCategories map[string]ControlStatus) *ComplianceReport {
	report := &ComplianceReport{
		Framework:   framework,
		GeneratedAt: time.Now(),
		Controls:    make([]Control, 0),
	}

	for _, mapping := range controlMappings {
		controlIDs, ok := mapping.Controls[framework]
		if !ok {
			continue
		}

		status := StatusNotAssessed
		if s, exists := assessedCategories[mapping.FindingCategory]; exists {
			status = s
		}

		for _, ctrlID := range controlIDs {
			ctrl := Control{
				ID:           fmt.Sprintf("%s-%s", framework, ctrlID),
				Framework:    framework,
				Title:        fmt.Sprintf("%s control %s", framework, ctrlID),
				Category:     mapping.FindingCategory,
				Status:       status,
				LastAssessed: time.Now(),
			}
			report.Controls = append(report.Controls, ctrl)
		}
	}

	report.TotalControls = len(report.Controls)
	for _, ctrl := range report.Controls {
		switch ctrl.Status {
		case StatusCompliant:
			report.Compliant++
		case StatusNonCompliant:
			report.NonCompliant++
		case StatusPartial:
			report.Partial++
		case StatusNotAssessed:
			report.NotAssessed++
		}
	}

	assessed := report.Compliant + report.NonCompliant + report.Partial
	if assessed > 0 {
		report.ComplianceScore = float64(report.Compliant) / float64(assessed) * 100
	}

	return report
}

// FormatComplianceReport renders a human-readable compliance report.
func FormatComplianceReport(report *ComplianceReport) string {
	var sb strings.Builder

	sb.WriteString("═══════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf(" %s COMPLIANCE REPORT\n", report.Framework))
	sb.WriteString("═══════════════════════════════════════════\n\n")
	sb.WriteString(fmt.Sprintf(" Generated: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf(" Score:     %.1f%% compliant\n", report.ComplianceScore))
	sb.WriteString(fmt.Sprintf(" Controls:  %d total | %d compliant | %d non-compliant | %d partial | %d not assessed\n\n",
		report.TotalControls, report.Compliant, report.NonCompliant, report.Partial, report.NotAssessed))

	// Group by category
	categories := make(map[string][]Control)
	for _, ctrl := range report.Controls {
		categories[ctrl.Category] = append(categories[ctrl.Category], ctrl)
	}

	for category, controls := range categories {
		sb.WriteString(fmt.Sprintf("┌─ %s\n", strings.ToUpper(category)))
		for _, ctrl := range controls {
			icon := "?"
			switch ctrl.Status {
			case StatusCompliant:
				icon = "✓"
			case StatusNonCompliant:
				icon = "✗"
			case StatusPartial:
				icon = "◐"
			case StatusNotAssessed:
				icon = "○"
			}
			sb.WriteString(fmt.Sprintf("│  %s [%s] %s\n", icon, ctrl.ID, ctrl.Title))
		}
		sb.WriteString("│\n")
	}

	sb.WriteString("═══════════════════════════════════════════\n")
	return sb.String()
}
