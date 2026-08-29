package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/agent"
	"github.com/0xP4X/drogonclaw-go/internal/core"
	"github.com/0xP4X/drogonclaw-go/internal/ctf"
	"github.com/0xP4X/drogonclaw-go/internal/health"
	"github.com/0xP4X/drogonclaw-go/internal/intel"
	tea "github.com/charmbracelet/bubbletea"
)

// ansiRE strips terminal escape sequences so copied output is plain text.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

var needSetup bool

// NeedSetup reports whether the operator requested /setup from inside the TUI.
func NeedSetup() bool { return needSetup }

// Command categories used by the registry and the /help renderer.
const (
	catOperations = "OPERATIONS"
	catControls   = "CONTROLS"
	catSession    = "SESSION"
	catUI         = "UI"
)

// slashCommand is a single entry in the slash-command registry. The first name
// is canonical; the rest are aliases. /help, the command palette, the inline
// hint bar, and the dispatcher all derive from this one table, so a new
// command cannot silently lose its documentation.
type slashCommand struct {
	names    []string
	category string
	desc     string
	args     string // argument hint (e.g. "<target>"); "" when the command takes none
	run      func(m *Model, args string) (*Model, tea.Cmd)
}

func (c slashCommand) canonical() string { return c.names[0] }

func (c slashCommand) matches(name string) bool {
	for _, n := range c.names {
		if n == name {
			return true
		}
	}
	return false
}

// slashCommands is the single source of truth for every in-TUI command. It is
// built in an init() because the /help handler renders via renderHelp, which
// reads this table back — a var-initializer literal would form an
// initialization cycle.
var slashCommands []slashCommand

func init() { slashCommands = buildSlashCommands() }

func buildSlashCommands() []slashCommand {
	var out []slashCommand
	out = append(out, operationsCommands()...)
	out = append(out, controlsCommands()...)
	out = append(out, sessionCommands()...)
	out = append(out, uiCommands()...)
	return out
}

