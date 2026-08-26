package agent

// SkillLearner discovers and persists reusable attack patterns from successful
// tool executions. Inspired by Hermes Agent's autonomous skill creation:
// after a complex task succeeds, the agent saves the technique as a skill
// that can be recalled and improved in future engagements.
//
// Instead of re-discovering the same SQL injection technique on every
// WordPress target, the learner saves "WordPress + SQLi via plugin X"
// and applies it directly on similar targets.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// LearnedSkill represents a discovered attack pattern from a successful execution.
type LearnedSkill struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Category    string            `json:"category"` // "recon", "exploit", "privesc", "enum", "web"
	Tool        string            `json:"tool"`
	Args        map[string]any    `json:"args,omitempty"`
	Target      string            `json:"target_pattern"` // e.g., "wordpress", "apache", "ssh"
	Findings    []string          `json:"findings"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	SuccessCount int              `json:"success_count"`
	FailCount   int               `json:"fail_count"`
	LastUsed    time.Time         `json:"last_used"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// SkillLearner tracks successful attacks and extracts reusable patterns.
type SkillLearner struct {
	mu      sync.RWMutex
	skills  map[string]*LearnedSkill
	dataDir string
}

// NewSkillLearner creates a learner that persists learned skills to disk.
func NewSkillLearner(dataDir string) *SkillLearner {
	sl := &SkillLearner{
		skills:  make(map[string]*LearnedSkill),
		dataDir: filepath.Join(dataDir, "learned_skills"),
	}
	os.MkdirAll(sl.dataDir, 0700)
	sl.loadAll()
	return sl
}

// ObserveExecution records a tool execution and its outcome.
// If the execution was successful and produced findings, it creates or
// updates a learned skill.
func (sl *SkillLearner) ObserveExecution(tool string, args map[string]any, result string, verified bool) {
	if !verified {
		return // Only learn from verified successes
	}

	// Extract the target pattern (what kind of system was this?)
	target := classifyTarget(result, args)
	if target == "" {
		target = "unknown"
	}

	// Extract findings for pattern matching
	findings := extractSkillFindings(result)

	// Generate a skill key
	skillKey := generateSkillKey(tool, target, findings)

	sl.mu.Lock()
	defer sl.mu.Unlock()

	if existing, ok := sl.skills[skillKey]; ok {
		// Existing skill — reinforce it
		existing.SuccessCount++
		existing.LastUsed = time.Now()
		existing.UpdatedAt = time.Now()
		// Merge findings
		findingSet := make(map[string]bool)
		for _, f := range existing.Findings {
			findingSet[f] = true
		}
		for _, f := range findings {
			if !findingSet[f] {
				existing.Findings = append(existing.Findings, f)
			}
		}
		sl.save(skillKey, existing)
		return
	}

	// New skill — create it
	skill := &LearnedSkill{
		ID:           skillKey,
		Name:         generateSkillName(tool, target, findings),
		Category:     categorizeTool(tool),
		Tool:         tool,
		Args:         args,
		Target:       target,
		Findings:     findings,
		Description:  generateDescription(tool, target, findings),
		Tags:         generateTags(tool, target, findings),
		SuccessCount: 1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	sl.skills[skillKey] = skill
	sl.save(skillKey, skill)
}

// RecordFailure records a failed attempt to update accuracy tracking.
func (sl *SkillLearner) RecordFailure(tool string, args map[string]any, result string) {
	target := classifyTarget(result, args)
	if target == "" {
		target = "unknown"
	}
	findings := extractSkillFindings(result)
	skillKey := generateSkillKey(tool, target, findings)

	sl.mu.Lock()
	defer sl.mu.Unlock()

	if existing, ok := sl.skills[skillKey]; ok {
		existing.FailCount++
		existing.UpdatedAt = time.Now()
		sl.save(skillKey, existing)
	}
}

// FindRelevantSkills returns skills that match a target pattern, sorted by
// success rate (most reliable first).
func (sl *SkillLearner) FindRelevantSkills(targetPattern string, category string) []*LearnedSkill {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	var matches []*LearnedSkill
	for _, skill := range sl.skills {
		if !matchesTarget(skill.Target, targetPattern) {
			continue
		}
		if category != "" && skill.Category != category {
			continue
		}
		matches = append(matches, skill)
	}

	// Sort by success rate (descending)
	sort.Slice(matches, func(i, j int) bool {
		rateI := float64(matches[i].SuccessCount) / float64(max(1, matches[i].SuccessCount+matches[i].FailCount))
		rateJ := float64(matches[j].SuccessCount) / float64(max(1, matches[j].SuccessCount+matches[j].FailCount))
		if rateI != rateJ {
			return rateI > rateJ
		}
		return matches[i].SuccessCount > matches[j].SuccessCount
	})

	return matches
}

// GetSkill returns a specific skill by ID.
func (sl *SkillLearner) GetSkill(id string) *LearnedSkill {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	if s, ok := sl.skills[id]; ok {
		cp := *s
		return &cp
	}
	return nil
}

// ListSkills returns all learned skills.
func (sl *SkillLearner) ListSkills() []*LearnedSkill {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	out := make([]*LearnedSkill, 0, len(sl.skills))
	for _, s := range sl.skills {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].SuccessCount > out[j].SuccessCount
	})
	return out
}

