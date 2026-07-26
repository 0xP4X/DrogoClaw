package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/openai/openai-go"
)

// ReportGenerator uses the LLM to draft professional penetration test reports.
type ReportGenerator struct {
	provider LLMProvider
	graph    *memory.Graph
}

func NewReportGenerator(provider LLMProvider, graph *memory.Graph) *ReportGenerator {
	return &ReportGenerator{
		provider: provider,
		graph:    graph,
	}
}

const reportPrompt = `You are an elite penetration testing report generator.
I will provide you with a raw JSON dump of the DrogonClaw Memory Graph, which contains the assets, open ports, and vulnerabilities discovered during an autonomous mission.

Write a highly professional, compliance-ready penetration test report in Markdown.
It must include:
1. Executive Summary
2. Discovered Assets & Attack Surface
3. Detailed Vulnerability Findings (with estimated CVSS scores)
4. Remediation Steps

Here is the raw memory graph:
%s`

// GenerateMarkdownReport queries the LLM to produce a final markdown report.
func (r *ReportGenerator) GenerateMarkdownReport(ctx context.Context) (string, error) {
	graphJSON := r.graph.GetFullJSON()
	if len(graphJSON) < 50 {
		return "", fmt.Errorf("memory graph is empty. run a pentest mission before generating a report")
	}

	sysPrompt := fmt.Sprintf(reportPrompt, graphJSON)

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You write professional pentest reports."),
		openai.UserMessage(sysPrompt),
	}

	content, err := r.provider.CompleteText(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate report: %w", err)
	}

	_ = os.MkdirAll("reports", 0755)
	filename := filepath.Join("reports", fmt.Sprintf("drogonclaw_report_%d.md", time.Now().Unix()))

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to save report to disk: %w", err)
	}

	return filename, nil
}
