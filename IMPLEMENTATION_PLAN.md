# DrogonClaw Implementation Plan

> Comprehensive plan for improving the TUI, logging, and benchmark systems.
> Generated from codebase analysis on 2026-08-26.

---

## Phase 1: High Priority (Core Improvements)

### 1.1 Enable Sidebar with Toggle (Ctrl+B)

**Goal:** Make the sidebar togglable so operators can reclaim screen space when focused on output, while keeping session metadata one keystroke away.

**Files to modify:**

| File | Change |
|------|--------|
| `internal/tui/model.go:27` | Add `showSidebar bool` field to `Model` struct |
| `internal/tui/layout.go:27-51` | Add `sidebarWidth int` constant (e.g., 30 chars); update `calculateLayout()` to set `hasSidebar` based on the toggle and compute `sidebarWidth` / reduced `mainWidth` |
| `internal/tui/view.go:18-60` | Conditionally render sidebar in `View()` — when `showSidebar` is true, split the width between `renderMainPane()` and `renderSidebar()` |
| `internal/tui/styles.go` | Add rendering styles for sidebar sub-components if needed (currently `SidebarPaneStyle` exists but may need border/background) |
| `internal/tui/model.go:159-488` | Add Ctrl+B handler in `Update()` to toggle `m.showSidebar` |

**Implementation details:**

1. Add field to struct:
   ```go
   // In Model struct (model.go:27-86)
   showSidebar bool
   ```

2. Add constant:
   ```go
   const sidebarDefaultWidth = 30
   ```

3. Add Ctrl+B handler in `Update()` (after the existing `tea.KeyF3` handler at view.go:201):
   ```go
   if msg.Type == tea.KeyCtrlB {
       m.showSidebar = !m.showSidebar
       return m, nil
   }
   ```

4. Update `View()` to conditionally include sidebar:
   ```go
   // In View() at view.go:18-60, after computing contentWidth/contentHeight:
   if m.showSidebar && m.width >= 60 {
       sidebarW := min(sidebarDefaultWidth, m.width/3)
       mainW := m.width - sidebarW
       // render main pane at mainW, sidebar at sidebarW
       // join with lipgloss.JoinHorizontal
   }
   ```

5. Update `calculateLayout()` at `layout.go:27-51` to accept and apply sidebar width.

6. Update `sidebarBounds()` at `layout.go:67-69` to return actual dimensions instead of `0, 0`.

---

### 1.2 Persistent Session Log File

**Goal:** Create a structured JSONL log per session that captures every tool execution, agent decision, and finding for post-mortem analysis.

**Files to create:**

| File | Purpose |
|------|---------|
| `internal/logging/session.go` | `SessionLogger` struct with structured JSONL output |

**Files to modify:**

| File | Change |
|------|--------|
| `internal/tui/model.go:27-86` | Add `sessionLog *logging.SessionLogger` field to `Model` |
| `internal/tui/events.go:13-139` | Hook into `handleAgentEvent()` to log tool starts, tool done, errors |
| `internal/tui/commands.go:34-301` | Add `/log` command to view session log in TUI |

**Implementation details:**

1. Create `internal/logging/session.go`:
   ```go
   type SessionLogEntry struct {
       Timestamp time.Time `json:"timestamp"`
       Event     string    `json:"event"`      // "tool_start", "tool_done", "error", "decision"
       Tool      string    `json:"tool,omitempty"`
       Args      string    `json:"args,omitempty"`
       Result    string    `json:"result,omitempty"`
       Duration  string    `json:"duration,omitempty"`
       Success   bool      `json:"success,omitempty"`
       Findings  []string  `json:"findings,omitempty"`
   }

   type SessionLogger struct {
       mu      sync.Mutex
       file    *os.File
       enc     *json.Encoder
       session string
   }
   ```

2. Write to `data/sessions/<sessionID>.jsonl` — create directory in constructor.

