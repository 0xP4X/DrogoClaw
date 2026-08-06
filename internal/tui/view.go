package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/core"
	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/0xP4X/drogonclaw-go/internal/skills"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading DrogonClaw..."
	}

	layout := calculateLayout(m.width, m.height, lipgloss.Height(m.renderInputArea()))
	mainWidth := layout.mainWidth
	inputArea := m.renderInputArea()
	_, vpHeight := m.viewportDimensions()
	output := m.viewport
	output.Width = max(8, mainWidth-4)
	output.Height = vpHeight

	var mainPane string
	if m.lines == nil || len(m.lines) == 0 {
		mainPane = MainPaneStyle.Width(max(8, mainWidth-2)).Height(vpHeight).Render(
			m.renderWelcome(),
		)
	} else {
		mainPane = MainPaneStyle.Width(max(8, mainWidth-2)).Height(vpHeight).Render(output.View())
	}

	var sidebar string
	if layout.sidebarWidth > 0 {
		sidebar = m.renderSidebar(layout.sidebarWidth, m.height)
	}

	var headerBar string
	if layout.sidebarWidth > 0 {
		headerBar = m.renderHeader(mainWidth + layout.sidebarWidth + 1)
	} else {
		headerBar = m.renderHeader(m.width)
	}

	statusBar := m.renderStatusBar()

	if sidebar != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			headerBar,
			lipgloss.JoinHorizontal(lipgloss.Left, mainPane, sidebar),
			inputArea,
			statusBar,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		headerBar,
		mainPane,
		inputArea,
		statusBar,
	)
}

func (m Model) renderWelcome() string {
	var sb strings.Builder
	
	title := HeaderBrandStyle.Render("DrogonClaw v2")
	subtitle := WelcomeSubtitleStyle.Render("Offensive Security & Red Team Operations Framework")

	sb.WriteString("\n")
	sb.WriteString(lipgloss.PlaceHorizontal(max(10, m.width-4), lipgloss.Center, title))
	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.PlaceHorizontal(max(10, m.width-4), lipgloss.Center, subtitle))
	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.PlaceHorizontal(max(10, m.width-4), lipgloss.Center, WelcomeHintStyle.Render("Enter a mission objective or type / for commands.")))
	sb.WriteString("\n\n")

	quickStart := lipgloss.JoinVertical(lipgloss.Left,
		WelcomeQuickStartStyle.Render("QUICK COMMANDS:"),
		fmt.Sprintf("  %-12s %s", HintCmdStyle.Render("/health"), HintDescStyle.Render("Verify runtime & dependencies")),
		fmt.Sprintf("  %-12s %s", HintCmdStyle.Render("/mode"), HintDescStyle.Render("Select attack methodology")),
		fmt.Sprintf("  %-12s %s", HintCmdStyle.Render("/skills"), HintDescStyle.Render("Browse loaded attack modules")),
		fmt.Sprintf("  %-12s %s", HintCmdStyle.Render("/status"), HintDescStyle.Render("Show session & graph metrics")),
	)

	sb.WriteString(lipgloss.PlaceHorizontal(max(10, m.width-4), lipgloss.Center, quickStart))

	return sb.String()
}

func (m Model) viewportDimensions() (width, height int) {
	layout := calculateLayout(m.width, m.height, lipgloss.Height(m.renderInputArea()))
	return layout.contentWidth, layout.contentHeight
}

