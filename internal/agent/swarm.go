package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/openai/openai-go"
)

type SwarmCommander struct {
	provider  *Provider
	tools     *ToolRegistry
	sysPrompt string
	sessionID string
	graph     *memory.Graph
}

func NewSwarmCommander(provider *Provider, tools *ToolRegistry, sysPrompt, sessionID string, graph *memory.Graph) *SwarmCommander {
	return &SwarmCommander{
		provider:  provider,
		tools:     tools,
		sysPrompt: sysPrompt,
		sessionID: sessionID,
		graph:     graph,
	}
}

func (s *SwarmCommander) ExecuteSwarm(ctx context.Context, mission string, events chan<- Event) (string, error) {
	events <- Event{Type: EvStatus, Content: "Commander analyzing mission for swarm distribution..."}
	
	// Fast generation of tasks without parsing JSON structure
	tasks, err := s.splitMission(ctx, mission)
	if err != nil {
		events <- Event{Type: EvError, Content: fmt.Sprintf("Warning: could not intelligently split mission. Executing as a single vector: %v", err)}
		tasks = []string{mission}
	}

	events <- Event{Type: EvToken, Content: fmt.Sprintf("\n[Swarm Commander] Sliced mission into %d concurrent vectors:\n", len(tasks))}
	for i, t := range tasks {
		events <- Event{Type: EvToken, Content: fmt.Sprintf("  Vector %d: %s\n", i+1, t)}
	}

	var wg sync.WaitGroup
	results := make([]string, len(tasks))
	errs := make([]error, len(tasks))

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t string) {
			defer wg.Done()
			
			// We must drain the orchestrator's event channel or it blocks
			agentEvents := make(chan Event, 32)
			orch := NewOrchestrator(s.provider, s.tools, s.sysPrompt, fmt.Sprintf("%s-agent-%d", s.sessionID, idx), s.graph)
			
			var finalOutput string
			go func() {
				for e := range agentEvents {
					// Route the agent's progress back to the main UI stream with a prefix
					if e.Type == EvStatus {
						events <- Event{Type: EvStatus, Content: fmt.Sprintf("[Agent %d] %s", idx+1, e.Content)}
					}
					if e.Type == EvDone {
						finalOutput = e.Content
					}
				}
			}()

			errs[idx] = orch.Execute(ctx, t, agentEvents)
			results[idx] = fmt.Sprintf("### Report from Agent %d (Vector: %s)\n%s\n", idx+1, t, finalOutput)
		}(i, task)
	}

	wg.Wait()

	events <- Event{Type: EvStatus, Content: "Swarm operation concluded. Aggregating results..."}

	var sb strings.Builder
	for i, res := range results {
		if errs[i] != nil {
			sb.WriteString(fmt.Sprintf("### Report from Agent %d (FAILED: %v)\n", i+1, errs[i]))
		} else {
			sb.WriteString(res)
		}
		sb.WriteString("\n" + strings.Repeat("─", 60) + "\n\n")
	}

	return sb.String(), nil
}

func (s *SwarmCommander) splitMission(ctx context.Context, mission string) ([]string, error) {
	prompt := fmt.Sprintf(`You are the Swarm Commander for DrogonClaw.
Your job is to take a high-level mission and split it into 1-3 highly independent parallel tasks that can be executed concurrently by separate agents.
If the mission is simple, return just 1 task.
Return the tasks as a comma separated list. Do NOT use markdown. Do NOT use JSON. Just comma separated text.

Mission: %s`, mission)

	resp, err := s.provider.CompleteText(ctx, []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(prompt),
	})
	if err != nil {
		return nil, err
	}

	raw := strings.Split(resp, ",")
	var tasks []string
	for _, r := range raw {
		clean := strings.TrimSpace(r)
		if clean != "" {
			tasks = append(tasks, clean)
		}
	}
	
	if len(tasks) == 0 {
		return []string{mission}, nil
	}
	return tasks, nil
}
