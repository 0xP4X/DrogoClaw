package tui

import (
	"fmt"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/config"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Provider identifiers shared by the wizard, the config keys they map to, and
// the model pickers.
const (
	providerOpenRouter = "openrouter"
	providerNVIDIA     = "nvidia"
	providerOpenAI     = "openai"
	providerGemini     = "gemini"
	providerOllama     = "ollama"
	provider9Router    = "9router"
)

// Wizard-reported states used in the configuration summary and key pickers.
const (
	stateSet    = "set"
	stateUnset  = "not set"
	stateTelOff = "disabled"
)

// Menu actions selected in the main setup loop.
const (
	actDone      = "done"
	actProvider  = "provider"
	actTelegram  = "telegram"
	actSecondary = "secondary"
	actIdentity  = "identity"
	actReset     = "reset"
)

// configReader is the subset of config.Manager the setup wizard and summary
// renderer depend on. Using an interface keeps the interactive wizard pure and
// lets the helpers be unit-tested against a fake reader instead of reading or
// writing the operator's real ~/.drogonclaw/config.json.
type configReader interface {
	GetProvider() string
	GetModel() string
	GetAPIKey() string
	GetBaseURL() string
	GetOperatorName() string
	GetAgentName() string
	GetString(string) string
}

// Setup-screen styling: modern dark-theme chrome that mirrors the TUI's own
// rounded-border / accent palette instead of flat text.
var (
	setupTitleStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)
	setupSubtitleStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)
	setupTagStyle = lipgloss.NewStyle().
			Background(ColorAccent).
			Foreground(ColorBg).
			Bold(true).
			Padding(0, 1)
	setupLabelStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)
	setupValueStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)
	setupPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 2).
			Background(ColorBgPanel)
	setupBannerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorAccent).
				Padding(1, 2).
				Background(ColorBgPanel)
)

// setupSection renders a modern section header: a filled accent tag plus an
// uppercase title and a subtle rule beneath.
func setupSection(step, title string) string {
	tag := setupTagStyle.Render(" " + step + " ")
	label := setupTitleStyle.Render(strings.ToUpper(title))
	return "\n  " + tag + "  " + label + "\n  " + SectionRuleStyle.Render(strings.Repeat("─", 52)) + "\n"
}

// setupBanner renders the wizard wordmark inside a bordered panel.
func setupBanner() string {
	title := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true).
		Render("🐉 DrogonClaw")
	subtitle := setupSubtitleStyle.Render("autonomous offensive security agent · setup & configuration")
	return "\n" + setupBannerStyle.Render(title+"\n\n"+subtitle) + "\n"
}

// setupMutedString renders dim secondary text.
func setupMutedString(s string) string { return setupSubtitleStyle.Render(s) }

// setupOnString renders a green "on" state for a stored credential.
func setupOnString(s string) string {
	return lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render(s)
}

// renderConfigSummary renders the "Current Configuration" pane: provider,
// model, the active API key, Telegram gateway state, the secondary keys in a
// two-column grid, and operator/agent identity. Secrets are only ever masked.
func renderConfigSummary(r configReader) string {
	label := func(s string) string {
		return lipgloss.NewStyle().Width(12).Render(setupLabelStyle.Render(s))
	}
	rendered := setupValueStyle.Render

	row := func(k, v string) string {
		return "  " + label(k) + rendered(v)
	}

	var sb strings.Builder
	sb.WriteString(row("Provider", providerLabel(r.GetProvider())))
	sb.WriteString("\n")
	sb.WriteString(row("Model", r.GetModel()))
	sb.WriteString("\n")

	if strings.EqualFold(r.GetProvider(), providerOllama) {
		sb.WriteString(row("Endpoint", r.GetString("OLLAMA_BASE_URL")))
		sb.WriteString("\n")
		sb.WriteString(row("API key", "local · no key required"))
	} else {
		keyVal := storedKeyStatus(r.GetAPIKey())
		if masked := maskSecret(r.GetAPIKey()); masked != "" {
			keyVal = masked
		}
		sb.WriteString(row("API key", keyVal))
	}
	sb.WriteString("\n")

	tg := telegramStatus(r)
	if tg == stateTelOff {
		sb.WriteString(row("Telegram", tg))
	} else {
		sb.WriteString(row("Telegram", setupOnString(tg)))
	}
	sb.WriteString("\n")

	sb.WriteString("  " + label("Secondary"))
	sb.WriteString("\n")
	const colW = 34
	var cells []string
	for _, k := range secondaryKeys {
		stored := r.GetString(k.configKey)
		if stored == "" {
			cells = append(cells, label(k.short)+setupMutedString("· ")+setupMutedString(stateUnset))
		} else {
			cells = append(cells, label(k.short)+setupOnString("● ")+setupMutedString(stateSet))
		}
	}
	for i := 0; i < len(cells); i += 2 {
		left := cells[i]
		right := ""
		if i+1 < len(cells) {
			right = cells[i+1]
		}
		sb.WriteString("    " + lipgloss.NewStyle().Width(colW).Render(left) + right + "\n")
	}

	operator := r.GetOperatorName()
	if operator == "" {
		operator = "—"
	}
	sb.WriteString(row("Operator", operator))
	sb.WriteString("\n")
	sb.WriteString(row("Agent", r.GetAgentName()))
	sb.WriteString("\n")
	return sb.String()
}

