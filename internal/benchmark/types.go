// Package benchmark provides a measurement harness for DrogonClaw's
// autonomous web-app / CTF / vulnerability-discovery capabilities. It runs a
// set of challenges through the agent's ReAct loop and reports a solve rate
// (XBEN-style), so DrogonClaw can be measured and iterated on the same axis as
// other autonomous pentest agents.
package benchmark

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Challenge describes a single benchmark task.
type Challenge struct {
	// ID is a stable short identifier (e.g. "web-001").
	ID string `json:"id"`
	// Class is a coarse category: "web", "code", "api", "ctf-web", "ctf-pwn"...
	Class string `json:"class"`
	// Target is what the agent should attack. For a local server started by a
	// shell command, use Cmd; for a reachable URL/domain, use URL; for source
	// review, use Path.
	URL  string `json:"url,omitempty"`
	Path string `json:"path,omitempty"`
	Cmd  string `json:"cmd,omitempty"`
	// Goal is the natural-language objective given to the agent.
	Goal string `json:"goal"`
	// FlagRegex is the pattern that proves success when found in any tool
	// output or agent message.
	FlagRegex string `json:"flagRegex,omitempty"`
	// MaxTurns is an optional hard cap on reasoning steps.
	MaxTurns int `json:"maxTurns,omitempty"`
}

// Set is a collection of challenges plus metadata.
type Set struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Challenges  []Challenge `json:"challenges"`
}

// Outcome is the per-challenge result.
type Outcome struct {
	ID        string  `json:"id"`
	Class     string  `json:"class"`
	Solved    bool    `json:"solved"`
	Flag      string  `json:"flag,omitempty"`
	Duration  string  `json:"duration"`
	Err       string  `json:"error,omitempty"`
	CostUSD   float64 `json:"costUsd"`
	Turns     int     `json:"turns"`
}

// Summary aggregates a run.
type Summary struct {
	Set         string    `json:"set"`
	RunAt       time.Time `json:"runAt"`
	Total       int       `json:"total"`
	Solved      int       `json:"solved"`
	SuccessRate float64   `json:"successRate"`
	AvgDuration string    `json:"avgDuration"`
	ByClass     map[string]ClassStat `json:"byClass"`
	Outcomes    []Outcome  `json:"outcomes"`
}

// ClassStat holds per-class breakdowns.
type ClassStat struct {
	Total   int     `json:"total"`
	Solved  int     `json:"solved"`
	Rate    float64 `json:"rate"`
}

// defaultFlagPattern mirrors agent.SuccessOracle's default.
var defaultFlagPattern = `(?i)(?:flag|ctf|picoctf|htb)\{[^\r\n{}]{1,200}\}`

// flagRegex returns the compiled pattern for a challenge.
func (c Challenge) flagRegex() *regexp.Regexp {
	pat := c.FlagRegex
	if pat == "" {
		pat = defaultFlagPattern
	}
	return regexp.MustCompile(pat)
}

// findFlag scans text for the challenge's flag pattern.
func (c Challenge) findFlag(text string) string {
	m := c.flagRegex().FindString(text)
	return m
}

// loadSet reads a benchmark set JSON file from disk.
func LoadSet(path string) (*Set, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading benchmark set: %w", err)
	}
	var s Set
	if err := jsonUnmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing benchmark set: %w", err)
	}
	if s.Name == "" {
		s.Name = strings.TrimSuffix(filepath.Base(path), ".json")
	}
	return &s, nil
}
