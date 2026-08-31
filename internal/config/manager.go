package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// Manager wraps Viper for DrogonClaw config, reading from ~/.drogonclaw/config.json and env vars.
type Manager struct {
	v *viper.Viper
}

var instance *Manager

func Get() *Manager {
	if instance == nil {
		instance = &Manager{v: viper.New()}
		instance.load()
	}
	return instance
}

func (m *Manager) load() {
	home, _ := os.UserHomeDir()
	cfgDir := filepath.Join(home, ".drogonclaw")
	_ = os.MkdirAll(cfgDir, 0700)

	m.v.SetConfigName("config")
	m.v.SetConfigType("json")
	m.v.AddConfigPath(cfgDir)
	m.v.AddConfigPath(".")

	// NOTE: AutomaticEnv() is intentionally NOT enabled. The Setup Wizard
	// (`./drogonclaw setup` / `/setup`) is the single source of truth for
	// provider/model/API-key configuration, persisted to
	// ~/.drogonclaw/config.json. Environment variables are only consulted as a
	// fallback (see the Get* helpers) and only for values the wizard does not
	// manage — e.g. OSINT keys like SHODAN_API_KEY. This prevents a stray
	// `export AI_PROVIDER=...` from silently overriding the saved config.

	// Defaults
	m.v.SetDefault("WORKSPACE_ROOT", home)

	_ = m.v.ReadInConfig() // OK if file missing — first run
}

func (m *Manager) GetString(key string) string {
	return m.v.GetString(key)
}

func (m *Manager) Set(key, value string) {
	m.v.Set(key, value)
	m.save()
}

func (m *Manager) Reload() {
	_ = m.v.ReadInConfig()
}

func (m *Manager) save() {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".drogonclaw", "config.json")
	_ = m.v.WriteConfigAs(path)
	_ = os.Chmod(path, 0600)
}

func (m *Manager) GetProvider() string {
	p := m.GetString("AI_PROVIDER")
	if p == "" {
		p = os.Getenv("AI_PROVIDER")
	}
	if p == "" {
		p = "openrouter"
	}
	return strings.ToLower(p)
}

func (m *Manager) GetRouterMode() string {
	mode := m.GetString("ROUTER_MODE")
	if mode == "" {
		mode = os.Getenv("ROUTER_MODE")
	}
	if mode == "" {
		mode = "off" // Routing disabled by default
	}
	return strings.ToLower(mode)
}

func (m *Manager) GetModel() string {
	mod := m.GetString("AI_MODEL")
	if mod == "" {
		mod = os.Getenv("AI_MODEL")
	}
	if mod == "" {
		mod = "meta-llama/llama-3.1-70b-instruct"
	}
	return mod
}

func (m *Manager) GetFastModel() string {
	mod := m.GetString("AI_MODEL_FAST")
	if mod == "" {
		mod = os.Getenv("AI_MODEL_FAST")
	}
	if mod == "" {
		mod = m.GetModel()
	}
	return mod
}

func (m *Manager) GetBenchmarkConcurrency() int {
	v := m.GetString("BENCH_CONCURRENCY")
	if v == "" {
		v = os.Getenv("BENCH_CONCURRENCY")
	}
	if v == "" {
		return 4
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 4
	}
	return n
}

func (m *Manager) GetAPIKey() string {
	provider := m.GetProvider()
	// The wizard-stored key (config file) always takes precedence over an env
	// var. Env vars are only a fallback for keys the wizard does not manage.
	switch provider {
	case "openai":
		return firstNonEmpty(m.GetString("OPENAI_API_KEY"), os.Getenv("OPENAI_API_KEY"))
	case "nvidia":
		return firstNonEmpty(m.GetString("NVIDIA_API_KEY"), os.Getenv("NVIDIA_API_KEY"))
	case "gemini":
		return firstNonEmpty(m.GetString("GOOGLE_API_KEY"), os.Getenv("GOOGLE_API_KEY"))
	case "ollama":
		return "ollama"
	case "9router":
		return firstNonEmpty(m.GetString("NINEROUTER_API_KEY"), os.Getenv("NINEROUTER_API_KEY"))
	default: // openrouter
		return firstNonEmpty(m.GetString("OPENROUTER_API_KEY"), os.Getenv("OPENROUTER_API_KEY"))
	}
}