// secondaryKeyEntry describes one optional OSINT/provider key the wizard
// manages. The short label is used in the compact status grid.
type secondaryKeyEntry struct {
	configKey string
	short     string
	label     string
}

var secondaryKeys = []secondaryKeyEntry{
	{"GITHUB_TOKEN", "GitHub", "GitHub Token (authenticated code/search)"},
	{"SHODAN_API_KEY", "Shodan", "Shodan (exposed-asset intelligence)"},
	{"VIRUSTOTAL_API_KEY", "VirusTotal", "VirusTotal (hash/URL reputation)"},
	{"BRAVE_SEARCH_API_KEY", "Brave", "Brave Search (web dorking)"},
	{"HUNTER_IO_API_KEY", "Hunter.io", "Hunter.io (email discovery)"},
	{"EXA_API_KEY", "Exa", "Exa (AI web search)"},
}

// maskSecret renders a credential for display: empty stays empty, otherwise a
// fixed-length mask plus the last four characters so the operator can recognise
// which key is stored without exposing it.
func maskSecret(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return "••••"
	}
	return "••••••••" + v[len(v)-4:]
}

// providerLabel returns a human label for a provider id, falling back to the id
// itself for unknown or future providers.
func providerLabel(id string) string {
	switch strings.ToLower(id) {
	case providerOpenRouter:
		return "OpenRouter"
	case providerNVIDIA:
		return "NVIDIA NIM"
	case providerOpenAI:
		return "OpenAI"
	case providerGemini:
		return "Google Gemini"
	case providerOllama:
		return "Ollama (local)"
	case provider9Router:
		return "9Router.ai"
	}
	return id
}

// storedKeyStatus summarises whether a stored credential is present.
func storedKeyStatus(v string) string {
	if v == "" {
		return stateUnset
	}
	return stateSet
}

// telegramStatus summarises the Telegram gateway state without exposing the
// full bot token (only the trailing four characters are shown).
func telegramStatus(r configReader) string {
	token := r.GetString("TELEGRAM_TOKEN")
	chat := r.GetString("TELEGRAM_CHAT_ID")
	if token == "" && chat == "" {
		return stateTelOff
	}
	var parts []string
	if t := maskSecret(token); t != "" {
		parts = append(parts, "token "+t)
	}
	if chat != "" {
		parts = append(parts, "chat "+chat)
	}
	return strings.Join(parts, " · ")
}

