package mitre

// Technique represents a single MITRE ATT&CK technique/sub-technique.
type Technique struct {
	ID          string   `json:"id"`          // e.g. "T1059.001"
	Name        string   `json:"name"`        // e.g. "PowerShell"
	Tactic      string   `json:"tactic"`      // e.g. "Execution"
	Description string   `json:"description"` // brief explanation
	DataSources []string `json:"dataSources"` // detection data sources
	Platforms   []string `json:"platforms"`   // Windows, Linux, macOS, etc.
	URL         string   `json:"url"`         // MITRE ATT&CK page link
}

// Mapping associates a DrogonClaw finding/action with ATT&CK techniques.
type Mapping struct {
	FindingID   string      `json:"findingId"`
	Action      string      `json:"action"`      // what DrogonClaw did
	Techniques  []Technique `json:"techniques"`   // matched ATT&CK TTPs
	Confidence  float64     `json:"confidence"`   // 0.0–1.0
}

// TacticOrder defines the ATT&CK kill chain order for display/reporting.
var TacticOrder = []string{
	"Reconnaissance",
	"Resource Development",
	"Initial Access",
	"Execution",
	"Persistence",
	"Privilege Escalation",
	"Defense Evasion",
	"Credential Access",
	"Discovery",
	"Lateral Movement",
	"Collection",
	"Command and Control",
	"Exfiltration",
	"Impact",
}

