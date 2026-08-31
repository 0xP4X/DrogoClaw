package tui

import (
	"fmt"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

// Add /router command handler
func (m *Model) handleRouterCommand(args string) (*Model, tea.Cmd) {
	if args == "" || args == "status" {
		return m.showRouterStatus()
	}

	switch strings.ToLower(args) {
	case "auto":
		m.cfg.Set("ROUTER_MODE", "auto")
		m.appendLine(ToolDoneStyle.Render("  [✓] Routing mode: AUTO (9router.ai with local fallback)"))
		m.appendLine(HintDescStyle.Render("      Intelligent routing enabled. Will use 9router.ai service with automatic"))
		m.appendLine(HintDescStyle.Render("      fallback to local rules if unavailable."))
	case "local":
		m.cfg.Set("ROUTER_MODE", "local")
		m.appendLine(ToolDoneStyle.Render("  [✓] Routing mode: LOCAL"))
		m.appendLine(HintDescStyle.Render("      Using built-in routing rules. No external API needed."))
	case "9router":
		apiKey := m.cfg.GetNineRouterAPIKey()
		if apiKey == "" {
			m.appendLine(ErrorStyle.Render("  [✗] 9router.ai API key not configured."))
			m.appendLine(InfoStyle.Render("      Get a free key at https://9router.ai"))
			m.appendLine(HintDescStyle.Render("      Then set it with: /set NINEROUTER_API_KEY <your-key>"))
			return m, nil
		}
		m.cfg.Set("ROUTER_MODE", "9router")
		m.appendLine(ToolDoneStyle.Render("  [✓] Routing mode: 9ROUTER"))
		m.appendLine(HintDescStyle.Render("      Using 9router.ai service for intelligent model routing."))
	case "off":
		m.cfg.Set("ROUTER_MODE", "off")
		m.appendLine(WarningStyle.Render("  [○] Routing disabled"))
		m.appendLine(HintDescStyle.Render("      Using configured provider without intelligent routing."))
	default:
		m.appendLine(ErrorStyle.Render(fmt.Sprintf("  [✗] Unknown routing mode: '%s'", args)))
		m.appendLine(HintDescStyle.Render("      Available: auto (recommended), local, 9router, off, status"))
		m.appendLine(InfoStyle.Render("      Example: /router auto"))
	}

	return m, nil
}

func (m *Model) showRouterStatus() (*Model, tea.Cmd) {
	mode := m.cfg.GetRouterMode()
	provider := m.cfg.GetProvider()
	model := m.cfg.GetModel()

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(SectionHeaderStyle.Render("  INTELLIGENT ROUTING STATUS") + "\n")
	sb.WriteString(SectionRuleStyle.Render("  " + strings.Repeat("─", 48)) + "\n\n")

	// Current mode
	modeStatus := StatusOffStyle.Render("○ OFF")
	switch mode {
	case "auto":
		modeStatus = StatusOnStyle.Render("● AUTO")
	case "local":
		modeStatus = StatusOnStyle.Render("● LOCAL")
	case "9router":
		modeStatus = StatusOnStyle.Render("● 9ROUTER")
	}
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", SidebarLabelStyle.Render("Mode"), modeStatus))

	// Current provider
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", SidebarLabelStyle.Render("Provider"), SidebarValueStyle.Render(provider)))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", SidebarLabelStyle.Render("Model"), SidebarValueStyle.Render(truncate(model, 30))))

	// Routing stats (placeholder - will be populated when router is wired)
	if mode != "off" {
		sb.WriteString("\n")
		sb.WriteString(SectionHeaderStyle.Render("  ROUTING RULES") + "\n")
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", SidebarLabelStyle.Render("Chat/Simple"), HintDescStyle.Render("Fast models ($0.10/1M)")))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", SidebarLabelStyle.Render("Recon"), HintDescStyle.Render("Fast models ($0.15/1M)")))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", SidebarLabelStyle.Render("Planning"), HintDescStyle.Render("Medium models ($3.00/1M)")))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", SidebarLabelStyle.Render("Exploitation"), HintDescStyle.Render("Premium models ($15.00/1M)")))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", SidebarLabelStyle.Render("Reporting"), HintDescStyle.Render("Writing models ($5.00/1M)")))
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", SidebarLabelStyle.Render("Analysis"), HintDescStyle.Render("High-quality ($10.00/1M)")))
	}

	sb.WriteString("\n")
	sb.WriteString(HintDescStyle.Render("  Commands: /router auto | local | 9router | off") + "\n")

	m.appendLine(sb.String())
	return m, nil
}

// Add to renderConfigSummary in commands.go
func renderRouterConfig(cfg *config.Manager) string {
	mode := cfg.GetRouterMode()
	apiKey := cfg.GetNineRouterAPIKey()

	var sb strings.Builder
	sb.WriteString(SectionHeaderStyle.Render("  ROUTING CONFIGURATION") + "\n")

	modeStatus := "OFF"
	if mode != "off" {
		modeStatus = strings.ToUpper(mode)
	}
	sb.WriteString(fmt.Sprintf("  %-20s %s\n", SidebarLabelStyle.Render("Routing Mode"), SidebarValueStyle.Render(modeStatus)))

	if mode == "9router" || mode == "auto" {
		keyStatus := "Not configured"
		if apiKey != "" {
			if len(apiKey) <= 4 {
				keyStatus = "● set"
			} else {
				keyStatus = "● ..." + apiKey[len(apiKey)-4:]
			}
		}
		sb.WriteString(fmt.Sprintf("  %-20s %s\n", SidebarLabelStyle.Render("9Router API Key"), SidebarValueStyle.Render(keyStatus)))
	}

	return sb.String()
}
