package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// The Setup Wizard config file is the single source of truth. A stale env var
// export must NOT override a value saved by `./drogonclaw setup` / `/setup`.
func TestConfigManager_GetAPIKeyFromConfig(t *testing.T) {
	home, err := os.MkdirTemp("", "dc-cfg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(home)
	cfgDir := filepath.Join(home, ".drogonclaw")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatal(err)
	}
	// Config says openrouter; a stale shell export says openai.
	cfgJSON := `{"AI_PROVIDER":"openrouter","AI_MODEL":"openai/gpt-4o","OPENROUTER_API_KEY":"sk-or-abc","USE_SANDBOX":"true"}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(cfgJSON), 0600); err != nil {
		t.Fatal(err)
	}

	os.Setenv("AI_PROVIDER", "openai")
	os.Setenv("OPENAI_API_KEY", "sk-stale")
	os.Setenv("HOME", home)
	instance = nil

	cfg := Get()
	if got := cfg.GetProvider(); got != "openrouter" {
		t.Fatalf("GetProvider = %q, want openrouter (config must beat env)", got)
	}
	if got := cfg.GetAPIKey(); got != "sk-or-abc" {
		t.Fatalf("GetAPIKey = %q, want sk-or-abc (config must beat env)", got)
	}
	if got := cfg.GetBaseURL(); got != "https://openrouter.ai/api/v1" {
		t.Fatalf("GetBaseURL = %q, want openrouter URL", got)
	}
}

func TestConfigManager_GetFastModelFallback(t *testing.T) {
	home, err := os.MkdirTemp("", "dc-cfg-fast")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(home)
	cfgDir := filepath.Join(home, ".drogonclaw")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatal(err)
	}
	cfgJSON := `{"AI_MODEL":"deepseek/deepseek-reasoner"}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(cfgJSON), 0600); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("AI_MODEL_FAST")
	os.Setenv("HOME", home)
	instance = nil

	m := &Manager{v: viper.New()}
	m.load()
	if got := m.GetFastModel(); got != "deepseek/deepseek-reasoner" {
		t.Fatalf("GetFastModel = %q, want fallback to primary model", got)
	}
	if got := m.GetBenchmarkConcurrency(); got != 4 {
		t.Fatalf("GetBenchmarkConcurrency default = %d, want 4", got)
	}
}

// Env vars are only a fallback when the wizard has not written a config value.
func TestConfigManager_GetAPIKeyEnvFallback(t *testing.T) {
	home, err := os.MkdirTemp("", "dc-cfg-env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(home)
	os.Setenv("HOME", home) // no config.json written -> fall back to env
	os.Setenv("AI_PROVIDER", "openai")
	os.Setenv("OPENAI_API_KEY", "sk-test123")
	instance = nil

	cfg := Get()
	key := cfg.GetAPIKey()
	if key != "sk-test123" {
		t.Errorf("expected sk-test123, got %s", key)
	}

	os.Setenv("AI_PROVIDER", "ollama")
	instance = nil
	cfg = Get()
	if cfg.GetAPIKey() != "ollama" {
		t.Errorf("expected ollama, got %s", cfg.GetAPIKey())
	}
}

// When the config file is absent, USE_SANDBOX is read from the environment
// (set by the `native` / `sandbox` subcommands).
func TestConfigManager_IsSandboxEnabled(t *testing.T) {
	home, err := os.MkdirTemp("", "dc-cfg-sb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(home)
	os.Setenv("HOME", home) // no config.json

	os.Setenv("USE_SANDBOX", "false")
	instance = nil
	if Get().IsSandboxEnabled() {
		t.Errorf("expected sandbox to be disabled")
	}

	os.Setenv("USE_SANDBOX", "true")
	instance = nil
	if !Get().IsSandboxEnabled() {
		t.Errorf("expected sandbox to be enabled")
	}
}
