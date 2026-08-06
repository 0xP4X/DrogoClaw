package intel

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const nvdFeedURL = "https://services.nvd.nist.gov/rest/json/cves/2.0"

// CVEEntry represents a single CVE from the NVD.
type CVEEntry struct {
	ID          string
	Description string
	CVSSScore   float64
	CVSSSeverity string
	Published   string
	ExploitAvailable bool
}

// CVEDatabase holds an in-memory index of CVEs keyed by normalized keyword.
type CVEDatabase struct {
	entries []CVEEntry
	path    string
}

var globalCVEDB *CVEDatabase

// LoadCVEDatabase loads or initializes the local CVE cache.
func LoadCVEDatabase() (*CVEDatabase, error) {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".drogonclaw", "cve_cache")
	_ = os.MkdirAll(cacheDir, 0700)
	dbPath := filepath.Join(cacheDir, "nvd_cache.json")

	db := &CVEDatabase{path: dbPath}

	// Load from cache if recent (< 24h)
	if fi, err := os.Stat(dbPath); err == nil {
		if time.Since(fi.ModTime()) < 24*time.Hour {
			if err := db.loadFromDisk(); err == nil {
				globalCVEDB = db
				return db, nil
			}
		}
	}

	// Fresh fetch from NVD (recent 30 days of CVEs to keep it manageable)
	if err := db.fetchRecent(); err != nil {
		// If fetch fails, try to use stale cache
		if loadErr := db.loadFromDisk(); loadErr == nil {
			globalCVEDB = db
			return db, nil
		}
		return nil, fmt.Errorf("CVE database unavailable: %w", err)
	}

	globalCVEDB = db
	return db, nil
}

func (db *CVEDatabase) fetchRecent() error {
	// NVD API: last 120 days, limit 2000 results
	pubStartDate := time.Now().AddDate(0, 0, -120).Format("2006-01-02T15:04:05.000")
	pubEndDate := time.Now().Format("2006-01-02T15:04:05.000")

	endpoint := fmt.Sprintf("%s?pubStartDate=%s&pubEndDate=%s&resultsPerPage=2000",
		nvdFeedURL, pubStartDate, pubEndDate)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("NVD request failed: %w", err)
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		defer gz.Close()
		reader = gz
	}

	var nvdResp struct {
		Vulnerabilities []struct {
			CVE struct {
				ID          string `json:"id"`
				Published   string `json:"published"`
				Descriptions []struct {
					Lang  string `json:"lang"`
					Value string `json:"value"`
				} `json:"descriptions"`
				Metrics struct {
					CVSSMetricV31 []struct {
						CVSSData struct {
							BaseScore    float64 `json:"baseScore"`
							BaseSeverity string  `json:"baseSeverity"`
						} `json:"cvssData"`
					} `json:"cvssMetricV31"`
					CVSSMetricV2 []struct {
						CVSSData struct {
							BaseScore float64 `json:"baseScore"`
						} `json:"cvssData"`
						BaseSeverity string `json:"baseSeverity"`
					} `json:"cvssMetricV2"`
				} `json:"metrics"`
			} `json:"cve"`
		} `json:"vulnerabilities"`
	}

	if err := json.NewDecoder(reader).Decode(&nvdResp); err != nil {
		return fmt.Errorf("NVD decode error: %w", err)
	}

	db.entries = nil
	for _, v := range nvdResp.Vulnerabilities {
		entry := CVEEntry{
			ID:        v.CVE.ID,
			Published: v.CVE.Published,
		}

		for _, d := range v.CVE.Descriptions {
			if d.Lang == "en" {
				entry.Description = d.Value
				break
			}
		}

		if len(v.CVE.Metrics.CVSSMetricV31) > 0 {
			entry.CVSSScore = v.CVE.Metrics.CVSSMetricV31[0].CVSSData.BaseScore
			entry.CVSSSeverity = v.CVE.Metrics.CVSSMetricV31[0].CVSSData.BaseSeverity
		} else if len(v.CVE.Metrics.CVSSMetricV2) > 0 {
			entry.CVSSScore = v.CVE.Metrics.CVSSMetricV2[0].CVSSData.BaseScore
			entry.CVSSSeverity = v.CVE.Metrics.CVSSMetricV2[0].BaseSeverity
		}

		db.entries = append(db.entries, entry)
	}

	// Save to disk
	data, _ := json.Marshal(db.entries)
	_ = os.WriteFile(db.path, data, 0600)

	return nil
}

func (db *CVEDatabase) loadFromDisk() error {
	data, err := os.ReadFile(db.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &db.entries)
}

// Search searches the CVE database for entries matching product/version keywords.
func (db *CVEDatabase) Search(keywords string) []CVEEntry {
	keywords = strings.ToLower(keywords)
	words := strings.Fields(keywords)

	var matches []CVEEntry
	for _, e := range db.entries {
		desc := strings.ToLower(e.Description)
		score := 0
		for _, w := range words {
			if strings.Contains(desc, w) {
				score++
			}
		}
		if score == len(words) || (len(words) > 1 && score >= len(words)-1) {
			matches = append(matches, e)
		}
		if len(matches) >= 20 {
			break
		}
	}

	return matches
}

// LookupCVE is the public entry point for the tool — searches by product+version.
func LookupCVE(productVersion string) string {
	var sb strings.Builder
	queryLow := strings.ToLower(productVersion)

	// 1. Search Static DB first
	var staticMatches []StaticCVE
	for key, entry := range StaticCVEDB {
		if strings.Contains(strings.ToLower(key), queryLow) || strings.Contains(strings.ToLower(entry.ID), queryLow) {
			staticMatches = append(staticMatches, entry)
		}
	}

	if len(staticMatches) > 0 {
		sb.WriteString(fmt.Sprintf("★ Static CVE results for '%s' (Offline Curated):\n\n", productVersion))
		for _, m := range staticMatches {
			sb.WriteString(fmt.Sprintf("▸ %s [CVSS: %.1f]\n", m.ID, m.CVSSScore))
			sb.WriteString(fmt.Sprintf("  Description: %s\n", m.Description))
			sb.WriteString(fmt.Sprintf("  MSF Module: %s\n", m.MSFModule))
			sb.WriteString(fmt.Sprintf("  Verify Cmd: %s\n\n", m.VerifyCmd))
		}
	}

	// 2. Search NVD Cache (load lazily on first use so lookup_cve works
	// without a prior explicit load).
	db := globalCVEDB
	if db == nil {
		if loaded, err := LoadCVEDatabase(); err == nil {
			db = loaded
			globalCVEDB = loaded
		}
	}
	if db == nil {
		if len(staticMatches) == 0 {
			return "[CVE DB] NVD Database not loaded yet, and no static matches found."
		}
		return sb.String()
	}

	matches := db.Search(productVersion)
	if len(matches) > 0 {
		sb.WriteString(fmt.Sprintf("NVD Recent CVE results for '%s' (%d found):\n\n", productVersion, len(matches)))
		for _, m := range matches {
			sb.WriteString(fmt.Sprintf("▸ %s [CVSS: %.1f %s]\n", m.ID, m.CVSSScore, m.CVSSSeverity))
			sb.WriteString(fmt.Sprintf("  Published: %s\n", m.Published[:10]))
			desc := m.Description
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("  %s\n\n", desc))
		}
	}

	if sb.Len() == 0 {
		return fmt.Sprintf("[CVE DB] No CVEs found matching: %s", productVersion)
	}

	return sb.String()
}
