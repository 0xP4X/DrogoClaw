package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/agent"
	"github.com/0xP4X/drogonclaw-go/internal/config"
	"github.com/0xP4X/drogonclaw-go/internal/core"
	"github.com/0xP4X/drogonclaw-go/internal/ctf"
	"github.com/0xP4X/drogonclaw-go/internal/health"
	"github.com/0xP4X/drogonclaw-go/internal/intel"
	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/0xP4X/drogonclaw-go/internal/opsec"
	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
	"github.com/0xP4X/drogonclaw-go/internal/shell"
	"github.com/0xP4X/drogonclaw-go/internal/skills"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// PromptBuilder is a function type to avoid circular imports with agent package.
type PromptBuilder func() string

// OutputLine is a single rendered line in the output pane.
type OutputLine struct {
	Content string
}

// Model is the root Bubbletea model — the entire DrogonClaw TUI state.
type Model struct {
	// Layout
	width, height int

	// Core components
	viewport viewport.Model
	input    textarea.Model
	spinner  spinner.Model

	// Agent
	orch     *agent.Orchestrator
	graph    *memory.Graph
	opsecMgr *opsec.Manager
	cfg      *config.Manager
	manifest *skills.Manifest

	// Session state
	executing       bool
	autopilot       bool
	noisy           bool
	personaOverride string
	sessionID       string
	phase           string
	phaseDetail     string
	lastStatus      string
	lastObjective   string
	lastTool        string
	lastToolResult  string
	lastPlan        *core.MissionPlan
	recovery        *memory.ActionRecord

	// Output
	lines       []string
	outputReady bool

	// Hints
	hints []cmdHint

	// Error
	lastError string

	// Streaming response accumulator
	currentResponse string

	// Markdown renderer
	mdRenderer *glamour.TermRenderer

	// Tool execution tracking
	execStartTime time.Time
	toolStartTime time.Time
	cancelFn      context.CancelFunc
	activeEvents  chan agent.Event

	// Sandbox reference for commands
	sandbox *sandbox.Docker

	// Prompt refresher (injected from main to avoid circular import)
	promptRefresher func() string

	// Autocomplete state
	selectedHint int

	// Startup banner shown
	bannerShown bool

	// ── HCI improvements ──────────────────────────────────────────────────
	// Scroll tracking: don't auto-scroll when user has scrolled up
	userScrolledUp bool

	// Command history
	history    []string
	historyIdx int    // -1 means "not browsing history"
	historyBuf string // stash current input when entering history

	// Confirmation gate for destructive commands
	pendingConfirm string // e.g. "new", "sandbox"

	// Active tool name for phase-aware spinner
	activeToolName string
	activeToolLine int

	// Double Ctrl+C guard
	ctrlCAt time.Time

	// Active operational mode (set by /mode or auto-analysis)
	activeMode     string
	activeChain    *intel.AttackChain
	targetAnalyzer *intel.TargetAnalyzer
}

type cmdHint struct {
	cmd  string
	desc string
}

// New creates a fresh DrogonClaw TUI model.
func New(
	orch *agent.Orchestrator,
	graph *memory.Graph,
	opsecMgr *opsec.Manager,
	cfg *config.Manager,
	manifest *skills.Manifest,
	sb *sandbox.Docker,
) (*Model, error) {
	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	// Textarea (single-line input for now, expandable)
	ta := textarea.New()
	ta.Placeholder = "Enter mission or /help for commands..."
	ta.Focus()
	ta.CharLimit = 4096
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false) // Enter submits, not newlines

	// Viewport for scrollable output
	vp := viewport.New(120, 30)
	vp.Style = OutputPaneStyle

	// Glamour markdown renderer
	mdRenderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		return nil, fmt.Errorf("glamour init: %w", err)
	}

	sessionID := orch.SessionID

	m := &Model{
		viewport:       vp,
		input:          ta,
		spinner:        sp,
		orch:           orch,
		graph:          graph,
		opsecMgr:       opsecMgr,
		cfg:            cfg,
		manifest:       manifest,
		sessionID:      sessionID,
		mdRenderer:     mdRenderer,
		sandbox:        sb,
		historyIdx:     -1,
		activeToolLine: -1,
		phase:          "idle",
		recovery:       orch.Recovery(),
		targetAnalyzer: &intel.TargetAnalyzer{},
	}

	return m, nil
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		textarea.Blink,
		func() tea.Msg { return showBannerMsg{} },
	)
}

type showBannerMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		layout := calculateLayout(msg.Width, msg.Height)
		m.input.SetWidth(layout.inputWidth)

	case showBannerMsg:
		m.appendBanner()

	case tea.KeyMsg:
		// ── Double Ctrl+C guard ────────────────────────────────────────
		if msg.Type == tea.KeyCtrlC {
			if m.executing && m.cancelFn != nil {
				if time.Since(m.ctrlCAt) < 2*time.Second {
					// Second press within 2s — actually abort
					m.cancelFn()
					m.appendLine(WarningStyle.Render("  [!] Execution halted by user."))
					m.executing = false
					m.activeToolName = ""
					m.ctrlCAt = time.Time{}
				} else {
					// First press — warn
					m.ctrlCAt = time.Now()
					m.appendLine(WarningStyle.Render("  [!] Press Ctrl+C again within 2s to abort execution."))
				}
			} else {
				return m, tea.Quit
			}
			return m, nil
		}

		// ── Confirmation gate intercept ────────────────────────────────
		if m.pendingConfirm != "" {
			if msg.Type == tea.KeyEnter {
				answer := strings.TrimSpace(m.input.Value())
				m.input.Reset()
				if m.confirmationAccepted(answer) {
					switch m.pendingConfirm {
					case "new":
						m.orch.NewSession()
						m.graph.Reset()
						m.recovery = nil
						m.appendLine(ToolDoneStyle.Render("  [OK] Session memory cleared. New session started."))
					case "sandbox":
						isSandboxEnabled := m.cfg.IsSandboxEnabled()
						nextEnabled := !isSandboxEnabled
						nextNative := !nextEnabled
						m.cfg.Set("USE_SANDBOX", fmt.Sprintf("%t", nextEnabled))
						m.executing = true
						m.execStartTime = time.Now()
						if nextEnabled {
							m.activeToolName = "docker-sandbox"
							m.appendLine(SpinnerStyle.Render("  [*] Enabling Docker sandbox..."))
						} else {
							m.activeToolName = "native-kali"
							m.appendLine(WarningStyle.Render("  [*] Switching tools to native Kali mode..."))
						}
						m.pendingConfirm = ""
						return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
							ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
							defer cancel()
							if m.sandbox == nil {
								return SandboxToggleResultMsg{Enabled: nextEnabled, Err: fmt.Errorf("runtime manager is unavailable")}
							}
							err := m.sandbox.Initialize(ctx, nextNative)
							if err != nil && nextEnabled {
								_ = m.sandbox.Initialize(ctx, true)
							}
							return SandboxToggleResultMsg{Enabled: nextEnabled, Err: err}
						})
					}
				} else {
					m.appendLine(SessionStyle.Render("  [·] Cancelled."))
				}
				m.pendingConfirm = ""
				return m, nil
			}
			// Allow typing y/n while confirmation is pending
			var inputCmd tea.Cmd
			m.input, inputCmd = m.input.Update(msg)
			cmds = append(cmds, inputCmd)
			return m, tea.Batch(cmds...)
		}

		// ── Autocomplete navigation (only when hints are visible) ──────
		if !m.executing && len(m.hints) > 0 {
			switch msg.Type {
			case tea.KeyUp:
				m.selectedHint--
				if m.selectedHint < 0 {
					m.selectedHint = len(m.hints) - 1
				}
				return m, nil
			case tea.KeyDown:
				m.selectedHint++
				if m.selectedHint >= len(m.hints) {
					m.selectedHint = 0
				}
				return m, nil
			case tea.KeyTab:
				if m.selectedHint >= 0 && m.selectedHint < len(m.hints) {
					m.input.SetValue(m.hints[m.selectedHint].cmd + " ")
					m.input.SetCursor(len(m.input.Value()))
					m.hints = nil
					m.selectedHint = 0
					return m, nil
				}
			}
		}

		// ── Command history (Alt+↑/Alt+↓ — never conflicts with scroll) ──
		// Plain ↑/↓ is reserved for viewport scrolling so the user can
		// always read past output without needing to be in executing mode.
		if !m.executing && len(m.hints) == 0 {
			if msg.String() == "alt+up" && len(m.history) > 0 {
				if m.historyIdx == -1 {
					// Entering history mode — stash current input
					m.historyBuf = m.input.Value()
					m.historyIdx = len(m.history) - 1
				} else if m.historyIdx > 0 {
					m.historyIdx--
				}
				m.input.SetValue(m.history[m.historyIdx])
				m.input.SetCursor(len(m.input.Value()))
				return m, nil
			}
			if msg.String() == "alt+down" && m.historyIdx != -1 {
				if m.historyIdx < len(m.history)-1 {
					m.historyIdx++
					m.input.SetValue(m.history[m.historyIdx])
				} else {
					// Past the end — restore stashed input
					m.historyIdx = -1
					m.input.SetValue(m.historyBuf)
				}
				m.input.SetCursor(len(m.input.Value()))
				return m, nil
			}
		}

		// ── Submit input ──────────────────────────────────────────────
		if msg.Type == tea.KeyEnter {
			rawInput := strings.TrimSpace(m.input.Value())
			m.input.Reset()
			m.historyIdx = -1
			m.historyBuf = ""
			if rawInput == "" {
				return m, nil
			}

			// Intercept for HitL
			if m.executing && core.GlobalHitL.HasPending() {
				m.appendLine(PromptUserStyle.Render(fmt.Sprintf("  [User] %s", rawInput)))
				core.GlobalHitL.Resolve(rawInput)
				return m, nil
			}

			if !m.executing {
				// Push to history (avoid consecutive duplicates)
				if len(m.history) == 0 || m.history[len(m.history)-1] != rawInput {
					m.history = append(m.history, rawInput)
				}
				return m.handleInput(rawInput)
			}
			return m, nil
		}

		// ── Viewport scrolling — ↑/↓ ALWAYS scroll, PgUp/PgDn for big jumps ──
		// This is the fix for the "scroll vs history" conflict: ↑/↓ no longer
		// navigate command history (which moved to Alt+↑/↓), so they are free
		// to scroll the viewport at all times — executing or idle.
		switch msg.Type {
		case tea.KeyUp:
			m.viewport.LineUp(3)
			m.userScrolledUp = true
			return m, nil
		case tea.KeyDown:
			m.viewport.LineDown(3)
			if m.viewport.AtBottom() {
				m.userScrolledUp = false
			}
			return m, nil
		case tea.KeyPgUp:
			m.viewport.HalfViewUp()
			m.userScrolledUp = true
			return m, nil
		case tea.KeyPgDown:
			m.viewport.HalfViewDown()
			if m.viewport.AtBottom() {
				m.userScrolledUp = false
			}
			return m, nil
		case tea.KeyEnd:
			m.viewport.GotoBottom()
			m.userScrolledUp = false
			return m, nil
		case tea.KeyHome:
			m.viewport.GotoTop()
			m.userScrolledUp = true
			return m, nil
		}

		// Pass keystrokes to input (but don't redraw viewport on every keystroke)
		if !m.executing || core.GlobalHitL.HasPending() {
			var inputCmd tea.Cmd
			m.input, inputCmd = m.input.Update(msg)
			cmds = append(cmds, inputCmd)

			// Live autocomplete on "/" — don't call updateViewportContent() here
			if !m.executing && strings.HasPrefix(m.input.Value(), "/") {
				m.hints = matchHints(m.input.Value())
				if m.selectedHint >= len(m.hints) {
					m.selectedHint = 0
				}
			} else {
				m.hints = nil
				m.selectedHint = 0
			}
		}

	case spinner.TickMsg:
		if m.executing {
			var spinCmd tea.Cmd
			m.spinner, spinCmd = m.spinner.Update(msg)
			cmds = append(cmds, spinCmd)
		}

	case AgentEventMsg:
		cmds = append(cmds, m.handleAgentEvent(msg.Event)...)

	case HealthResultMsg:
		m.activeToolName = ""
		m.executing = false
		if m.cancelFn != nil {
			m.cancelFn()
			m.cancelFn = nil
		}
		m.activeEvents = nil
		m.appendLine(strings.TrimRight(msg.Output, "\n"))
		cmds = append(cmds, textarea.Blink)

	case SandboxToggleResultMsg:
		m.executing = false
		m.activeToolName = ""
		if msg.Err != nil {
			m.cfg.Set("USE_SANDBOX", fmt.Sprintf("%t", !msg.Enabled))
			m.appendLine(ErrorStyle.Render(fmt.Sprintf("  [x] Runtime switch failed: %v", msg.Err)))
			if !msg.Enabled {
				m.appendLine(WarningStyle.Render("  [!] Still using Docker sandbox mode. Use /health to verify runtime state."))
			} else {
				m.appendLine(WarningStyle.Render("  [!] Still using native Kali mode. Use /health to verify runtime state."))
			}
		} else if msg.Enabled {
			m.appendLine(ToolDoneStyle.Render("  [OK] Docker sandbox enabled. Tools now run inside the Kali container."))
		} else {
			m.appendLine(WarningStyle.Render("  [OK] Native Kali mode enabled. Tools now run directly on this machine."))
		}
		cmds = append(cmds, textarea.Blink)

	case tea.QuitMsg:
		return m, tea.Quit
	}

	// Keep Bubble Tea's stored viewport geometry aligned with the rendered
	// layout. This is essential for correct scroll bounds after hints, status
	// lines, confirmation prompts, or the execution rail change height.
	if m.width > 0 {
		contentWidth, contentHeight := m.viewportDimensions()
		m.viewport.Width = contentWidth
		m.viewport.Height = contentHeight
		if _, resized := msg.(tea.WindowSizeMsg); resized && m.mdRenderer != nil {
			m.mdRenderer, _ = glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(max(1, contentWidth-2)),
			)
		}
	}

	// Update viewport scrolling
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) handleInput(raw string) (Model, tea.Cmd) {
	// Display what the user typed
	op := m.graph.GetOperatorProfile()
	ag := m.graph.GetAgentProfile()
	opName := "Unknown"
	if op != nil {
		opName = op.Name
	}
	agName := "drogonclaw"
	if ag != nil {
		agName = strings.ToLower(ag.Name)
	}
	sid := m.sessionID
	if len(sid) > 18 {
		sid = sid[:18]
	}

	promptLine := PromptGlyphStyle.Render("┗━❯ ") + PromptUserStyle.Render(raw)
	_ = opName
	_ = agName
	m.appendLine(promptLine)
	m.hints = nil

	// Slash command routing
	if strings.HasPrefix(raw, "/") {
		return m.handleSlashCommand(raw)
	}

	m.lastObjective = raw
	m.phase = "planning"
	m.phaseDetail = "Mission accepted"
	m.lastPlan = nil

	// Auto-classify target and inject attack chain if no mode is active
	if m.targetAnalyzer != nil && m.activeChain == nil {
		profile := m.targetAnalyzer.Analyze(raw)
		if profile.Class != intel.ClassUnknown && profile.Chain != nil {
			m.activeChain = profile.Chain
			m.activeMode = profile.Chain.Name
			// Inject mode prompt into agent
			if m.promptRefresher != nil {
				base := m.promptRefresher()
				m.orch.UpdateSystemPrompt(base + profile.ModePromptInjection())
			}
			m.appendLine(SidebarTitleStyle.Render(fmt.Sprintf("  [◈] TARGET: %s  →  MODE: %s  (%.0f%% confidence)",
				profile.Raw, profile.Class.String(), profile.Confidence*100)))
			m.appendLine(SpinnerStyle.Render(fmt.Sprintf("  [⛓] Attack chain activated: %s (%d steps)",
				profile.Chain.Name, len(profile.Chain.Steps))))
		}
	}

	// Launch agent
	m.executing = true
	m.execStartTime = time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Minute)
	m.cancelFn = cancel

	events := make(chan agent.Event, 32)
	m.activeEvents = events

	var runFn func()
	if agent.IsChatOnly(raw, m.graph, m.opsecMgr) {
		runFn = func() { m.orch.ExecuteChat(ctx, raw, events) }
	} else {
		runFn = func() { m.orch.Execute(ctx, raw, events) }
	}

	go runFn()

	return *m, tea.Batch(m.spinner.Tick, waitForEvent(events))
}

// waitForEvent returns a Cmd that blocks until one event arrives, then delivers it.
func waitForEvent(events <-chan agent.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			return AgentEventMsg{Event: agent.Event{Type: agent.EvDone}}
		}
		return AgentEventMsg{Event: ev}
	}
}