// ollamaModelOptions discovers models installed in the local Ollama instance so
// the offline runtime only offers what is actually present. Falls back to a
// small static list when Ollama is unreachable (e.g. not yet started).
func ollamaModelOptions(baseURL string) []huh.Option[string] {
	fallback := []huh.Option[string]{
		huh.NewOption("Llama 3.1 8B (Local)", "llama3.1"),
		huh.NewOption("Llama 3.1 70B (Local)", "llama3.1:70b"),
		huh.NewOption("Mistral (Local)", "mistral"),
	}
	names, err := config.OllamaModels(baseURL)
	if err != nil || len(names) == 0 {
		return fallback
	}
	opts := make([]huh.Option[string], 0, len(names))
	for _, n := range names {
		opts = append(opts, huh.NewOption(n+" (Local)", n))
	}
	return opts
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

// providerModelOptions returns the model picker for a provider. Live catalogs
// (OpenRouter, NVIDIA, Ollama) are preferred and fall back to curated lists.
func providerModelOptions(provider string, r configReader) []huh.Option[string] {
	switch provider {
	case providerOpenRouter:
		return openRouterModelOptions()
	case providerNVIDIA:
		opts := []huh.Option[string]{
			huh.NewOption("Poolside Laguna XS 2.1 (verified)", "poolside/laguna-xs-2.1"),
			huh.NewOption("Nemotron 3 Ultra 550B", "nvidia/nemotron-3-ultra-550b-a55b"),
			huh.NewOption("Qwen 3.5: 397B Instruct", "qwen/qwen3.5-397b-a17b"),
			huh.NewOption("DeepSeek: DeepSeek V4 Pro", "deepseek-ai/deepseek-v4-pro"),
			huh.NewOption("Llama 3.1 Nemotron 70B", "nvidia/llama-3.1-nemotron-70b-instruct"),
		}
		if apiKey := r.GetAPIKey(); apiKey != "" {
			if names, err := config.NVIDIAModels(r.GetBaseURL(), apiKey); err == nil && len(names) > 0 {
				opts = make([]huh.Option[string], 0, len(names))
				for _, n := range names {
					opts = append(opts, huh.NewOption(n, n))
				}
			}
		}
		return opts
	case providerOpenAI:
		return []huh.Option[string]{
			huh.NewOption("gpt-4o (Recommended)", "gpt-4o"),
			huh.NewOption("gpt-4o-mini (Fast)", "gpt-4o-mini"),
		}
	case providerGemini:
		return []huh.Option[string]{
			huh.NewOption("Gemini 2.5 Pro", "gemini-2.5-pro"),
			huh.NewOption("Gemini 2.5 Flash", "gemini-2.5-flash"),
		}
	case providerOllama:
		return ollamaModelOptions(ollamaDefaultBaseURL(r))
	case provider9Router:
		return []huh.Option[string]{
			huh.NewOption("Auto (intelligent routing)", "auto"),
			huh.NewOption("Claude 3.5 Sonnet", "anthropic/claude-3.5-sonnet"),
			huh.NewOption("GPT-4o", "openai/gpt-4o"),
			huh.NewOption("Llama 3.1 70B", "meta-llama/llama-3.1-70b-instruct"),
		}
	}
	return nil
}

// ollamaDefaultBaseURL returns the stored Ollama endpoint or the well-known
// local default, trimming any trailing slash so "/v1" concatenation is clean.
func ollamaDefaultBaseURL(r configReader) string {
	base := r.GetString("OLLAMA_BASE_URL")
	if base == "" {
		return "http://localhost:11434"
	}
	return strings.TrimRight(base, "/")
}

func setupMuted() func(string) string {
	return func(s string) string { return HintDescStyle.Render(s) }
}

func setupSuccess(v string) {
	fmt.Println(ToolOutputSuccessStyle.Render("  " + v))
}

// RunSetup drives the interactive configuration wizard. Unlike a single linear
// pass, it is a controller: the current configuration is always shown first and
// every section is re-runnable, prefilled with stored values, and cancellable
// back to the menu. Credentials are only ever masked in the summary pane.
func RunSetup(cfg *config.Manager) {
	muted := HintDescStyle.Render
	dim := SidebarLabelStyle.Render

	fmt.Println(setupBanner())
	fmt.Println(muted("  Review and manage your local security environment."))
	fmt.Println(dim("  Credentials remain strictly local (~/.drogonclaw/config.json)."))

	if !cfg.IsVerified() {
		fmt.Println(setupSection("AUTH", "Authorisation Check"))
		var authorised bool
		err := huh.NewConfirm().
			Title("I authorize DrogonClaw for explicitly scoped operations only").
			Description("You are strictly responsible for scope and compliance.").
			Value(&authorised).
			Affirmative("Acknowledge").
			Negative("Abort").
			WithTheme(CustomHuhTheme()).
			Run()
		if err != nil || !authorised {
			fmt.Println(muted("  Setup aborted."))
			fmt.Println()
			return
		}
		cfg.SetVerified(true)
		setupSuccess("Authorised — returning users skip this step on later runs.")
		fmt.Println()
	}

	for {
		fmt.Println(setupSection("CONFIG", "Current Configuration"))
		fmt.Println(setupPanelStyle.Render(
			"\n  " + setupLabelStyle.Render("CURRENT CONFIGURATION") + "\n\n" +
				renderConfigSummary(cfg) + "  \n",
		))
		fmt.Println(setupSection("ACTION", "What would you like to do?"))
		fmt.Println()

		var action string
		err := huh.NewSelect[string]().
			Title("Setup Menu").
			Options(
				huh.NewOption("Finish — save & exit", actDone),
				huh.NewOption("Change AI provider or model", actProvider),
				huh.NewOption("Telegram C2 gateway — view / change / disable", actTelegram),
				huh.NewOption("Secondary API keys — view / add / clear", actSecondary),
				huh.NewOption("Operator & agent identity", actIdentity),
				huh.NewOption("Reset everything to defaults", actReset),
			).
			Value(&action).
			WithTheme(CustomHuhTheme()).
			Run()
		if err != nil || action == actDone {
			break
		}

		switch action {
		case actProvider:
			runProviderSection(cfg)
		case actTelegram:
			runTelegramSection(cfg)
		case actSecondary:
			runSecondarySection(cfg)
		case actIdentity:
			runIdentitySection(cfg)
		case actReset:
			runResetSection(cfg)
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println(setupPanelStyle.Render(
		"\n  " + setupTitleStyle.Render("✓  SETUP COMPLETE") + "\n\n" +
			"  " + muted("Configuration saved to ~/.drogonclaw/config.json.") + "\n" +
			"  " + setupOnString("Launch workspace with:  ./drogonclaw") + "\n" +
			"  ",
	))
	fmt.Println()
}

// runProviderSection lets the operator switch the AI provider and model. Every
// field is prefilled with the stored value; leaving the API key blank keeps the
// existing key.
func runProviderSection(cfg *config.Manager) {
	muted := setupMuted()

	fmt.Println(setupSection("PROVIDER", "AI Provider & Model"))
	fmt.Println(muted("  Leave the API key blank to keep the currently stored key."))
	fmt.Println()

	var provider string
	err := huh.NewSelect[string]().
		Title("Select Intelligence Provider").
		Description("Choose where DrogonClaw delegates reasoning tasks.").
		Options(
			huh.NewOption("OpenRouter — Flexible multi-model gateway", providerOpenRouter),
			huh.NewOption("NVIDIA NIM — High-performance inference", providerNVIDIA),
			huh.NewOption("OpenAI — Direct API runtime", providerOpenAI),
			huh.NewOption("Google Gemini — Enterprise reasoning core", providerGemini),
			huh.NewOption("Ollama — Autonomous offline runtime", providerOllama),
			huh.NewOption("9Router.ai — Intelligent auto-routing", provider9Router),
		).
		Value(&provider).
		WithTheme(CustomHuhTheme()).
		Run()
	if err != nil || provider == "" {
		fmt.Println(muted("  Cancelled."))
		return
	}

	cfg.Set("AI_PROVIDER", provider)

	var apiKey, model string
	model = cfg.GetModel()
	options := providerModelOptions(provider, cfg)
	if options == nil {
		fmt.Println(WarningStyle.Render("  [!] No models available for this provider."))
		return
	}

	if provider == providerOllama {
		ollamaURL := ollamaDefaultBaseURL(cfg)
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Ollama Endpoint").
					Description("Local server URL for the offline runtime.").
					Value(&ollamaURL).
					Placeholder("http://localhost:11434"),
				huh.NewSelect[string]().
					Title("Select Model").
					Options(options...).
					Value(&model),
			),
		).WithTheme(CustomHuhTheme()).Run()
		if err != nil {
			fmt.Println(muted("  Cancelled."))
			return
		}
		cfg.Set("OLLAMA_BASE_URL", strings.TrimRight(ollamaURL, "/"))
	} else {
		keyRow := fmt.Sprintf("Stored: %s. Leave blank to keep.", maskSecret(cfg.GetAPIKey()))
		if maskSecret(cfg.GetAPIKey()) == "" {
			keyRow = "Stored locally in encrypted configuration."
		}
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("%s API Key", strings.ToUpper(provider))).
					Description(keyRow).
					EchoMode(huh.EchoModePassword).
					Value(&apiKey),
				huh.NewSelect[string]().
					Title("Select Model Target").
					Options(options...).
					Value(&model),
			),
		).WithTheme(CustomHuhTheme()).Run()
		if err != nil {
			fmt.Println(muted("  Cancelled."))
			return
		}
		if apiKey != "" {
			cfg.Set(providerAPIKeyConfig(provider), apiKey)
		}
	}

	if model != "" {
		cfg.Set("AI_MODEL", model)
	}
	setupSuccess(fmt.Sprintf("AI provider set to %s running %s.", providerLabel(provider), model))
}