func (m *Manager) GetNineRouterAPIKey() string {
	return firstNonEmpty(m.GetString("NINEROUTER_API_KEY"), os.Getenv("NINEROUTER_API_KEY"))
}

func (m *Manager) GetBaseURL() string {
	switch m.GetProvider() {
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "openai":
		return "https://api.openai.com/v1"
	case "nvidia":
		return "https://integrate.api.nvidia.com/v1"
	case "ollama":
		base := firstNonEmpty(m.GetString("OLLAMA_BASE_URL"), os.Getenv("OLLAMA_BASE_URL"), "http://localhost:11434")
		base = strings.TrimRight(base, "/")
		base = strings.TrimSuffix(base, "/v1")
		return base + "/v1"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta/openai/"
	case "9router":
		return "https://api.9router.ai/v1"
	default:
		return "https://openrouter.ai/api/v1"
	}
}

func (m *Manager) GetOperatorName() string {
	return m.GetString("OPERATOR_NAME")
}

func (m *Manager) GetWorkspaceRoot() string {
	return m.GetString("WORKSPACE_ROOT")
}

// GetAutopilot reports whether delegating autopilot mode is enabled. When on,
// long-running low-risk tools auto-accept instead of pausing for the operator.
func (m *Manager) GetAutopilot() bool {
	v := strings.ToLower(m.GetString("AUTOPILOT"))
	if v == "" {
		v = strings.ToLower(os.Getenv("AUTOPILOT"))
	}
	return v == "true" || v == "1" || v == "on"
}

func (m *Manager) SetAutopilot(enabled bool) {
	m.Set("AUTOPILOT", map[bool]string{true: "true", false: "false"}[enabled])
}

func (m *Manager) GetAgentName() string {
	n := m.GetString("AGENT_NAME")
	if n == "" {
		return "DrogonClaw"
	}
	return n
}

func (m *Manager) IsSandboxEnabled() bool {
	v := strings.ToLower(m.GetString("USE_SANDBOX"))
	if v == "" {
		v = strings.ToLower(os.Getenv("USE_SANDBOX"))
	}
	return v != "false" // sandbox ON by default
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func (m *Manager) GetBraveAPIKey() string {
	return firstNonEmpty(m.GetString("BRAVE_SEARCH_API_KEY"), os.Getenv("BRAVE_SEARCH_API_KEY"))
}

func (m *Manager) GetShodanAPIKey() string {
	return firstNonEmpty(m.GetString("SHODAN_API_KEY"), os.Getenv("SHODAN_API_KEY"))
}

func (m *Manager) GetVirusTotalAPIKey() string {
	return firstNonEmpty(m.GetString("VIRUSTOTAL_API_KEY"), os.Getenv("VIRUSTOTAL_API_KEY"))
}

func (m *Manager) GetHunterAPIKey() string {
	return firstNonEmpty(m.GetString("HUNTER_IO_API_KEY"), os.Getenv("HUNTER_IO_API_KEY"))
}

func (m *Manager) GetGitHubToken() string {
	return firstNonEmpty(m.GetString("GITHUB_TOKEN"), os.Getenv("GITHUB_TOKEN"))
}

func (m *Manager) IsVerified() bool {
	return strings.ToLower(m.GetString("DROGONCLAW_VERIFIED")) == "true"
}

func (m *Manager) GetMaxIterations() int {
	v := m.GetString("MAX_ITERATIONS")
	if v == "" {
		return 20
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 20
	}
	return n
}

func (m *Manager) SetVerified(verified bool) {
	if verified {
		m.Set("DROGONCLAW_VERIFIED", "true")
	} else {
		m.Set("DROGONCLAW_VERIFIED", "false")
	}
}

func (m *Manager) GetTheme() string {
	t := m.GetString("THEME")
	if t == "" {
		return "dark"
	}
	return t
}

func (m *Manager) SetTheme(name string) {
	m.Set("THEME", name)
}

func (m *Manager) SetAutopilot(enabled bool) {
	val := "false"
	if enabled {
		val = "true"
	}
	m.Set("AUTOPILOT", val)
}

func (m *Manager) IsAutopilot() bool {
	s := strings.ToLower(m.GetString("AUTOPILOT"))
	return s == "true" || s == "on"
}
