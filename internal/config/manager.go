package config

import (
	"os"
	"path/filepath"
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

	// Env vars override config file
	m.v.AutomaticEnv()
	m.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

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
	_ = m.v.WriteConfigAs(filepath.Join(home, ".drogonclaw", "config.json"))
}

func (m *Manager) GetProvider() string {
	p := strings.ToLower(m.GetString("AI_PROVIDER"))
	if p == "" {
		p = strings.ToLower(os.Getenv("AI_PROVIDER"))
	}
	if p == "" {
		p = "openrouter"
	}
	return p
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

func (m *Manager) GetAPIKey() string {
	provider := m.GetProvider()
	switch provider {
	case "openai":
		return firstNonEmpty(m.GetString("OPENAI_API_KEY"), os.Getenv("OPENAI_API_KEY"))
	case "nvidia":
		return firstNonEmpty(m.GetString("NVIDIA_API_KEY"), os.Getenv("NVIDIA_API_KEY"))
	case "gemini":
		return firstNonEmpty(m.GetString("GOOGLE_API_KEY"), os.Getenv("GOOGLE_API_KEY"))
	case "ollama":
		return "ollama"
	default: // openrouter
		return firstNonEmpty(m.GetString("OPENROUTER_API_KEY"), os.Getenv("OPENROUTER_API_KEY"))
	}
}

func (m *Manager) GetBaseURL() string {
	switch m.GetProvider() {
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "nvidia":
		return "https://integrate.api.nvidia.com/v1"
	case "ollama":
		base := firstNonEmpty(m.GetString("OLLAMA_BASE_URL"), os.Getenv("OLLAMA_BASE_URL"), "http://localhost:11434")
		return strings.TrimRight(base, "/") + "/v1"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta/openai/"
	default:
		return "https://api.openai.com/v1"
	}
}

func (m *Manager) GetOperatorName() string {
	return m.GetString("OPERATOR_NAME")
}

func (m *Manager) GetWorkspaceRoot() string {
	return m.GetString("WORKSPACE_ROOT")
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

func (m *Manager) SetVerified(verified bool) {
	if verified {
		m.Set("DROGONCLAW_VERIFIED", "true")
	} else {
		m.Set("DROGONCLAW_VERIFIED", "false")
	}
}