func operationsCommands() []slashCommand {
	return []slashCommand{
		{
			names:    []string{"/mode"},
			category: catOperations,
			desc:     "Select the active attack workflow",
			args:     "[name|off]",
			run: func(m *Model, args string) (*Model, tea.Cmd) {
				m.handleModeCommand(args)
				return m, nil
			},
		},
		{
			names:    []string{"/analyze"},
			category: catOperations,
			desc:     "Classify a target and determine the attack path",
			args:     "<target>",
			run: func(m *Model, args string) (*Model, tea.Cmd) {
				if args == "" {
					m.appendLine(ErrorStyle.Render("  [✗] Usage: /analyze <target>"))
					return m, nil
				}
				profile := m.targetAnalyzer.Analyze(args)
				m.appendLine(SpinnerStyle.Render(profile.Summarize()))
				if profile.Chain != nil {
					m.appendLine(HintDescStyle.Render(fmt.Sprintf("  Next: activate chain with /mode %s", profile.Chain.Name)))
				}
				return m, nil
			},
		},
		{
			names:    []string{"/skills"},
			category: catOperations,
			desc:     "List and search available execution modules",
			args:     "[query]",
			run: func(m *Model, args string) (*Model, tea.Cmd) {
				m.appendLine(renderSkills(m.manifest, args))
				return m, nil
			},
		},
		{
			names:    []string{"/health"},
			category: catOperations,
			desc:     "Verify runtime environment and dependencies",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				m.appendLine(SpinnerStyle.Render("  [*] Running diagnostic checks..."))
				m.phase = "verifying"
				m.phaseDetail = "Diagnostics"
				m.executing = true
				m.execStartTime = time.Now()
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
				m.cancelFn = cancel
				m.activeEvents = nil
				healthWidth := m.viewport.Width
				if healthWidth <= 0 {
					healthWidth = m.width - 4
				}
				return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
					return HealthResultMsg{Output: health.RunDiagnosticsWithWidth(ctx, m.sandbox, healthWidth)}
				})
			},
		},
		{
			names:    []string{"/profile"},
			category: catOperations,
			desc:     "Build a passive intelligence profile",
			args:     "<target>",
			run: func(m *Model, args string) (*Model, tea.Cmd) {
				if args == "" {
					m.appendLine(ErrorStyle.Render("  [✗] Usage: /profile <target-domain-or-ip>"))
					return m, nil
				}
				m2, ctx, events, cmd := m.beginTask(90*time.Second, "planning", "Building target profile", 8)
				go func() {
					defer close(events)
					if err := ctx.Err(); err != nil {
						events <- agent.Event{Type: agent.EvError, Content: "Target profile cancelled: " + err.Error()}
						return
					}
					events <- agent.Event{Type: agent.EvStatus, Content: "Gathering passive intelligence for target..."}
					profile, err := intel.BuildPublicProfile(args, m.cfg.GetShodanAPIKey(), m.cfg.GetVirusTotalAPIKey(), intel.DefaultProfileDependencies())
					if err != nil {
						events <- agent.Event{Type: agent.EvError, Content: fmt.Sprintf("Target profile failed: %v", err)}
						return
					}
					if err := ctx.Err(); err != nil {
						events <- agent.Event{Type: agent.EvError, Content: "Target profile cancelled: " + err.Error()}
						return
					}
					events <- agent.Event{Type: agent.EvDone, Content: intel.FormatPublicProfile(profile)}
				}()
				return m2, cmd
			},
		},
		{
			names:    []string{"/ctf"},
			category: catOperations,
			desc:     "Run local CTF artifact triage",
			args:     "<file-or-dir>",
			run: func(m *Model, args string) (*Model, tea.Cmd) {
				if args == "" {
					m.appendLine(ErrorStyle.Render("  [✗] Usage: /ctf <local-file-or-directory>"))
					return m, nil
				}
				m2, ctx, events, cmd := m.beginTask(2*time.Minute, "planning", "Running CTF triage", 8)
				go func() {
					defer close(events)
					events <- agent.Event{Type: agent.EvStatus, Content: "Running local CTF solver (scan -> decode -> verify)..."}
					rs, err := ctf.Solve(ctx, ctf.LocalTask{Path: args})
					if err != nil {
						events <- agent.Event{Type: agent.EvError, Content: fmt.Sprintf("CTF solve failed: %v", err)}
						return
					}
					events <- agent.Event{Type: agent.EvDone, Content: ctf.FormatSolve(rs)}
				}()
				return m2, cmd
			},
		},
		{
			names:    []string{"/report"},
			category: catOperations,
			desc:     "Generate a structured penetration test report",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				m2, ctx, events, cmd := m.beginTask(5*time.Minute, "planning", "Generating penetration test report", 32)
				go func() {
					defer close(events)
					events <- agent.Event{Type: agent.EvStatus, Content: "Drafting structured penetration test report..."}
					reporter := core.NewReportGenerator(m.orch.GetProvider(), m.graph)
					path, err := reporter.GenerateMarkdownReport(ctx)
					if err != nil {
						events <- agent.Event{Type: agent.EvError, Content: fmt.Sprintf("Report generation failed: %v", err)}
					} else {
						events <- agent.Event{Type: agent.EvDone, Content: fmt.Sprintf("Report generated: %s", path)}
					}
				}()
				return m2, cmd
			},
		},
		{
			names:    []string{"/swarm"},
			category: catOperations,
			desc:     "Dispatch a parallel sub-agent swarm",
			args:     "<objective>",
			run: func(m *Model, args string) (*Model, tea.Cmd) {
				if args == "" {
					m.appendLine(ErrorStyle.Render("  [✗] Usage: /swarm <mission objective>"))
					return m, nil
				}
				m2, ctx, events, cmd := m.beginTask(120*time.Minute, "planning", "Swarm commander engaged", 32)
				objective := args
				go func() {
					defer close(events)
					events <- agent.Event{Type: agent.EvStatus, Content: "Dispatching autonomous sub-agent swarm..."}
					sysPrompt := ""
					if m.promptRefresher != nil {
						sysPrompt = m.promptRefresher()
					}
					commander := agent.NewSwarmCommander(m.orch.GetProvider(), m.orch.GetTools(), sysPrompt, m.sessionID, m.graph)
					result, err := commander.ExecuteSwarm(ctx, objective, events)
					if err != nil {
						events <- agent.Event{Type: agent.EvError, Content: fmt.Sprintf("Swarm failed: %v", err)}
					} else {
						events <- agent.Event{Type: agent.EvDone, Content: result}
					}
				}()
				return m2, cmd
			},
		},
		{
			names:    []string{"/queue"},
			category: catOperations,
			desc:     "Show prompts queued behind the running task",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				if len(m.promptQueue) == 0 {
					m.appendLine(QueueStyle.Render("  [⏳] Prompt queue is empty."))
					return m, nil
				}
				m.appendLine(QueueStyle.Render(fmt.Sprintf("  [⏳] PROMPT QUEUE (%d pending):", len(m.promptQueue))))
				for i, q := range m.promptQueue {
					m.appendLine(QueueItemStyle.Render(fmt.Sprintf("      %d. %s", i+1, q)))
				}
				m.appendLine(HintDescStyle.Render("      Queued prompts run automatically after the current task finishes."))
				return m, nil
			},
		},
		{
			names:    []string{"/benchmarks"},
			category: catOperations,
			desc:     "Show benchmark statistics and Mermaid charts",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				m.appendLine(m.renderBenchmarks())
				return m, nil
			},
		},
		{
			names:    []string{"/timeline"},
			category: catOperations,
			desc:     "Show the execution timeline of tools and findings",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				m.appendLine(m.renderTimeline())
				return m, nil
			},
		},
	}
}

