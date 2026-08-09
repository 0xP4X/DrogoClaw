package ctf

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Step is one typed action in a solving workflow.
type Step struct {
	ID           string
	Kind         string
	Description  string
	Status       string // pending | running | done | failed | skipped
	ObservationID string
	Error        string
}

// Fact is a typed, provenance-tracked claim. Every fact references the
// observation (file or archive member) that produced it, so a solver cannot
// assert a finding without evidence.
type Fact struct {
	Claim        string
	ObservationID string
	Kind         string // flag | encoding | protection | artifact
}

// RunState is the durable state of one solve attempt.
type RunState struct {
	TaskID       string
	Category     string
	Steps        []Step
	Facts        []Fact
	Candidates   []string
	State        string // solved | needs_analysis | failed
	Verified     bool
	VerifierNote string
	Duration     time.Duration
}

func addStep(rs *RunState, kind, desc string) {
	rs.Steps = append(rs.Steps, Step{
		ID:          fmt.Sprintf("step-%d", len(rs.Steps)+1),
		Kind:        kind,
		Description: desc,
		Status:      "running",
	})
}

func doneStep(rs *RunState, kind string) {
	for i := range rs.Steps {
		if rs.Steps[i].Kind == kind && rs.Steps[i].Status == "running" {
			rs.Steps[i].Status = "done"
			return
		}
	}
}

func failStep(rs *RunState, kind string, err error) {
	for i := range rs.Steps {
		if rs.Steps[i].Kind == kind {
			rs.Steps[i].Status = "failed"
			if err != nil {
				rs.Steps[i].Error = err.Error()
			}
			return
		}
	}
}

func pushUnique(s *[]string, v string) {
	for _, x := range *s {
		if x == v {
			return
		}
	}
	*s = append(*s, v)
}

func mustFlagRE(pattern string) *regexp.Regexp {
	if pattern == "" {
		pattern = defaultFlagPattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		re = regexp.MustCompile(defaultFlagPattern)
	}
	return re
}

// Solve runs the local CTF execution kernel: scan -> decode/solve -> verify.
// It operates only on the user-supplied local artifact (no network), and every
// candidate flag is checked by the deterministic verifier with provenance.
func Solve(ctx context.Context, task LocalTask) (RunState, error) {
	started := time.Now()
	flagRE := mustFlagRE(task.FlagPattern)
	rs := RunState{TaskID: task.Path, Category: task.ChallengeType}
	if rs.Category == "" || rs.Category == "auto" {
		rs.Category = classifyPath(task.Path)
	}

	// Step 1: literal + archive flag scan (reuses the triage scanner).
	addStep(&rs, "scan", "Scan files and archives for flag-shaped strings")
	scan, err := RunLocalTriage(ctx, task)
	if err != nil {
		failStep(&rs, "scan", err)
		rs.State = "failed"
		rs.Duration = time.Since(started)
		return rs, err
	}
	for _, c := range scan.Candidates {
		pushUnique(&rs.Candidates, c)
		rs.Facts = append(rs.Facts, Fact{Claim: c, ObservationID: "scan", Kind: "flag"})
	}
	doneStep(&rs, "scan")

	// Step 2: decode / solve — surface flags hidden behind common encodings.
	addStep(&rs, "decode", "Decode base64/hex/rot13/caesar and analyze binaries for hidden flags")
	walkForSolve(ctx, task, flagRE, &rs)
	doneStep(&rs, "decode")

	// Step 3: deterministic verifier (LLM cannot declare success by prose).
	addStep(&rs, "verify", "Deterministic verifier checks flag format and provenance")
	rs.verify(flagRE)
	doneStep(&rs, "verify")

	rs.Duration = time.Since(started)
	if rs.State == "" {
		rs.State = "needs_analysis"
	}
	return rs, nil
}

// verify marks the run solved only when a candidate matches the expected flag
// pattern and is tracked with provenance.
func (rs *RunState) verify(flagRE *regexp.Regexp) {
	for _, c := range rs.Candidates {
		if flagRE.MatchString(c) {
			rs.Verified = true
			rs.State = "solved"
			rs.VerifierNote = "flag matches expected pattern; finding tracked with provenance"
			return
		}
	}
	if rs.State == "" {
		rs.State = "needs_analysis"
	}
	rs.VerifierNote = "no verified flag found"
}