// providerAPIKeyConfig maps a provider id to the config key that stores its API
// key. Kept next to the wizard so the mapping cannot drift from setup.
func providerAPIKeyConfig(provider string) string {
	switch provider {
	case providerOpenAI:
		return "OPENAI_API_KEY"
	case providerNVIDIA:
		return "NVIDIA_API_KEY"
	case providerGemini:
		return "GOOGLE_API_KEY"
	case providerOllama:
		return "OLLAMA_BASE_URL"
	case provider9Router:
		return "NINEROUTER_API_KEY"
	default:
		return "OPENROUTER_API_KEY"
	}
}

// runTelegramSection displays the current gateway state and lets the operator
// keep, replace, or disable it. This is the entry point returning users need to
// see what is already configured without re-entering credentials.
func runTelegramSection(cfg *config.Manager) {
	muted := setupMuted()
	token := cfg.GetString("TELEGRAM_TOKEN")
	chat := cfg.GetString("TELEGRAM_CHAT_ID")

	fmt.Println(setupSection("C2", "Telegram Gateway"))
	fmt.Println(muted("  Current: " + telegramStatus(cfg)))
	fmt.Println()

	var choice string
	options := []huh.Option[string]{
		huh.NewOption("Back to menu — keep current gateway", "keep"),
	}
	if token == "" {
		options = append(options, huh.NewOption("Enable gateway — set bot token & chat ID", "replace"))
	} else {
		options = append(options, huh.NewOption("Replace token or chat ID", "replace"))
		options = append(options, huh.NewOption("Disable gateway (clear credentials)", "disable"))
	}

	err := huh.NewSelect[string]().
		Title("Telegram Gateway").
		Options(options...).
		Value(&choice).
		WithTheme(CustomHuhTheme()).
		Run()
	if err != nil || choice == "keep" {
		return
	}

	switch choice {
	case "replace":
		var tgToken, tgChatID string
		tgChatID = chat
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Telegram Bot Token").
					Description("Provided by @BotFather. Blank keeps the current token.").
					EchoMode(huh.EchoModePassword).
					Value(&tgToken),
				huh.NewInput().
					Title("Authorized Chat ID").
					Description("Strict whitelist for command execution").
					Value(&tgChatID),
			),
		).WithTheme(CustomHuhTheme()).Run()
		if err != nil {
			return
		}
		if tgToken != "" {
			cfg.Set("TELEGRAM_TOKEN", tgToken)
		}
		if tgChatID != "" {
			cfg.Set("TELEGRAM_CHAT_ID", tgChatID)
		}
		setupSuccess("Telegram gateway updated.")
	case "disable":
		var confirm bool
		err = huh.NewConfirm().
			Title("Disable the Telegram gateway?").
			Description("Clears the saved bot token and chat ID.").
			Affirmative("Disable").
			Negative("Cancel").
			Value(&confirm).
			WithTheme(CustomHuhTheme()).
			Run()
		if err != nil || !confirm {
			return
		}
		cfg.Set("TELEGRAM_TOKEN", "")
		cfg.Set("TELEGRAM_CHAT_ID", "")
		fmt.Println(WarningStyle.Render("  [−] Telegram gateway disabled."))
	}
}