func (m Model) renderHeader(width int) string {
	if width <= 0 {
		return ""
	}
	op := m.graph.GetOperatorProfile()
	opName := "operator"
	if op != nil && op.Name != "" {
		opName = op.Name
	}
	ag := m.graph.GetAgentProfile()
	agName := "drogonclaw"
	if ag != nil && ag.Name != "" {
		agName = strings.ToLower(ag.Name)
	}
	sid := m.sessionID
	if len(sid) > 18 {
		sid = sid[:18]
	}
	provider := m.cfg.GetProvider()
	model := m.cfg.GetModel()
	runtime := runtimeLabel(m.sandbox)

	sep := HeaderSepStyle.Render(" │ ")

	parts := []string{
		HeaderBrandStyle.Render("DROGONCLAW"),
		HeaderInfoStyle.Render(fmt.Sprintf("%s@%s", opName, agName)),
		HeaderDimStyle.Render(fmt.Sprintf("section:%s", sid)),
	}

	if provider != "" {
		parts = append(parts, HeaderInfoStyle.Render(fmt.Sprintf("%s/%s", provider, model)))
	}
	parts = append(parts, HeaderInfoStyle.Render(runtime))

	line := strings.Join(parts, sep)
	if lipgloss.Width(line) > width {
		// Fallback for tight spaces
		line = HeaderBrandStyle.Render("DROGONCLAW") + sep + HeaderInfoStyle.Render(fmt.Sprintf("%s/%s", provider, model))
	}

	return HeaderBarBorderStyle.Width(width).Render(
		HeaderBarStyle.Width(width).Render(line),
	)
}

func (m Model) renderStatusBar() string {
	phase, _ := renderPhaseBadge(m.phase)
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
		tool = "ready"
	}

	left := fmt.Sprintf(" %s %s ", phase, HeaderDimStyle.Render("│ "+elapsed))
	center := fmt.Sprintf(" %s %s ", HeaderDimStyle.Render("tool:"), lipgloss.NewStyle().Foreground(ColorWhite).Render(tool))
	
	scrollIndicator := ""
	if m.userScrolledUp {
		scrollIndicator = WarningStyle.Render(" ▲ SCROLLED ") + " "
	}
	right := scrollIndicator + HeaderDimStyle.Render("type /help")

	statusWidth := m.width
	if statusWidth <= 0 {
		statusWidth = 80
	}

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right) + 1
	centerW := max(0, statusWidth-leftW-rightW)

	var sb strings.Builder
	sb.WriteString(left)
	if centerW > 0 {
		sb.WriteString(lipgloss.NewStyle().Width(centerW).Render(center))
	}
	sb.WriteString(right)

	return StatusBarStyle.Width(statusWidth).Render(sb.String())
}

func (m Model) renderSidebar(width, height int) string {
	if width <= 0 {
		return ""
	}

	var sb strings.Builder
	row := func(label, value string) {
		sb.WriteString(fmt.Sprintf(" %-10s %s\n", SidebarLabelStyle.Render(label), value))
	}
	section := func(title string) {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(" " + SectionHeaderStyle.Render(title) + "\n")
		sb.WriteString(" " + SectionRuleStyle.Render(strings.Repeat("─", width-3)) + "\n")
	}

	section("SESSION")
	activeMode := "default"
	if m.activeMode != "" {
		activeMode = m.activeMode
	}
	elapsed := "idle"
	if !m.execStartTime.IsZero() && m.executing {
		elapsed = time.Since(m.execStartTime).Round(time.Second).String()
	}
	row("Section", HeaderInfoStyle.Render(m.sessionID))
	row("Mode", SidebarValueStyle.Render(truncate(activeMode, max(1, width-14))))
	row("Runtime", lipgloss.NewStyle().Foreground(ColorWarning).Render(elapsed))
	row("Phase", renderSidebarPhase(m.phase))
	if m.lastObjective != "" {
		row("Objective", HintDescStyle.Render(truncate(m.lastObjective, max(1, width-14))))
	}

	section("MEMORY")
	row("Entities", StatusNodeStyle.Render(fmt.Sprintf("%d", m.graph.NodeCount())))
	row("Links", SidebarValueStyle.Render(fmt.Sprintf("%d", m.graph.EdgeCount())))

	if m.cfg.GetString("TELEGRAM_TOKEN") != "" {
		section("GATEWAY")
		telegramStatus := StatusOffStyle.Render("○ OFF")
		if m.cfg.GetString("TELEGRAM_CHAT_ID") != "" {
			telegramStatus = StatusOnStyle.Render("● READY")
		}
		row("Telegram", telegramStatus)
	}

	section("CONTROLS")
	row("Sandbox", SidebarValueStyle.Render(runtimeLabel(m.sandbox)))
	if m.autopilot {
		row("Autopilot", StatusOnStyle.Render("● ON"))
	} else {
		row("Autopilot", StatusOffStyle.Render("○ OFF"))
	}
	if m.opsecMgr.IsActive() {
		row("Stealth", StatusOnStyle.Render("● ON"))
	} else {
		row("Stealth", StatusOffStyle.Render("○ OFF"))
	}

	if m.lastPlan != nil {
		section("TACTICAL PLAN")
		row("Steps", SidebarValueStyle.Render(fmt.Sprintf("%d", len(m.lastPlan.Steps))))
		row("Detail", HintDescStyle.Render(truncate(m.phaseDetail, max(1, width-14))))
	}

	return SidebarPaneStyle.Width(width).Height(height).Render(sb.String())
}