func classifyPath(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "forensics"
	}
	if info.IsDir() {
		return "forensics"
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "forensics"
	}
	return classifyChallenge(path, content)
}

// walkForSolve reads each artifact (and archive members) and runs the decode
// solver plus binary analysis, recording provenance-tracked facts.
func walkForSolve(ctx context.Context, task LocalTask, flagRE *regexp.Regexp, rs *RunState) {
	root, err := filepath.Abs(task.Path)
	if err != nil {
		return
	}
	info, err := os.Stat(root)
	if err != nil {
		return
	}
	paths := []string{root}
	if info.IsDir() {
		paths = nil
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if d.IsDir() {
				return nil
			}
			paths = append(paths, p)
			return nil
		})
	}
	for _, p := range paths {
		if ctx.Err() != nil {
			return
		}
		content, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		obsID := p
		if classifyChallenge(p, content) == "binary" {
			analyzeBinary(content, obsID, rs)
		}
		solveContent(content, obsID, flagRE, rs)
		decodeArchive(p, content, flagRE, rs)
	}
}

var (
	reB64 = regexp.MustCompile(`[A-Za-z0-9+/]{8,}={0,2}`)
	reHex = regexp.MustCompile(`[0-9a-fA-F]{8,}`)
)

// solveContent extracts printable strings and tries common decodings, flagging
// any decoded output that matches the flag pattern. It decodes both the whole
// string and embedded base64/hex tokens so flags hidden inside larger blobs
// (e.g. "data=Q1RGe2...= end") are still recovered.
func solveContent(content []byte, obsID string, flagRE *regexp.Regexp, rs *RunState) {
	for _, s := range extractStrings(content, 4) {
		checkDecode(s, obsID, flagRE, rs)
		for _, tok := range tokenize(s) {
			checkDecode(tok, obsID, flagRE, rs)
		}
	}
}

func checkDecode(s string, obsID string, flagRE *regexp.Regexp, rs *RunState) {
	for _, dec := range tryDecode(s) {
		if flagRE.MatchString(dec) {
			pushUnique(&rs.Candidates, dec)
			rs.Facts = append(rs.Facts, Fact{Claim: dec, ObservationID: obsID, Kind: "encoding"})
		}
	}
}

func tokenize(s string) []string {
	var toks []string
	toks = append(toks, reB64.FindAllString(s, -1)...)
	toks = append(toks, reHex.FindAllString(s, -1)...)
	return toks
}

// analyzeBinary records checksec-style protection states and possible win
// primitives as provenance-tracked facts.
func analyzeBinary(content []byte, obsID string, rs *RunState) {
	indicators := []struct {
		name    string
		pattern string
	}{
		{"NX disabled (executable stack)", "execstack"},
		{"PIE disabled", "No PIE"},
		{"CANARY disabled", "No canary"},
		{"RELRO disabled", "No RELRO"},
		{"FORTIFY_SOURCE disabled", "FORTIFY_SOURCE"},
	}
	low := string(content)
	for _, ind := range indicators {
		if strings.Contains(low, ind.pattern) {
			rs.Facts = append(rs.Facts, Fact{Claim: ind.name, ObservationID: obsID, Kind: "protection"})
		}
	}
	if strings.Contains(low, "flag{") || strings.Contains(low, "FLAG") {
		rs.Facts = append(rs.Facts, Fact{Claim: "flag string present in binary", ObservationID: obsID, Kind: "artifact"})
	}
	for _, w := range []string{"system(", "/bin/sh", "win(", "shell"} {
		if strings.Contains(low, w) {
			rs.Facts = append(rs.Facts, Fact{Claim: "possible win primitive: " + w, ObservationID: obsID, Kind: "artifact"})
		}
	}
}