func controlsCommands() []slashCommand {
	return []slashCommand{
		{
			names:    []string{"/stealth"},
			category: catControls,
			desc:     "Toggle evasive rate-limiting policy",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				on := m.opsecMgr.Toggle()
				if on {
					m.appendLine(ToolDoneStyle.Render("  [⬡] Rate limiting enabled — evasive timing active"))
				} else {
					m.appendLine(WarningStyle.Render("  [⬡] Rate limiting disabled — high-concurrency active"))
				}
				if m.promptRefresher != nil {
					m.orch.UpdateSystemPrompt(m.promptRefresher())
				}
				return m, nil
			},
		},
		{
			names:    []string{"/auto"},
			category: catControls,
			desc:     "Toggle autonomous execution mode",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				m.autopilot = !m.autopilot
				m.orch.Autopilot = m.autopilot
				state := "MANUAL"
				if m.autopilot {
					state = "AUTOPILOT"
				}
				m.appendLine(WarningStyle.Render(fmt.Sprintf("  [⚠] Execution Mode: %s", state)))
				return m, nil
			},
		},
		{
			names:    []string{"/sandbox"},
			category: catControls,
			desc:     "Toggle container sandbox execution",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				m.pendingConfirm = "sandbox"
				m.appendLine(WarningStyle.Render("  [CONFIRM] Type TOGGLE SANDBOX to switch execution environment."))
				return m, nil
			},
		},
		{
			names:    []string{"/persona"},
			category: catControls,
			desc:     "Inject a custom agent persona directive",
			args:     "<directive>",
			run: func(m *Model, args string) (*Model, tea.Cmd) {
				if args == "" {
					m.appendLine(ErrorStyle.Render("  [✗] Usage: /persona <custom directive>"))
					return m, nil
				}
				m.personaOverride = args
				if m.promptRefresher != nil {
					m.orch.UpdateSystemPrompt(m.promptRefresher())
				}
				m.appendLine(ToolDoneStyle.Render(fmt.Sprintf("  [+] Persona directive updated: %s", truncate(args, 60))))
				return m, nil
			},
		},
	}
}

func sessionCommands() []slashCommand {
	return []slashCommand{
		{
			names:    []string{"/help", "/commands"},
			category: catSession,
			desc:     "Show the complete command reference",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				m.appendLine(renderHelp())
				return m, nil
			},
		},
		{
			names:    []string{"/status"},
			category: catSession,
			desc:     "Show session, runtime and workspace details",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				for _, line := range strings.Split(m.renderStatusReport(), "\n") {
					m.appendLine(line)
				}
				return m, nil
			},
		},
		{
			names:    []string{"/cost"},
			category: catSession,
			desc:     "Show API token usage and estimated cost",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				if m.tracker == nil {
					m.appendLine(WarningStyle.Render("  [!] No token tracker active."))
				} else {
					m.appendLine(m.tracker.Render())
				}
				return m, nil
			},
		},
		{
			names:    []string{"/sections"},
			category: catSession,
			desc:     "List all previously saved session sections",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				m.appendLine(m.renderSections())
				return m, nil
			},
		},
		{
			names:    []string{"/section"},
			category: catSession,
			desc:     "Switch to a previously saved session section",
			args:     "<id>",
			run: func(m *Model, args string) (*Model, tea.Cmd) {
				if args == "" {
					m.appendLine(ErrorStyle.Render("  [✗] Usage: /section <id> — switch to a previous section"))
					return m, nil
				}
				m.switchSection(args)
				return m, nil
			},
		},
		{
			names:    []string{"/setup"},
			category: catSession,
			desc:     "Run the interactive configuration wizard and restart",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				needSetup = true
				m.appendLine(InfoStyle.Render("  [i] Launching setup wizard — DrogonClaw will restart afterward."))
				return m, tea.Quit
			},
		},
		{
			names:    []string{"/new"},
			category: catSession,
			desc:     "Clear session memory and start clean",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				m.pendingConfirm = "new"
				m.appendLine(WarningStyle.Render("  [CONFIRM] This clears session memory. Type CLEAR SESSION to continue."))
				return m, nil
			},
		},
		{
			names:    []string{"/resume"},
			category: catSession,
			desc:     "Resume the interrupted execution checkpoint",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				if m.recovery == nil {
					m.appendLine(WarningStyle.Render("  [RECOVERY] No interrupted mission is available."))
					return m, nil
				}
				m.lastObjective = m.recovery.Objective
				m2, ctx, events, cmd := m.beginTask(120*time.Minute, "recovering", "Restoring last durable checkpoint", 32)
				m.appendLine(SpinnerStyle.Render("  [RECOVERY] Restoring context. Interrupted tool will be verified before retry."))
				m.recovery = nil
				orch := m.orch
				go func() { _ = orch.Resume(ctx, events) }()
				return m2, cmd
			},
		},
		{
			names:    []string{"/copy"},
			category: catSession,
			desc:     "Copy the full transcript to clipboard / file",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				m.appendLine(m.copyConversation())
				return m, nil
			},
		},
		{
			names:    []string{"/clear"},
			category: catSession,
			desc:     "Clears the visible terminal output",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				m.lines = nil
				m.viewport.SetContent("")
				return m, nil
			},
		},
		{
			names:    []string{"/exit", "/quit"},
			category: catSession,
			desc:     "Terminate the session gracefully",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				return m, tea.Quit
			},
		},
	}
}

