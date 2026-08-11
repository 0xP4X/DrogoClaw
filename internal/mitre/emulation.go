package mitre

import (
	"fmt"
	"sort"
	"strings"
)

// EmulationStep is one ordered step in an ATT&CK emulation plan.
type EmulationStep struct {
	Order         int
	Tactic        string
	TechniqueID   string
	TechniqueName string
	Tool          string
	Notes         string
}

// DefaultToolSet returns the tool/action names DrogonClaw already maps to ATT&CK
// (what the operator can emulate today without adding capability).
func DefaultToolSet() []string {
	tools := make([]string, 0, len(techniqueDB))
	for k := range techniqueDB {
		tools = append(tools, k)
	}
	sort.Strings(tools)
	return tools
}

// BuildEmulationPlan produces a kill-chain-ordered emulation plan from the
// available tools, reusing the embedded techniqueDB. Only techniques whose tool
// is in `available` are included, so the plan reflects real, present capability
// — promoting the static map into a Caldera-style planner without duplication.
func BuildEmulationPlan(available []string) []EmulationStep {
	have := make(map[string]bool, len(available))
	for _, t := range available {
		have[strings.ToLower(strings.TrimSpace(t))] = true
	}
	var steps []EmulationStep
	order := 0
	seen := map[string]bool{}
	for _, tactic := range TacticOrder {
		for tool, techniques := range techniqueDB {
			if !have[strings.ToLower(tool)] {
				continue
			}
			for _, tech := range techniques {
				if tech.Tactic != tactic {
					continue
				}
				key := tactic + "/" + tech.ID + "/" + tool
				if seen[key] {
					continue
				}
				seen[key] = true
				order++
				steps = append(steps, EmulationStep{
					Order:         order,
					Tactic:        tactic,
					TechniqueID:   tech.ID,
					TechniqueName: tech.Name,
					Tool:          tool,
					Notes:         tech.URL,
				})
			}
		}
	}
	return steps
}

// RenderEmulationPlan returns a readable, kill-chain-ordered plan.
func RenderEmulationPlan(steps []EmulationStep) string {
	if len(steps) == 0 {
		return "[ATT&CK] No emulation steps for the provided tool set."
	}
	var b strings.Builder
	b.WriteString("═══════════════════════════════════════════\n")
	b.WriteString(" MITRE ATT&CK® Emulation Plan\n")
	b.WriteString("═══════════════════════════════════════════\n\n")
	phase := 0
	cur := ""
	for _, s := range steps {
		if s.Tactic != cur {
			cur = s.Tactic
			phase++
			fmt.Fprintf(&b, "Phase %d — %s\n", phase, cur)
		}
		fmt.Fprintf(&b, "  [%s] %s  (tool: %s)\n", s.TechniqueID, s.TechniqueName, s.Tool)
	}
	b.WriteString("\n═══════════════════════════════════════════\n")
	return b.String()
}
