package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ActionRecord is a compact, durable account of a mission. It intentionally
// stores summaries rather than full model transcripts, keeping recovery useful
// without allowing long-running sessions to grow memory without bound.
type ActionRecord struct {
	ID             string    `json:"id"`
	Objective      string    `json:"objective"`
	Status         string    `json:"status"`
	Plan           []string  `json:"plan,omitempty"`
	CurrentTool    string    `json:"currentTool,omitempty"`
	CurrentArgs    string    `json:"currentArgs,omitempty"`
	CompletedSteps []string  `json:"completedSteps,omitempty"`
	LastError      string    `json:"lastError,omitempty"`
	StartedAt      time.Time `json:"startedAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// ActionJournal checkpoints the current mission before and after each tool.
// A process can never safely resume an arbitrary process mid-tool, so recovery
// resumes from the last completed checkpoint and explicitly identifies the
// interrupted tool for the operator and agent.
type ActionJournal struct {
	mu     sync.RWMutex
	path   string
	record *ActionRecord
}

func NewActionJournal(name string) *ActionJournal {
	if name == "" {
		name = "default"
	}
	_ = os.MkdirAll("data", 0755)
	j := &ActionJournal{path: filepath.Join("data", fmt.Sprintf("actions_%s.json", name))}
	j.load()
	return j
}

func (j *ActionJournal) Begin(objective string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := time.Now().UTC()
	j.record = &ActionRecord{ID: fmt.Sprintf("run_%d", now.UnixNano()), Objective: objective, Status: "running", StartedAt: now, UpdatedAt: now}
	j.saveLocked()
}

func (j *ActionJournal) SetPlan(plan []string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.record == nil {
		return
	}
	j.record.Plan = append([]string(nil), plan...)
	j.touchLocked()
}

func (j *ActionJournal) ToolStarted(tool, args string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.record == nil {
		return
	}
	j.record.CurrentTool, j.record.CurrentArgs = tool, truncateActionText(args, 1000)
	j.record.Status = "running"
	j.touchLocked()
}

func (j *ActionJournal) ToolFinished(tool, result string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.record == nil {
		return
	}
	j.record.CompletedSteps = append(j.record.CompletedSteps, fmt.Sprintf("%s: %s", tool, truncateActionText(result, 2000)))
	j.record.CurrentTool, j.record.CurrentArgs = "", ""
	j.touchLocked()
}

func (j *ActionJournal) Finish(status, errText string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.record == nil {
		return
	}
	j.record.Status, j.record.LastError = status, truncateActionText(errText, 1000)
	j.record.CurrentTool, j.record.CurrentArgs = "", ""
	j.touchLocked()
}

// Recovery returns an interrupted mission once. A mission marked running by a
// previous process is converted to interrupted during startup.
func (j *ActionJournal) Recovery() *ActionRecord {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.record == nil {
		return nil
	}
	if j.record.Status == "running" {
		j.record.Status = "interrupted"
		j.record.LastError = "DrogonClaw stopped before this action completed."
		j.touchLocked()
	}
	if j.record.Status != "interrupted" {
		return nil
	}
	return copyActionRecord(j.record)
}

func (j *ActionJournal) Clear() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.record = nil
	_ = os.Remove(j.path)
}

func (j *ActionJournal) load() {
	b, err := os.ReadFile(j.path)
	if err != nil {
		return
	}
	var record ActionRecord
	if json.Unmarshal(b, &record) == nil {
		j.record = &record
	}
}

func (j *ActionJournal) touchLocked() { j.record.UpdatedAt = time.Now().UTC(); j.saveLocked() }

func (j *ActionJournal) saveLocked() {
	if j.record == nil {
		return
	}
	b, err := json.MarshalIndent(j.record, "", "  ")
	if err != nil {
		return
	}
	tmp := j.path + ".tmp"
	if os.WriteFile(tmp, b, 0600) == nil {
		_ = os.Rename(tmp, j.path)
	}
}

func copyActionRecord(in *ActionRecord) *ActionRecord {
	out := *in
	out.Plan = append([]string(nil), in.Plan...)
	out.CompletedSteps = append([]string(nil), in.CompletedSteps...)
	return &out
}

func truncateActionText(s string, limit int) string {
	s = string([]rune(s))
	if len([]rune(s)) <= limit {
		return s
	}
	return string([]rune(s)[:limit]) + "…"
}
