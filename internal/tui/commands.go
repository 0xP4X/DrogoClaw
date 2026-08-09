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

func (m *Model) handleSlashCommand(raw string) (Model, tea.Cmd) {
	parts := strings.Fields(raw)
	cmd := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	switch cmd {
	case "/exit", "/quit":
		return *m, tea.Quit

	case "/clear":
		m.lines = nil
		m.viewport.SetContent("")

	case "/new":
		m.pendingConfirm = "new"
		m.appendLine(WarningStyle.Render("  [CONFIRM] This clears session memory. Type CLEAR SESSION to continue."))

	case "/resume":
		if m.recovery == nil {
			m.appendLine(WarningStyle.Render("  [RECOVERY] No interrupted mission is available."))
			break
		}
		m.lastObjective = m.recovery.Objective
		m.phase = "recovering"
		m.phaseDetail = "Restoring last durable checkpoint"
		m.executing = true
		m.execStartTime = time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Minute)
		m.cancelFn = cancel
		events := make(chan agent.Event, 32)
		m.activeEvents = events
		m.appendLine(SpinnerStyle.Render("  [RECOVERY] Restoring context. Interrupted tool will be verified before retry."))
		go func() { _ = m.orch.Resume(ctx, events) }()
		m.recovery = nil

	case "/stealth":
		on := m.opsecMgr.Toggle()
		if on {
			m.appendLine(ToolDoneStyle.Render("  [⬡] Rate limiting enabled — evasive timing active"))
		} else {
			m.appendLine(WarningStyle.Render("  [⬡] Rate limiting disabled — high-concurrency active"))
		}
		if m.promptRefresher != nil {
			m.orch.UpdateSystemPrompt(m.promptRefresher())
		}

	case "/auto":
		m.autopilot = !m.autopilot
		m.orch.Autopilot = m.autopilot
		state := "MANUAL"
		if m.autopilot {
			state = "AUTOPILOT"
		}
		m.appendLine(WarningStyle.Render(fmt.Sprintf("  [⚠] Execution Mode: %s", state)))

	case "/persona":
		if args == "" {
			m.appendLine(ErrorStyle.Render("  [✗] Usage: /persona <custom directive>"))
		} else {
			m.personaOverride = args
			if m.promptRefresher != nil {
				m.orch.UpdateSystemPrompt(m.promptRefresher())
			}
			m.appendLine(ToolDoneStyle.Render(fmt.Sprintf("  [+] Persona directive updated: %s", truncate(args, 60))))
		}

	case "/setup":
		m.appendLine(SpinnerStyle.Render("  [*] Launching setup wizard..."))
		return *m, func() tea.Msg {
			cmd := exec.Command(os.Args[0], "setup")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			return SetupResultMsg{Err: err}
		}

	case "/report":
		m.phase = "planning"
		m.phaseDetail = "Generating penetration test report"
		m.executing = true
		m.execStartTime = time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		m.cancelFn = cancel
		events := make(chan agent.Event, 32)
		m.activeEvents = events
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

	case "/ctf":
		if args == "" {
			m.appendLine(ErrorStyle.Render("  [✗] Usage: /ctf <local-file-or-directory>"))
		} else {
			m.phase = "planning"
			m.phaseDetail = "Running CTF triage"
			m.executing = true
			m.execStartTime = time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			m.cancelFn = cancel
			events := make(chan agent.Event, 8)
			m.activeEvents = events
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
		}

	case "/profile":
		if args == "" {
			m.appendLine(ErrorStyle.Render("  [✗] Usage: /profile <target-domain-or-ip>"))
		} else {
			m.phase = "planning"
			m.phaseDetail = "Building target profile"
			m.executing = true
			m.execStartTime = time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			m.cancelFn = cancel
			events := make(chan agent.Event, 8)
			m.activeEvents = events
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
		}

	case "/swarm":
		if args == "" {
			m.appendLine(ErrorStyle.Render("  [✗] Usage: /swarm <mission objective>"))
		} else {
			m.phase = "planning"
			m.phaseDetail = "Swarm commander engaged"
			m.executing = true
			m.execStartTime = time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Minute)
			m.cancelFn = cancel
			events := make(chan agent.Event, 32)
			m.activeEvents = events
			go func() {
				defer close(events)
				events <- agent.Event{Type: agent.EvStatus, Content: "Dispatching autonomous sub-agent swarm..."}
				sysPrompt := ""
				if m.promptRefresher != nil {
					sysPrompt = m.promptRefresher()
				}
				commander := agent.NewSwarmCommander(m.orch.GetProvider(), m.orch.GetTools(), sysPrompt, m.sessionID, m.graph)
				result, err := commander.ExecuteSwarm(ctx, args, events)
				if err != nil {
					events <- agent.Event{Type: agent.EvError, Content: fmt.Sprintf("Swarm failed: %v", err)}
				} else {
					events <- agent.Event{Type: agent.EvDone, Content: result}
				}
			}()
		}

	case "/mode":
		m.handleModeCommand(args)

	case "/analyze":
		if args == "" {
			m.appendLine(ErrorStyle.Render("  [✗] Usage: /analyze <target>"))
		} else {
			profile := m.targetAnalyzer.Analyze(args)
			m.appendLine(SpinnerStyle.Render(profile.Summarize()))
			if profile.Chain != nil {
				m.appendLine(HintDescStyle.Render(fmt.Sprintf("  Next: activate chain with /mode %s", profile.Chain.Name)))
			}
		}

	case "/sandbox":
		m.pendingConfirm = "sandbox"
		m.appendLine(WarningStyle.Render("  [CONFIRM] Type TOGGLE SANDBOX to switch execution environment."))

	case "/status":
		m.appendLine(m.renderStatusReport())

	case "/health":
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
		return *m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			return HealthResultMsg{Output: health.RunDiagnosticsWithWidth(ctx, m.sandbox, healthWidth)}
		})

	case "/skills":
		m.appendLine(renderSkills(m.manifest, args))

	case "/queue":
		if len(m.promptQueue) == 0 {
			m.appendLine(QueueStyle.Render("  [⏳] Prompt queue is empty."))
		} else {
			m.appendLine(QueueStyle.Render(fmt.Sprintf("  [⏳] PROMPT QUEUE (%d pending):", len(m.promptQueue))))
			for i, q := range m.promptQueue {
				m.appendLine(QueueItemStyle.Render(fmt.Sprintf("      %d. %s", i+1, q)))
			}
			m.appendLine(HintDescStyle.Render("      Queued prompts run automatically after the current task finishes."))
		}

	case "/sections":
		m.appendLine(m.renderSections())

	case "/section":
		if args == "" {
			m.appendLine(ErrorStyle.Render("  [✗] Usage: /section <id> — switch to a previous section"))
			break
		}
		m.switchSection(args)

	case "/copy":
		m.appendLine(m.copyConversation())

	case "/help":
		m.appendLine(renderHelp())

	default:
		m.appendLine(WarningStyle.Render(fmt.Sprintf("  [?] Unknown command: %s. Type /help for reference.", cmd)))
	}

	if m.executing && m.activeEvents != nil {
		return *m, tea.Batch(m.spinner.Tick, waitForEvent(m.activeEvents))
	}
	return *m, nil
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
