package incident

import (
	"fmt"
	"strings"
)

// Playbook represents an incident response playbook.
type Playbook struct {
	ID          string
	Name        string
	Description string
	Trigger     string // e.g., "Ransomware_Detected"
	Steps       []string
}

// defaultPlaybooks contains industry-standard IR procedures.
var defaultPlaybooks = []Playbook{
	{
		ID:          "IR-01",
		Name:        "Ransomware Outbreak",
		Description: "Response procedures for active ransomware encryption.",
		Trigger:     "Ransomware_Detected",
		Steps: []string{
			"1. Isolate infected hosts from the network immediately (do not power off to preserve memory).",
			"2. Disable compromised user and service accounts in Active Directory.",
			"3. Block known C2 IP addresses and domains at the perimeter firewall.",
			"4. Capture RAM from isolated hosts for forensics.",
			"5. Identify the initial access vector (Phishing, RDP, VPN).",
			"6. Begin restoration from offline immutable backups.",
		},
	},
	{
		ID:          "IR-02",
		Name:        "Data Exfiltration (Cloud)",
		Description: "Response procedures for unauthorized AWS S3 / Cloud data access.",
		Trigger:     "Mass_Data_Download",
		Steps: []string{
			"1. Identify and revoke the compromised AWS IAM access keys immediately.",
			"2. Identify the source IP address performing the download.",
			"3. Review CloudTrail logs to determine what other actions the key performed.",
			"4. Determine the classification of the exfiltrated data (PII, PHI, PCI).",
			"5. Engage legal and compliance teams for mandatory breach notification.",
		},
	},
}

// GetPlaybook returns the response playbook for a given trigger condition.
func GetPlaybook(trigger string) *Playbook {
	for _, pb := range defaultPlaybooks {
		if strings.EqualFold(pb.Trigger, trigger) {
			return &pb
		}
	}
	return nil
}

// FormatPlaybook prints the steps for incident responders.
func FormatPlaybook(pb *Playbook) string {
	if pb == nil {
		return "No playbook found for the given trigger."
	}
	
	var sb strings.Builder
	sb.WriteString("═══════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf(" INCIDENT RESPONSE PLAYBOOK: %s\n", pb.Name))
	sb.WriteString("═══════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf(" Description: %s\n", pb.Description))
	sb.WriteString(fmt.Sprintf(" Trigger:     %s\n\n", pb.Trigger))
	
	sb.WriteString(" IMMEDIATE ACTIONS:\n")
	for _, step := range pb.Steps {
		sb.WriteString(fmt.Sprintf("  %s\n", step))
	}
	
	return sb.String()
}
