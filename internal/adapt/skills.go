package adapt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// SkillEntry represents a dynamically created skill.
type SkillEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Command     string `json:"command"` // shell command template with {{param}} placeholders
}

var (
	mu            sync.RWMutex
	dynamicSkills []SkillEntry
	skillsPath    string
)

func init() {
	home, _ := os.UserHomeDir()
	skillsPath = filepath.Join(home, ".drogonclaw", "dynamic_skills.json")
	_ = loadSkills()
}

func loadSkills() error {
	data, err := os.ReadFile(skillsPath)
	if err != nil {
		return nil // first run
	}
	return json.Unmarshal(data, &dynamicSkills)
}

func saveSkills() error {
	data, err := json.MarshalIndent(dynamicSkills, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(skillsPath, data, 0600)
}

// CreateSkill adds a new dynamic skill to the runtime registry.
func CreateSkill(name, description, commandTemplate string) error {
	mu.Lock()
	defer mu.Unlock()

	name = strings.ToLower(strings.ReplaceAll(name, " ", "_"))

	// Check for duplicates
	for i, s := range dynamicSkills {
		if s.Name == name {
			dynamicSkills[i].Description = description
			dynamicSkills[i].Command = commandTemplate
			return saveSkills()
		}
	}

	dynamicSkills = append(dynamicSkills, SkillEntry{
		Name:        name,
		Description: description,
		Command:     commandTemplate,
	})
	return saveSkills()
}

// GetSkill returns a dynamic skill by name.
func GetSkill(name string) (SkillEntry, bool) {
	mu.RLock()
	defer mu.RUnlock()
	for _, s := range dynamicSkills {
		if s.Name == name {
			return s, true
		}
	}
	return SkillEntry{}, false
}

// ListSkills returns all dynamic skills.
func ListSkills() []SkillEntry {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]SkillEntry, len(dynamicSkills))
	copy(out, dynamicSkills)
	return out
}

// ExecuteSkill runs a dynamic skill with the provided parameters.
// It validates the rendered command against a dangerous-pattern denylist
// and returns the command string along with whether it passed validation.
func ExecuteSkill(name string, params map[string]string) (string, bool, error) {
	skill, ok := GetSkill(name)
	if !ok {
		return "", false, fmt.Errorf("dynamic skill '%s' not found", name)
	}

	cmd := skill.Command
	for k, v := range params {
		cmd = strings.ReplaceAll(cmd, "{{"+k+"}}", v)
	}

	// Validate the rendered command against dangerous patterns.
	if isDangerousCommand(cmd) {
		return cmd, false, fmt.Errorf("command blocked by safety policy: %s", name)
	}

	return cmd, true, nil
}

// dangerousPatterns is a denylist checked before dynamic skill execution.
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)rm\s+-[a-z]*r[a-z]*f`),
	regexp.MustCompile(`(?i)mkfs`),
	regexp.MustCompile(`(?i)dd\s+if=`),
	regexp.MustCompile(`(?i)shutdown`),
	regexp.MustCompile(`(?i)reboot`),
	regexp.MustCompile(`(?i)halt`),
}

func isDangerousCommand(cmd string) bool {
	for _, p := range dangerousPatterns {
		if p.MatchString(cmd) {
			return true
		}
	}
	return false
}
