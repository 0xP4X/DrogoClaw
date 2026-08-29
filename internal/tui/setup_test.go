package tui

import (
	"strings"
	"testing"
)

// fakeReader is a deterministic configReader for pure-function tests. It never
// touches the operator's real ~/.drogonclaw/config.json.
type fakeReader struct {
	vals map[string]string
}

func (f fakeReader) GetProvider() string     { return f.vals["PROVIDER"] }
func (f fakeReader) GetModel() string        { return f.vals["MODEL"] }
func (f fakeReader) GetAPIKey() string       { return f.vals["APIKEY"] }
func (f fakeReader) GetBaseURL() string      { return f.vals["BASE_URL"] }
func (f fakeReader) GetOperatorName() string { return f.vals["OPERATOR_NAME"] }
func (f fakeReader) GetAgentName() string    { return f.vals["AGENT_NAME"] }
func (f fakeReader) GetString(k string) string {
	return f.vals[k]
}

func TestMaskSecret(t *testing.T) {
	if got := maskSecret(""); got != "" {
		t.Errorf("maskSecret(empty) = %q, want empty", got)
	}
	if got := maskSecret("abcd"); got != "••••" {
		t.Errorf("maskSecret(short) = %q, want fixed mask", got)
	}
	if got := maskSecret("sk-abcdefghijkl"); got != "••••••••ijkl" {
		t.Errorf("maskSecret(long) = %q, want mask + last 4", got)
	}
}

func TestProviderLabel(t *testing.T) {
	cases := map[string]string{
		"openrouter": "OpenRouter",
		"OPENAI":     "OpenAI",
		"ollama":     "Ollama (local)",
		"future-svc": "future-svc",
	}
	for in, want := range cases {
		if got := providerLabel(in); got != want {
			t.Errorf("providerLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStoredKeyStatus(t *testing.T) {
	if storedKeyStatus("") != "not set" {
		t.Error("empty value should report not set")
	}
	if storedKeyStatus("x") != "set" {
		t.Error("non-empty value should report set")
	}
}

func TestTelegramStatus(t *testing.T) {
	r := fakeReader{vals: map[string]string{}}
	if got := telegramStatus(r); got != "disabled" {
		t.Errorf("empty gateway = %q, want disabled", got)
	}

	r.vals["TELEGRAM_CHAT_ID"] = "987654"
	if got := telegramStatus(r); got != "chat 987654" {
		t.Errorf("token-only-out? got %q", got)
	}

	r.vals["TELEGRAM_TOKEN"] = "123456:AAabcdefgh"
	got := telegramStatus(r)
	if !strings.Contains(got, "chat 987654") || !strings.Contains(got, "•") {
		t.Errorf("telegram status should show masked token and chat, got %q", got)
	}
	if strings.Contains(got, "AAabcdefgh") {
		t.Errorf("telegram status leaked the full token: %q", got)
	}
}

func TestRenderConfigSummaryCloudProvider(t *testing.T) {
	r := fakeReader{vals: map[string]string{
		"PROVIDER":         "openai",
		"MODEL":            "gpt-4o",
		"APIKEY":           "sk-abcdefghijkl",
		"TELEGRAM_TOKEN":   "123456:AAabcdefgh",
		"TELEGRAM_CHAT_ID": "987654",
		"GITHUB_TOKEN":     "ghp_secret",
		"OPERATOR_NAME":    "jiggon",
		"AGENT_NAME":       "DrogonClaw",
		"BASE_URL":         "https://api.openai.com/v1",
	}}
	out := renderConfigSummary(r)

	for _, want := range []string{"OpenAI", "gpt-4o", "••••••••ijkl", "987654", "jiggon", "DrogonClaw"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "sk-abcdefghijkl") || strings.Contains(out, "AAabcdefgh") {
		t.Error("summary leaked a full secret")
	}
	if !strings.Contains(out, "set") || !strings.Contains(out, "not set") {
		t.Error("summary should mark set and not-set keys")
	}
}

func TestRenderConfigSummaryOllama(t *testing.T) {
	r := fakeReader{vals: map[string]string{
		"PROVIDER":        "ollama",
		"MODEL":           "llama3.1",
		"APIKEY":          "ollama",
		"OLLAMA_BASE_URL": "http://localhost:11434",
		"BASE_URL":        "http://localhost:11434/v1",
	}}
	out := renderConfigSummary(r)
	if !strings.Contains(out, "no key required") {
		t.Errorf("ollama summary should mention no key is required:\n%s", out)
	}
	if !strings.Contains(out, "http://localhost:11434") {
		t.Errorf("ollama summary should show the endpoint:\n%s", out)
	}
}

func TestSecondaryKeysTableSync(t *testing.T) {
	want := map[string]bool{
		"GITHUB_TOKEN": true, "SHODAN_API_KEY": true, "VIRUSTOTAL_API_KEY": true,
		"BRAVE_SEARCH_API_KEY": true, "HUNTER_IO_API_KEY": true, "EXA_API_KEY": true,
	}
	seen := map[string]bool{}
	for _, k := range secondaryKeys {
		if k.configKey == "" || k.short == "" || k.label == "" {
			t.Fatalf("secondary key entry incomplete: %+v", k)
		}
		if seen[k.configKey] {
			t.Fatalf("duplicate secondary key %s", k.configKey)
		}
		seen[k.configKey] = true
	}
	for key := range want {
		if !seen[key] {
			t.Errorf("secondary keys table missing config key %s", key)
		}
	}
}

func TestProviderAPIKeyConfigMapping(t *testing.T) {
	cases := map[string]string{
		"openai": "OPENAI_API_KEY", "nvidia": "NVIDIA_API_KEY",
		"gemini": "GOOGLE_API_KEY", "ollama": "OLLAMA_BASE_URL",
		"openrouter": "OPENROUTER_API_KEY", "unknown": "OPENROUTER_API_KEY",
	}
	for in, want := range cases {
		if got := providerAPIKeyConfig(in); got != want {
			t.Errorf("providerAPIKeyConfig(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProviderModelOptionsCurated(t *testing.T) {
	r := fakeReader{vals: map[string]string{"PROVIDER": "openai"}}
	opts := providerModelOptions("openai", r)
	if len(opts) != 2 || opts[0].Value != "gpt-4o" {
		t.Errorf("openai curated options wrong: %+v", opts)
	}
	if got := providerModelOptions("bogus-provider", r); got != nil {
		t.Error("unknown provider should yield no model options")
	}
}

func TestManagedConfigKeysCoverSecondaryKeys(t *testing.T) {
	for _, k := range secondaryKeys {
		found := false
		for _, managed := range managedConfigKeys {
			if managed == k.configKey {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("reset list missing secondary key %s", k.configKey)
		}
	}
}