// runSecondarySection shows which OSINT keys are stored and lets the operator
// add/update any selection, then clear any currently-stored keys.
func runSecondarySection(cfg *config.Manager) {
	muted := setupMuted()

	fmt.Println(setupSection("KEYS", "Secondary API Keys"))
	fmt.Println(muted("  Navigate with space/x, confirm with enter."))
	fmt.Println()

	var selected []string
	options := make([]huh.Option[string], 0, len(secondaryKeys))
	for _, k := range secondaryKeys {
		stored := storedKeyStatus(cfg.GetString(k.configKey))
		options = append(options, huh.NewOption(k.label+"  ("+stored+")", k.configKey))
	}

	multi := huh.NewMultiSelect[string]().
		Title("Add or update which secondary API keys?").
		Description("Leave empty to skip this step.").
		Options(options...).
		Value(&selected).
		Filterable(false).
		WithTheme(CustomHuhTheme())
	if err := multi.Run(); err != nil {
		return
	}

	for _, key := range selected {
		var val string
		err := huh.NewInput().
			Title(key).
			Description("Stored locally in ~/.drogonclaw/config.json.").
			EchoMode(huh.EchoModePassword).
			Value(&val).
			WithTheme(CustomHuhTheme()).
			Run()
		if err == nil && val != "" {
			cfg.Set(key, val)
			setupSuccess(key + " updated.")
		}
	}

	var setKeys []string
	for _, k := range secondaryKeys {
		if cfg.GetString(k.configKey) != "" {
			setKeys = append(setKeys, k.configKey)
		}
	}
	if len(setKeys) == 0 {
		return
	}

	var toClear []string
	clearOptions := make([]huh.Option[string], 0, len(setKeys))
	for _, key := range setKeys {
		clearOptions = append(clearOptions, huh.NewOption(key+"  (stored)", key))
	}
	clear := huh.NewMultiSelect[string]().
		Title("Clear any of the stored secondary keys?").
		Description("Leave empty to keep everything.").
		Options(clearOptions...).
		Value(&toClear).
		Filterable(false).
		WithTheme(CustomHuhTheme())
	if err := clear.Run(); err != nil {
		return
	}
	for _, key := range toClear {
		cfg.Set(key, "")
		fmt.Println(WarningStyle.Render(fmt.Sprintf("  [−] %s cleared.", key)))
	}
}