func (m Model) renderStatusReport() string {
	heading := SectionHeaderStyle.Render
	label := SidebarLabelStyle.Render
	value := SidebarValueStyle.Render
	rule := SectionRuleStyle.Render("  " + strings.Repeat("─", max(20, m.viewport.Width-4)))

	onOff := func(on bool, yes string) string {
		if on {
			return StatusOnStyle.Render("● " + yes)
		}
		return StatusOffStyle.Render("○ OFF")
	}
	row := func(key, val string) string {
		return fmt.Sprintf("  %-16s %s", label(key), val)
	}

	phase, _ := renderPhaseBadge(m.phase)
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

	var sb strings.Builder
	sb.WriteString("\n  " + HeaderBrandStyle.Render("WORKSPACE STATUS REPORT") + "\n")
	sb.WriteString(rule + "\n\n")
	sb.WriteString("  " + heading("SESSION CONTEXT") + "\n")
	sb.WriteString(row("Section", value(m.sessionID)) + "\n")
	sb.WriteString(row("Active Phase", phase) + "\n")
	sb.WriteString(row("Attack Mode", value(mode)) + "\n")
	sb.WriteString(row("Elapsed Time", value(elapsed)) + "\n")
	sb.WriteString(row("Current Tool", value(tool)) + "\n\n")
	sb.WriteString("  " + heading("OPERATIONAL CONTROLS") + "\n")
	sb.WriteString(row("Stealth Policy", onOff(m.opsecMgr.IsActive(), "ACTIVE")) + "\n")
	sb.WriteString(row("Autopilot", onOff(m.autopilot, "ENABLED")) + "\n\n")
	sb.WriteString("  " + heading("ENVIRONMENT") + "\n")
	sb.WriteString(row("Execution Engine", value(runtimeLabel(m.sandbox))) + "\n")
	sb.WriteString(row("Telegram Gateway", onOff(telegramReady, "READY")) + "\n\n")
	sb.WriteString("  " + heading("INTELLIGENCE GRAPH") + "\n")
	sb.WriteString(row("Graph Nodes", StatusNodeStyle.Render(fmt.Sprintf("%d", m.graph.NodeCount()))) + "\n")
	sb.WriteString(row("Graph Edges", value(fmt.Sprintf("%d", m.graph.EdgeCount()))) + "\n")
	sb.WriteString(rule + "\n")

	return sb.String()
}