3. Integrate into `handleAgentEvent()` at `events.go:47` (EvToolStart) and `events.go:68` (EvToolDone):
   ```go
   case agent.EvToolStart:
       if m.sessionLog != nil {
           m.sessionLog.LogToolStart(ev.Tool, ev.Args)
       }
       // ... existing code
   case agent.EvToolDone:
       if m.sessionLog != nil {
           m.sessionLog.LogToolDone(ev.Tool, ev.Result, time.Since(m.toolStartTime), !isError)
       }
       // ... existing code
   ```

4. Add `/log` command in `commands.go:42` switch:
   ```go
   case "/log":
       if m.sessionLog == nil {
           m.appendLine(WarningStyle.Render("  [!] No active session log."))
       } else {
           m.appendLine(m.sessionLog.RecentEntries(50))
       }
   ```

---

### 1.3 Execution Timeline View (/timeline command)

**Goal:** Provide a formatted, chronological view of all tool executions with durations and key findings, enabling quick post-mission review.

**Files to create:**

| File | Purpose |
|------|---------|
| `internal/tui/timeline.go` | `TimelineEntry` struct and `renderTimeline()` function |

**Files to modify:**

| File | Change |
|------|--------|
| `internal/tui/model.go:27-86` | Add `timeline []TimelineEntry` field |
| `internal/tui/events.go:13-139` | Populate timeline on EvToolStart and EvToolDone |
| `internal/tui/commands.go:42` | Add `/timeline` command case |
| `internal/tui/commands.go:42` | Add `/timeline --json` variant |

**Implementation details:**

1. Create `internal/tui/timeline.go`:
   ```go
   type TimelineEntry struct {
       Timestamp time.Time
       Tool      string
       Args      string
       Result    string
       Duration  time.Duration
       Findings  []string
       Success   bool
       EndTime   time.Time
   }
   ```

2. `renderTimeline()` outputs formatted entries:
   ```
   [00:12] ▶ nmap -sV target.com
   [00:45] ✓ nmap (33s) — 3 ports open
   [00:46] ▶ nuclei -target target.com
   [02:15] ✓ nuclei (1m29s) — CVE-2024-XXXX detected
   ```

3. Add to `Model` struct at `model.go:27`:
   ```go
   timeline []TimelineEntry
   ```

4. Populate in `events.go`:
   - On `EvToolStart` (line 47): append a new `TimelineEntry` with timestamp and tool/args.
   - On `EvToolDone` (line 68): find the matching open entry, set Duration/Result/Success/Findings.

5. Add command in `commands.go`:
   ```go
   case "/timeline":
       if args == "--json" {
           m.appendLine(m.renderTimelineJSON())
       } else {
           for _, line := range strings.Split(m.renderTimeline(), "\n") {
               m.appendLine(line)
           }
       }
   ```

---

### 1.4 Structured Logging with slog

**Goal:** Replace ad-hoc `fmt.Printf` debugging with structured `log/slog` logging, enabling runtime level control, file output, and sensitive field redaction.

**Files to create:**

| File | Purpose |
|------|---------|
| `internal/logging/logger.go` | slog initialization, multi-handler, redaction, session-scoped logger |

**Files to modify:**

| File | Change |
|------|--------|
| `internal/agent/orchestrator.go:177` | Instrument `Execute()` to log each tool call with slog |
| `internal/agent/toolwrappers.go:89` | Instrument `registerToolWrappers()` wrappers to log execution details |
| `internal/tui/commands.go:42` | Add `/debug` command to toggle debug logging |

**Implementation details:**

1. Create `internal/logging/logger.go`:
   ```go
   func Init(level slog.LevelVar, logPath string) {
       // Create MultiHandler: stderr + file
       // Configure redaction for sensitive field names
   }

   func SessionLogger(sessionID string) *slog.Logger {
       return slog.Default().With(slog.String("session_id", sessionID))
   }
   ```

2. Use `slog.LevelVar` for runtime level control — the `/debug` command toggles between `slog.LevelInfo` and `slog.LevelDebug`.

3. Add redaction handler that masks fields matching patterns like `api_key`, `token`, `password`, `secret`.