// decodeArchive recursively solves inside zip / tar / gzip containers.
func decodeArchive(path string, content []byte, flagRE *regexp.Regexp, rs *RunState) {
	lower := strings.ToLower(path)
	if zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content))); err == nil {
		for _, entry := range zr.File {
			r, err := entry.Open()
			if err != nil {
				continue
			}
			member, err := io.ReadAll(io.LimitReader(r, 32<<20))
			r.Close()
			if err != nil {
				continue
			}
			solveContent(member, path+"::"+entry.Name, flagRE, rs)
		}
		return
	}
	if gr, err := gzip.NewReader(bytes.NewReader(content)); err == nil {
		dec, err := io.ReadAll(io.LimitReader(gr, 64<<20))
		gr.Close()
		if err == nil {
			if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
				iterTar(dec, path, flagRE, rs)
			} else {
				solveContent(dec, path+"::gzip", flagRE, rs)
			}
		}
		return
	}
	if strings.HasSuffix(lower, ".tar") {
		iterTar(content, path, flagRE, rs)
	}
}

func iterTar(content []byte, path string, flagRE *regexp.Regexp, rs *RunState) {
	tr := tar.NewReader(bytes.NewReader(content))
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return
		}
		if err != nil {
			return
		}
		if header.FileInfo().IsDir() {
			continue
		}
		member, err := io.ReadAll(io.LimitReader(tr, 32<<20))
		if err != nil {
			continue
		}
		solveContent(member, path+"::"+header.Name, flagRE, rs)
	}
}

// ---------------------------------------------------------------------------
// Decoding primitives
// ---------------------------------------------------------------------------

func extractStrings(content []byte, min int) []string {
	var out []string
	var cur []byte
	flush := func() {
		if len(cur) >= min {
			out = append(out, string(cur))
		}
		cur = nil
	}
	for _, b := range content {
		if (b >= 0x20 && b < 0x7f) || b == '\n' || b == '\t' {
			cur = append(cur, b)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func isPrintable(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	for _, c := range b {
		if c < 0x20 && c != '\n' && c != '\t' && c != '\r' {
			return false
		}
	}
	return true
}

func tryDecode(s string) []string {
	var out []string
	t := strings.TrimSpace(s)
	if dec, err := base64.StdEncoding.DecodeString(t); err == nil && len(dec) > 0 && isPrintable(dec) {
		out = append(out, string(dec))
	}
	if len(t)%2 == 0 {
		if dec, err := hex.DecodeString(t); err == nil && len(dec) > 0 && isPrintable(dec) {
			out = append(out, string(dec))
		}
	}
	out = append(out, rot13(t))
	for shift := 1; shift <= 25; shift++ {
		out = append(out, caesar(t, shift))
	}
	return out
}

func rot13(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return 'a' + (r-'a'+13)%26
		case r >= 'A' && r <= 'Z':
			return 'A' + (r-'A'+13)%26
		}
		return r
	}, s)
}

func caesar(s string, shift int) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return 'a' + (r-'a'+rune(shift))%26
		case r >= 'A' && r <= 'Z':
			return 'A' + (r-'A'+rune(shift))%26
		}
		return r
	}, s)
}

// FormatSolve renders the solve run for the TUI.
func FormatSolve(rs RunState) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[LOCAL CTF SOLVE — %s]\n", strings.ToUpper(rs.State))
	fmt.Fprintf(&b, "Category: %s | Verified: %v | Duration: %s\n", rs.Category, rs.Verified, rs.Duration.Round(time.Millisecond))
	b.WriteString("Steps:\n")
	for _, s := range rs.Steps {
		fmt.Fprintf(&b, "- [%s] %s: %s\n", strings.ToUpper(s.Status), s.Kind, s.Description)
	}
	if rs.VerifierNote != "" {
		fmt.Fprintf(&b, "Verifier: %s\n", rs.VerifierNote)
	}
	if len(rs.Candidates) > 0 {
		b.WriteString("Flag candidates (verified):\n")
		for _, c := range rs.Candidates {
			fmt.Fprintf(&b, "- %s\n", c)
		}
	}
	if len(rs.Facts) > 0 {
		b.WriteString("Facts (provenance-tracked):\n")
		for _, f := range rs.Facts {
			fmt.Fprintf(&b, "- [%s] %s (src: %s)\n", f.Kind, f.Claim, f.ObservationID)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