func (m Model) renderInputArea() string {
	op := m.graph.GetOperatorProfile()

	opName := "operator"
	if op != nil && op.Name != "" {
		opName = op.Name
	} else if m.cfg != nil && m.cfg.GetOperatorName() != "" {
		opName = m.cfg.GetOperatorName()
	}

	agName := "drogonclaw"

	var lines []string

	for i, h := range m.hints {
		prefix := " "
		cmdStr := HintCmdStyle.Render(h.cmd)
		if i == m.selectedHint {
			prefix = "▸ "
			cmdStr = HintSelectedStyle.Render(h.cmd)
		}
		pad := strings.Repeat(" ", max(1, 14-len(h.cmd)))
		lines = append(lines, HintBorderStyle.Render(prefix)+cmdStr+pad+HintDescStyle.Render(h.desc))
	}

	var promptGlyph string
	switch {
	case m.pendingConfirm != "":
		promptGlyph = WarningStyle.Render("CONFIRMATION REQUIRED > ")
	case m.executing && core.GlobalHitL.HasPending():
		promptGlyph = WarningStyle.Render("OPERATOR APPROVAL REQUIRED > ")
	case m.executing:
		elapsed := int(time.Since(m.execStartTime).Seconds())
		phaseStr := m.phase
		if phaseStr == "" {
			phaseStr = "reasoning"
		}
		if m.activeToolName != "" {
			phaseStr = "executing " + m.activeToolName
		}
		promptGlyph = m.spinner.View() + " " + SpinnerStyle.Render(fmt.Sprintf("%s [%02d:%02d]", phaseStr, elapsed/60, elapsed%60)) + " "
	default:
		promptGlyph = PromptGlyphStyle.Render(fmt.Sprintf("%s@%s ❯ ", opName, agName))
	}

	lines = append(lines, promptGlyph+m.input.View())
	if m.pendingConfirm != "" {
		lines = append(lines, WarningStyle.Render("Action requires exact confirmation: type "+m.confirmationPhrase()+" or Enter to cancel."))
	}

	return InputPaneStyle.Width(max(8, m.width-2)).Render(strings.Join(lines, "\n"))
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
	opName := "Operator"
	if op != nil && op.Name != "" {
		opName = op.Name
	}

	m.appendLine("")
	m.appendLine(HeaderBrandStyle.Render("  DrogonClaw v2"))
	m.appendLine(HintDescStyle.Render(fmt.Sprintf("  Operator: %s  ·  Engine: %s/%s  ·  Runtime: %s", opName, provider, model, runtimeLabel(m.sandbox))))
	if m.recovery != nil {
		tool := m.recovery.CurrentTool
		if tool == "" {
			tool = "planning or execution"
		}
		m.appendLine(WarningStyle.Render(fmt.Sprintf("  [RECOVERY] Previous mission interrupted at checkpoint: %s. Type /resume to continue.", tool)))
	}
	m.appendLine("")
}

func (m *Model) updateViewportContent() {
	inputHeight := lipgloss.Height(m.renderInputArea())
	layout := calculateLayout(m.width, m.height, inputHeight)
	base := strings.Join(m.lines, "\n")
	if m.executing && m.currentResponse != "" {
		base += "\n" + m.renderAgentResponseString(m.currentResponse)
	}
	m.viewport.Width = layout.contentWidth
	m.viewport.Height = layout.contentHeight
	m.viewport.SetContent(base)
	if !m.userScrolledUp {
		m.viewport.GotoBottom()
	}
}

func (m *Model) appendLine(line string) {
	line = truncateLine(line)
	m.lines = append(m.lines, line)
	if len(m.lines) > maxOutputLines {
		m.lines = truncateOutput(m.lines)
	}
	m.updateViewportContent()
}