func (m *Model) handleAgentEvent(ev agent.Event) []tea.Cmd {
	var cmds []tea.Cmd

	switch ev.Type {
	case agent.EvThinking:
		m.lastStatus = ev.Content
		m.phase = phaseFromStatus(ev.Content, "reasoning")
		m.phaseDetail = ev.Content

	case agent.EvPlan:
		m.lastPlan = ev.Plan
		m.phase = "planning"
		if ev.Plan != nil {
			m.phaseDetail = fmt.Sprintf("%d step mission plan", len(ev.Plan.Steps))
		}

	case agent.EvStatus:
		m.lastStatus = ev.Content
		m.phase = phaseFromStatus(ev.Content, m.phase)
		m.phaseDetail = ev.Content

	case agent.EvToolStart:
		// Flush any accumulated text BEFORE tool execution
		if m.currentResponse != "" {
			for _, line := range strings.Split(m.renderAgentResponseString(m.currentResponse), "\n") {
				m.lines = append(m.lines, line)
			}
			m.currentResponse = ""
		}

		m.activeToolName = ev.Tool
		m.lastTool = ev.Tool
		m.phase = "executing"
		m.phaseDetail = ev.Tool
		m.toolStartTime = time.Now()
		m.appendLine(ToolStartStyle.Render(fmt.Sprintf("  • %s…", ev.Tool)))
		m.activeToolLine = len(m.lines) - 1

	case agent.EvToolDone:
		m.activeToolName = ""
		m.lastToolResult = summarizeResult(ev.Result, 220)
		m.phase = "verifying"
		m.phaseDetail = ev.Tool
		elapsed := time.Since(m.toolStartTime)
		result := summarizeResult(ev.Result, 180)
		line := fmt.Sprintf("  • %s completed (%s)", ev.Tool, elapsed.Round(100*time.Millisecond))
		if result != "" {
			line += " — " + result
		}
		if m.activeToolLine >= 0 && m.activeToolLine < len(m.lines) {
			m.lines[m.activeToolLine] = ToolDoneStyle.Render(line)
			m.updateViewportContent()
		} else {
			m.appendLine(ToolDoneStyle.Render(line))
		}
		m.activeToolLine = -1

	case agent.EvToken:
		m.currentResponse += ev.Content
		m.updateViewportContent()

	case agent.EvDone:
		if m.currentResponse != "" {
			for _, line := range strings.Split(m.renderAgentResponseString(m.currentResponse), "\n") {
				m.lines = append(m.lines, line)
			}
			m.currentResponse = ""
		} else if ev.Content != "" {
			for _, line := range strings.Split(m.renderAgentResponseString(ev.Content), "\n") {
				m.lines = append(m.lines, line)
			}
		}
		m.executing = false
		m.phase = "complete"
		m.phaseDetail = summarizeResult(ev.Content, 120)
		m.lastPlan = nil
		m.cancelFn = nil
		cmds = append(cmds, textarea.Blink)
		m.updateViewportContent()
		return cmds // stop listening

	case agent.EvError:
		m.activeToolName = ""
		m.activeToolLine = -1
		m.phase = "error"
		m.phaseDetail = ev.Content
		m.lastPlan = nil
		m.appendLine(ErrorStyle.Render(fmt.Sprintf("  [✗] Error: %s", ev.Content)))
		m.executing = false
		m.cancelFn = nil
		return cmds // stop listening
	}

	// Keep listening for next event
	cmds = append(cmds, waitForEvent(m.eventsCh()))
	return cmds
}

func phaseFromStatus(status, fallback string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.Contains(s, "plan"):
		return "planning"
	case strings.Contains(s, "think"):
		return "reasoning"
	case strings.Contains(s, "tool") || strings.Contains(s, "execute") || strings.Contains(s, "running"):
		return "executing"
	case strings.Contains(s, "approve") || strings.Contains(s, "suspend"):
		return "waiting"
	case strings.Contains(s, "compose") || strings.Contains(s, "aggregate") || strings.Contains(s, "verify"):
		return "verifying"
	case strings.Contains(s, "done") || strings.Contains(s, "complete"):
		return "complete"
	case strings.Contains(s, "error") || strings.Contains(s, "fail"):
		return "error"
	default:
		return fallback
	}
}

func summarizeResult(result string, limit int) string {
	cleaned := strings.TrimSpace(result)
	if cleaned == "" {
		return ""
	}
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return truncate(cleaned, limit)
}

// toolCategory classifies a tool name into a display category for badge coloring.
func toolCategory(name string) (badge string, style lipgloss.Style) {
	switch {
	case strings.HasPrefix(name, "osint_") || name == "web_search" || name == "fetch_url" || name == "deep_research" || name == "lookup_cve":
		return "INTEL", ToolBadgeIntelStyle
	case name == "run_nmap" || name == "run_nuclei" || name == "run_gobuster" || name == "run_ffuf" ||
		name == "run_subfinder" || name == "run_httpx" || name == "run_forensics_triage":
		return "RECON", ToolBadgeReconStyle
	case name == "run_sqlmap" || name == "run_hydra" || name == "run_pwntools":
		return "EXPLOIT", ToolBadgeExploitStyle
	case name == "run_checksec" || name == "run_angr" || name == "run_ropper" || name == "run_one_gadget":
		return "BINARY", ToolBadgeMemoryStyle
	case name == "run_volatility3":
		return "FORENSICS", ToolBadgeIntelStyle
	case strings.HasPrefix(name, "nmap") || strings.HasPrefix(name, "gobuster") || strings.HasPrefix(name, "ffuf") || strings.HasPrefix(name, "nuclei") || strings.HasPrefix(name, "subfinder") || strings.HasPrefix(name, "httpx"):
		return "RECON", ToolBadgeReconStyle
	case name == "shell_execute" || strings.HasPrefix(name, "shell_session") || name == "catch_shell":
		return "SHELL", ToolBadgeExploitStyle
	case name == "update_neural_memory":
		return "MEMORY", ToolBadgeMemoryStyle
	case name == "create_skill" || name == "update_directive" || name == "install_tool" || name == "github_download" || name == "write_and_run_script":
		return "SYSTEM", ToolBadgeSystemStyle
	case name == "download_loot":
		return "LOOT", ToolBadgeIntelStyle
	default:
		return "TOOL", ToolBadgeReconStyle
	}
}

// renderToolStartPanel builds the rich noisy-mode tool execution header.
func (m *Model) renderToolStartPanel(toolName, argsStr string) string {
	w := m.width - 4
	if w < 20 {
		w = 20
	}

	badge, badgeStyle := toolCategory(toolName)

	// ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
	dashes := strings.Repeat("━", max(0, w-len(toolName)-len(badge)-8))
	header := ToolPanelBorderStyle.Render(" ┏━") +
		" " + badgeStyle.Render(" "+badge+" ") +
		" " + ToolPanelHeaderStyle.Render(toolName) +
		" " + ToolPanelBorderStyle.Render(dashes+"━┓")

	var lines []string
	lines = append(lines, header)

	// Parse and display args as key: value pairs
	if argsStr != "" {
		lines = append(lines, ToolPanelBorderStyle.Render(" ┃")+
			ToolArgKeyStyle.Render("  ▸ INPUT"))

		// argsStr is "key: value | key: value" format from formatArgs()
		for _, part := range strings.Split(argsStr, " | ") {
			kv := strings.SplitN(part, ": ", 2)
			if len(kv) == 2 {
				val := kv[1]
				if len(val) > w-20 {
					val = val[:w-20] + "…"
				}
				line := ToolPanelBorderStyle.Render(" ┃") +
					"    " + ToolArgKeyStyle.Render(kv[0]+":") +
					" " + ToolArgValStyle.Render(val)
				lines = append(lines, line)
			}
		}
	}

	lines = append(lines, ToolPanelBorderStyle.Render(" ┃")+
		ToolArgKeyStyle.Render("  ▸ EXECUTING..."))

	return strings.Join(lines, "\n")
}