// FormatForLLM formats relevant skills as context for the LLM prompt.
func (sl *SkillLearner) FormatForLLM(target string) string {
	skills := sl.FindRelevantSkills(target, "")
	if len(skills) == 0 {
		return ""
	}

	// Limit to top 5 most reliable
	limit := 5
	if len(skills) < limit {
		limit = len(skills)
	}

	var sb strings.Builder
	sb.WriteString("LEARNED ATTACK PATTERNS (from previous successes):\n")
	for _, s := range skills[:limit] {
		rate := float64(s.SuccessCount) / float64(max(1, s.SuccessCount+s.FailCount)) * 100
		sb.WriteString(fmt.Sprintf("- [%s] %s (success rate: %.0f%%, used %d times)\n", s.Category, s.Name, rate, s.SuccessCount))
		sb.WriteString(fmt.Sprintf("  Tool: %s\n", s.Tool))
		if s.Description != "" {
			sb.WriteString(fmt.Sprintf("  How: %s\n", s.Description))
		}
	}
	return sb.String()
}

// SaveHistory saves a compressed trajectory for future training data.
func (sl *SkillLearner) SaveHistory(tool string, args map[string]any, result string, duration time.Duration) {
	// This is a placeholder for trajectory compression (inspired by Hermes)
	// In a full implementation, this would use semantic compression to
	// reduce the trajectory size while preserving the attack logic.
}

func generateSkillKey(tool, target string, findings []string) string {
	parts := []string{tool, target}
	for _, f := range findings {
		parts = append(parts, f)
	}
	return strings.Join(parts, "::")
}

func generateSkillName(tool, target string, findings []string) string {
	name := fmt.Sprintf("%s on %s", strings.ReplaceAll(tool, "run_", ""), target)
	if len(findings) > 0 {
		name += " — " + findings[0]
	}
	return name
}

func generateDescription(tool, target string, findings []string) string {
	return fmt.Sprintf("Successfully used %s against %s targets, finding: %s",
		tool, target, strings.Join(findings, ", "))
}

func categorizeTool(tool string) string {
	switch {
	case strings.Contains(tool, "nmap") || strings.Contains(tool, "recon") || strings.Contains(tool, "subfinder") || strings.Contains(tool, "enum"):
		return "recon"
	case strings.Contains(tool, "exploit") || strings.Contains(tool, "sqlmap") || strings.Contains(tool, "nuclei") || strings.Contains(tool, "xss"):
		return "exploit"
	case strings.Contains(tool, "privesc") || strings.Contains(tool, "sudo") || strings.Contains(tool, "suid"):
		return "privesc"
	case strings.Contains(tool, "gobuster") || strings.Contains(tool, "ffuf") || strings.Contains(tool, "dir") || strings.Contains(tool, "fuzz"):
		return "enum"
	case strings.Contains(tool, "http") || strings.Contains(tool, "web") || strings.Contains(tool, "curl") || strings.Contains(tool, "request"):
		return "web"
	case strings.Contains(tool, "lateral") || strings.Contains(tool, "smb") || strings.Contains(tool, "ssh"):
		return "lateral"
	default:
		return "other"
	}
}

func generateTags(tool, target string, findings []string) []string {
	tags := []string{target}
	tags = append(tags, categorizeTool(tool))
	for _, f := range findings {
		if strings.HasPrefix(f, "cve:") {
			tags = append(tags, f)
		}
	}
	return tags
}

