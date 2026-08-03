// Package ctf contains deterministic, local-only CTF helpers.
package ctf

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// LocalTask describes a user-supplied local challenge. It intentionally
// performs offline triage only: no network access and no external commands.
type LocalTask struct {
	Path          string
	FlagPattern   string
	ChallengeType string // "forensics", "crypto", "web", "binary", "auto"
}

type Artifact struct {
	Path   string
	Size   int64
	SHA256 string
}

// Observation is a bounded, reproducible record of one local inspection.
// Source is either a filesystem path or a virtual archive member path.
type Observation struct {
	ID       string
	Kind     string
	Source   string
	SHA256   string
	Findings int
	Error    string
}

// CandidateEvidence ties a verified candidate to the observation that
// produced it. A flag is never considered verified from model prose alone.
type CandidateEvidence struct {
	Candidate     string
	ObservationID string
	Source        string
}

type Result struct {
	State        string
	Summary      string
	Artifacts    []Artifact
	Observations []Observation
	Evidence     []CandidateEvidence
	Candidates   []string
	Scanned      int
	Skipped      int
	Duration     time.Duration
}

const defaultFlagPattern = `(?i)(?:flag|ctf|picoctf|htb)\{[^\r\n{}]{1,200}\}`

const (
	maxFiles          = 2000
	maxFileSize       = 32 << 20 // 32 MiB: enough for common fixtures, bounded for responsive UX.
	maxArchiveMembers = 200
	maxArchiveBytes   = 64 << 20
)

// RunLocalTriage creates a reproducible first observation for a local CTF
// artifact. A candidate is reported only when it matches the verifier regex.
func RunLocalTriage(ctx context.Context, task LocalTask) (Result, error) {
	started := time.Now()
	result := Result{}

	if strings.TrimSpace(task.Path) == "" {
		return result, fmt.Errorf("challenge path is required")
	}
	pattern := task.FlagPattern
	if pattern == "" {
		pattern = defaultFlagPattern
	}
	flagRE, err := regexp.Compile(pattern)
	if err != nil {
		return result, fmt.Errorf("invalid flag pattern: %w", err)
	}

	root, err := filepath.Abs(task.Path)
	if err != nil {
		return result, fmt.Errorf("resolve challenge path: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return result, fmt.Errorf("inspect challenge path: %w", err)
	}

	paths := []string{root}
	if info.IsDir() {
		paths = paths[:0]
		err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if d.Type()&os.ModeSymlink != 0 || d.IsDir() {
				return nil
			}
			if len(paths) >= maxFiles {
				return filepath.SkipDir
			}
			paths = append(paths, path)
			return nil
		})
		if err != nil {
			return result, fmt.Errorf("walk challenge directory: %w", err)
		}
	}

	seen := map[string]struct{}{}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		fileInfo, err := os.Stat(path)
		if err != nil || !fileInfo.Mode().IsRegular() || fileInfo.Size() > maxFileSize {
			result.Skipped++
			continue
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			result.Skipped++
			continue
		}
		digest := sha256.Sum256(contents)
		result.Artifacts = append(result.Artifacts, Artifact{Path: path, Size: fileInfo.Size(), SHA256: fmt.Sprintf("%x", digest)})
		result.Scanned++
		challengeType := classifyChallenge(path, contents)
		if challengeType == "binary" {
			binaryObs := validateBinary(path, contents)
			result.Observations = append(result.Observations, binaryObs...)
		}
		scanObservation(&result, seen, flagRE, "file", path, contents)
		inspectArchive(ctx, &result, seen, flagRE, path, contents)
	}

	sort.Strings(result.Candidates)
	result.Duration = time.Since(started)
	if len(result.Candidates) > 0 {
		result.State = "verified"
		result.Summary = fmt.Sprintf("Verified %d flag candidate(s) from %d local artifact(s).", len(result.Candidates), result.Scanned)
	} else {
		result.State = "needs_analysis"
		result.Summary = fmt.Sprintf("No verified flag found in %d local artifact(s); use the artifact inventory to choose the next analysis step.", result.Scanned)
	}
	return result, nil
}