// renderToolDonePanel builds the rich noisy-mode tool result panel.
func (m *Model) renderToolDonePanel(toolName, result string, elapsed time.Duration) string {
	w := m.width - 4
	if w < 20 {
		w = 20
	}

	var lines []string

	// Output section header
	lines = append(lines, ToolPanelBorderStyle.Render(" ┃")+
		ToolArgKeyStyle.Render("  ▸ OUTPUT"))

	if result != "" {
		// Classify each output line for coloring
		resultLines := strings.Split(strings.TrimRight(result, "\n"), "\n")
		maxLines := 40
		if len(resultLines) > maxLines {
			resultLines = resultLines[:maxLines]
		}

		for _, line := range resultLines {
			if len(line) > w-8 {
				line = line[:w-8] + "…"
			}

			// Smart colorization based on content
			var styled string
			low := strings.ToLower(line)
			switch {
			case strings.Contains(low, "error") || strings.Contains(low, "failed") || strings.Contains(low, "denied"):
				styled = ToolOutputErrorStyle.Render("    " + line)
			case strings.Contains(low, "open") || strings.Contains(low, "found") || strings.Contains(low, "success") ||
				strings.Contains(low, "[+]") || strings.Contains(low, "✓"):
				styled = ToolOutputSuccessStyle.Render("    " + line)
			case strings.HasPrefix(strings.TrimSpace(line), "#") || strings.HasPrefix(strings.TrimSpace(line), "//"):
				styled = ToolTimingStyle.Render("    " + line)
			default:
				styled = ToolOutputStyle.Render("    " + line)
			}

			lines = append(lines, ToolPanelBorderStyle.Render(" ┃")+styled)
		}

		if len(strings.Split(strings.TrimRight(result, "\n"), "\n")) > maxLines {
			lines = append(lines, ToolPanelBorderStyle.Render(" ┃")+
				ToolTimingStyle.Render(fmt.Sprintf("    … %d more lines truncated", len(strings.Split(result, "\n"))-maxLines)))
		}
	} else {
		lines = append(lines, ToolPanelBorderStyle.Render(" ┃")+ToolTimingStyle.Render("    (no output)"))
	}

	// Footer with timing
	elapsedStr := fmt.Sprintf("%.2fs", elapsed.Seconds())
	footer := ToolPanelBorderStyle.Render(" ┗━") +
		ToolDoneStyle.Render(fmt.Sprintf(" ✓ %s DONE ", toolName)) +
		ToolTimingStyle.Render("took "+elapsedStr) +
		ToolPanelBorderStyle.Render(strings.Repeat("━", max(0, w-len(elapsedStr)-len(toolName)-16))+"━┛")

	lines = append(lines, footer)
	return strings.Join(lines, "\n")
}

func (m *Model) renderAgentResponseString(content string) string {
	ag := m.graph.GetAgentProfile()
	agName := "DrogonClaw"
	if ag != nil && ag.Name != "" {
		agName = ag.Name
	}

	var boxLines []string

	// Box top border
	width := max(12, m.width)
	dashes := strings.Repeat("━", max(0, width-len(agName)-6))
	boxLines = append(boxLines, AgentBoxTopStyle.Render(fmt.Sprintf(" ┏━ %s %s┓", agName, dashes)))

	// Render markdown
	rendered := content
	if m.mdRenderer != nil {
		if md, err := m.mdRenderer.Render(content); err == nil {
			rendered = md
		}
	}

	// Each line prefixed with ┃
	for _, line := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
		boxLines = append(boxLines, AgentBoxTopStyle.Render(" ┃  ")+AgentTextStyle.Render(line))
	}

	// Box bottom border
	boxLines = append(boxLines, AgentBoxTopStyle.Render(" ┗"+strings.Repeat("━", max(0, width-2))+"┛"))
	return strings.Join(boxLines, "\n")
}

func (m *Model) appendBanner() {
	if m.bannerShown {
		return
	}
	m.bannerShown = true

	cfg := m.cfg
	provider := cfg.GetProvider()
	model := cfg.GetModel()

	op := m.graph.GetOperatorProfile()
	opName := "Unknown Operator"
	if op != nil && op.Name != "" {
		opName = op.Name
	}

	accent := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(ColorMuted)
	white := lipgloss.NewStyle().Foreground(ColorWhite)
	border := lipgloss.NewStyle().Foreground(ColorDim)
	label := func(k, v string) string {
		return "  " + muted.Render(k) + "  " + white.Render(v)
	}

	sep := border.Render(strings.Repeat("─", max(24, m.width-4)))
	m.appendLine("")
	m.appendLine(accent.Render("  DrogonClaw") + muted.Render("  /  security operations workspace"))
	m.appendLine(muted.Render("  A focused workspace for authorised, evidence-led assessments."))
	m.appendLine(sep)
	m.appendLine(label("OPERATOR", opName) + label("ENGINE", provider+" / "+model))
	m.appendLine(label("RUNTIME ", runtimeLabel(m.sandbox)) + label("SESSION", truncate(m.sessionID, 18)))
	m.appendLine(sep)
	m.appendLine(white.Render("  Start with a clear objective.") + muted.Render("  Use /help to browse commands or /status for workspace details."))
	m.appendLine(muted.Render("  Tip: /profile <domain> gathers passive context before you begin."))
	if m.recovery != nil {
		tool := m.recovery.CurrentTool
		if tool == "" {
			tool = "planning or response generation"
		}
		m.appendLine(WarningStyle.Render(fmt.Sprintf("  [RECOVERY] An earlier mission stopped during %s. Type /resume to continue safely from its last checkpoint.", tool)))
	}
	m.appendLine("")
}

func (m *Model) updateViewportContent() {
	base := strings.Join(m.lines, "\n")
	if m.executing && m.currentResponse != "" {
		base += "\n" + m.renderAgentResponseString(m.currentResponse)
	}
	m.viewport.SetContent(base)
	if !m.userScrolledUp {
		m.viewport.GotoBottom()
	}
}

func (m *Model) appendLine(line string) {
	m.lines = append(m.lines, line)
	m.updateViewportContent()
}

func (m Model) confirmationAccepted(answer string) bool {
	return answer == m.confirmationPhrase()
}

func (m Model) confirmationPhrase() string {
	switch m.pendingConfirm {
	case "new":
		return "CLEAR SESSION"
	case "sandbox":
		return "TOGGLE SANDBOX"
	default:
		return "CONFIRM"
	}
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading DrogonClaw..."
	}

	layout := calculateLayout(m.width, m.height)
	mainWidth := layout.mainWidth

	// Measure the rendered input rather than estimating its line count. Hints,
	// confirmation prompts, and wrapped context lines can all add height. Using
	// the same value for the viewport and sidebar prevents either pane from
	// drifting or being pushed below the terminal while the output scrolls.
	inputArea := m.renderInputArea()
	_, vpHeight := m.viewportDimensions()
	output := m.viewport
	output.Width = max(8, mainWidth-4)
	output.Height = vpHeight
	mainPane := DashboardPaneStyle.Width(max(8, mainWidth-2)).Height(vpHeight).Render(output.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		mainPane,
		inputArea,
	)
}

// viewportDimensions returns the geometry shared by Bubble Tea's scroll model
// and the visible dashboard. Keeping this in one place prevents scroll range
// errors when dynamic UI regions appear or disappear.
func (m Model) viewportDimensions() (width, height int) {
	layout := calculateLayout(m.width, m.height)
	inputHeight := lipgloss.Height(m.renderInputArea())
	return layout.contentWidth, max(3, m.height-inputHeight)
}

func (m Model) renderExecutionRail(width int) string {
	if width < 60 {
		return ""
	}

	objective := m.lastObjective
	if objective == "" {
		objective = "Waiting for a mission objective."
	}
	status := m.lastStatus
	if status == "" {
		status = "Idle"
	}
	tool := "none"
	if m.activeToolName != "" {
		tool = m.activeToolName
	} else if m.lastTool != "" {
		tool = m.lastTool
	}

	phase := m.phase
	if phase == "" {
		phase = "idle"
	}
	phaseLabel, phaseStyle := renderPhaseBadge(phase)
	elapsed := "0s"
	if !m.execStartTime.IsZero() && m.executing {
		elapsed = time.Since(m.execStartTime).Round(time.Second).String()
	}

	var sb strings.Builder
	sb.WriteString(ActivityRailStyle.Width(width).Render(
		ActivityTitleStyle.Render("  LIVE EXECUTION") + "\n" +
			ActivityDimStyle.Render("  what you are seeing is the agent's actual control loop") + "\n\n" +
			fmt.Sprintf("  %s  %s\n", ControlRailLabelStyle.Render("phase"), phaseStyle.Render(phaseLabel)) +
			fmt.Sprintf("  %s  %s\n", ControlRailLabelStyle.Render("objective"), ControlRailValueStyle.Render(truncate(objective, max(20, width-16)))) +
			fmt.Sprintf("  %s  %s\n", ControlRailLabelStyle.Render("status"), ControlRailMutedStyle.Render(truncate(status, max(20, width-16)))) +
			fmt.Sprintf("  %s  %s\n", ControlRailLabelStyle.Render("tool"), ControlRailAccentStyle.Render(truncate(tool, max(12, width-16)))) +
			fmt.Sprintf("  %s  %s\n", ControlRailLabelStyle.Render("elapsed"), ControlRailSuccessStyle.Render(elapsed)),
	))

	if m.lastToolResult != "" {
		sb.WriteString("\n")
		sb.WriteString(ControlRailStyle.Width(width).Render(
			ControlRailLabelStyle.Render("  verification") + "\n  " + ControlRailMutedStyle.Render(truncate(m.lastToolResult, max(20, width-14))),
		))
	}

	if m.lastPlan != nil && len(m.lastPlan.Steps) > 0 {
		sb.WriteString("\n")
		var planLines []string
		limit := min(len(m.lastPlan.Steps), 3)
		planLines = append(planLines, ControlRailLabelStyle.Render("  plan"))
		for i := 0; i < limit; i++ {
			step := m.lastPlan.Steps[i]
			planLines = append(planLines, fmt.Sprintf("  %d. %s", i+1, truncate(step.Action, max(12, width-10))))
		}
		if len(m.lastPlan.Steps) > limit {
			planLines = append(planLines, fmt.Sprintf("  ... %d more steps", len(m.lastPlan.Steps)-limit))
		}
		sb.WriteString(ControlRailStyle.Width(width).Render(strings.Join(planLines, "\n")))
	}

	return sb.String()
}