// techniqueDB is the embedded mapping of DrogonClaw tool names → ATT&CK techniques.
// This covers the most common mappings used by the 52 pentest skills.
var techniqueDB = map[string][]Technique{
	// ── Reconnaissance ──────────────────────────────────────
	"nmap_scan": {
		{ID: "T1046", Name: "Network Service Discovery", Tactic: "Discovery", Platforms: []string{"Linux", "Windows", "macOS"}},
		{ID: "T1595.001", Name: "Active Scanning: Scanning IP Blocks", Tactic: "Reconnaissance", Platforms: []string{"PRE"}},
	},
	"masscan": {
		{ID: "T1046", Name: "Network Service Discovery", Tactic: "Discovery", Platforms: []string{"Linux", "Windows"}},
	},
	"dns_enum": {
		{ID: "T1018", Name: "Remote System Discovery", Tactic: "Discovery", Platforms: []string{"Linux", "Windows", "macOS"}},
		{ID: "T1596.001", Name: "Search Open Technical Databases: DNS/Passive DNS", Tactic: "Reconnaissance", Platforms: []string{"PRE"}},
	},
	"osint_recon": {
		{ID: "T1593", Name: "Search Open Websites/Domains", Tactic: "Reconnaissance", Platforms: []string{"PRE"}},
		{ID: "T1589", Name: "Gather Victim Identity Information", Tactic: "Reconnaissance", Platforms: []string{"PRE"}},
	},
	"subdomain_enum": {
		{ID: "T1596.001", Name: "Search Open Technical Databases: DNS/Passive DNS", Tactic: "Reconnaissance", Platforms: []string{"PRE"}},
	},

	// ── Web Enumeration ─────────────────────────────────────
	"dir_bruteforce": {
		{ID: "T1595.003", Name: "Active Scanning: Wordlist Scanning", Tactic: "Reconnaissance", Platforms: []string{"PRE"}},
	},
	"web_fuzz": {
		{ID: "T1190", Name: "Exploit Public-Facing Application", Tactic: "Initial Access", Platforms: []string{"Linux", "Windows"}},
	},

	// ── Exploitation ────────────────────────────────────────
	"sql_injection": {
		{ID: "T1190", Name: "Exploit Public-Facing Application", Tactic: "Initial Access", Platforms: []string{"Linux", "Windows"}},
	},
	"exploit_cve": {
		{ID: "T1190", Name: "Exploit Public-Facing Application", Tactic: "Initial Access", Platforms: []string{"Linux", "Windows"}},
		{ID: "T1203", Name: "Exploitation for Client Execution", Tactic: "Execution", Platforms: []string{"Linux", "Windows", "macOS"}},
	},
	"metasploit": {
		{ID: "T1190", Name: "Exploit Public-Facing Application", Tactic: "Initial Access", Platforms: []string{"Linux", "Windows"}},
		{ID: "T1059", Name: "Command and Scripting Interpreter", Tactic: "Execution", Platforms: []string{"Linux", "Windows", "macOS"}},
	},
	"shell_execute": {
		{ID: "T1059.004", Name: "Command and Scripting Interpreter: Unix Shell", Tactic: "Execution", Platforms: []string{"Linux", "macOS"}},
	},
	"python_execute": {
		{ID: "T1059.006", Name: "Command and Scripting Interpreter: Python", Tactic: "Execution", Platforms: []string{"Linux", "Windows", "macOS"}},
	},

	// ── Credential Access ───────────────────────────────────
	"bruteforce": {
		{ID: "T1110", Name: "Brute Force", Tactic: "Credential Access", Platforms: []string{"Linux", "Windows", "macOS"}},
	},
	"kerberoasting": {
		{ID: "T1558.003", Name: "Steal or Forge Kerberos Tickets: Kerberoasting", Tactic: "Credential Access", Platforms: []string{"Windows"}},
	},
	"pass_the_hash": {
		{ID: "T1550.002", Name: "Use Alternate Authentication Material: Pass the Hash", Tactic: "Lateral Movement", Platforms: []string{"Windows"}},
	},
	"dcsync": {
		{ID: "T1003.006", Name: "OS Credential Dumping: DCSync", Tactic: "Credential Access", Platforms: []string{"Windows"}},
	},
	"credential_dump": {
		{ID: "T1003", Name: "OS Credential Dumping", Tactic: "Credential Access", Platforms: []string{"Linux", "Windows"}},
	},

	// ── Privilege Escalation ────────────────────────────────
	"privesc_linux": {
		{ID: "T1548.001", Name: "Abuse Elevation Control Mechanism: Setuid and Setgid", Tactic: "Privilege Escalation", Platforms: []string{"Linux"}},
		{ID: "T1068", Name: "Exploitation for Privilege Escalation", Tactic: "Privilege Escalation", Platforms: []string{"Linux"}},
	},
	"privesc_windows": {
		{ID: "T1068", Name: "Exploitation for Privilege Escalation", Tactic: "Privilege Escalation", Platforms: []string{"Windows"}},
		{ID: "T1574", Name: "Hijack Execution Flow", Tactic: "Privilege Escalation", Platforms: []string{"Windows"}},
	},

	// ── Lateral Movement ────────────────────────────────────
	"pivot": {
		{ID: "T1021", Name: "Remote Services", Tactic: "Lateral Movement", Platforms: []string{"Linux", "Windows"}},
		{ID: "T1572", Name: "Protocol Tunneling", Tactic: "Command and Control", Platforms: []string{"Linux", "Windows"}},
	},
	"ad_lateral": {
		{ID: "T1021.002", Name: "Remote Services: SMB/Windows Admin Shares", Tactic: "Lateral Movement", Platforms: []string{"Windows"}},
		{ID: "T1021.006", Name: "Remote Services: Windows Remote Management", Tactic: "Lateral Movement", Platforms: []string{"Windows"}},
	},

	// ── Defense Evasion ─────────────────────────────────────
	"generate_fud_payload": {
		{ID: "T1027", Name: "Obfuscated Files or Information", Tactic: "Defense Evasion", Platforms: []string{"Linux", "Windows"}},
		{ID: "T1027.013", Name: "Obfuscated Files or Information: Encrypted/Encoded File", Tactic: "Defense Evasion", Platforms: []string{"Linux", "Windows"}},
	},
	"opsec_cleanup": {
		{ID: "T1070", Name: "Indicator Removal", Tactic: "Defense Evasion", Platforms: []string{"Linux", "Windows"}},
		{ID: "T1070.004", Name: "Indicator Removal: File Deletion", Tactic: "Defense Evasion", Platforms: []string{"Linux", "Windows"}},
	},

	// ── Exfiltration ────────────────────────────────────────
	"exfiltrate_data": {
		{ID: "T1048", Name: "Exfiltration Over Alternative Protocol", Tactic: "Exfiltration", Platforms: []string{"Linux", "Windows"}},
		{ID: "T1041", Name: "Exfiltration Over C2 Channel", Tactic: "Exfiltration", Platforms: []string{"Linux", "Windows"}},
	},

	// ── Persistence ─────────────────────────────────────────
	"implant_deploy": {
		{ID: "T1505.003", Name: "Server Software Component: Web Shell", Tactic: "Persistence", Platforms: []string{"Linux", "Windows"}},
	},

	// ── Collection ──────────────────────────────────────────
	"screenshot": {
		{ID: "T1113", Name: "Screen Capture", Tactic: "Collection", Platforms: []string{"Linux", "Windows", "macOS"}},
	},

	// ── Cloud ───────────────────────────────────────────────
	"aws_enum_iam": {
		{ID: "T1087.004", Name: "Account Discovery: Cloud Account", Tactic: "Discovery", Platforms: []string{"IaaS"}},
		{ID: "T1580", Name: "Cloud Infrastructure Discovery", Tactic: "Discovery", Platforms: []string{"IaaS"}},
	},
	"aws_escalate_privs": {
		{ID: "T1098", Name: "Account Manipulation", Tactic: "Persistence", Platforms: []string{"IaaS"}},
		{ID: "T1078.004", Name: "Valid Accounts: Cloud Accounts", Tactic: "Privilege Escalation", Platforms: []string{"IaaS"}},
	},
	"aws_dump_s3": {
		{ID: "T1530", Name: "Data from Cloud Storage", Tactic: "Collection", Platforms: []string{"IaaS"}},
	},

	// ── Social Engineering ──────────────────────────────────
	"phishing_campaign": {
		{ID: "T1566", Name: "Phishing", Tactic: "Initial Access", Platforms: []string{"Linux", "Windows", "macOS"}},
		{ID: "T1598", Name: "Phishing for Information", Tactic: "Reconnaissance", Platforms: []string{"PRE"}},
	},
}