4. Instrument `orchestrator.go:306` (tool execution loop):
   ```go
   // Before tools.Execute():
   logger.Info("tool_start", "tool", tc.Function.Name, "args", prettyArgs)
   // After tools.Execute():
   logger.Info("tool_done", "tool", tc.Function.Name, "duration", time.Since(start), "success", !isError)
   ```

5. Instrument `toolwrappers.go` — add a log line at the top of each wrapper:
   ```go
   r.builtins["run_nmap"] = func(ctx context.Context, args map[string]any) string {
       logger.Debug("nmap_execute", "target", target, "mode", mode)
       // ... existing code
   }
   ```

6. Add `/debug` command in `commands.go`:
   ```go
   case "/debug":
       logging.ToggleDebug()
       m.appendLine(InfoStyle.Render("  [i] Debug logging toggled."))
   ```

---

### 1.5 Tool Execution Detail Panel

**Goal:** Allow operators to inspect the full, unsanitized output of any tool execution in a dedicated panel, essential for debugging failed commands.

**Files to modify:**

| File | Change |
|------|--------|
| `internal/tui/model.go:27-86` | Add `showToolDetail bool` and `activeToolDetail *ToolDetail` fields |
| `internal/tui/view.go` | Create `renderToolDetailPanel()` function |
| `internal/tui/events.go:47-94` | Capture unsanitized tool output before sanitization |

**Implementation details:**

1. Add structs to `model.go`:
   ```go
   type ToolDetail struct {
       Tool      string
       Args      string
       RawOutput string
       Findings  []string
       Timestamp time.Time
       Duration  time.Duration
   }
   ```

2. In `events.go:85` (EvToolDone), capture `ev.Result` before `sanitizeToolOutputLines()`:
   ```go
   case agent.EvToolDone:
       // Capture raw output for detail panel before sanitization
       m.activeToolDetail = &ToolDetail{
           Tool:      ev.Tool,
           RawOutput: ev.Result,
           Duration:  elapsed,
           Timestamp: m.toolStartTime,
       }
       // ... existing sanitization and rendering
   ```

3. Add toggle in `Update()` (Ctrl+T or `/detail` command):
   ```go
   if msg.Type == tea.KeyCtrlT {
       m.showToolDetail = !m.showToolDetail
       return m, nil
   }
   ```

4. Create `renderToolDetailPanel()` that shows formatted full output with scrollable content.

5. When `showToolDetail` is true and `showSidebar` is true, display in sidebar. When only `showToolDetail`, show as overlay or split view.

---

## Phase 2: Medium Priority (Enhancements)

### 2.1 Improve Streaming Throttle

**Goal:** Replace the character-count-based viewport update throttle with time-based throttling for smoother streaming.

**Files to modify:**

| File | Change |
|------|--------|
| `internal/tui/events.go:96-101` | Replace `len(m.currentResponse)%50 == 0` with time-based check |
| `internal/tui/model.go:27-86` | Add `lastStreamUpdate time.Time` field |

**Implementation details:**

Current code at `events.go:99`:
```go
if len(m.currentResponse)%50 == 0 {
    m.updateViewportContent()
}
```

Replace with:
```go
if time.Since(m.lastStreamUpdate) > 100*time.Millisecond {
    m.updateViewportContent()
    m.lastStreamUpdate = time.Now()
}
```

Add field to `Model`:
```go
lastStreamUpdate time.Time
```

This provides consistent ~10fps updates regardless of token streaming speed.

---

### 2.2 Increase Action Journal Truncation

**Goal:** Preserve more tool output context in the action journal for debugging and recovery.

**Files to modify:**

| File | Change |
|------|--------|
| `internal/memory/actions.go:83` | Change `truncateActionText(result, 600)` to `truncateActionText(result, 2000)` |

**Implementation details:**

Line 83 currently:
```go
j.record.CompletedSteps = append(j.record.CompletedSteps, fmt.Sprintf("%s: %s", tool, truncateActionText(result, 600)))
```

