package adapt

import (
	"fmt"
	"regexp"
	"strings"
)

// ScriptRiskLevel classifies a script's risk.
type ScriptRiskLevel int

const (
	RiskLow    ScriptRiskLevel = iota
	RiskMedium                 // network connections
	RiskHigh                   // privilege escalation, file writes outside tmp
)

// ScriptAnalysis is the result of analyzing a script before execution.
type ScriptAnalysis struct {
	Risk     ScriptRiskLevel
	Reasons  []string
	NeedsHitL bool
}

var (
	networkPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(socket|requests\.get|urllib|wget|curl|nc\s|netcat|connect\()`),
		regexp.MustCompile(`(?i)(import\s+socket|import\s+requests|import\s+urllib)`),
	}
	privescPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(sudo\s|chmod\s+[0-9]*[67][0-9]*|chown\s+root|setuid|setgid)`),
		regexp.MustCompile(`(?i)(\/etc\/passwd|\/etc\/shadow|\/root\/|\/proc\/)`),
	}
	writePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(open\([^)]+['"](w|a|wb|ab)['"]\)|os\.write|shutil\.copy)`),
		regexp.MustCompile(`(?i)(echo\s+.*>\s*/(etc|root|var|bin|usr|sbin))`),
	}
)

// AnalyzeScript examines script code for dangerous patterns.
func AnalyzeScript(code string) *ScriptAnalysis {
	analysis := &ScriptAnalysis{Risk: RiskLow}

	for _, p := range networkPatterns {
		if p.MatchString(code) {
			analysis.Risk = RiskMedium
			analysis.Reasons = append(analysis.Reasons, "contains network I/O")
			break
		}
	}

	for _, p := range privescPatterns {
		if p.MatchString(code) {
			if analysis.Risk < RiskHigh {
				analysis.Risk = RiskHigh
			}
			analysis.Reasons = append(analysis.Reasons, "contains privilege escalation patterns")
			break
		}
	}

	for _, p := range writePatterns {
		if p.MatchString(code) {
			if analysis.Risk < RiskMedium {
				analysis.Risk = RiskMedium
			}
			analysis.Reasons = append(analysis.Reasons, "contains file write operations")
			break
		}
	}

	analysis.NeedsHitL = analysis.Risk >= RiskMedium
	return analysis
}

// BuildScriptCommand constructs the sandbox command to write and run a script.
func BuildScriptCommand(filename, language, code string) (string, error) {
	var interpreter string
	switch strings.ToLower(language) {
	case "python", "python3", "py":
		interpreter = "python3"
		if !strings.HasSuffix(filename, ".py") {
			filename += ".py"
		}
	case "bash", "sh", "shell":
		interpreter = "bash"
		if !strings.HasSuffix(filename, ".sh") {
			filename += ".sh"
		}
	case "ruby", "rb":
		interpreter = "ruby"
	case "perl":
		interpreter = "perl"
	default:
		return "", fmt.Errorf("unsupported script language: %s", language)
	}

	// Escape single quotes in code for safe shell embedding
	escapedCode := strings.ReplaceAll(code, "'", "'\\''")

	// Write to /tmp and execute
	cmd := fmt.Sprintf("cat > /tmp/%s << 'DROGON_EOF'\n%s\nDROGON_EOF\n%s /tmp/%s",
		filename, escapedCode, interpreter, filename)

	return cmd, nil
}

// HitLSuspendScript returns the HitL suspension string for a risky script.
func HitLSuspendScript(filename, language string, analysis *ScriptAnalysis) string {
	reasons := strings.Join(analysis.Reasons, ", ")
	return fmt.Sprintf("[HitL_SUSPENDED] | Script: %s (%s) requires approval — Risk: %s (%s)",
		filename, language, riskName(analysis.Risk), reasons)
}

func riskName(r ScriptRiskLevel) string {
	switch r {
	case RiskLow:
		return "LOW"
	case RiskMedium:
		return "MEDIUM"
	case RiskHigh:
		return "HIGH"
	default:
		return "UNKNOWN"
	}
}