// runIdentitySection updates the operator and agent display names.
func runIdentitySection(cfg *config.Manager) {
	muted := setupMuted()
	operator := cfg.GetOperatorName()
	agent := cfg.GetAgentName()

	fmt.Println(setupSection("PROFILE", "Operator & Agent Identity"))
	fmt.Println(muted("  Used in headers and reports."))
	fmt.Println()

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Operator name").
				Value(&operator),
			huh.NewInput().
				Title("Agent name").
				Value(&agent),
		),
	).WithTheme(CustomHuhTheme()).Run()
	if err != nil {
		return
	}
	cfg.Set("OPERATOR_NAME", operator)
	cfg.Set("AGENT_NAME", agent)
	setupSuccess("Identity updated.")
}

// managedConfigKeys is the full set of credentials the wizard can reset.
var managedConfigKeys = []string{
	"OPENROUTER_API_KEY", "NVIDIA_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY",
	"NINEROUTER_API_KEY", "ROUTER_MODE",
	"OLLAMA_BASE_URL", "TELEGRAM_TOKEN", "TELEGRAM_CHAT_ID",
	"OPERATOR_NAME", "AGENT_NAME",
	"GITHUB_TOKEN", "SHODAN_API_KEY", "VIRUSTOTAL_API_KEY",
	"BRAVE_SEARCH_API_KEY", "HUNTER_IO_API_KEY", "EXA_API_KEY",
}

// runResetSection clears every wizard-managed credential and returns the
// provider/model to defaults so a fresh authorised run is required.
func runResetSection(cfg *config.Manager) {
	muted := setupMuted()
	fmt.Println(setupSection("RESET", "Reset Configuration"))
	fmt.Println(WarningStyle.Render("  This clears AI provider keys, Telegram credentials and all secondary keys."))
	fmt.Println()

	var confirm bool
	err := huh.NewConfirm().
		Title("Reset everything to defaults?").
		Description("You will be asked to authorise again on the next run.").
		Affirmative("Reset").
		Negative("Cancel").
		Value(&confirm).
		WithTheme(CustomHuhTheme()).
		Run()
	if err != nil || !confirm {
		fmt.Println(muted("  Cancelled."))
		return
	}

	for _, key := range managedConfigKeys {
		cfg.Set(key, "")
	}
	cfg.Set("AI_PROVIDER", "openrouter")
	cfg.Set("AI_MODEL", "")
	cfg.SetVerified(false)
	fmt.Println(WarningStyle.Render("  [−] All configuration reset to defaults."))
}
