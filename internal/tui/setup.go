package tui

import (
	"fmt"
	"os"

	"github.com/0xP4X/drogonclaw-go/internal/config"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// RunSetup launches the interactive configuration menu using huh.
func RunSetup(cfg *config.Manager) {
	var provider string
	var err error

	banner := `
  ██████╗  ██████╗  ██████╗  ██████╗  ██████╗ ███╗   ██╗ ██████╗██╗      █████╗ ██╗    ██╗
  ██╔══██╗██╔══██╗██╔═══██╗██╔════╝ ██╔═══██╗████╗  ██║██╔════╝██║     ██╔══██╗██║    ██║
  ██║  ██║██████╔╝██║   ██║██║  ███╗██║   ██║██╔██╗ ██║██║     ██║     ███████║██║ █╗ ██║
  ██║  ██║██╔══██╗██║   ██║██║   ██║██║   ██║██║╚██╗██║██║     ██║     ██╔══██║██║███╗██║
  ██████╔╝██║  ██║╚██████╔╝╚██████╔╝╚██████╔╝██║ ╚████║╚██████╗███████╗██║  ██║╚███╔███╔╝
  ╚═════╝ ╚═╝  ╚═╝ ╚═════╝  ╚═════╝  ╚═════╝ ╚═╝  ╚═══╝ ╚══════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝`
	_ = banner // Kept for compatibility with older terminal launchers.
	brand := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(ColorMuted)
	rule := lipgloss.NewStyle().Foreground(ColorDim)
	fmt.Println()
	fmt.Println(brand.Render("  DrogonClaw") + muted.Render("  /  workspace setup"))
	fmt.Println(muted.Render("  Configure a model provider and local execution preferences."))
	fmt.Println(rule.Render("  ───────────────────────────────────────────────────────────"))
	fmt.Println(muted.Render("  Your credentials are stored locally and are never shown again."))
	fmt.Println()

	var authorised bool
	err = huh.NewConfirm().
		Title("I will use DrogonClaw only for systems I am authorised to assess").
		Description("You remain responsible for confirming scope and permission before running any operation.").
		Value(&authorised).
		Affirmative("Continue").
		Negative("Exit setup").
		WithTheme(CustomHuhTheme()).
		Run()
	if err != nil || !authorised {
		fmt.Println(muted.Render("  Setup cancelled. No configuration was changed."))
		return
	}

	err = huh.NewSelect[string]().
		Title("1 of 3 — Choose a model provider").
		Description("Select where DrogonClaw sends model requests. You can rerun setup at any time.").
		Options(
			huh.NewOption("OpenRouter — flexible hosted model access", "openrouter"),
			huh.NewOption("NVIDIA NIM — hosted NVIDIA inference", "nvidia"),
			huh.NewOption("OpenAI — hosted OpenAI models", "openai"),
			huh.NewOption("Google Gemini — hosted Gemini models", "gemini"),
			huh.NewOption("Ollama — local model runtime", "ollama"),
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
		modelOptions = []huh.Option[string]{
			huh.NewOption("Meta: Llama 3.1 405B Instruct", "meta-llama/llama-3.1-405b-instruct"),
			huh.NewOption("Meta: Llama 3.1 70B Instruct", "meta-llama/llama-3.1-70b-instruct"),
			huh.NewOption("Mistral: Mixtral 8x22B Instruct", "mistralai/mixtral-8x22b-instruct"),
			huh.NewOption("OpenAI: GPT-4o", "openai/gpt-4o"),
			huh.NewOption("Google: Gemini 2.5 Pro", "google/gemini-2.5-pro"),
		}
	case "nvidia":
		modelOptions = []huh.Option[string]{
			huh.NewOption("Nemotron 3 Ultra 550B (Best)", "nvidia/nemotron-3-ultra-550b-a55b"),
			huh.NewOption("Qwen 3.5: 397B Instruct (Fast)", "qwen/qwen3.5-397b-a17b"),
			huh.NewOption("DeepSeek: DeepSeek V4 Pro", "deepseek-ai/deepseek-v4-pro"),
			huh.NewOption("Llama 3.1 Nemotron 70B Instruct", "nvidia/llama-3.1-nemotron-70b-instruct"),
			huh.NewOption("Meta: Llama 3.1 405B Instruct", "meta/llama-3.1-405b-instruct"),
		}
	case "openai":
		modelOptions = []huh.Option[string]{
			huh.NewOption("gpt-4o (Best overall)", "gpt-4o"),
			huh.NewOption("gpt-4o-mini (Faster & cheaper)", "gpt-4o-mini"),
		}
	case "gemini":
		modelOptions = []huh.Option[string]{
			huh.NewOption("Gemini 2.5 Pro (Best performance)", "gemini-2.5-pro"),
			huh.NewOption("Gemini 2.5 Flash (Fastest)", "gemini-2.5-flash"),
		}
	case "ollama":
		modelOptions = []huh.Option[string]{
			huh.NewOption("Llama 3.1 8B (Fast/Local)", "llama3.1"),
			huh.NewOption("Llama 3.1 70B (Requires 40GB VRAM)", "llama3.1:70b"),
			huh.NewOption("Mistral (Fast)", "mistral"),
			huh.NewOption("Custom/Other (Type manually in config later)", "llama3.1"),
		}
	}

	var ollamaURL string

	if provider == "ollama" {
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("2 of 3 — Ollama server address").
					Value(&ollamaURL).
					Placeholder("http://localhost:11434"),
				huh.NewSelect[string]().
					Title("Select a model").
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
					Title(fmt.Sprintf("2 of 3 — %s API key", provider)).
					Description("Stored locally in DrogonClaw configuration.").
					EchoMode(huh.EchoModePassword).
					Value(&apiKey),
				huh.NewSelect[string]().
					Title("Select a model").
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

	var enableTelegram bool
	err = huh.NewConfirm().
		Title("3 of 3 — Enable the optional Telegram gateway?").
		Description("This lets you receive and send approved workspace requests from Telegram.").
		Value(&enableTelegram).
		WithTheme(CustomHuhTheme()).
		Run()

	if err == nil && enableTelegram {
		var tgToken string
		var tgChatID string

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Telegram bot token").
					Description("Get this from @BotFather").
					Value(&tgToken),
				huh.NewInput().
					Title("Allowed Telegram chat ID").
					Description("Only this chat will be allowed to use the gateway.").
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
	fmt.Println(lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render("  Setup complete."))
	fmt.Println(muted.Render("  Configuration saved to ~/.drogonclaw/config.json. Start a session with drogonclaw."))
}