func renderPhaseBadge(phase string) (string, lipgloss.Style) {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "planning":
		return "planning", PhasePlanningStyle
	case "reasoning":
		return "reasoning", PhaseReasoningStyle
	case "executing", "running":
		return "executing", PhaseExecutingStyle
	case "verifying", "waiting":
		return "verifying", PhaseVerifyingStyle
	case "complete", "done":
		return "complete", PhaseCompleteStyle
	case "error", "failed":
		return "error", PhaseErrorStyle
	default:
		return "idle", ControlRailMutedStyle
	}
}

func renderSidebarPhase(phase string) string {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "planning":
		return "Planning"
	case "reasoning":
		return "Reasoning"
	case "executing", "running":
		return "Executing"
	case "verifying", "waiting":
		return "Verifying"
	case "complete", "done":
		return "Complete"
	case "error", "failed":
		return "Error"
	default:
		return "Idle"
	}
}

func (m Model) renderSidebar(height, width int) string {
	if width <= 0 {
		return ""
	}

	var sb strings.Builder
	row := func(label, value string) {
		sb.WriteString(fmt.Sprintf("  %-12s %s\n", label, value))
	}
	section := func(title string) {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(SidebarTitleStyle.Render("  " + title))
		sb.WriteString("\n")
	}

	toggleOn := func(on bool, onText string) string {
		if on {
			return StatusOnStyle.Render(onText)
		}
		return StatusOffStyle.Render("OFF")
	}

	section("SWITCHES")
	row("Stealth", toggleOn(m.opsecMgr.IsActive(), "ON"))
	row("Autopilot", toggleOn(m.autopilot, "ON"))
	row("Telemetry", toggleOn(m.noisy, "VERBOSE"))

	section("RUNTIME")
	// View is called for every keypress and spinner tick. Do not make a Docker
	// API call here; a slow daemon must never freeze input or redraws.
	row("Engine", lipgloss.NewStyle().Foreground(ColorWhite).Render(runtimeLabel(m.sandbox)))
	row("Network", lipgloss.NewStyle().Foreground(ColorCyan).Render("managed"))

	section("SESSION")
	activeMode := "none"
	if m.activeMode != "" {
		activeMode = m.activeMode
	}
	elapsed := "idle"
	if !m.execStartTime.IsZero() && m.executing {
		elapsed = time.Since(m.execStartTime).Round(time.Second).String()
	}
	row("Mode", lipgloss.NewStyle().Foreground(ColorWhite).Render(truncate(activeMode, max(1, width-16))))
	row("Run time", lipgloss.NewStyle().Foreground(ColorGold).Render(elapsed))
	row("Phase", lipgloss.NewStyle().Foreground(ColorWhite).Render(renderSidebarPhase(m.phase)))

	section("GRAPH")
	row("Entities", StatusNodeStyle.Render(fmt.Sprintf("%d", m.graph.NodeCount())))
	row("Links", lipgloss.NewStyle().Foreground(ColorWhite).Render(fmt.Sprintf("%d", m.graph.EdgeCount())))

	section("GATEWAYS")
	telegramStatus := StatusOffStyle.Render("OFF")
	if m.cfg.GetString("TELEGRAM_TOKEN") != "" && m.cfg.GetString("TELEGRAM_CHAT_ID") != "" {
		telegramStatus = StatusOnStyle.Render("READY")
	}
	row("Telegram", telegramStatus)

	return DashboardPaneStyle.Width(width).Height(height).Render(sb.String())
}

// renderStatusReport is the full-screen equivalent of the former sidebar. It
// keeps operational context available without permanently taking space from the
// conversation and evidence stream.
func (m Model) renderStatusReport() string {
	heading := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	label := lipgloss.NewStyle().Foreground(ColorMuted)
	value := lipgloss.NewStyle().Foreground(ColorWhite)
	rule := lipgloss.NewStyle().Foreground(ColorDim)
	ok := lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)

	onOff := func(on bool, yes string) string {
		if on {
			return ok.Render(yes)
		}
		return StatusOffStyle.Render("OFF")
	}
	row := func(key, val string) string {
		return fmt.Sprintf("  %-14s %s", label.Render(key), val)
	}

	phase, phaseStyle := renderPhaseBadge(m.phase)
	mode := m.activeMode
	if mode == "" {
		mode = "default"
	}
	elapsed := "idle"
	if !m.execStartTime.IsZero() && m.executing {
		elapsed = time.Since(m.execStartTime).Round(time.Second).String()
	}
	tool := m.activeToolName
	if tool == "" {
		tool = m.lastTool
	}
	if tool == "" {
		tool = "none"
	}
	telegramReady := m.cfg.GetString("TELEGRAM_TOKEN") != "" && m.cfg.GetString("TELEGRAM_CHAT_ID") != ""
	width := max(34, m.viewport.Width-2)
	divider := rule.Render("  " + strings.Repeat("─", max(20, width-4)))

	var sb strings.Builder
	sb.WriteString(heading.Render("  Workspace status") + "\n")
	sb.WriteString(label.Render("  Live configuration and session context.") + "\n")
	sb.WriteString(divider + "\n")
	sb.WriteString(heading.Render("  SESSION") + "\n")
	sb.WriteString(row("ID", value.Render(m.sessionID)) + "\n")
	sb.WriteString(row("Phase", phaseStyle.Render(phase)) + "\n")
	sb.WriteString(row("Mode", value.Render(mode)) + "\n")
	sb.WriteString(row("Elapsed", value.Render(elapsed)) + "\n")
	sb.WriteString(row("Active tool", value.Render(tool)) + "\n\n")
	sb.WriteString(heading.Render("  CONTROLS") + "\n")
	sb.WriteString(row("Stealth policy", onOff(m.opsecMgr.IsActive(), "ON")) + "\n")
	sb.WriteString(row("Autopilot", onOff(m.autopilot, "ON")) + "\n")
	sb.WriteString(row("Verbose events", onOff(m.noisy, "ON")) + "\n\n")
	sb.WriteString(heading.Render("  ENVIRONMENT") + "\n")
	sb.WriteString(row("Engine", value.Render(runtimeLabel(m.sandbox))) + "\n")
	sb.WriteString(row("Network", lipgloss.NewStyle().Foreground(ColorCyan).Render("managed")) + "\n")
	sb.WriteString(row("Telegram", onOff(telegramReady, "READY")) + "\n\n")
	sb.WriteString(heading.Render("  MEMORY") + "\n")
	sb.WriteString(row("Entities", StatusNodeStyle.Render(fmt.Sprintf("%d", m.graph.NodeCount()))) + "\n")
	sb.WriteString(row("Links", value.Render(fmt.Sprintf("%d", m.graph.EdgeCount()))) + "\n")
	sb.WriteString(divider + "\n")
	sb.WriteString(label.Render("  /health verifies dependencies · /skills lists available capabilities"))
	return sb.String()
}

func (m Model) renderStatusBar() string {
	phase, _ := renderPhaseBadge(m.phase)
	if m.width < 54 {
		return StatusBarStyle.Width(m.width).Render(fmt.Sprintf(" %s · %s ", phase, runtimeLabel(m.sandbox)))
	}
	policy := "standard"
	if m.opsecMgr.IsActive() {
		policy = "stealth"
	}
	control := "manual"
	if m.autopilot {
		control = "autopilot"
	}
	tool := m.activeToolName
	if tool == "" {
		tool = m.lastTool
	}
	if tool == "" {
		tool = "none"
	}
	bar := fmt.Sprintf(" policy=%s · control=%s · phase=%s · tool=%s · memory=%d · shells=%d · %s",
		policy, control, phase, truncate(tool, 18), m.graph.NodeCount(), shell.GlobalShells.Count(), runtimeLabel(m.sandbox))
	return StatusBarStyle.Width(m.width).Render(truncate(bar, max(1, m.width-2)))
}

