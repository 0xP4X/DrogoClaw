package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// openRouterModelsURL is a package-level var (not const) so tests can point it
// at a local server.
var openRouterModelsURL = "https://openrouter.ai/api/v1/models"

type openRouterModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type openRouterModelsResp struct {
	Data []openRouterModel `json:"data"`
}

// OpenRouterModels returns the live list of OpenRouter model IDs. It prefers a
// fresh fetch from the API, then falls back to a local cache
// (~/.drogonclaw/models.json), then the bundled catalog shipped with the
// binary. This keeps the model picker current instead of a stale static file.
func OpenRouterModels() ([]string, error) {
	if ids, err := fetchOpenRouterModels(context.Background()); err == nil && len(ids) > 0 {
		_ = writeModelCache(ids)
		return ids, nil
	}
	if ids, err := readModelCache(); err == nil && len(ids) > 0 {
		return ids, nil
	}
	if ids, err := readBundledModels(); err == nil && len(ids) > 0 {
		return ids, nil
	}
	return nil, fmt.Errorf("unable to load OpenRouter models (offline and no cached/bundled catalog)")
}

func fetchOpenRouterModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openRouterModelsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DrogonClaw/2.0 (security-assessment-tool)")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter models error %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	var out openRouterModelsResp
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(out.Data))
	for _, m := range out.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("openrouter returned no models")
	}
	return ids, nil
}

func modelCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".drogonclaw", "models.json")
}

// writeModelCache persists the fetched IDs to ~/.drogonclaw/models.json so the
// picker still works offline. It preserves the OpenRouter {"object":"list",
// "data":[...]} shape used by the bundled catalog.
func writeModelCache(ids []string) error {
	dir := filepath.Dir(modelCachePath())
	_ = os.MkdirAll(dir, 0700)

	data := struct {
		Object string              `json:"object"`
		Data   []map[string]string `json:"data"`
	}{Object: "list"}
	for _, id := range ids {
		data.Data = append(data.Data, map[string]string{"id": id})
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(modelCachePath(), b, 0600)
}

func readModelCache() ([]string, error) {
	return readModelsFile(modelCachePath())
}

// readBundledModels searches for the shipped models.json so the wizard still
// has a fallback when fully offline and no cache exists yet.
func readBundledModels() ([]string, error) {
	candidates := []string{"models.json", filepath.Join("..", "models.json")}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "models.json"))
	}
	for _, c := range candidates {
		if ids, err := readModelsFile(c); err == nil && len(ids) > 0 {
			return ids, nil
		}
	}
	return nil, fmt.Errorf("bundled models.json not found")
}

func readModelsFile(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out openRouterModelsResp
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(out.Data))
	for _, m := range out.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids, nil
}
