package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// providerInfo is a lightweight health snapshot for the dashboard.
type providerInfo struct {
	name      string
	configKey string
	hasKey    bool
	baseURL   string
	latency   string
	status    string
}

func (m *Model) handleProvidersCommand(_ string) (*Model, tea.Cmd) {
	// Build static snapshot — no network calls in the TUI thread.
	providers := []providerInfo{
		{
			name:      "openrouter",
			configKey: "OPENROUTER_API_KEY",
			hasKey:    m.cfg.GetString("OPENROUTER_API_KEY") != "",
			baseURL:   "https://openrouter.ai/api/v1",
		},
		{
			name:      "openai",
			configKey: "OPENAI_API_KEY",
			hasKey:    m.cfg.GetString("OPENAI_API_KEY") != "",
			baseURL:   "https://api.openai.com/v1",
		},
		{
			name:      "nvidia",
			configKey: "NVIDIA_API_KEY",
			hasKey:    m.cfg.GetString("NVIDIA_API_KEY") != "",
			baseURL:   "https://integrate.api.nvidia.com/v1",
		},
		{
			name:      "gemini",
			configKey: "GOOGLE_API_KEY",
			hasKey:    m.cfg.GetString("GOOGLE_API_KEY") != "",
			baseURL:   "https://generativelanguage.googleapis.com/v1beta/openai/",
		},
		{
			name:      "ollama",
			configKey: "OLLAMA_BASE_URL",
			hasKey:    true, // local, no key required
			baseURL:   m.cfg.GetString("OLLAMA_BASE_URL"),
		},
		{
			name:      "9router",
			configKey: "NINEROUTER_API_KEY",
			hasKey:    m.cfg.GetNineRouterAPIKey() != "",
			baseURL:   "https://api.9router.ai/v1",
		},
	}

	activeProvider := m.cfg.GetProvider()
	activeModel := m.cfg.GetModel()
	routerMode := m.cfg.GetRouterMode()

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(SectionHeaderStyle.Render("  PROVIDER HEALTH DASHBOARD") + "\n")
	sb.WriteString(SectionRuleStyle.Render("  "+strings.Repeat("─", 68)) + "\n\n")

	// Active provider banner
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", SidebarLabelStyle.Render("Active"),
		SidebarValueStyle.Render(fmt.Sprintf("%s / %s", activeProvider, truncate(activeModel, 32)))))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", SidebarLabelStyle.Render("Router"),
		SidebarValueStyle.Render(strings.ToUpper(routerMode))))
	sb.WriteString("\n")

	// Table header
	headerStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted).Bold(true)
	sb.WriteString(headerStyle.Render(fmt.Sprintf("  %-12s %-8s %-10s %s",
		"PROVIDER", "KEY", "STATUS", "ENDPOINT")) + "\n")
	sb.WriteString(SectionRuleStyle.Render("  "+strings.Repeat("─", 68)) + "\n")

	for _, p := range providers {
		keyBadge := StatusOffStyle.Render("○ none")
		if p.hasKey {
			keyBadge = StatusOnStyle.Render("● set")
		}
		if p.name == "ollama" {
			keyBadge = lipgloss.NewStyle().Foreground(m.theme.TextDim).Render("— local")
		}

		statusBadge := lipgloss.NewStyle().Foreground(m.theme.TextDim).Render("○ idle")
		if p.name == activeProvider {
			statusBadge = StatusOnStyle.Render("● active")
		} else if p.hasKey {
			statusBadge = lipgloss.NewStyle().Foreground(m.theme.Warning).Render("○ ready")
		}

		endpoint := truncate(p.baseURL, 36)
		if p.name == "ollama" && endpoint == "" {
			endpoint = "http://localhost:11434/v1"
		}

		nameCell := p.name
		if p.name == activeProvider {
			nameCell = lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render(p.name)
		} else {
			nameCell = lipgloss.NewStyle().Foreground(m.theme.Text).Render(p.name)
		}

		sb.WriteString(fmt.Sprintf("  %-12s %-8s %-10s %s\n",
			nameCell, keyBadge, statusBadge, lipgloss.NewStyle().Foreground(m.theme.TextDim).Render(endpoint)))
	}

	sb.WriteString("\n")
	sb.WriteString(HintDescStyle.Render("  Tip: configure keys via /setup or edit ~/.drogonclaw/config.json") + "\n")
	sb.WriteString(HintDescStyle.Render("  Switch provider: set AI_PROVIDER in config, then /setup or restart") + "\n")
	if m.tracker != nil {
		total := m.tracker.Total()
		cost := m.tracker.TotalCost()
		sb.WriteString(fmt.Sprintf("\n  %s  %d tokens  ·  $%.4f  ·  %s elapsed\n",
			SidebarLabelStyle.Render("Session usage:"),
			total.TotalTokens,
			cost,
			time.Since(m.startTime).Round(time.Second).String(),
		))
	}

	m.appendLine(sb.String())
	return m, nil
}
