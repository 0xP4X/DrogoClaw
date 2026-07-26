package toolmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

var ghClient = &http.Client{Timeout: 30 * time.Second}

// GitHubRelease holds release asset info.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// DownloadFromGitHub downloads a file from a GitHub repository into the sandbox workspace.
// repo format: "owner/repo", filePath is the file within the release assets or repo.
func DownloadFromGitHub(ctx context.Context, repo, filePattern string, sb *sandbox.Docker) (string, error) {
	// Get latest release info
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, _ := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "DrogonClaw/2.0")

	resp, err := ghClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 404 {
		// No releases — try raw file download from main branch
		return downloadRawFile(ctx, repo, filePattern, sb)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("release decode error: %w", err)
	}

	// Find matching asset
	filePattern = strings.ToLower(filePattern)
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, filePattern) || filePattern == "" {
			destName := filepath.Base(asset.Name)
			cmd := fmt.Sprintf(
				"wget -q -O /workspace/%s '%s' && chmod +x /workspace/%s && echo '[OK] Downloaded %s (release %s)'",
				destName, asset.BrowserDownloadURL, destName, destName, release.TagName,
			)
			out, err := sb.Execute(ctx, cmd)
			if err != nil {
				return "", fmt.Errorf("download failed: %w", err)
			}
			return out, nil
		}
	}

	// No matching asset — list available ones
	var assetNames []string
	for _, a := range release.Assets {
		assetNames = append(assetNames, a.Name)
	}
	return "", fmt.Errorf("no asset matching '%s' in release %s. Available: %s",
		filePattern, release.TagName, strings.Join(assetNames, ", "))
}

// downloadRawFile downloads a specific file directly from a GitHub repository.
func downloadRawFile(ctx context.Context, repo, filePath string, sb *sandbox.Docker) (string, error) {
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/%s", repo, filePath)
	destName := filepath.Base(filePath)
	cmd := fmt.Sprintf(
		"wget -q -O /workspace/%s '%s' && chmod +x /workspace/%s && echo '[OK] Downloaded %s from %s'",
		destName, rawURL, destName, destName, repo,
	)
	out, err := sb.Execute(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("raw download failed: %w\nOutput: %s", err, out)
	}
	return out, nil
}
