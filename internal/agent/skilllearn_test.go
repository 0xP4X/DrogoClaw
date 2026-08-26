package agent

import (
	"testing"
)

func TestSkillLearnerObserveExecution(t *testing.T) {
	sl := NewSkillLearner(t.TempDir())

	// Observe a successful SQL injection
	sl.ObserveExecution("run_sqlmap", map[string]any{"target": "http://wordpress.com/page?id=1"}, "SQL injection found: parameter 'id' is vulnerable CVE-2024-1234", true)

	skills := sl.ListSkills()
	if len(skills) != 1 {
		t.Fatalf("Expected 1 skill, got %d", len(skills))
	}
	if skills[0].SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", skills[0].SuccessCount)
	}
}

func TestSkillLearnerReinforcement(t *testing.T) {
	sl := NewSkillLearner(t.TempDir())

	// Observe same technique twice
	sl.ObserveExecution("run_sqlmap", map[string]any{"target": "http://wordpress.com/page?id=1"}, "SQL injection found", true)
	sl.ObserveExecution("run_sqlmap", map[string]any{"target": "http://wordpress.com/page?id=1"}, "SQL injection found", true)

	skills := sl.ListSkills()
	if len(skills) != 1 {
		t.Fatalf("Expected 1 skill (reinforced), got %d", len(skills))
	}
	if skills[0].SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2", skills[0].SuccessCount)
	}
}

func TestSkillLearnerIgnoresFailures(t *testing.T) {
	sl := NewSkillLearner(t.TempDir())

	// Only verified successes create skills
	sl.ObserveExecution("run_nmap", nil, "connection refused", false)

	skills := sl.ListSkills()
	if len(skills) != 0 {
		t.Errorf("Expected 0 skills from unverified results, got %d", len(skills))
	}
}

func TestSkillLearnerFindRelevant(t *testing.T) {
	sl := NewSkillLearner(t.TempDir())

	sl.ObserveExecution("run_nuclei", nil, "WordPress vulnerability found CVE-2024-1234", true)
	sl.ObserveExecution("run_nuclei", nil, "Apache vulnerability found CVE-2024-5678", true)

	// Find WordPress skills
	skills := sl.FindRelevantSkills("wordpress", "")
	if len(skills) != 1 {
		t.Fatalf("Expected 1 WordPress skill, got %d", len(skills))
	}
}

func TestSkillLearnerFormatForLLM(t *testing.T) {
	sl := NewSkillLearner(t.TempDir())

	sl.ObserveExecution("run_nuclei", nil, "WordPress vulnerability found", true)

	output := sl.FormatForLLM("wordpress")
	if output == "" {
		t.Error("Expected non-empty LLM context")
	}
}

func TestSkillLearnerFormatForLLMNoMatch(t *testing.T) {
	sl := NewSkillLearner(t.TempDir())

	output := sl.FormatForLLM("nonexistent")
	if output != "" {
		t.Errorf("Expected empty for no match, got %q", output)
	}
}

func TestSkillLearnerPersistence(t *testing.T) {
	dir := t.TempDir()

	sl1 := NewSkillLearner(dir)
	sl1.ObserveExecution("run_nmap", nil, "open ports found", true)

	sl2 := NewSkillLearner(dir)
	skills := sl2.ListSkills()
	if len(skills) != 1 {
		t.Errorf("Expected 1 persisted skill, got %d", len(skills))
	}
}

func TestClassifyTarget(t *testing.T) {
	tests := []struct {
		output string
		want   string
	}{
		{"WordPress 6.4 installed", "wordpress"},
		{"Apache/2.4.41", "apache"},
		{"nginx/1.18", "nginx"},
		{"OpenSSH_8.2", "ssh"},
		{"MySQL 8.0", "mysql"},
		{"random text", ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := classifyTarget(tt.output, nil)
			if got != tt.want {
				t.Errorf("classifyTarget() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractSkillFindings(t *testing.T) {
	output := "CVE-2024-1234 found, 80/open, status: 200, SQL injection detected"
	findings := extractSkillFindings(output)

	if len(findings) == 0 {
		t.Error("Expected findings, got none")
	}

	hasCVE := false
	for _, f := range findings {
		if f == "cve:CVE-2024-1234" {
			hasCVE = true
		}
	}
	if !hasCVE {
		t.Errorf("Expected CVE finding, got %v", findings)
	}
}

func TestRecordFailure(t *testing.T) {
	sl := NewSkillLearner(t.TempDir())

	// Create a skill first
	sl.ObserveExecution("run_nuclei", nil, "WordPress found", true)
	sl.RecordFailure("run_nuclei", nil, "WordPress found")

	skills := sl.ListSkills()
	if len(skills) != 1 {
		t.Fatalf("Expected 1 skill, got %d", len(skills))
	}
	if skills[0].FailCount != 1 {
		t.Errorf("FailCount = %d, want 1", skills[0].FailCount)
	}
}
