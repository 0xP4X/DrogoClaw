package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/config"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

func stepIndicator(current, total int, label string) string {
	barWidth := 20
	completed := float64(current) / float64(total)
	filledWidth := int(completed * float64(barWidth))
	if filledWidth > barWidth {
		filledWidth = barWidth
	}

	var bar strings.Builder
	for i := 0; i < filledWidth; i++ {
		bar.WriteString(HeaderBrandStyle.Render("━"))
	}
	for i := filledWidth; i < barWidth; i++ {
		bar.WriteString(SectionRuleStyle.Render("─"))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s  %s  %s\n",
		HeaderBrandStyle.Render(fmt.Sprintf("[%d/%d]", current, total)),
		bar.String(),
		SidebarLabelStyle.Render(label),
	))
	return sb.String()
}

func sectionHeader(number, title string) string {
	return lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true).
		Render(fmt.Sprintf("  %s %s", number, title))
}

func RunSetup(cfg *config.Manager) {
	var provider string
	var err error

	brand := HeaderBrandStyle.Render
	muted := HintDescStyle.Render
	dim := SidebarLabelStyle.Render

	fmt.Println()
	fmt.Println(brand("  DrogonClaw Setup Wizard"))
	fmt.Println(muted("  Configure your local security environment and neural pathways."))
	fmt.Println(dim("  Credentials remain strictly local (~/.drogonclaw/config.json)."))
	fmt.Println()

	fmt.Println(stepIndicator(1, 4, "Authorisation Check"))
	fmt.Println()

	var authorised bool
	err = huh.NewConfirm().
		Title("I authorize DrogonClaw for explicitly scoped operations only").
		Description("You are strictly responsible for scope and compliance.").
		Value(&authorised).
		Affirmative("Acknowledge").
		Negative("Abort").
		WithTheme(CustomHuhTheme()).
		Run()
	if err != nil || !authorised {
		fmt.Println(muted("  Setup aborted."))
		return
	}

	fmt.Println()
	fmt.Println(stepIndicator(2, 4, "Select Neural Provider"))
	fmt.Println()

	err = huh.NewSelect[string]().
		Title("Select Intelligence Provider").
		Description("Choose where DrogonClaw delegates reasoning tasks.").
		Options(
			huh.NewOption("OpenRouter — Flexible multi-model gateway", "openrouter"),
			huh.NewOption("NVIDIA NIM — High-performance inference", "nvidia"),
			huh.NewOption("OpenAI — Direct API runtime", "openai"),
			huh.NewOption("Google Gemini — Enterprise reasoning core", "gemini"),
			huh.NewOption("Ollama — Autonomous offline runtime", "ollama"),
		).
		Value(&provider).
		WithTheme(CustomHuhTheme()).
		Run()

	if err != nil {
		fmt.Println("  [!] Setup cancelled.")
		os.Exit(1)
	}

	var apiKey string
	var model string
	var modelOptions []huh.Option[string]

	switch provider {
	case "openrouter":
		modelOptions = openRouterModelOptions()
	case "nvidia":
		modelOptions = []huh.Option[string]{
			huh.NewOption("Nemotron 3 Ultra 550B", "nvidia/nemotron-3-ultra-550b-a55b"),
			huh.NewOption("Qwen 3.5: 397B Instruct", "qwen/qwen3.5-397b-a17b"),
			huh.NewOption("DeepSeek: DeepSeek V4 Pro", "deepseek-ai/deepseek-v4-pro"),
			huh.NewOption("Llama 3.1 Nemotron 70B", "nvidia/llama-3.1-nemotron-70b-instruct"),
		}
	case "openai":
		modelOptions = []huh.Option[string]{
			huh.NewOption("gpt-4o (Recommended)", "gpt-4o"),
			huh.NewOption("gpt-4o-mini (Fast)", "gpt-4o-mini"),
		}
	case "gemini":
		modelOptions = []huh.Option[string]{
			huh.NewOption("Gemini 2.5 Pro", "gemini-2.5-pro"),
			huh.NewOption("Gemini 2.5 Flash", "gemini-2.5-flash"),
		}
	case "ollama":
		modelOptions = []huh.Option[string]{
			huh.NewOption("Llama 3.1 8B (Local)", "llama3.1"),
			huh.NewOption("Llama 3.1 70B (Local)", "llama3.1:70b"),
			huh.NewOption("Mistral (Local)", "mistral"),
		}
	}

	var ollamaURL string

	fmt.Println()
	fmt.Println(stepIndicator(3, 4, "Credentials & Model Selection"))
	fmt.Println()

	if provider == "ollama" {
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Ollama Endpoint").
					Description("Defaults to http://localhost:11434").
					Value(&ollamaURL).
					Placeholder("http://localhost:11434"),
				huh.NewSelect[string]().
					Title("Select Model").
					Options(modelOptions...).
					Value(&model),
			),
		).WithTheme(CustomHuhTheme()).Run()
		if ollamaURL == "" {
			ollamaURL = "http://localhost:11434"
		}
	} else {
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("%s API Key", strings.ToUpper(provider))).
					Description("Stored locally in encrypted configuration.").
					EchoMode(huh.EchoModePassword).
					Value(&apiKey),
				huh.NewSelect[string]().
					Title("Select Model Target").
					Options(modelOptions...).
					Value(&model),
			),
		).WithTheme(CustomHuhTheme()).Run()
	}

	if err != nil {
		fmt.Println("  [!] Setup cancelled.")
		os.Exit(1)
	}

	cfg.Set("AI_PROVIDER", provider)
	cfg.Set("AI_MODEL", model)

	switch provider {
	case "openai":
		cfg.Set("OPENAI_API_KEY", apiKey)
	case "nvidia":
		cfg.Set("NVIDIA_API_KEY", apiKey)
	case "gemini":
		cfg.Set("GOOGLE_API_KEY", apiKey)
	case "ollama":
		cfg.Set("OLLAMA_BASE_URL", ollamaURL)
	default:
		cfg.Set("OPENROUTER_API_KEY", apiKey)
	}

	fmt.Println()
	fmt.Println(stepIndicator(4, 4, "Remote C2 Gateway (Optional)"))
	fmt.Println()

	var enableTelegram bool
	err = huh.NewConfirm().
		Title("Enable Telegram C2 Gateway?").
		Description("Allows remote interaction via encrypted Telegram bot.").
		Value(&enableTelegram).
		WithTheme(CustomHuhTheme()).
		Run()

	if err == nil && enableTelegram {
		var tgToken string
		var tgChatID string

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Telegram Bot Token").
					Description("Provided by @BotFather").
					Value(&tgToken),
				huh.NewInput().
					Title("Authorized Chat ID").
					Description("Strict whitelist for command execution").
					Value(&tgChatID),
			),
		).WithTheme(CustomHuhTheme()).Run()

		if err == nil && tgToken != "" && tgChatID != "" {
			cfg.Set("TELEGRAM_TOKEN", tgToken)
			cfg.Set("TELEGRAM_CHAT_ID", tgChatID)
		} else {
			fmt.Println("  [!] Telegram setup skipped.")
		}
	}

	fmt.Println()
	fmt.Println(SectionRuleStyle.Render("  ────────────────────────────────────────────────────────────────"))
	fmt.Println()
	fmt.Println(ToolOutputSuccessStyle.Render("  ✔ Setup Complete."))
	fmt.Println(muted("  Configuration saved successfully."))
	fmt.Println(muted("  Launch workspace with:  ./drogonclaw"))
	fmt.Println()
}

// openRouterModelOptions builds the model picker from the live OpenRouter
// catalog. On any failure (offline / API down) it falls back to a small curated
// list so setup is never blocked.
func openRouterModelOptions() []huh.Option[string] {
	fallback := []huh.Option[string]{
		huh.NewOption("Meta: Llama 3.1 405B Instruct", "meta-llama/llama-3.1-405b-instruct"),
		huh.NewOption("Meta: Llama 3.1 70B Instruct", "meta-llama/llama-3.1-70b-instruct"),
		huh.NewOption("Mistral: Mixtral 8x22B Instruct", "mistralai/mixtral-8x22b-instruct"),
		huh.NewOption("OpenAI: GPT-4o", "openai/gpt-4o"),
		huh.NewOption("Google: Gemini 2.5 Pro", "google/gemini-2.5-pro"),
	}

	ids, err := config.OpenRouterModels()
	if err != nil || len(ids) == 0 {
		return fallback
	}

	const maxOptions = 300
	if len(ids) > maxOptions {
		ids = ids[:maxOptions]
	}
	opts := make([]huh.Option[string], 0, len(ids))
	for _, id := range ids {
		opts = append(opts, huh.NewOption(id, id))
	}
	return opts
}