func (m Model) renderInputArea() string {
	op := m.graph.GetOperatorProfile()

	// Keep the prompt personal without exposing implementation state. The saved
	// operator profile takes precedence; setup's configured name is the fallback.
	opName := "alias"
	if op != nil && op.Name != "" {
		opName = op.Name
	} else if m.cfg != nil && m.cfg.GetOperatorName() != "" {
		opName = m.cfg.GetOperatorName()
	}

	agName := "drogonclaw"

	// Session IDs are generated as session_<timestamp>; show the useful portion.
	sid := strings.TrimPrefix(m.sessionID, "session_")
	if len(sid) > 8 {
		sid = sid[:8]
	}

	{
		var lines []string
		for i, h := range m.hints {
			prefix := "    "
			cmdStr := HintCmdStyle.Render(h.cmd)
			if i == m.selectedHint {
				prefix = "  > "
				cmdStr = lipgloss.NewStyle().Foreground(ColorBg).Background(ColorAccent).Bold(true).Render(h.cmd)
			}
			pad := strings.Repeat(" ", max(1, 14-len(h.cmd)))
			lines = append(lines, HintBorderStyle.Render(prefix)+cmdStr+pad+HintDescStyle.Render(h.desc))
		}

		contextLine := fmt.Sprintf("  %s@%s  session=%s", opName, agName, sid)
		lines = append(lines, PromptBorderStyle.Render(contextLine))
		if m.lastObjective != "" && !m.executing {
			lines = append(lines, ActivityDimStyle.Render("  objective: "+truncate(m.lastObjective, 88)))
		}
		if m.lastStatus != "" {
			lines = append(lines, ActivityDimStyle.Render("  status: "+truncate(m.lastStatus, 88)))
		}

		var promptGlyph string
		switch {
		case m.pendingConfirm != "":
			promptGlyph = WarningStyle.Render("  confirm > ")
		case m.executing && core.GlobalHitL.HasPending():
			promptGlyph = WarningStyle.Render("  approval > ")
		case m.executing:
			elapsed := int(time.Since(m.execStartTime).Seconds())
			phaseStr := m.phase
			if phaseStr == "" {
				phaseStr = "reasoning"
			}
			if m.activeToolName != "" {
				phaseStr = "running " + m.activeToolName
			}
			promptGlyph = m.spinner.View() + " " + SpinnerStyle.Render(fmt.Sprintf("%s %02d:%02d  ", phaseStr, elapsed/60, elapsed%60))
		default:
			promptGlyph = PromptBorderStyle.Render("  > ")
		}

		lines = append(lines, promptGlyph+m.input.View())
		if m.pendingConfirm != "" {
			lines = append(lines, WarningStyle.Render("  Required: type "+m.confirmationPhrase()+" exactly, or press Enter to cancel."))
		}

		return strings.Join(lines, "\n")
	}

	var lines []string

	// Hint lines above prompt
	for i, h := range m.hints {
		glyph := "┣"
		if i == len(m.hints)-1 {
			glyph = "┗"
		}
		if i == 0 && len(m.hints) > 1 {
			glyph = "┏"
		}
		if len(m.hints) == 1 {
			glyph = "┣"
		}

		prefix := "  " + glyph + "   "
		if i == m.selectedHint {
			prefix = "  " + glyph + " ❯ "
		}
		pad := strings.Repeat(" ", max(0, 12-len(h.cmd)))

		cmdStr := HintCmdStyle.Render(h.cmd)
		if i == m.selectedHint {
			cmdStr = lipgloss.NewStyle().Foreground(ColorBg).Background(ColorAccent).Bold(true).Render(h.cmd)
		}

		lines = append(lines, HintBorderStyle.Render(prefix)+
			cmdStr+pad+
			HintDescStyle.Render(h.desc))
	}

	// Prompt header line
	promptHeader := PromptBorderStyle.Render("┏━ ") +
		PromptAliasStyle.Render(opName) +
		PromptAtStyle.Render("@") +
		PromptAgentStyle.Render(agName) +
		PromptSessionStyle.Render(fmt.Sprintf(" [Session: %s]", sid))

	lines = append(lines, promptHeader)

	// Spinner or static prompt glyph
	var promptGlyph string
	if m.executing && !core.GlobalHitL.HasPending() {
		elapsed := int(time.Since(m.execStartTime).Seconds())
		phaseStr := "Reasoning..."
		if m.activeToolName != "" {
			phaseStr = "Executing " + m.activeToolName + "..."
		}
		promptGlyph = m.spinner.View() + " " + SpinnerStyle.Render(fmt.Sprintf("%s [%02d:%02d]", phaseStr, elapsed/60, elapsed%60))
	} else if m.executing && core.GlobalHitL.HasPending() {
		promptGlyph = WarningStyle.Render("  [HitL] Awaiting Response ❯ ")
	} else {
		promptGlyph = PromptBorderStyle.Render("┗━❯ ")
	}

	inputLine := promptGlyph + m.input.View()
	lines = append(lines, inputLine)

	return strings.Join(lines, "\n")
}