Change to:
```go
j.record.CompletedSteps = append(j.record.CompletedSteps, fmt.Sprintf("%s: %s", tool, truncateActionText(result, 2000)))
```

This preserves 3.3x more output context per step, significantly aiding post-mortem analysis while keeping memory bounded.

---

### 2.3 Persist FailureMemory to Disk

**Goal:** Prevent the agent from retrying failed commands across sessions by persisting failure records.

**Files to modify:**

| File | Change |
|------|--------|
| `internal/memory/failure.go:21-25` | Add `path string` field to `FailureMemory` struct |
| `internal/memory/failure.go` | Add `Load(path)` and `Save()` methods |
| `internal/agent/orchestrator.go:104-124` | Load failures on startup, save after each `Record()` |

**Implementation details:**

1. Add path field:
   ```go
   type FailureMemory struct {
       mu       sync.RWMutex
       failures []FailedAttempt
       counter  int
       path     string
   }
   ```

2. Add persistence methods:
   ```go
   func (f *FailureMemory) Load(path string) {
       f.mu.Lock()
       defer f.mu.Unlock()
       f.path = path
       data, err := os.ReadFile(path)
       if err != nil { return }
       json.Unmarshal(data, &f.failures)
       f.counter = len(f.failures)
   }

   func (f *FailureMemory) Save() {
       f.mu.RLock()
       defer f.mu.RUnlock()
       data, _ := json.MarshalIndent(f.failures, "", "  ")
       os.WriteFile(f.path, data, 0600)
   }
   ```

3. Call `Save()` at the end of `Record()`.

4. In orchestrator or TUI init, call `GlobalFailures.Load("data/failures_<session>.json")`.

---

### 2.4 Wire Billing to Benchmarks

**Goal:** Track API costs during benchmark runs to report per-challenge and total cost.

**Files to modify:**

| File | Change |
|------|--------|
| `internal/benchmark/runner.go:65-118` | Create `billing.Tracker` in `runChallenge()`, wire into orchestrator, populate `Outcome.CostUSD` |
| `internal/benchmark/types.go:46-55` | Verify `Outcome.CostUSD` field exists (it does at line 53) |
| `internal/benchmark/report.go:13-62` | Add cost column to report output |

**Implementation details:**

1. In `runChallenge()` at `runner.go:65`, after creating the orchestrator:
   ```go
   tracker := billing.NewTracker()
   orch.SetTracker(tracker)  // Model.SetTracker exists at model.go:634
   ```

   Note: Need to add `SetTracker` to `Orchestrator` or pass through the existing provider.

2. After `scanEvents()` at `runner.go:109`:
   ```go
   if tracker != nil {
       out.CostUSD = tracker.TotalCost()
   }
   ```

3. In `report.go`, add cost to the challenges table:
   ```go
   // Add Cost column to markdown table
   fmt.Fprintf(md, "| %s | %s | %t | %s | %s | $%.4f |\n", ...)
   ```

---

### 2.5 Enhanced Status Bar

**Goal:** Show step progress during multi-step missions with ETA calculation.

**Files to modify:**

| File | Change |
|------|--------|
| `internal/tui/view.go:289-347` | Enhance `renderStatusBar()` with step counter and ETA |

**Implementation details:**

1. Add fields to `Model`:
   ```go
   currentStep   int
   totalSteps    int
   avgStepDuration time.Duration
   ```

2. Populate from `Event.StepIndex`/`Event.StepTotal` (already defined at `orchestrator.go:43-44`).

3. In `renderStatusBar()` at `view.go:289`, add step counter:
   ```go
   if m.totalSteps > 0 {
       stepInfo = fmt.Sprintf("Step %d/%d", m.currentStep, m.totalSteps)
       if m.avgStepDuration > 0 {
           remaining := time.Duration(m.totalSteps-m.currentStep) * m.avgStepDuration
           eta = fmt.Sprintf(" ETA: %s", remaining.Round(time.Second))
       }
   }
   ```