// LookupTechniques returns the ATT&CK techniques associated with a tool/action name.
func LookupTechniques(toolName string) []Technique {
	if techniques, ok := techniqueDB[toolName]; ok {
		return techniques
	}
	return nil
}

// MapFinding creates a Mapping from a finding to its ATT&CK techniques.
func MapFinding(findingID, action, toolName string) *Mapping {
	techniques := LookupTechniques(toolName)
	if len(techniques) == 0 {
		return &Mapping{
			FindingID:  findingID,
			Action:     action,
			Techniques: nil,
			Confidence: 0.0,
		}
	}

	return &Mapping{
		FindingID:  findingID,
		Action:     action,
		Techniques: techniques,
		Confidence: 0.85, // high confidence for known tool→TTP mappings
	}
}

// GetTechniquesByTactic returns all mapped techniques grouped by tactic.
func GetTechniquesByTactic(mappings []*Mapping) map[string][]Technique {
	byTactic := make(map[string][]Technique)
	seen := make(map[string]bool)

	for _, m := range mappings {
		for _, t := range m.Techniques {
			key := t.ID + ":" + t.Tactic
			if !seen[key] {
				seen[key] = true
				byTactic[t.Tactic] = append(byTactic[t.Tactic], t)
			}
		}
	}
	return byTactic
}

// GetCoveredTactics returns a list of ATT&CK tactics that have at least one mapped technique.
func GetCoveredTactics(mappings []*Mapping) []string {
	byTactic := GetTechniquesByTactic(mappings)
	var covered []string
	for _, tactic := range TacticOrder {
		if _, ok := byTactic[tactic]; ok {
			covered = append(covered, tactic)
		}
	}
	return covered
}

// FormatATTCKMatrix returns a formatted text representation of the ATT&CK coverage.
func FormatATTCKMatrix(mappings []*Mapping) string {
	byTactic := GetTechniquesByTactic(mappings)
	var result string

	result += "═══════════════════════════════════════════\n"
	result += " MITRE ATT&CK® Technique Coverage\n"
	result += "═══════════════════════════════════════════\n\n"

	for _, tactic := range TacticOrder {
		techniques, ok := byTactic[tactic]
		if !ok {
			continue
		}
		result += "┌─ " + tactic + "\n"
		for _, t := range techniques {
			result += "│  ├─ [" + t.ID + "] " + t.Name + "\n"
		}
		result += "│\n"
	}

	result += "═══════════════════════════════════════════\n"
	return result
}
