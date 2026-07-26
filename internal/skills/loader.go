package skills

import (
	"encoding/json"
	"fmt"
	"os"
)

// ParameterDef describes one parameter of a skill tool.
type ParameterDef struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required,omitempty"`
}

// Skill is one entry in skills_manifest.json — mirrors a TypeScript DynamicStructuredTool.
type Skill struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Parameters  map[string]ParameterDef `json:"parameters"`
	ExecutesVia string                  `json:"executes_via"` // "docker_shell"
}

// Manifest holds all loaded skills.
type Manifest struct {
	Skills []Skill
}

// Load reads skills_manifest.json from the project root.
func Load(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("skills manifest not found at %s — run: node scripts/gen_skill_manifest.mjs", path)
	}
	var skills []Skill
	if err := json.Unmarshal(b, &skills); err != nil {
		return nil, fmt.Errorf("invalid skills manifest: %w", err)
	}
	return &Manifest{Skills: skills}, nil
}

// Count returns number of loaded skills.
func (m *Manifest) Count() int {
	return len(m.Skills)
}