---

### 2.6 Finding Highlights

**Goal:** Detect and visually highlight vulnerabilities, credentials, and flags in tool output.

**Files to modify:**

| File | Change |
|------|--------|
| `internal/tui/events.go:160-177` | Add regex-based finding detection in `colorizeOutputLine()` |
| `internal/tui/styles.go` | Add `FindingVulnStyle`, `FindingCredentialStyle`, `FindingFlagStyle` |

**Implementation details:**

1. Add styles in `styles.go`:
   ```go
   FindingVulnStyle = lipgloss.NewStyle().Foreground(ColorDanger).Bold(true)
   FindingCredentialStyle = lipgloss.NewStyle().Foreground(ColorGold).Bold(true)
   FindingFlagStyle = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
   ```

2. Define finding type enum:
   ```go
   type FindingType int
   const (
       FindingInfo FindingType = iota
       FindingVulnerability
       FindingCredential
       FindingFlag
   )
   ```

3. In `colorizeOutputLine()` at `events.go:160`, add detection patterns:
   ```go
   // Detect CVE patterns
   if matched, _ := regexp.MatchString(`CVE-\d{4}-\d+`, lower); matched {
       return FindingVulnStyle.Render(raw)
   }
   // Detect credential patterns
   if strings.Contains(lower, "password") || strings.Contains(lower, "credential") {
       return FindingCredentialStyle.Render(raw)
   }
   // Detect flag patterns
   if matched, _ := regexp.MatchString(`(?i)(?:flag|ctf|htb)\{`, lower); matched {
       return FindingFlagStyle.Render(raw)
   }
   ```

---

## Phase 3: Low Priority (Polish)

### 3.1 Historical Benchmark Comparison

**Goal:** Track benchmark results over time and detect performance regressions.

**Files to modify:**

| File | Change |
|------|--------|
| `internal/benchmark/runner.go:31-62` | Save each run with timestamp to `benchmark_runs/` |
| `internal/benchmark/report.go:13-62` | Add `CompareRuns()` function and comparison report |

**Implementation details:**

1. In `Run()` at `runner.go:31`, after generating summary:
   ```go
   runDir := filepath.Join("benchmark_runs", fmt.Sprintf("run_%s", s.RunAt.Format("20060102_150405")))
   s.Write(runDir)
   ```

2. Add `CompareRuns()`:
   ```go
   func CompareRuns(run1, run2 *Summary) string {
       // Compare success rates, per-class stats, duration changes
       // Flag regressions (success rate drop > 5%)
   }
   ```

3. Generate comparison markdown showing deltas between runs.

---

### 3.2 Mermaid Charts in Reports

**Goal:** Generate visual Mermaid diagrams in benchmark reports for execution flow and success distribution.

**Files to modify:**

| File | Change |
|------|--------|
| `internal/benchmark/report.go:13-62` | Add Mermaid chart generation to `Write()` |

**Implementation details:**

1. Add flowchart of execution steps per challenge:
   ```markdown
   ```mermaid
   graph TD
       A[Target Analysis] -->|nmap| B[Port Discovery]
       B -->|nuclei| C[Vuln Scan]
       C -->|sqlmap| D[Exploitation]
   ```
   ```

2. Add pie chart for success rate by class:
   ```markdown
   ```mermaid
   pie title Success Rate by Class
       "web" : 75
       "code" : 90
       "api" : 60
   ```
   ```

3. Add timeline chart for challenge durations.

4. Insert into the markdown report before the challenges table.

---

### 3.3 Dead Code Cleanup

**Goal:** Remove unused functions and consolidate duplicate utility functions.

**Files to modify:**

| File | Change |
|------|--------|
| `internal/tui/view.go:515-581` | Remove `renderInputArea()` — never called |
| `internal/tui/view.go:982-994` | Consolidate `truncate()` usage |
| `internal/tui/model.go:697-699` | Remove `truncateStyled()` — it just calls `truncateVisible()` |

**Implementation details:**

1. `renderInputArea()` at `view.go:515-581` is a dead function. The active version is `renderInputLine()` at `view.go:132-188`. Delete the dead function.

2. `truncateStyled()` at `model.go:697-699`:
   ```go
   func truncateStyled(line string, width int) string {
       return truncateVisible(line, width)
   }
   ```
   This is only called from `renderSidebar()` at `view.go:359`. Replace all calls with `truncateVisible()` directly and remove the wrapper.

3. `truncate()` at `view.go:982-994` is used in several places — keep it but verify it's not duplicating `truncateVisible()` semantics. They differ: `truncate()` operates on runes, `truncateVisible()` on ANSI-aware width. Both are needed.

---

## Implementation Order

Implement in this sequence to maximize dependency satisfaction:

| Order | Phase | Task | Rationale |
|-------|-------|------|-----------|
| 1 | 1.4 | slog structured logging | Foundation for all logging — everything else builds on this |
| 2 | 1.2 | Session log file | Depends on slog; provides data for timeline |
| 3 | 1.3 | Timeline view | Consumes session log data; provides /timeline command |
| 4 | 1.1 | Sidebar toggle | UI foundation for detail panel; independent of logging |
| 5 | 1.5 | Tool detail panel | Uses sidebar infrastructure; captures raw output |
| 6 | 2.1 | Streaming throttle | Quick win, no dependencies |
| 7 | 2.2 | Journal truncation | One-line change |
| 8 | 2.3 | FailureMemory persistence | Standalone memory improvement |
| 9 | 2.4 | Billing to benchmarks | Standalone benchmark improvement |
| 10 | 2.5 | Enhanced status bar | UI polish, depends on Event.StepIndex/StepTotal |
| 11 | 2.6 | Finding highlights | Output styling, can be done anytime |
| 12 | 3.1 | Historical benchmarks | Depends on benchmark infrastructure |
| 13 | 3.2 | Mermaid charts | Report formatting, low priority |
| 14 | 3.3 | Dead code cleanup | Should be done last to avoid conflicts |

---

## Testing Strategy

| Test Type | Scope | Tools |
|-----------|-------|-------|
| Unit tests | `internal/logging/session.go` | Test SessionLogger JSONL output, file creation, concurrent writes |
| Unit tests | `internal/logging/logger.go` | Test slog initialization, level toggle, redaction |
| Unit tests | `internal/tui/timeline.go` | Test TimelineEntry formatting, duration calculation |
| Unit tests | `internal/memory/failure.go` | Test Load/Save roundtrip, concurrent access |
| Integration tests | Timeline rendering | Render timeline with mock entries, verify output format |
| Integration tests | Sidebar toggle | Toggle sidebar, verify layout dimensions update correctly |
| Manual tests | Sidebar toggle (Ctrl+B) | Visual verification across terminal sizes (80x24, 120x40, 200x60) |
| Manual tests | Streaming throttle | Observe smooth streaming at various speeds |
| Manual tests | Tool detail panel | Verify unsanitized output capture |
| Benchmark regression | `internal/benchmark/` | Run benchmarks before/after changes, compare success rates |
| Lint/typecheck | All modified files | `go vet ./...`, `staticcheck ./...`, `go build ./...` |

---

## Risk Assessment

| Change | Risk | Mitigation |
|--------|------|------------|
| slog instrumentation | Low — additive only | Log level defaults to Info; debug is opt-in |
| Sidebar toggle | Low — UI only | Graceful degradation when width < 60 |
| Session log file writes | Low — new file only | Atomic writes via temp file + rename (existing pattern in actions.go:138-149) |
| FailureMemory persistence | Medium — changes global state | Thread-safe via existing mutex; fallback to in-memory on file errors |
| Billing in benchmarks | Medium — new dependency | billing package already imported in model.go:11; wire carefully |
| Dead code removal | Low — remove unused code | Verify no callers with `grep` before deletion |

---

*This plan covers 14 discrete changes across 3 priority phases. Each item is scoped to be implementable and testable independently.*
