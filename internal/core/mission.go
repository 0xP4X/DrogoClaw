package core

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/openai/openai-go"
)

type MissionStep struct {
	ID              string `json:"id"`
	Action          string `json:"action"`
	TargetAssetID   string `json:"targetAssetId"`
	ExpectedOutcome string `json:"expectedOutcome"`
	Status          string `json:"status"`
}

type MissionPlan struct {
	IsValidMission bool          `json:"isValidMission"`
	Objective      string        `json:"objective"`
	Steps          []MissionStep `json:"steps"`
}

type LLMProvider interface {
	CompleteText(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) (string, error)
}

// MissionPlanner parses high-level objectives into actionable ReAct steps.
type MissionPlanner struct {
	provider LLMProvider
	graph    *memory.Graph
}

func NewMissionPlanner(provider LLMProvider, graph *memory.Graph) *MissionPlanner {
	return &MissionPlanner{
		provider: provider,
		graph:    graph,
	}
}

const plannerPromptTemplate = `You are the central intelligence core of an autonomous Offensive Security framework.
You translate high-level objectives from the operator, **{{OPERATOR_NAME}}**, into an actionable sequence of security analysis steps.

EFFICIENCY IS PARAMOUNT:
- If the input is a short greeting (hi, hello, hey), be extremely concise.
- If the user's input is a direct security objective (e.g. "scan this IP", "fuzz this directory"), output a technical execution plan:
  1. Set "isValidMission" to true.
  2. Break the plan down into discrete steps.

- If the user asks a general question or simply chats:
  1. Set "isValidMission" to false.
  2. Provide a highly intelligent conversational response in the "objective" field.

Return strictly JSON matching this schema:
{
  "isValidMission": true, 
  "objective": "Plan or response",
  "steps": [
    {
      "id": "step-1",
      "action": "Description of the tool or action to execute",
      "targetAssetId": "The ID of the asset from the graph",
      "expectedOutcome": "What data we expect to capture",
      "status": "PENDING"
    }
  ]
}`

// GeneratePlan queries the LLM to structure the mission.
func (m *MissionPlanner) GeneratePlan(ctx context.Context, objective string) (*MissionPlan, error) {
	opName := "zero"
	if p := m.graph.GetOperatorProfile(); p != nil && p.Name != "" {
		opName = p.Name
	}

	sysPrompt := strings.ReplaceAll(plannerPromptTemplate, "{{OPERATOR_NAME}}", opName)
	
	// Compress graph (using JSON for now)
	graphState := m.graph.GetFullJSON()
	if len(graphState) > 2000 {
		graphState = graphState[:2000] + "... (truncated)"
	}

	prompt := fmt.Sprintf("Current Intelligence Context:\n%s\n\nUser Objective: %s", graphState, objective)

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(sysPrompt),
		openai.UserMessage(prompt),
	}

	content, err := m.provider.CompleteText(ctx, messages)
	if err != nil {
		return m.fallbackPlan(objective), nil
	}

	re := regexp.MustCompile(`(?s)\{.*\}`)
	match := re.FindString(content)
	if match != "" {
		content = match
	}

	var plan MissionPlan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return m.fallbackPlan(objective), nil
	}

	return &plan, nil
}

func (m *MissionPlanner) fallbackPlan(objective string) *MissionPlan {
	return &MissionPlan{
		IsValidMission: false,
		Objective:      objective,
		Steps: []MissionStep{
			{
				ID:              "fallback-1",
				Action:          "Execute manual agent evaluation",
				TargetAssetID:   "unknown",
				ExpectedOutcome: "Evaluate the target manually",
				Status:          "PENDING",
			},
		},
	}
}