func (m Model) renderAgentResponseString(content string) string {
	var lines []string
	for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		line = truncateLine(line)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderGraphSummary(graph *memory.Graph) string {
	if graph == nil {
		return WarningStyle.Render("[MEMORY] Graph unavailable")
	}

	var sb strings.Builder
	sb.WriteString(HintBorderStyle.Render(fmt.Sprintf("[MEMORY] %d entities, %d relationships", graph.NodeCount(), graph.EdgeCount())) + "\n")

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
		sb.WriteString(ToolArgsStyle.Render("Entities: "+strings.Join(parts, ", ")) + "\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

func renderSkills(manifest *skills.Manifest, query string) string {
	if manifest == nil || manifest.Count() == 0 {
		return WarningStyle.Render("  [!] No skills loaded.")
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
	sb.WriteString(SectionHeaderStyle.Render(fmt.Sprintf("  LOADED SKILLS (%d total)", manifest.Count())) + "\n")
	sb.WriteString(HintDescStyle.Render("  Type /skills <term> to search modules or /skills <name> for options.") + "\n\n")
	for _, cat := range categories {
		sb.WriteString(fmt.Sprintf("    %-24s %s\n", SidebarLabelStyle.Render(cat), SidebarValueStyle.Render(fmt.Sprintf("%d", counts[cat]))))
	}
	return sb.String()
}

func renderSkillSearch(manifest *skills.Manifest, query string) string {
	q := strings.ToLower(query)
	var matches []skills.Skill
	for _, skill := range manifest.Skills {
		if strings.Contains(strings.ToLower(skill.Name), q) ||
			strings.Contains(strings.ToLower(skill.Description), q) {
			matches = append(matches, skill)
		}
	}

	var sb strings.Builder
	sb.WriteString(SectionHeaderStyle.Render(fmt.Sprintf("  SKILLS MATCHING '%s' (%d)", query, len(matches))) + "\n\n")
	if len(matches) == 0 {
		sb.WriteString(WarningStyle.Render("    No skills found matching search criteria."))
		return sb.String()
	}
	limit := min(len(matches), 12)
	for i := 0; i < limit; i++ {
		skill := matches[i]
		sb.WriteString(fmt.Sprintf("    %-28s %-16s %s\n", HintCmdStyle.Render(skill.Name), SidebarLabelStyle.Render(classifySkill(skill)), HintDescStyle.Render(truncate(skill.Description, 60))))
	}
	return sb.String()
}

func renderSkillDetail(skill skills.Skill) string {
	var sb strings.Builder
	sb.WriteString(HeaderBrandStyle.Render("  "+skill.Name) + "\n")
	sb.WriteString(HintDescStyle.Render("  "+skill.Description) + "\n\n")
	sb.WriteString(fmt.Sprintf("  Category: %s | Backend: %s\n", classifySkill(skill), fallback(skill.ExecutesVia, "system")))
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
	case strings.Contains(text, "active directory") || strings.Contains(text, "domain"):
		return "Active Directory"
	case strings.Contains(text, "cloud") || strings.Contains(text, "aws") || strings.Contains(text, "azure"):
		return "Cloud Security"
	case strings.Contains(text, "exploit") || strings.Contains(text, "payload"):
		return "Exploitation"
	case strings.Contains(text, "nmap") || strings.Contains(text, "recon") || strings.Contains(text, "scan"):
		return "Reconnaissance"
	case strings.Contains(text, "web") || strings.Contains(text, "sql") || strings.Contains(text, "xss"):
		return "Web Application"
	default:
		return "General Security"
	}
}

func fallback(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

var allHints = []cmdHint{
	{"/health", "Verify runtime environment and dependencies"},
	{"/mode", "Select active attack methodology"},
	{"/analyze", "Classify a target and determine attack path"},
	{"/skills", "List and search available execution modules"},
	{"/status", "Display current session and workspace details"},
	{"/stealth", "Toggle evasive timing policy"},
	{"/auto", "Toggle autonomous execution mode"},
	{"/profile", "Build passive target profile"},
	{"/ctf", "Run local CTF artifact triage"},
	{"/report", "Generate structured pentest report"},
	{"/swarm", "Dispatch parallel autonomous sub-agents"},
	{"/sandbox", "Toggle container sandbox execution"},
	{"/persona", "Inject custom agent persona prompt"},
	{"/new", "Clear session memory and start clean"},
	{"/resume", "Resume interrupted execution checkpoint"},
	{"/clear", "Clear terminal output screen"},
	{"/exit", "Terminate session gracefully"},
}

func matchHints(input string) []cmdHint {
	query := strings.ToLower(strings.TrimSpace(input))
	if query == "" || !strings.HasPrefix(query, "/") {
		return nil
	}
	var out []cmdHint
	for _, h := range allHints {
		if strings.HasPrefix(h.cmd, query) {
			out = append(out, h)
			if len(out) >= 5 {
				break
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

func renderHelp() string {
	var sb strings.Builder
	sb.WriteString("\n  " + SectionHeaderStyle.Render("COMMAND REFERENCE") + "\n")
	sb.WriteString("  " + SectionRuleStyle.Render("──────────────────────────────────────────────────") + "\n\n")

	categories := map[string][]cmdHint{
		"OPERATIONS": {
			{"/health", "Verify runtime environment and dependencies"},
			{"/mode", "Select active attack methodology"},
			{"/analyze", "Classify a target and determine attack path"},
			{"/skills", "List and search available execution modules"},
			{"/profile", "Build passive target profile"},
			{"/ctf", "Run local CTF artifact triage"},
			{"/report", "Generate structured pentest report"},
			{"/swarm", "Dispatch parallel autonomous sub-agents"},
		},
		"CONTROLS": {
			{"/stealth", "Toggle evasive timing policy"},
			{"/auto", "Toggle autonomous execution mode"},
			{"/sandbox", "Toggle container sandbox execution"},
			{"/persona", "Inject custom agent persona prompt"},
		},
		"SESSION": {
			{"/status", "Display current session and workspace details"},
			{"/sections", "List all previous sections"},
			{"/section <id>", "Switch to a previous section"},
			{"/setup", "Run interactive configuration wizard"},
			{"/new", "Clear session memory and start clean"},
			{"/resume", "Resume interrupted execution checkpoint"},
			{"/clear", "Clear terminal output screen"},
			{"/help", "Show command reference"},
			{"/exit", "Terminate session gracefully"},
		},
	}

	for cat, hints := range categories {
		sb.WriteString("  " + SectionHeaderStyle.Render(cat) + "\n")
		for _, c := range hints {
			pad := strings.Repeat(" ", max(1, 14-len(c.cmd)))
			sb.WriteString("    " + HintCmdStyle.Render(c.cmd) + pad + HintDescStyle.Render(c.desc) + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("  " + SectionHeaderStyle.Render("KEYBOARD CONTROLS") + "\n")
	sb.WriteString("  " + HintDescStyle.Render("  ↑/↓ scroll output · Alt+↑/↓ history · PgUp/PgDn page · Tab accept suggestion · Ctrl+C abort") + "\n\n")

	return sb.String()
}

func renderSidebarPhase(phase string) string {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "planning":
		return PhasePlanningStyle.Render("Planning")
	case "reasoning":
		return PhaseReasoningStyle.Render("Reasoning")
	case "executing", "running":
		return PhaseExecutingStyle.Render("Executing")
	case "verifying", "waiting":
		return PhaseVerifyingStyle.Render("Verifying")
	case "complete", "done":
		return PhaseCompleteStyle.Render("Complete")
	case "error", "failed":
		return PhaseErrorStyle.Render("Error")
	default:
		return PhaseIdleStyle.Render("Idle")
	}
}

func renderPhaseBadge(phase string) (string, lipgloss.Style) {
	s := strings.ToLower(strings.TrimSpace(phase))
	switch s {
	case "planning":
		return PhasePlanningStyle.Render("PLANNING"), PhasePlanningStyle
	case "reasoning":
		return PhaseReasoningStyle.Render("REASONING"), PhaseReasoningStyle
	case "executing", "running":
		return PhaseExecutingStyle.Render("EXECUTING"), PhaseExecutingStyle
	case "verifying", "waiting":
		return PhaseVerifyingStyle.Render("VERIFYING"), PhaseVerifyingStyle
	case "complete", "done":
		return PhaseCompleteStyle.Render("COMPLETE"), PhaseCompleteStyle
	case "error", "failed":
		return PhaseErrorStyle.Render("ERROR"), PhaseErrorStyle
	default:
		return PhaseIdleStyle.Render("IDLE"), PhaseIdleStyle
	}
}
