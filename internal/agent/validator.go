package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
)

type ValidationResult struct {
	IsValid           bool     `json:"isValid"`
	ConfidenceScore   int      `json:"confidenceScore"`
	Reasoning         string   `json:"reasoning"`
	ExtractedEntities struct {
		Assets          []string `json:"assets,omitempty"`
		Ports           []int    `json:"ports,omitempty"`
		Vulnerabilities []string `json:"vulnerabilities,omitempty"`
	} `json:"extractedEntities"`
}

type EvidenceValidator struct {
	provider *Provider
}

func NewEvidenceValidator(provider *Provider) *EvidenceValidator {
	return &EvidenceValidator{provider: provider}
}

const validatorPrompt = `You are the DrogonClaw Evidence Validator — a critical component of an AI-driven offensive security OS.
Your job is to review raw tool outputs (from nmap, curl, sqlmap, etc.) alongside the agent's claim about what that output means.
You must prevent hallucinations. If the agent claims a vulnerability exists but the raw tool output does NOT explicitly prove it, you must REJECT it.

Analyze the provided data and return a JSON object with the following schema:
{
  "isValid": boolean, // True only if the evidence directly supports the claim
  "confidenceScore": number, // 0-100. 100 = Absolute mathematical proof. 0 = Pure hallucination.
  "reasoning": string, // Briefly explain why you accepted or rejected the claim based strictly on the evidence
  "extractedEntities": { // Extract verified entities to populate the Intelligence Graph
    "assets": string[], // e.g., ["10.10.10.1", "target.com"]
    "ports": number[], // e.g., [80, 443, 22]
    "vulnerabilities": string[] // e.g., ["CVE-2021-44228", "SQL Injection in /login"]
  }
}

Respond ONLY with valid JSON.`

func (v *EvidenceValidator) Validate(ctx context.Context, toolName, rawOutput, agentClaim string) (*ValidationResult, error) {
	prompt := fmt.Sprintf("Tool Executed: %s\n\nRaw Output:\n```\n%.5000s\n```\n\nAgent's Claim/Finding: %s", toolName, rawOutput, agentClaim)

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(validatorPrompt),
		openai.UserMessage(prompt),
	}

	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    v.provider.model,
	}

	resp, err := v.provider.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("llm error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned from validator")
	}

	content := resp.Choices[0].Message.Content
	
	var res ValidationResult
	if err := json.Unmarshal([]byte(content), &res); err != nil {
		return nil, fmt.Errorf("failed to parse validator JSON: %w", err)
	}

	return &res, nil
}