// classifyTarget identifies what kind of system the tool output describes.
func classifyTarget(output string, args map[string]any) string {
	low := strings.ToLower(output)

	// Check args for explicit target info
	if t, ok := args["target"].(string); ok {
		tl := strings.ToLower(t)
		if strings.Contains(tl, "wordpress") || strings.Contains(tl, "wp-") {
			return "wordpress"
		}
		if strings.Contains(tl, "drupal") {
			return "drupal"
		}
		if strings.Contains(tl, "joomla") {
			return "joomla"
		}
	}

	// Classify from output
	switch {
	case strings.Contains(low, "wordpress") || strings.Contains(low, "wp-content") || strings.Contains(low, "wp-admin"):
		return "wordpress"
	case strings.Contains(low, "apache") || strings.Contains(low, "apache/2"):
		return "apache"
	case strings.Contains(low, "nginx") || strings.Contains(low, "nginx/"):
		return "nginx"
	case strings.Contains(low, "iis") || strings.Contains(low, "microsoft-iis"):
		return "iis"
	case strings.Contains(low, "php/") || strings.Contains(low, ".php"):
		return "php"
	case strings.Contains(low, "asp.net") || strings.Contains(low, "x-aspnet"):
		return "aspnet"
	case strings.Contains(low, "ssh") || strings.Contains(low, "openssh"):
		return "ssh"
	case strings.Contains(low, "smb") || strings.Contains(low, "samba") || strings.Contains(low, "microsoft-ds"):
		return "smb"
	case strings.Contains(low, "mysql") || strings.Contains(low, "mariadb"):
		return "mysql"
	case strings.Contains(low, "postgresql") || strings.Contains(low, "postgres"):
		return "postgres"
	case strings.Contains(low, "mssql") || strings.Contains(low, "microsoft sql"):
		return "mssql"
	case strings.Contains(low, "oracle"):
		return "oracle"
	case strings.Contains(low, "tomcat"):
		return "tomcat"
	case strings.Contains(low, "jboss") || strings.Contains(low, "wildfly"):
		return "jboss"
	case strings.Contains(low, "docker"):
		return "docker"
	case strings.Contains(low, "kubernetes") || strings.Contains(low, "k8s"):
		return "kubernetes"
	default:
		return ""
	}
}

// extractSkillFindings extracts structured findings from tool output.
func extractSkillFindings(output string) []string {
	var findings []string

	// CVEs
	reCVE := regexp.MustCompile(`CVE-\d{4}-\d+`)
	for _, m := range reCVE.FindAllString(output, -1) {
		findings = append(findings, "cve:"+m)
	}

	// Open ports
	rePort := regexp.MustCompile(`(\d+)/(open|filtered)`)
	for _, m := range rePort.FindAllString(output, -1) {
		findings = append(findings, "port:"+m)
	}

	// HTTP status
	reHTTP := regexp.MustCompile(`status[:\s]+(\d{3})`)
	if m := reHTTP.FindString(output); m != "" {
		findings = append(findings, "http:"+m)
	}

	// Vulnerability confirmed
	low := strings.ToLower(output)
	if strings.Contains(low, "is vulnerable") || strings.Contains(low, "vulnerability confirmed") {
		findings = append(findings, "vuln:confirmed")
	}

	// SQL injection
	if strings.Contains(low, "sql") && (strings.Contains(low, "injection") || strings.Contains(low, "sqli")) {
		findings = append(findings, "vuln:sqli")
	}

	// XSS
	if strings.Contains(low, "xss") || strings.Contains(low, "cross-site") {
		findings = append(findings, "vuln:xss")
	}

	// RCE
	if strings.Contains(low, "remote code execution") || strings.Contains(low, "rce") || strings.Contains(low, "command injection") {
		findings = append(findings, "vuln:rce")
	}

	// Auth bypass
	if strings.Contains(low, "authentication bypass") || strings.Contains(low, "auth bypass") {
		findings = append(findings, "vuln:auth_bypass")
	}

	return findings
}

func matchesTarget(skillTarget, query string) bool {
	if skillTarget == query {
		return true
	}
	// Fuzzy match: skill target contains query or vice versa
	return strings.Contains(skillTarget, query) || strings.Contains(query, skillTarget)
}

func (sl *SkillLearner) save(key string, skill *LearnedSkill) {
	b, err := json.MarshalIndent(skill, "", "  ")
	if err != nil {
		return
	}
	safeKey := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, key)
	os.WriteFile(filepath.Join(sl.dataDir, safeKey+".json"), b, 0600)
}

func (sl *SkillLearner) loadAll() {
	entries, err := os.ReadDir(sl.dataDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(sl.dataDir, e.Name()))
		if err != nil {
			continue
		}
		var skill LearnedSkill
		if json.Unmarshal(b, &skill) == nil && skill.ID != "" {
			sl.skills[skill.ID] = &skill
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