// handleSlashCommand processes /commands.
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
		m.appendLine(SpinnerStyle.Render("  [RECOVERY] Restoring context. The interrupted tool will be verified before any retry."))
		go func() { _ = m.orch.Resume(ctx, events) }()
		m.recovery = nil

	case "/noisy":
		m.noisy = !m.noisy
		state := StatusOffStyle.Render("DISABLED")
		if m.noisy {
			state = StatusOnStyle.Render("ENABLED")
		}
		m.appendLine(SpinnerStyle.Render(fmt.Sprintf("  [⚡] Verbose Telemetry is now %s.", state)))

	case "/stealth":
		on := m.opsecMgr.Toggle()
		if on {
			m.appendLine(ToolDoneStyle.Render("  ⬡ Rate limiting enabled — timing jitter on, concurrency reduced"))
		} else {
			m.appendLine(WarningStyle.Render("  ⬡ Rate limiting disabled — high-concurrency active"))
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
		m.appendLine(WarningStyle.Render(fmt.Sprintf("  [⚠] Autopilot mode: %s", state)))

	case "/persona":
		if args == "" {
			m.appendLine(ErrorStyle.Render("  [x] Usage: /persona <custom directive>"))
		} else {
			m.personaOverride = args
			if m.promptRefresher != nil {
				m.orch.UpdateSystemPrompt(m.promptRefresher())
			}
			m.appendLine(ToolDoneStyle.Render(fmt.Sprintf("  [+] Persona override injected: %s", truncate(args, 60))))
		}

	case "/setup":
		m.appendLine(WarningStyle.Render("  [!] To reconfigure, please exit the session and run 'drogonclaw setup' from your terminal."))

	case "/install":
		m.appendLine(SpinnerStyle.Render(fmt.Sprintf("  [*] Fetching remote plugin from %s...", args)))
		m.appendLine(WarningStyle.Render("  [!] Warning: Executing untrusted 3rd party plugins can compromise your system."))
		m.appendLine(ErrorStyle.Render("  [x] Feature temporarily disabled for security auditing. Cannot load unsigned plugin."))

	case "/report":
		m.phase = "planning"
		m.phaseDetail = "Generating compliance report"
		m.executing = true
		m.execStartTime = time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		m.cancelFn = cancel

		events := make(chan agent.Event, 32)
		m.activeEvents = events

		go func() {
			defer close(events)
			events <- agent.Event{Type: agent.EvStatus, Content: "Drafting compliance-ready penetration test report..."}

			// We can generate this without adding a direct dependency on core by asking the orchestrator
			// Actually, we can use the Orchestrator's provider and graph
			reporter := core.NewReportGenerator(m.orch.GetProvider(), m.graph)
			path, err := reporter.GenerateMarkdownReport(ctx)

			if err != nil {
				events <- agent.Event{Type: agent.EvError, Content: fmt.Sprintf("Report generation failed: %v", err)}
			} else {
				events <- agent.Event{Type: agent.EvDone, Content: fmt.Sprintf("Report successfully generated: %s", path)}
			}
		}()

	case "/ctf":
		if args == "" {
			m.appendLine(ErrorStyle.Render("  [x] Usage: /ctf <local-file-or-directory>"))
		} else {
			m.phase = "planning"
			m.phaseDetail = "Running local triage"
			m.executing = true
			m.execStartTime = time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			m.cancelFn = cancel
			events := make(chan agent.Event, 8)
			m.activeEvents = events

			go func() {
				defer close(events)
				events <- agent.Event{Type: agent.EvStatus, Content: "Running offline local-CTF triage and flag verification..."}
				result, err := ctf.RunLocalTriage(ctx, ctf.LocalTask{Path: args})
				if err != nil {
					events <- agent.Event{Type: agent.EvError, Content: fmt.Sprintf("Local CTF triage failed: %v", err)}
					return
				}
				events <- agent.Event{Type: agent.EvDone, Content: ctf.FormatResult(result)}
			}()
		}

	case "/profile":
		if args == "" {
			m.appendLine(ErrorStyle.Render("  [x] Usage: /profile <domain-ip-or-url>"))
		} else {
			m.phase = "planning"
			m.phaseDetail = "Building passive profile"
			m.executing = true
			m.execStartTime = time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			m.cancelFn = cancel
			events := make(chan agent.Event, 8)
			m.activeEvents = events

			go func() {
				defer close(events)
				if err := ctx.Err(); err != nil {
					events <- agent.Event{Type: agent.EvError, Content: fmt.Sprintf("Target profile cancelled: %v", err)}
					return
				}
				events <- agent.Event{Type: agent.EvStatus, Content: "Building evidence-led passive target profile..."}
				profile, err := intel.BuildPublicProfile(args, m.cfg.GetShodanAPIKey(), m.cfg.GetVirusTotalAPIKey(), intel.DefaultProfileDependencies())
				if err != nil {
					events <- agent.Event{Type: agent.EvError, Content: fmt.Sprintf("Target profile failed: %v", err)}
					return
				}
				if err := ctx.Err(); err != nil {
					events <- agent.Event{Type: agent.EvError, Content: fmt.Sprintf("Target profile cancelled: %v", err)}
					return
				}
				events <- agent.Event{Type: agent.EvDone, Content: intel.FormatPublicProfile(profile)}
			}()
		}

	case "/swarm":
		if args == "" {
			m.appendLine(ErrorStyle.Render("  [x] Usage: /swarm <mission objective>"))
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
				events <- agent.Event{Type: agent.EvStatus, Content: "Engaging Swarm Command. Analyzing mission vectors..."}

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
		switch args {
		case "":
			// List available modes
			m.appendLine(SidebarTitleStyle.Render("  [◈] Available Operational Modes:"))
			for _, name := range intel.ListModes() {
				m.appendLine(HintDescStyle.Render(fmt.Sprintf("      /mode %-20s", name)))
			}
			if m.activeMode != "" {
				m.appendLine(StatusOnStyle.Render(fmt.Sprintf("  [✓] Current mode: %s", m.activeMode)))
			} else {
				m.appendLine(StatusOffStyle.Render("  [○] No mode active. Type /mode <name> to activate."))
			}
		case "off", "clear", "reset":
			m.activeChain = nil
			m.activeMode = ""
			if m.promptRefresher != nil {
				m.orch.UpdateSystemPrompt(m.promptRefresher())
			}
			m.appendLine(WarningStyle.Render("  [○] Operational mode cleared."))
		default:
			chain, ok := intel.GetChainByName(args)
			if !ok {
				m.appendLine(ErrorStyle.Render(fmt.Sprintf("  [✗] Unknown mode: %s. Type /mode for list.", args)))
			} else {
				m.activeChain = chain
				m.activeMode = chain.Name
				// Build a dummy profile to get the injection string
				profile := &intel.TargetProfile{Raw: "manual", Class: chain.Class, Chain: chain, Confidence: 1.0}
				if m.promptRefresher != nil {
					base := m.promptRefresher()
					m.orch.UpdateSystemPrompt(base + profile.ModePromptInjection())
				}
				m.appendLine(StatusOnStyle.Render(fmt.Sprintf("  [◈] MODE ACTIVATED: %s", chain.Name)))
				m.appendLine(SpinnerStyle.Render(fmt.Sprintf("  [⛓] %d-step attack chain loaded:", len(chain.Steps))))
				for _, step := range chain.Steps {
					m.appendLine(HintDescStyle.Render(fmt.Sprintf("      [%02d] %-18s %s", step.Priority, step.Tool, step.Description)))
				}
			}
		}

	case "/analyze":
		if args == "" {
			m.appendLine(ErrorStyle.Render("  [✗] Usage: /analyze <target>"))
		} else {
			profile := m.targetAnalyzer.Analyze(args)
			m.appendLine(SpinnerStyle.Render(profile.Summarize()))
			if profile.Chain != nil {
				m.appendLine(HintDescStyle.Render(fmt.Sprintf("  Next: review with /mode %s, then enter a mission objective.", profile.Chain.Name)))
			}
		}

	case "/sandbox":
		m.pendingConfirm = "sandbox"
		m.appendLine(WarningStyle.Render("  [CONFIRM] This changes where tools execute. Type TOGGLE SANDBOX to continue."))

	case "/status":
		m.appendLine(m.renderStatusReport())

	case "/health":
		m.appendLine(SpinnerStyle.Render("  [*] Running diagnostics..."))

		m.phase = "verifying"
		m.phaseDetail = "Running diagnostics"
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

	case "/help":
		m.appendLine(renderHelp())

	default:
		m.appendLine(WarningStyle.Render(fmt.Sprintf("  [?] Unknown command: %s. Type /help.", cmd)))
	}

	_ = args
	if m.executing && m.activeEvents != nil {
		return *m, tea.Batch(m.spinner.Tick, waitForEvent(m.activeEvents))
	}
	return *m, nil
}

func renderHelp() string {
	var sb strings.Builder
	sb.WriteString(HintBorderStyle.Render("  Commands — type / to browse; Tab accepts a highlighted suggestion") + "\n")
	for _, c := range allHints {
		pad := strings.Repeat(" ", max(1, 16-len(c.cmd)))
		sb.WriteString(HintBorderStyle.Render("    ") + HintCmdStyle.Render(c.cmd) + pad + HintDescStyle.Render(c.desc) + "\n")
	}
	sb.WriteString(HintDescStyle.Render("  ↑/↓ scroll output · Alt+↑/↓ command history · PgUp/PgDn page · End resumes live output · Ctrl+C twice cancels a run"))
	return sb.String()
}