// classifyChallenge determines the challenge type based on file extensions
// and content inspection.
func classifyChallenge(path string, contents []byte) string {
	lower := strings.ToLower(path)
	binaryExts := []string{".elf", ".exe", ".bin", ".so", ".dll", ".o", ".out", ".ko"}
	for _, ext := range binaryExts {
		if strings.HasSuffix(lower, ext) {
			return "binary"
		}
	}
	cryptoExts := []string{".enc", ".cipher", ".encrypted", ".key", ".pem", ".cer", ".crt"}
	for _, ext := range cryptoExts {
		if strings.HasSuffix(lower, ext) {
			return "crypto"
		}
	}
	webExts := []string{".html", ".php", ".js", ".jsp", ".asp", ".aspx", ".py", ".rb"}
	for _, ext := range webExts {
		if strings.HasSuffix(lower, ext) {
			return "web"
		}
	}
	forensicsExts := []string{".pcap", ".pcapng", ".mem", ".vmem", ".img", ".iso", ".jpg", ".png", ".wav", ".mp3", ".zip", ".tar", ".gz", ".rar", ".7z"}
	for _, ext := range forensicsExts {
		if strings.HasSuffix(lower, ext) {
			return "forensics"
		}
	}
	// Inspect content for binary signatures
	if len(contents) >= 4 {
		if bytes.HasPrefix(contents, []byte{0x7f, 'E', 'L', 'F'}) {
			return "binary"
		}
		if bytes.HasPrefix(contents, []byte{'M', 'Z'}) {
			return "binary"
		}
	}
	return "forensics"
}

// validateBinary performs basic binary analysis on the artifact contents.
// It checks for common exploitation primitives and reports findings.
func validateBinary(path string, contents []byte) []Observation {
	var obs []Observation
	digest := sha256.Sum256(contents)
	id := fmt.Sprintf("obs-binary-%d", len(obs)+1)
	observation := Observation{ID: id, Kind: "binary_analysis", Source: path, SHA256: fmt.Sprintf("%x", digest)}

	// Check for common binary exploitation indicators
	content := string(contents)
	indicators := []struct {
		name  string
		pattern string
	}{
		{"NX disabled (executable stack)", "execstack"},
		{"PIE disabled", "No PIE"},
		{"CANARY disabled", "No canary"},
		{"RELRO disabled", "No RELRO"},
		{"Fortify source disabled", "FORTIFY_SOURCE"},
	}
	for _, ind := range indicators {
		if strings.Contains(content, ind.pattern) {
			observation.Findings++
		}
	}

	// Check for embedded files (common in binary CTF challenges)
	if strings.Contains(content, "FLAG") || strings.Contains(content, "flag{") {
		observation.Findings++
	}

	observation.Error = ""
	obs = append(obs, observation)
	return obs
}

func scanObservation(result *Result, seen map[string]struct{}, flagRE *regexp.Regexp, kind, source string, contents []byte) {
	digest := sha256.Sum256(contents)
	id := fmt.Sprintf("obs-%03d", len(result.Observations)+1)
	observation := Observation{ID: id, Kind: kind, Source: source, SHA256: fmt.Sprintf("%x", digest)}
	for _, candidate := range flagRE.FindAllString(string(contents), -1) {
		observation.Findings++
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		result.Candidates = append(result.Candidates, candidate)
		result.Evidence = append(result.Evidence, CandidateEvidence{Candidate: candidate, ObservationID: id, Source: source})
	}
	result.Observations = append(result.Observations, observation)
}

