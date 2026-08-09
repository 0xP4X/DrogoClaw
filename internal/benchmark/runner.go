package benchmark

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/agent"
	"github.com/0xP4X/drogonclaw-go/internal/config"
	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/0xP4X/drogonclaw-go/internal/opsec"
	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
	"github.com/0xP4X/drogonclaw-go/internal/skills"
)

// RunMode configures how the agent is built for benchmarking.
type RunMode struct {
	// ManifestPath defaults to "skills_manifest.json".
	ManifestPath string
	// ChallengeTimeout bounds a single challenge (hard ceiling).
	ChallengeTimeout time.Duration
}

// Run executes every challenge in the set and returns a Summary.
//
// It builds the agent components the same way the CLI does (skills manifest,
// sandbox, provider, tools, memory graph) and drives the orchestrator's
// headless Execute path. A challenge is marked solved when its flag pattern is
// observed in any streamed event before the run ends or times out.
func Run(ctx context.Context, set *Set, cfg *config.Manager, mode RunMode) (*Summary, error) {
	if mode.ManifestPath == "" {
		mode.ManifestPath = "skills_manifest.json"
	}
	if mode.ChallengeTimeout == 0 {
		mode.ChallengeTimeout = 15 * time.Minute
	}

	manifest, _ := skills.Load(mode.ManifestPath)

	summary := &Summary{
		Set:     set.Name,
		RunAt:   time.Now(),
		Total:   len(set.Challenges),
		ByClass: map[string]ClassStat{},
	}

	for _, ch := range set.Challenges {
		outcome := runChallenge(ctx, ch, cfg, manifest, mode)
		summary.Outcomes = append(summary.Outcomes, outcome)
		if outcome.Solved {
			summary.Solved++
		}
		bumpClass(summary.ByClass, ch.Class, outcome.Solved)
	}

	if summary.Total > 0 {
		summary.SuccessRate = float64(summary.Solved) / float64(summary.Total) * 100
	}
	summary.AvgDuration = avgDuration(summary.Outcomes)
	return summary, nil
}

// runChallenge executes a single challenge and reports its outcome.
func runChallenge(ctx context.Context, ch Challenge, cfg *config.Manager, manifest *skills.Manifest, mode RunMode) Outcome {
	start := time.Now()
	out := Outcome{ID: ch.ID, Class: ch.Class}

	// Spawn the target if it is a local command.
	handle, err := startTarget(ch)
	if err != nil {
		out.Err = err.Error()
		out.Duration = elapsed(start)
		return out
	}
	defer handle.Close()

	// Build agent components (same as the CLI startup path).
	sb, err := sandbox.New()
	if err != nil {
		out.Err = fmt.Sprintf("sandbox init failed: %v", err)
		out.Duration = elapsed(start)
		return out
	}
	_ = sb.Initialize(ctx, !cfg.IsSandboxEnabled())

	provider := agent.NewProvider(cfg)
	validator := agent.NewEvidenceValidator(provider)
	loot, _ := memory.NewLootDB()
	graph := memory.NewGraph(fmt.Sprintf("bench-%s", ch.ID))
	opsecMgr := opsec.NewManager()
	sysPrompt := agent.BuildSystemPrompt(graph, opsecMgr, "", "benchmark sandbox")

	tools := agent.NewToolRegistry(manifest, sb, validator, loot, cfg, graph, provider)
	sessionID := fmt.Sprintf("bench-%s", ch.ID)
	orch := agent.NewOrchestratorWithJournal(provider, tools, sysPrompt, sessionID, graph, memory.NewActionJournal(sessionID), 100)
	orch.Autopilot = true // headless: no operator to approve gates

	goal := buildGoal(ch)

	cctx, cancel := context.WithTimeout(ctx, mode.ChallengeTimeout)
	defer cancel()

	events := make(chan agent.Event, 64)
	go func() {
		_ = orch.Execute(cctx, goal, events)
	}()

	solved, flag, turns := scanEvents(events, ch)
	out.Solved = solved
	out.Flag = flag
	out.Turns = turns
	out.Duration = elapsed(start)
	if !solved && cctx.Err() != nil {
		out.Err = "challenge timed out or budget exceeded"
	}
	return out
}

// buildGoal produces the natural-language objective for the agent.
func buildGoal(ch Challenge) string {
	var b strings.Builder
	b.WriteString("You are in an authorized security benchmark. ")
	if ch.URL != "" {
		b.WriteString(fmt.Sprintf("Target: %s\n", ch.URL))
	}
	if ch.Path != "" {
		b.WriteString(fmt.Sprintf("Source under review: %s\n", ch.Path))
	}
	b.WriteString(ch.Goal)
	b.WriteString("\nWhen you have captured the flag, state it clearly. Do not stop until you have found and reported the flag or exhausted reasonable approaches.")
	return b.String()
}

// scanEvents consumes orchestrator events and detects the flag.
func scanEvents(events <-chan agent.Event, ch Challenge) (bool, string, int) {
	turns := 0
	for ev := range events {
		if ev.Type == agent.EvToolStart {
			turns++
		}
		text := ev.Content + "\n" + ev.Result + "\n" + ev.Tool + ":" + ev.Args
		if f := ch.findFlag(text); f != "" {
			return true, f, turns
		}
	}
	return false, "", turns
}

func elapsed(start time.Time) string {
	return time.Since(start).Round(time.Second).String()
}

func bumpClass(m map[string]ClassStat, class string, solved bool) {
	s := m[class]
	s.Total++
	if solved {
		s.Solved++
	}
	if s.Total > 0 {
		s.Rate = float64(s.Solved) / float64(s.Total) * 100
	}
	m[class] = s
}

func avgDuration(outcomes []Outcome) string {
	if len(outcomes) == 0 {
		return "0s"
	}
	var total time.Duration
	for _, o := range outcomes {
		d, err := time.ParseDuration(o.Duration)
		if err == nil {
			total += d
		}
	}
	return (total / time.Duration(len(outcomes))).Round(time.Second).String()
}
