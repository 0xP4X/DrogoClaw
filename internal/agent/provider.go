package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/config"
	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/0xP4X/drogonclaw-go/internal/opsec"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Provider wraps the OpenAI-compatible client for all LLM providers.
type Provider struct {
	client *openai.Client
	model  string
}

// NewProvider constructs the LLM client, pointing to the correct provider base URL.
func NewProvider(cfg *config.Manager) *Provider {
	p := &Provider{model: cfg.GetModel()}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.GetAPIKey()),
		option.WithBaseURL(cfg.GetBaseURL()),
	}

	// OpenRouter requires these headers for ranking/analytics
	if cfg.GetProvider() == "openrouter" {
		opts = append(opts,
			option.WithHeader("HTTP-Referer", "https://drogonclaw.xyz"),
			option.WithHeader("X-Title", "DrogonClaw"),
		)
	}

	client := openai.NewClient(opts...)
	p.client = &client
	return p
}

// CompletionResponse from the LLM.
type CompletionResponse struct {
	Message   openai.ChatCompletionMessage
	ToolCalls []openai.ChatCompletionMessageToolCall
}

// Complete sends messages to the LLM and returns the response.
// Non-streaming: used during tool-calling loops where we need the full response.
func (p *Provider) Complete(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion, tools []openai.ChatCompletionToolParam) (*CompletionResponse, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		params := openai.ChatCompletionNewParams{
			Model:    p.model,
			Messages: messages,
		}
		if len(tools) > 0 {
			params.Tools = tools
		}

		resp, err := p.client.Chat.Completions.New(ctx, params)
		if err == nil {
			if len(resp.Choices) == 0 {
				return nil, fmt.Errorf("LLM returned no choices")
			}
			choice := resp.Choices[0]
			return &CompletionResponse{
				Message:   choice.Message,
				ToolCalls: choice.Message.ToolCalls,
			}, nil
		}

		lastErr = err
		if !isRetryableLLMError(err) || attempt == 2 {
			break
		}
		if err := sleepWithContext(ctx, retryBackoff(attempt)); err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("LLM completion failed after retries: %w", lastErr)
}

// CompleteText is a convenience function that sends messages and returns just the string content.
func (p *Provider) CompleteText(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) (string, error) {
	resp, err := p.Complete(ctx, messages, nil)
	if err != nil {
		return "", err
	}
	return resp.Message.Content, nil
}

// StreamFinal streams the final text response (no tools) token by token.
func (p *Provider) StreamFinal(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion, onToken func(string)) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		stream := p.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
			Model:    p.model,
			Messages: messages,
		})

		var sb strings.Builder
		for stream.Next() {
			chunk := stream.Current()
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta.Content
				if delta != "" {
					sb.WriteString(delta)
					onToken(delta)
				}
			}
		}
		if err := stream.Err(); err != nil {
			lastErr = err
			if !isRetryableLLMError(err) || attempt == 2 || sb.Len() > 0 {
				return sb.String(), fmt.Errorf("stream error: %w", err)
			}
			if err := sleepWithContext(ctx, retryBackoff(attempt)); err != nil {
				return sb.String(), err
			}
			continue
		}
		return sb.String(), nil
	}
	return "", fmt.Errorf("stream error after retries: %w", lastErr)
}

// Ping tests connectivity to the LLM provider.
func (p *Provider) Ping(ctx context.Context) error {
	// Ping the models endpoint rather than generating a completion
	// This avoids 504 timeouts on cold models (e.g. NVIDIA NIM or Ollama)
	_, err := p.client.Models.List(ctx)
	if err != nil {
		// Fallback to chat completion ping if the provider doesn't support /models
		_, fallbackErr := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model: p.model,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("ping"),
			},
			MaxTokens: openai.Int(1),
		})
		return fallbackErr
	}
	return nil
}

// BuildMessages constructs the initial message array from history + new user message.
func BuildMessages(systemPrompt string, history []openai.ChatCompletionMessageParamUnion, userMsg string) []openai.ChatCompletionMessageParamUnion {
	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}
	msgs = append(msgs, history...)
	msgs = append(msgs, openai.UserMessage(userMsg))
	return msgs
}

// IsChatOnly returns true if the message is conversational and needs no tools.
func IsChatOnly(msg string, graph *memory.Graph, opsecMgr *opsec.Manager) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))

	// Mission keywords that always trigger the full agent
	missionKeywords := []string{
		"scan", "hack", "exploit", "fuzz", "enumerate", "brute", "attack",
		"recon", "nmap", "sqlmap", "metasploit", "osint", "whois",
		"shodan", "nuclei", "gobuster", "ffuf", "payload", "shell",
		"reverse", "privesc", "ctf", "target", "pentest", "vuln",
		"crack", "hash", "password", "bypass", "inject", "rce", "lfi",
		"ssrf", "xss", "sqli", "directory", "subdomain", "port",
	}

	for _, kw := range missionKeywords {
		if strings.Contains(lower, kw) {
			return false
		}
	}

	// IP/URL pattern suggests a target
	if containsTarget(lower) {
		return false
	}

	return true
}

func containsTarget(s string) bool {
	// Very simple heuristic — proper regex would be overkill here
	return strings.Contains(s, "http://") ||
		strings.Contains(s, "https://") ||
		strings.Contains(s, ".htb") ||
		strings.Contains(s, ".thm") ||
		countDots(s) >= 3 // IPv4-like
}

func retryBackoff(attempt int) time.Duration {
	switch attempt {
	case 0:
		return 350 * time.Millisecond
	case 1:
		return 900 * time.Millisecond
	default:
		return 1500 * time.Millisecond
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func isRetryableLLMError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout"),
		strings.Contains(msg, "temporarily"),
		strings.Contains(msg, "try again"),
		strings.Contains(msg, "eof"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "broken pipe"),
		strings.Contains(msg, "503"),
		strings.Contains(msg, "502"),
		strings.Contains(msg, "504"):
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}
	if _, ok := err.(*url.Error); ok {
		return true
	}
	return false
}

func countDots(s string) int {
	return strings.Count(s, ".")
}