func uiCommands() []slashCommand {
	return []slashCommand{
		{
			names:    []string{"/sidebar"},
			category: catUI,
			desc:     "Toggle the sidebar panel",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				if m.showSidebar || m.width >= sidebarMinWidth+20 {
					m.showSidebar = !m.showSidebar
				}
				state := "OFF"
				if m.showSidebar {
					state = "ON"
				}
				m.appendLine(InfoStyle.Render(fmt.Sprintf("  [-sidebar] Sidebar %s", state)))
				return m, nil
			},
		},
		{
			names:    []string{"/details"},
			category: catUI,
			desc:     "Toggle the tool detail panel",
			run: func(m *Model, _ string) (*Model, tea.Cmd) {
				m.showToolDetail = !m.showToolDetail
				state := "OFF"
				if m.showToolDetail {
					state = "ON"
				}
				m.appendLine(InfoStyle.Render(fmt.Sprintf("  [details] Tool details %s", state)))
				return m, nil
			},
		},
	}
}

// beginTask sets up the standard async-task machinery used by the long-running
// slash commands: phase/detail labels, executing state, timer, cancel func,
// and an event channel. It returns the mutated model, the cancellable context
// the task goroutine must honor, the event channel it must drive (and close),
// and the batch command that feeds events to the UI loop.
func (m *Model) beginTask(timeout time.Duration, phase, detail string, cap int) (*Model, context.Context, chan agent.Event, tea.Cmd) {
	m.phase = phase
	m.phaseDetail = detail
	m.executing = true
	m.execStartTime = time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	m.cancelFn = cancel
	events := make(chan agent.Event, cap)
	m.activeEvents = events
	return m, ctx, events, tea.Batch(m.spinner.Tick, waitForEvent(events))
}

// handleSlashCommand dispatches a raw slash input through the command registry.
func (m *Model) handleSlashCommand(raw string) (*Model, tea.Cmd) {
	parts := strings.Fields(raw)
	name := strings.ToLower(parts[0])
	if name == "/" {
		m.appendLine(renderHelp())
		return m, nil
	}
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}
	for _, c := range slashCommands {
		if c.matches(name) {
			return c.run(m, args)
		}
	}
	m.appendLine(WarningStyle.Render(fmt.Sprintf("  [?] Unknown command: %s. Type /help for reference.", name)))
	return m, nil
}

