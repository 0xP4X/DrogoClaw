package config

import (
	"os"
	"testing"
)

func TestConfigManager_GetAPIKey(t *testing.T) {
	os.Setenv("AI_PROVIDER", "openai")
	os.Setenv("OPENAI_API_KEY", "sk-test123")
	
	// Reset singleton for testing
	instance = nil
	cfg := Get()

	key := cfg.GetAPIKey()
	if key != "sk-test123" {
		t.Errorf("expected sk-test123, got %s", key)
	}
	
	os.Setenv("AI_PROVIDER", "ollama")
	
	// Reset singleton
	instance = nil
	cfg = Get()
	
	if cfg.GetAPIKey() != "ollama" {
		t.Errorf("expected ollama, got %s", cfg.GetAPIKey())
	}
}

func TestConfigManager_IsSandboxEnabled(t *testing.T) {
	os.Setenv("USE_SANDBOX", "false")
	
	// Reset singleton
	instance = nil
	cfg := Get()
	
	if cfg.IsSandboxEnabled() {
		t.Errorf("expected sandbox to be disabled")
	}
	
	os.Setenv("USE_SANDBOX", "true")
	
	// Reset singleton
	instance = nil
	cfg = Get()
	
	if !cfg.IsSandboxEnabled() {
		t.Errorf("expected sandbox to be enabled")
	}
}
