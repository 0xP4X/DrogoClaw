package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/core"
	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/openai/openai-go"
)

// EventType classifies events emitted during agent execution.
type EventType string

const (
	EvThinking  EventType = "thinking"
	EvPlan      EventType = "plan"
	EvToolStart EventType = "tool_start"
	EvToolDone  EventType = "tool_done"
	EvToken     EventType = "token"
	EvDone      EventType = "done"
	EvError     EventType = "error"
	EvStatus    EventType = "status"
)

// Event is emitted to the TUI during agent execution.
type Event struct {
	Type      EventType
	Tool      string
	Args      string
	Result    string
	Content   string
	Plan      *core.MissionPlan
	StepIndex int
	StepTotal int
}

// Orchestrator is the DrogonClaw agent — a hand-rolled ReAct loop.
type Orchestrator struct {
	provider  *Provider
	tools     *ToolRegistry
	history   []openai.ChatCompletionMessageParamUnion
	sysPrompt string

	// Session state
	Autopilot bool
	Telemetry bool
	SessionID string

	// Subsystems
	missionPlanner *core.MissionPlanner
	actions        *memory.ActionJournal
}

const maxHistoryMessages = 24

// NewOrchestrator creates the agent core.
func NewOrchestrator(provider *Provider, tools *ToolRegistry, sysPrompt, sessionID string, graph *memory.Graph) *Orchestrator {
	return NewOrchestratorWithJournal(provider, tools, sysPrompt, sessionID, graph, memory.NewActionJournal(sessionID))
}

// NewOrchestratorWithJournal permits the primary UI to use a stable journal
// name across application restarts while worker agents keep isolated journals.
func NewOrchestratorWithJournal(provider *Provider, tools *ToolRegistry, sysPrompt, sessionID string, graph *memory.Graph, actions *memory.ActionJournal) *Orchestrator {
	return &Orchestrator{
		provider:       provider,
		tools:          tools,
		sysPrompt:      sysPrompt,
		SessionID:      sessionID,
		missionPlanner: core.NewMissionPlanner(provider, graph),
		actions:        actions,
	}
}

func (o *Orchestrator) GetProvider() *Provider {
	return o.provider
}

func (o *Orchestrator) GetTools() *ToolRegistry {
	return o.tools
}

// UpdateSystemPrompt hot-swaps the system prompt (e.g., after /persona or /stealth).
func (o *Orchestrator) UpdateSystemPrompt(prompt string) {
	o.sysPrompt = prompt
}

// NewSession wipes conversation history.
func (o *Orchestrator) NewSession() {
	o.history = nil
	if o.actions != nil {
		o.actions.Clear()
	}
}

// retainHistory keeps conversations useful while bounding prompt size and RAM
// use in long-running interactive sessions. Each exchange contributes a user
// and assistant message, so retaining an even number preserves turn pairs.
func (o *Orchestrator) retainHistory() {
	if len(o.history) > maxHistoryMessages {
		o.history = append([]openai.ChatCompletionMessageParamUnion(nil), o.history[len(o.history)-maxHistoryMessages:]...)
	}
}

func (o *Orchestrator) Recovery() *memory.ActionRecord {
	if o.actions == nil {
		return nil
	}
	return o.actions.Recovery()
}

// Resume re-enters the normal planner with the durable checkpoint as context.
// It never claims an interrupted external tool completed, preventing unsafe
// duplicate actions from being hidden from the operator.
func (o *Orchestrator) Resume(ctx context.Context, events chan<- Event) error {
	recovery := o.Recovery()
	if recovery == nil {
		return fmt.Errorf("there is no interrupted action to resume")
	}
	context := fmt.Sprintf("Resume the interrupted mission: %s\nCompleted checkpoints: %s\nInterrupted tool (its outcome is unknown; verify before retrying): %s\nContinue from the safest next step without repeating completed work.", recovery.Objective, strings.Join(recovery.CompletedSteps, " | "), recovery.CurrentTool)
	return o.Execute(ctx, context, events)
}