// inspectArchive reads common local archive formats in-process. The limits make
// this safe for untrusted challenge files: no paths are extracted and total
// decompressed content is capped.
func inspectArchive(ctx context.Context, result *Result, seen map[string]struct{}, flagRE *regexp.Regexp, path string, contents []byte) {
	if len(contents) == 0 {
		return
	}
	if zr, err := zip.NewReader(bytes.NewReader(contents), int64(len(contents))); err == nil {
		var total int64
		for i, entry := range zr.File {
			if i >= maxArchiveMembers || entry.FileInfo().IsDir() || entry.UncompressedSize64 > maxFileSize || total+int64(entry.UncompressedSize64) > maxArchiveBytes {
				result.Skipped++
				continue
			}
			if err := ctx.Err(); err != nil {
				return
			}
			r, err := entry.Open()
			if err != nil {
				result.Skipped++
				continue
			}
			member, readErr := io.ReadAll(io.LimitReader(r, maxFileSize+1))
			r.Close()
			if readErr != nil || len(member) > maxFileSize {
				result.Skipped++
				continue
			}
			total += int64(len(member))
			scanObservation(result, seen, flagRE, "zip_member", path+"::"+entry.Name, member)
		}
		return
	}

	reader := bytes.NewReader(contents)
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		gz, err := gzip.NewReader(reader)
		if err != nil {
			return
		}
		decompressed, err := io.ReadAll(io.LimitReader(gz, maxArchiveBytes+1))
		gz.Close()
		if err != nil || len(decompressed) > maxArchiveBytes {
			result.Skipped++
			return
		}
		if strings.HasSuffix(strings.ToLower(strings.TrimSuffix(path, ".gz")), ".tar") {
			inspectTar(ctx, result, seen, flagRE, path, decompressed)
		} else {
			scanObservation(result, seen, flagRE, "gzip_content", path+"::gzip-content", decompressed)
		}
		return
	}
	if strings.HasSuffix(strings.ToLower(path), ".tar") {
		inspectTar(ctx, result, seen, flagRE, path, contents)
	}
}

func inspectTar(ctx context.Context, result *Result, seen map[string]struct{}, flagRE *regexp.Regexp, path string, contents []byte) {
	tr := tar.NewReader(bytes.NewReader(contents))
	var total int64
	for count := 0; count < maxArchiveMembers; count++ {
		if err := ctx.Err(); err != nil {
			return
		}
		header, err := tr.Next()
		if err == io.EOF {
			return
		}
		if err != nil {
			result.Skipped++
			return
		}
		if header.FileInfo().IsDir() || header.Size < 0 || header.Size > maxFileSize || total+header.Size > maxArchiveBytes {
			result.Skipped++
			continue
		}
		member, err := io.ReadAll(io.LimitReader(tr, maxFileSize+1))
		if err != nil || len(member) > maxFileSize {
			result.Skipped++
			continue
		}
		total += int64(len(member))
		scanObservation(result, seen, flagRE, "tar_member", path+"::"+header.Name, member)
	}
}

// FormatResult is intentionally concise so the TUI exposes evidence, not a
// fabricated claim of success.
func FormatResult(result Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[LOCAL CTF TRIAGE — %s]\n%s\nScanned: %d | Skipped: %d | Duration: %s\n", strings.ToUpper(result.State), result.Summary, result.Scanned, result.Skipped, result.Duration.Round(time.Millisecond))
	if len(result.Candidates) > 0 {
		b.WriteString("Verified candidates:\n")
		for _, evidence := range result.Evidence {
			fmt.Fprintf(&b, "- %s (evidence: %s)\n", evidence.Candidate, evidence.Source)
		}
	}
	if len(result.Artifacts) > 0 {
		b.WriteString("Artifacts:\n")
		for _, artifact := range result.Artifacts[:min(10, len(result.Artifacts))] {
			fmt.Fprintf(&b, "- %s (%d bytes, sha256:%s)\n", artifact.Path, artifact.Size, artifact.SHA256[:12])
		}
		if len(result.Artifacts) > 10 {
			fmt.Fprintf(&b, "- … %d additional artifact(s)\n", len(result.Artifacts)-10)
		}
	}
	return strings.TrimSpace(b.String())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
