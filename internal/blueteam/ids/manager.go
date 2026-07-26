package ids

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Rule represents a parsed Suricata/Snort IDS rule.
type Rule struct {
	Action   string // alert, drop, pass, etc.
	Protocol string // tcp, udp, icmp, ip
	SrcIP    string
	SrcPort  string
	DstIP    string
	DstPort  string
	Msg      string
	SID      string
	Raw      string
}

// Manager handles loading and processing IDS rules.
type Manager struct {
	Rules []Rule
}

func NewManager() *Manager {
	return &Manager{
		Rules: make([]Rule, 0),
	}
}

// LoadRulesFile reads a Suricata/Snort rule file and parses the rules.
func (m *Manager) LoadRulesFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open rules file: %w", err)
	}
	defer file.Close()

	// Regex to extract msg and sid from rule options
	msgRegex := regexp.MustCompile(`msg:\s*"([^"]+)"`)
	sidRegex := regexp.MustCompile(`sid:\s*(\d+)`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "(", 2)
		if len(parts) < 2 {
			continue // Invalid rule format
		}

		header := strings.Fields(parts[0])
		if len(header) < 7 {
			continue
		}

		rule := Rule{
			Action:   header[0],
			Protocol: header[1],
			SrcIP:    header[2],
			SrcPort:  header[3],
			DstIP:    header[5],
			DstPort:  header[6],
			Raw:      line,
		}

		// Extract msg
		msgMatch := msgRegex.FindStringSubmatch(parts[1])
		if len(msgMatch) > 1 {
			rule.Msg = msgMatch[1]
		}

		// Extract sid
		sidMatch := sidRegex.FindStringSubmatch(parts[1])
		if len(sidMatch) > 1 {
			rule.SID = sidMatch[1]
		}

		m.Rules = append(m.Rules, rule)
	}

	return scanner.Err()
}

// GetRulesCount returns the number of loaded rules.
func (m *Manager) GetRulesCount() int {
	return len(m.Rules)
}

// SearchRules returns rules that match a given keyword in their message.
func (m *Manager) SearchRules(keyword string) []Rule {
	var results []Rule
	keyword = strings.ToLower(keyword)
	
	for _, rule := range m.Rules {
		if strings.Contains(strings.ToLower(rule.Msg), keyword) {
			results = append(results, rule)
		}
	}
	return results
}

// FormatRuleList returns a formatted string of the given rules.
func FormatRuleList(rules []Rule) string {
	var sb strings.Builder
	sb.WriteString("═══════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf(" IDS RULES MATCH (%d)\n", len(rules)))
	sb.WriteString("═══════════════════════════════════════════\n\n")

	for _, rule := range rules {
		sb.WriteString(fmt.Sprintf("[%s] %s (Protocol: %s)\n", rule.SID, rule.Msg, rule.Protocol))
		sb.WriteString(fmt.Sprintf("  ↳ %s %s -> %s %s\n", rule.SrcIP, rule.SrcPort, rule.DstIP, rule.DstPort))
		sb.WriteString("\n")
	}

	return sb.String()
}