func renderGraphSummary(graph *memory.Graph) string {
	if graph == nil {
		return WarningStyle.Render("  [MEMORY] graph unavailable")
	}

	var sb strings.Builder
	sb.WriteString(HintBorderStyle.Render(fmt.Sprintf("  [MEMORY] %d entities, %d links", graph.NodeCount(), graph.EdgeCount())) + "\n")

	labelCounts := graph.LabelCounts()
	labels := make([]string, 0, len(labelCounts))
	for label := range labelCounts {
		labels = append(labels, string(label))
	}
	sort.Strings(labels)
	if len(labels) > 0 {
		var parts []string
		for _, label := range labels {
			parts = append(parts, fmt.Sprintf("%s=%d", label, labelCounts[memory.NodeLabel(label)]))
		}
		sb.WriteString(ToolArgsStyle.Render("    Entities: "+strings.Join(parts, ", ")) + "\n")
	}

	relCounts := graph.RelationshipCounts()
	rels := make([]string, 0, len(relCounts))
	for rel := range relCounts {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	if len(rels) > 0 {
		var parts []string
		for _, rel := range rels {
			parts = append(parts, fmt.Sprintf("%s=%d", rel, relCounts[rel]))
		}
		sb.WriteString(ToolArgsStyle.Render("    Links: " + strings.Join(parts, ", ")))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func renderSkills(manifest *skills.Manifest, query string) string {
	if manifest == nil || manifest.Count() == 0 {
		return WarningStyle.Render("  [!] No skills are loaded. Regenerate skills_manifest.json and restart DrogonClaw.")
	}

	query = strings.TrimSpace(query)
	if query != "" {
		for _, skill := range manifest.Skills {
			if strings.EqualFold(skill.Name, query) {
				return renderSkillDetail(skill)
			}
		}
		return renderSkillSearch(manifest, query)
	}

	counts := map[string]int{}
	for _, skill := range manifest.Skills {
		counts[classifySkill(skill)]++
	}
	categories := make([]string, 0, len(counts))
	for cat := range counts {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	var sb strings.Builder
	sb.WriteString(HintBorderStyle.Render(fmt.Sprintf("  Skills loaded: %d executable modules", manifest.Count())) + "\n")
	sb.WriteString(HintDescStyle.Render("  Use /skills <term> to search or /skills <exact_name> for parameters.") + "\n\n")
	for _, cat := range categories {
		sb.WriteString(ToolArgsStyle.Render(fmt.Sprintf("    %-22s %d\n", cat, counts[cat])))
	}
	sb.WriteString("\n" + HintDescStyle.Render("  Examples: /skills nmap, /skills credential, /skills autonomous_ad_exploiter"))
	return sb.String()
}

func renderSkillSearch(manifest *skills.Manifest, query string) string {
	q := strings.ToLower(query)
	var matches []skills.Skill
	for _, skill := range manifest.Skills {
		if strings.Contains(strings.ToLower(skill.Name), q) ||
			strings.Contains(strings.ToLower(skill.Description), q) ||
			skillHasParam(skill, q) ||
			strings.Contains(strings.ToLower(classifySkill(skill)), q) {
			matches = append(matches, skill)
		}
	}

	var sb strings.Builder
	sb.WriteString(HintBorderStyle.Render(fmt.Sprintf("  Skills matching %q: %d", query, len(matches))) + "\n")
	if len(matches) == 0 {
		sb.WriteString(WarningStyle.Render("    No matching skill found. Try a domain term like web, ad, cloud, credential, recon, exploit, or report."))
		return sb.String()
	}
	limit := min(len(matches), 12)
	for i := 0; i < limit; i++ {
		skill := matches[i]
		sb.WriteString(ToolArgsStyle.Render(fmt.Sprintf("    %-30s %-18s %s\n", skill.Name, classifySkill(skill), truncate(skill.Description, 70))))
	}
	if len(matches) > limit {
		sb.WriteString(HintDescStyle.Render(fmt.Sprintf("    ...%d more. Use a narrower term or exact skill name for details.", len(matches)-limit)))
	}
	return sb.String()
}

func renderSkillDetail(skill skills.Skill) string {
	var sb strings.Builder
	sb.WriteString(HintBorderStyle.Render("  "+skill.Name) + "\n")
	sb.WriteString(HintDescStyle.Render("  "+skill.Description) + "\n")
	sb.WriteString(ToolArgsStyle.Render(fmt.Sprintf("  Category: %s | Executes via: %s\n", classifySkill(skill), fallback(skill.ExecutesVia, "tool registry"))))
	if len(skill.Parameters) == 0 {
		sb.WriteString(HintDescStyle.Render("  Parameters: none"))
		return sb.String()
	}

	names := make([]string, 0, len(skill.Parameters))
	for name := range skill.Parameters {
		names = append(names, name)
	}
	sort.Strings(names)
	sb.WriteString(HintBorderStyle.Render("  Parameters") + "\n")
	for _, name := range names {
		param := skill.Parameters[name]
		req := "optional"
		if param.Required {
			req = "required"
		}
		sb.WriteString(ToolArgsStyle.Render(fmt.Sprintf("    %-22s %-8s %-9s %s\n", name, param.Type, req, truncate(param.Description, 72))))
	}
	return sb.String()
}

func skillHasParam(skill skills.Skill, query string) bool {
	for name, param := range skill.Parameters {
		if strings.Contains(strings.ToLower(name), query) || strings.Contains(strings.ToLower(param.Description), query) {
			return true
		}
	}
	return false
}

func classifySkill(skill skills.Skill) string {
	text := strings.ToLower(skill.Name + " " + skill.Description)
	switch {
	case strings.Contains(text, "active directory") || strings.Contains(text, "kerberoast") || strings.Contains(text, "ldap") || strings.Contains(text, "domain"):
		return "Active Directory"
	case strings.Contains(text, "cloud") || strings.Contains(text, "aws") || strings.Contains(text, "azure") || strings.Contains(text, "s3") || strings.Contains(text, "iam"):
		return "Cloud"
	case strings.Contains(text, "binary") || strings.Contains(text, "pwn") || strings.Contains(text, "reverse") || strings.Contains(text, "apk"):
		return "Binary/Mobile"
	case strings.Contains(text, "credential") || strings.Contains(text, "brute") || strings.Contains(text, "hash") || strings.Contains(text, "password"):
		return "Credentials"
	case strings.Contains(text, "exploit") || strings.Contains(text, "payload") || strings.Contains(text, "metasploit") || strings.Contains(text, "cve"):
		return "Exploitation"
	case strings.Contains(text, "nmap") || strings.Contains(text, "osint") || strings.Contains(text, "recon") || strings.Contains(text, "subdomain") || strings.Contains(text, "scan"):
		return "Recon"
	case strings.Contains(text, "web") || strings.Contains(text, "sql") || strings.Contains(text, "xss") || strings.Contains(text, "browser") || strings.Contains(text, "ffuf"):
		return "Web"
	case strings.Contains(text, "report") || strings.Contains(text, "memory") || strings.Contains(text, "health"):
		return "Operations"
	default:
		return "General"
	}
}

func fallback(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

var allHints = []cmdHint{
	{"/ctf", "Offline triage and verified flag scan for a local challenge"},
	{"/profile", "Build a passive, evidence-led target profile"},
	{"/mode", "Select or clear an operational mode"},
	{"/analyze", "Classify a target before launching work"},
	{"/stealth", "Toggle cautious execution policy"},
	{"/noisy", "Toggle verbose telemetry"},
	{"/auto", "Toggle autonomous control mode"},
	{"/health", "System diagnostics"},
	{"/status", "Show workspace, runtime, and session details"},
	{"/skills", "List all loaded skills"},
	{"/persona", "Inject custom persona"},
	{"/report", "Generate markdown pentest report"},
	{"/swarm", "Execute mission using parallel agents"},
	{"/sandbox", "Toggle Docker sandbox"},
	{"/new", "Clear session memory"},
	{"/resume", "Continue an interrupted mission from its last checkpoint"},
	{"/clear", "Clear the screen"},
	{"/exit", "Terminate session"},
}

func matchHints(input string) []cmdHint {
	query := strings.ToLower(strings.TrimSpace(input))
	if query == "" || !strings.HasPrefix(query, "/") {
		return nil
	}
	var out []cmdHint
	// Prefix matches preserve the expected command-completion behaviour.
	for _, h := range allHints {
		if strings.HasPrefix(h.cmd, query) {
			out = append(out, h)
			if len(out) >= 5 {
				break
			}
		}
	}
	// If no command prefix matched, search visible descriptions as well. This
	// lets an operator recognize an action by intent (for example, "/memory")
	// without having to recall the exact command name.
	if len(out) == 0 && len(query) > 1 {
		term := strings.TrimPrefix(query, "/")
		for _, h := range allHints {
			if strings.Contains(strings.ToLower(h.cmd), term) || strings.Contains(strings.ToLower(h.desc), term) {
				out = append(out, h)
				if len(out) >= 5 {
					break
				}
			}
		}
	}
	return out
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-1]) + "…"
}

type tuiLayout struct {
	mainWidth, sidebarWidth, contentWidth, contentHeight, inputWidth int
}

// calculateLayout is the single source of truth for responsive geometry.
// The main pane is never allowed to collapse merely to keep the side rail.
func calculateLayout(width, height int) tuiLayout {
	width = max(1, width)
	height = max(1, height)
	l := tuiLayout{mainWidth: width, contentWidth: max(8, width-4), contentHeight: max(3, height-6), inputWidth: max(8, width-6)}
	// Keep the transcript full width. Workspace details live in /status, where
	// they can be read deliberately instead of competing with mission output.
	return l
}

func (m Model) inputAreaHeight() int {
	height := len(m.hints) + 2 // context line + one-line input
	if m.lastObjective != "" && !m.executing {
		height++
	}
	if m.lastStatus != "" {
		height++
	}
	if m.pendingConfirm != "" {
		height++
	}
	return height
}

func runtimeLabel(sb *sandbox.Docker) string {
	if sb == nil {
		return "unavailable"
	}
	if sb.IsNativeMode() {
		return "native host"
	}
	return "Docker sandbox"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// PromptRefreshFn is set by main.go to avoid circular imports.
var PromptRefreshFn func() string

// SetPromptRefresher allows main to inject the prompt builder.
func (m *Model) SetPromptRefresher(fn func() string) {
	m.promptRefresher = fn
}

// eventsCh returns the current active event channel (nil-safe).
func (m *Model) eventsCh() <-chan agent.Event {
	if m.activeEvents == nil {
		// Return a permanently closed channel if no active execution
		ch := make(chan agent.Event)
		close(ch)
		return ch
	}
	return m.activeEvents
}
