package config

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenRouterModelsLive(t *testing.T) {
	orig := openRouterModelsURL
	defer func() { openRouterModelsURL = orig }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"meta-llama/llama-3.1-405b-instruct"},{"id":"openai/gpt-4o"}]}`))
	}))
	defer srv.Close()
	openRouterModelsURL = srv.URL

	ids, err := OpenRouterModels()
	if err != nil {
		t.Fatalf("OpenRouterModels error: %v", err)
	}
	if len(ids) != 2 || ids[0] != "meta-llama/llama-3.1-405b-instruct" || ids[1] != "openai/gpt-4o" {
		t.Fatalf("unexpected ids: %v", ids)
	}
}

func TestOpenRouterModelsBundledFallback(t *testing.T) {
	orig := openRouterModelsURL
	defer func() { openRouterModelsURL = orig }()

	// Point at an unreachable server so fetch fails and we fall back to the
	// bundled catalog shipped at repo root.
	openRouterModelsURL = "http://127.0.0.1:1/unreachable"

	ids, err := OpenRouterModels()
	if err != nil {
		t.Fatalf("expected bundled fallback, got error: %v", err)
	}
	if len(ids) == 0 {
		t.Fatalf("bundled fallback returned no models")
	}
}