func (m *Model) handleModeCommand(args string) {
	switch args {
	case "":
		m.appendLine(SectionHeaderStyle.Render("  AVAILABLE ATTACK METHODOLOGIES:"))
		for _, name := range intel.ListModes() {
			m.appendLine(HintDescStyle.Render(fmt.Sprintf("      /mode %-20s", name)))
		}
		if m.activeMode != "" {
			m.appendLine(StatusOnStyle.Render(fmt.Sprintf("  [✓] Active methodology: %s", m.activeMode)))
		} else {
			m.appendLine(StatusOffStyle.Render("  [○] No methodology active. Type /mode <name> to activate."))
		}
	case "off", "clear", "reset":
		m.activeChain = nil
		m.activeMode = ""
		if m.promptRefresher != nil {
			m.orch.UpdateSystemPrompt(m.promptRefresher())
		}
		m.appendLine(WarningStyle.Render("  [○] Attack methodology cleared."))
	default:
		chain, ok := intel.GetChainByName(args)
		if !ok {
			m.appendLine(ErrorStyle.Render(fmt.Sprintf("  [✗] Unknown methodology: %s. Type /mode to list.", args)))
			return
		}
		m.activeChain = chain
		m.activeMode = chain.Name
		profile := &intel.TargetProfile{Raw: "manual", Class: chain.Class, Chain: chain, Confidence: 1.0}
		if m.promptRefresher != nil {
			base := m.promptRefresher()
			m.orch.UpdateSystemPrompt(base + profile.ModePromptInjection())
		}
		m.appendLine(StatusOnStyle.Render(fmt.Sprintf("  [◈] METHODOLOGY ACTIVATED: %s", chain.Name)))
		m.appendLine(SpinnerStyle.Render(fmt.Sprintf("  [⛓] %d-step execution chain loaded:", len(chain.Steps))))
		for _, step := range chain.Steps {
			m.appendLine(HintDescStyle.Render(fmt.Sprintf("      [%02d] %-18s %s", step.Priority, step.Tool, step.Description)))
		}
	}
}

func currentSessionID() string {
	return ""
}

// copyConversation dumps the on-screen transcript to a file and tries to push it
// to the system clipboard. DrogonClaw runs in the terminal's alternate screen,
// which has no scrollback, so the only reliable way to copy real output is to
// export it. The plain-text file is always written as a fallback.
func (m *Model) copyConversation() string {
	if len(m.lines) == 0 {
		return WarningStyle.Render("  [!] Nothing to copy yet.")
	}

	plain := stripANSI(stripXMLTags(strings.Join(m.lines, "\n")))

	dir, err := core.LootDir()
	if err != nil {
		dir = "."
	}
	path := filepath.Join(dir, "drogonclaw_transcript.txt")
	if werr := os.WriteFile(path, []byte(plain+"\n"), 0600); werr != nil {
		return ErrorStyle.Render(fmt.Sprintf("  [x] Could not write transcript: %v", werr))
	}

	if copyToClipboard(plain) {
		return ToolOutputSuccessStyle.Render(
			fmt.Sprintf("  [✓] Transcript (%d lines) copied to clipboard and saved to %s", len(m.lines), path))
	}
	return InfoStyle.Render(
		fmt.Sprintf("  [i] Transcript (%d lines) saved to %s — open it outside DrogonClaw to copy (clipboard unavailable here).", len(m.lines), path))
}

// PagerFinishedMsg is sent after the external pager process completes.
type PagerFinishedMsg struct {
	Err  error
	Path string
}

// openViewportInPager writes the current viewport content to a temp file and
// opens it in the user's pager using tea.ExecProcess so terminal raw mode
// is safely suspended and restored.
func (m *Model) openViewportInPager() (tea.Cmd, string) {
	content := m.viewport.View()
	if strings.TrimSpace(content) == "" {
		return nil, WarningStyle.Render("  [!] Nothing to view yet.")
	}

	plain := stripANSI(stripXMLTags(content))

	tmpFile, err := os.CreateTemp("", "drogonclaw-output-*.txt")
	if err != nil {
		return nil, ErrorStyle.Render(fmt.Sprintf("  [x] Could not create temp file: %v", err))
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()

	if werr := os.WriteFile(tmpPath, []byte(plain+"\n"), 0600); werr != nil {
		os.Remove(tmpPath)
		return nil, ErrorStyle.Render(fmt.Sprintf("  [x] Could not write output: %v", werr))
	}

	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less -R"
	}

	c := exec.Command("sh", "-c", pager+" "+tmpPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return PagerFinishedMsg{Err: err, Path: tmpPath}
	}), ""
}

// copyToClipboard attempts a native clipboard tool (xclip/wl-copy/pbcopy/...).
// It returns false if none is available; the transcript file is the fallback.
func copyToClipboard(text string) bool {
	if runtime.GOOS == "darwin" {
		return runClip("pbcopy", text)
	}
	if runtime.GOOS == "linux" {
		if runClip("wl-copy", text) {
			return true
		}
		if runClip("xclip", text, "-selection", "clipboard") {
			return true
		}
		return runClip("xsel", text, "--clipboard", "--input")
	}
	if runtime.GOOS == "windows" {
		return runClip("clip", text)
	}
	return false
}

func runClip(name string, text string, args ...string) bool {
	path, err := exec.LookPath(name)
	if err != nil {
		return false
	}
	cmd := exec.Command(path, args...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run() == nil
}