// Execute runs the ReAct loop for a user message, emitting events to the channel.
// The caller must close or drain the channel.
func (o *Orchestrator) Execute(ctx context.Context, userMsg string, events chan<- Event) error {
	defer close(events)
	if o.actions != nil {
		o.actions.Begin(userMsg)
	}

	// 1. Mission Planning Phase
	events <- Event{Type: EvThinking, Content: "Analyzing objective and generating execution plan..."}
	plan, err := o.missionPlanner.GeneratePlan(ctx, userMsg)
	if err == nil && plan != nil && plan.IsValidMission {
		events <- Event{Type: EvPlan, Plan: plan, Content: "Mission plan generated"}
		var planLines []string
		for i, step := range plan.Steps {
			planLines = append(planLines, fmt.Sprintf("%d. %s (Target: %s)", i+1, step.Action, step.TargetAssetID))
		}
		if o.actions != nil {
			o.actions.SetPlan(planLines)
		}
		events <- Event{Type: EvToken, Content: fmt.Sprintf("```\n[MISSION PLAN GENERATED]\nObjective: %s\nSteps:\n%s\n```\n\n", plan.Objective, strings.Join(planLines, "\n"))}
	} else {
		events <- Event{Type: EvStatus, Content: "Planning fallback engaged; proceeding with direct reasoning..."}
	}

	messages := BuildMessages(o.sysPrompt, o.history, userMsg)

	const maxIter = 25 // hard cap on tool call loops
	for i := 0; i < maxIter; i++ {
		events <- Event{Type: EvThinking, Content: fmt.Sprintf("Thinking... step %d/%d", i+1, maxIter)}

		resp, err := o.provider.Complete(ctx, messages, o.tools.Definitions())
		if err != nil {
			if o.actions != nil {
				o.actions.Finish("failed", err.Error())
			}
			events <- Event{Type: EvError, Content: fmt.Sprintf("LLM error: %v", err)}
			return err
		}

		// Append assistant message to history
		messages = append(messages, resp.Message.ToParam())

		// --- Fallback Parser for Raw JSON Tool Calls ---
		// Some uncensored models fail to use the native ToolCalls API and output JSON directly.
		if len(resp.ToolCalls) == 0 && (strings.Contains(resp.Message.Content, `{"name"`) || strings.Contains(resp.Message.Content, `{"tool"`)) {
			re := regexp.MustCompile(`(?s)\{"(?:name|tool)"\s*:\s*"[^"]+",\s*"arguments"\s*:\s*\{.*?\}\}`)
			matches := re.FindAllString(resp.Message.Content, -1)

			for i, match := range matches {
				var parsed struct {
					Name      string                 `json:"name"`
					Tool      string                 `json:"tool"`
					Arguments map[string]interface{} `json:"arguments"`
				}
				if err := json.Unmarshal([]byte(match), &parsed); err == nil {
					name := parsed.Name
					if name == "" {
						name = parsed.Tool
					}
					argsBytes, _ := json.Marshal(parsed.Arguments)
					tc := openai.ChatCompletionMessageToolCall{
						ID: fmt.Sprintf("call_fb_%d_%d", time.Now().Unix(), i),
						Function: openai.ChatCompletionMessageToolCallFunction{
							Name:      name,
							Arguments: string(argsBytes),
						},
					}
					resp.ToolCalls = append(resp.ToolCalls, tc)
				}
			}

			if len(resp.ToolCalls) > 0 {
				resp.Message.Content = strings.TrimSpace(re.ReplaceAllString(resp.Message.Content, ""))
			}
		}
		// ---------------------------------------------

		// No tool calls = final answer — stream it
		if len(resp.ToolCalls) == 0 {
			events <- Event{Type: EvStatus, Content: "Composing response..."}
			finalContent := resp.Message.Content
			events <- Event{Type: EvToken, Content: finalContent}
			events <- Event{Type: EvDone, Content: finalContent}

			// Save to history
			o.history = append(o.history,
				openai.UserMessage(userMsg),
				resp.Message.ToParam(),
			)
			o.retainHistory()
			if o.actions != nil {
				o.actions.Finish("completed", "")
			}
			return nil
		}

		// Execute each tool call
		for _, tc := range resp.ToolCalls {
			prettyArgs := formatArgs(tc.Function.Arguments)
			events <- Event{Type: EvToolStart, Tool: tc.Function.Name, Args: prettyArgs}
			if o.actions != nil {
				o.actions.ToolStarted(tc.Function.Name, prettyArgs)
			}

			result := o.tools.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if o.actions != nil {
				o.actions.ToolFinished(tc.Function.Name, result)
			}

			events <- Event{Type: EvToolDone, Tool: tc.Function.Name, Result: result}
			messages = append(messages, openai.ToolMessage(tc.ID, result))

			if strings.Contains(result, "[HitL_SUSPENDED]") {
				events <- Event{Type: EvStatus, Content: "Agent execution suspended. Awaiting human approval..."}

				// Block on the approval event rather than consuming a CPU core while waiting.
				if err := core.GlobalHitL.Wait(ctx); err != nil {
					if o.actions != nil {
						o.actions.Finish("interrupted", err.Error())
					}
					return err
				}

				// Once answered, inject the answer into the loop context as if the tool returned it
				ans := core.GlobalHitL.ConsumeAnswer()
				messages = append(messages, openai.UserMessage(fmt.Sprintf("Human-in-the-loop response: %s", ans)))
				events <- Event{Type: EvStatus, Content: "Approval received. Resuming execution..."}
			}
		}
	}

	err = fmt.Errorf("agent hit max iteration limit (%d) — possible infinite loop", maxIter)
	if o.actions != nil {
		o.actions.Finish("failed", err.Error())
	}
	events <- Event{Type: EvError, Content: err.Error()}
	return err
}

// ExecuteChat runs a lightweight no-tools completion for conversational messages.
func (o *Orchestrator) ExecuteChat(ctx context.Context, userMsg string, events chan<- Event) error {
	defer close(events)

	messages := BuildMessages(o.sysPrompt, o.history, userMsg)

	events <- Event{Type: EvStatus, Content: "Thinking..."}

	// For chat we want streaming — more responsive feel
	finalContent, err := o.provider.StreamFinal(ctx, messages, func(token string) {
		events <- Event{Type: EvToken, Content: token}
	})
	if err != nil {
		events <- Event{Type: EvError, Content: fmt.Sprintf("Chat error: %v", err)}
		return err
	}

	events <- Event{Type: EvDone, Content: finalContent}
	o.history = append(o.history,
		openai.UserMessage(userMsg),
		openai.AssistantMessage(finalContent),
	)
	o.retainHistory()
	return nil
}

// formatArgs pretty-prints JSON tool arguments for display.
func formatArgs(raw string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return raw
	}
	var parts []string
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := m[k]
		parts = append(parts, fmt.Sprintf("%s: %v", k, v))
	}
	return strings.Join(parts, " | ")
}
