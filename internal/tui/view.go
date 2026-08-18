package tui

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/0xP4X/drogonclaw-go/internal/core"
	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/0xP4X/drogonclaw-go/internal/skills"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading DrogonClaw..."
	}

	var sb strings.Builder

	// Render command autocomplete hints if user is typing a slash command
	if len(m.hints) > 0 {
		sb.WriteString(m.renderHints())
		sb.WriteString("\n")
	}

	// Render compact 1-line execution status while tool/agent is running
	if m.executing {
		tool := m.activeToolName
		if tool == "" {
			tool = m.lastTool
		}
		if tool == "" {
			tool = "thinking"
		}
		elapsed := ""
		if !m.execStartTime.IsZero() {
			elapsed = time.Since(m.execStartTime).Round(time.Second).String()
		}
		phase := m.phase
		if phase == "" {
			phase = "executing"
		}

		statusLine := fmt.Sprintf(" %s%s %s %s",
			m.spinner.View(),
			WarningStyle.Render(fmt.Sprintf("[%s]", strings.ToUpper(phase))),
			HeaderDimStyle.Render(elapsed),
			HeaderInfoStyle.Render("tool: "+tool),
		)
		sb.WriteString(statusLine)
		sb.WriteString("\n")
	}

	// Render input prompt line
	sb.WriteString(m.renderInputArea(m.width))

	return sb.String()
}

func (m Model) renderWelcome() string {
	var sb strings.Builder
	width := max(28, m.viewport.Width)
	if width <= 0 {
		width = max(28, m.width-MainPaneStyle.GetHorizontalFrameSize())
	}

	provider := fallback(m.cfg.GetProvider(), "provider")
	model := fallback(m.cfg.GetModel(), "model")
	runtime := m.sandbox.RuntimeLabel()
	mode := m.activeMode
	if mode == "" {
		mode = "default"
	}

	sb.WriteString(HeaderBrandStyle.Render("DrogonClaw v2") + " ")
	sb.WriteString(WelcomeSubtitleStyle.Render("operator workspace"))
	sb.WriteString("\n")
	sb.WriteString(SectionRuleStyle.Render(strings.Repeat("─", max(12, width-1))) + "\n\n")

	status := []string{
		fmt.Sprintf("%s %s", SidebarLabelStyle.Render("runtime"), SidebarValueStyle.Render(runtime)),
		fmt.Sprintf("%s %s", SidebarLabelStyle.Render("engine"), SidebarValueStyle.Render(truncate(provider+"/"+model, max(10, width/2)))),
		fmt.Sprintf("%s %s", SidebarLabelStyle.Render("mode"), SidebarValueStyle.Render(mode)),
	}
	sb.WriteString(strings.Join(status, HeaderDimStyle.Render("  │  ")))
	sb.WriteString("\n\n")

	sb.WriteString(WelcomeHintStyle.Render("Enter a concrete objective, for example: profile example.com and identify the safest next checks."))
	sb.WriteString("\n\n")

	quickStart := lipgloss.JoinVertical(lipgloss.Left,
		WelcomeQuickStartStyle.Render("Useful commands"),
		fmt.Sprintf("  %-12s %s", HintCmdStyle.Render("/health"), HintDescStyle.Render("Verify runtime & dependencies")),
		fmt.Sprintf("  %-12s %s", HintCmdStyle.Render("/profile"), HintDescStyle.Render("Build passive target context")),
		fmt.Sprintf("  %-12s %s", HintCmdStyle.Render("/mode"), HintDescStyle.Render("Select workflow")),
		fmt.Sprintf("  %-12s %s", HintCmdStyle.Render("/skills"), HintDescStyle.Render("Browse loaded attack modules")),
		fmt.Sprintf("  %-12s %s", HintCmdStyle.Render("/status"), HintDescStyle.Render("Show session metrics")),
	)

	sb.WriteString(quickStart)

	return sb.String()
}

func (m Model) viewportDimensions() (width, height int) {
	// First pass to get widths.
	layout := calculateLayout(m.width, m.height, 0)

	// Measure the same way View() does so the two stay in sync.
	headerBar := m.renderHeader(m.width)
	headerBarHeight := lipgloss.Height(headerBar)
	inputHeight := lipgloss.Height(m.renderInputArea(layout.mainWidth))
	fixedHeight := inputHeight + headerBarHeight

	layout = calculateLayout(m.width, m.height, fixedHeight)

	vpWidth := max(8, layout.mainWidth-MainPaneStyle.GetHorizontalFrameSize()-OutputPaneStyle.GetHorizontalFrameSize())
	vpHeight := max(3, layout.mainHeight-MainPaneStyle.GetVerticalFrameSize())
	return vpWidth, vpHeight
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
	runtime := m.sandbox.RuntimeLabel()

	sep := HeaderSepStyle.Render(" │ ")

	parts := []string{HeaderBrandStyle.Render("DROGONCLAW")}

	if width >= 56 {
		parts = append(parts, HeaderInfoStyle.Render(fmt.Sprintf("%s@%s", opName, agName)))
	}
	if width >= 76 {
		parts = append(parts, HeaderDimStyle.Render(fmt.Sprintf("session:%s", sid)))
	}

	if provider != "" && width >= 92 {
		parts = append(parts, HeaderInfoStyle.Render(fmt.Sprintf("%s/%s", provider, model)))
	}
	parts = append(parts, HeaderInfoStyle.Render(runtime))

	line := strings.Join(parts, sep)
	if lipgloss.Width(line) > width {
		line = HeaderBrandStyle.Render("DROGONCLAW") + sep + HeaderInfoStyle.Render(truncateVisible(runtime, max(8, width-16)))
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

	anim := ""
	if m.executing {
		anim = m.spinner.View() + " "
	}

	left := fmt.Sprintf(" %s%s %s ", anim, phase, HeaderDimStyle.Render("│ "+elapsed))
	centerValue := lipgloss.NewStyle().Foreground(ColorWhite).Render(truncateVisible(tool, max(8, m.width/3)))
	center := fmt.Sprintf(" %s %s ", HeaderDimStyle.Render("tool:"), centerValue)

	scrollIndicator := ""
	if m.userScrolledUp {
		scrollIndicator = WarningStyle.Render(" ▲ SCROLLED ") + " "
	}
	queueIndicator := ""
	if len(m.promptQueue) > 0 {
		queueIndicator = QueueStyle.Render(fmt.Sprintf(" ⏳ QUEUE %d ", len(m.promptQueue))) + " "
	}
	rightHint := "type /help"
	if m.width < 70 {
		rightHint = "/help"
	}
	right := scrollIndicator + queueIndicator + HeaderDimStyle.Render(rightHint)

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
		valueWidth := max(4, width-16)
		sb.WriteString(fmt.Sprintf(" %-10s %s\n", SidebarLabelStyle.Render(label), truncateStyled(value, valueWidth)))
	}
	section := func(title string) {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(" " + SectionHeaderStyle.Render(title) + "\n")
		sb.WriteString(" " + SectionRuleStyle.Render(strings.Repeat("─", max(4, width-3))) + "\n")
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
	row("Session", HeaderInfoStyle.Render(truncateVisible(m.sessionID, max(4, width-14))))
	row("Workflow", SidebarValueStyle.Render(truncate(activeMode, max(1, width-14))))
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
	row("Sandbox", SidebarValueStyle.Render(m.sandbox.RuntimeLabel()))
	if m.autopilot {
		row("Auto-run", StatusOnStyle.Render("● ON"))
	} else {
		row("Auto-run", StatusOffStyle.Render("○ OFF"))
	}
	if m.opsecMgr.IsActive() {
		row("Rate limit", StatusOnStyle.Render("● ON"))
	} else {
		row("Rate limit", StatusOffStyle.Render("○ OFF"))
	}

	if m.lastPlan != nil {
		section("EXECUTION PLAN")
		row("Steps", SidebarValueStyle.Render(fmt.Sprintf("%d", len(m.lastPlan.Steps))))
		row("Detail", HintDescStyle.Render(truncate(m.phaseDetail, max(1, width-14))))
	}

	if m.tracker != nil {
		section("COST")
		total := m.tracker.Total()
		cost := m.tracker.TotalCost()
		row("Prompt tokens", fmt.Sprintf("%d", total.PromptTokens))
		row("Completion tokens", fmt.Sprintf("%d", total.CompletionTokens))
		row("Total tokens", fmt.Sprintf("%d", total.TotalTokens))
		row("Est. cost", fmt.Sprintf("$%.4f", cost))
	}

	if len(m.promptQueue) > 0 {
		section("QUEUE")
		limit := len(m.promptQueue)
		if limit > 3 {
			limit = 3
		}
		for i := 0; i < limit; i++ {
			row(fmt.Sprintf("%d.", i+1), truncate(m.promptQueue[i], max(1, width-16)))
		}
		if len(m.promptQueue) > 3 {
			row("…", fmt.Sprintf("+%d more", len(m.promptQueue)-3))
		}
	}

	return SidebarPaneStyle.Width(max(8, width-SidebarPaneStyle.GetHorizontalFrameSize())).Height(height).Render(sb.String())
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
	sb.WriteString(row("Workflow", value(mode)) + "\n")
	sb.WriteString(row("Elapsed Time", value(elapsed)) + "\n")
	sb.WriteString(row("Current Tool", value(tool)) + "\n\n")
	sb.WriteString("  " + heading("OPERATIONAL CONTROLS") + "\n")
	sb.WriteString(row("Rate Limit", onOff(m.opsecMgr.IsActive(), "ACTIVE")) + "\n")
	sb.WriteString(row("Auto-run", onOff(m.autopilot, "ENABLED")) + "\n\n")
	sb.WriteString("  " + heading("ENVIRONMENT") + "\n")
	sb.WriteString(row("Execution Engine", value(m.sandbox.RuntimeLabel())) + "\n")
	sb.WriteString(row("Telegram Gateway", onOff(telegramReady, "READY")) + "\n\n")
	sb.WriteString("  " + heading("INTELLIGENCE GRAPH") + "\n")
	sb.WriteString(row("Graph Nodes", StatusNodeStyle.Render(fmt.Sprintf("%d", m.graph.NodeCount()))) + "\n")
	sb.WriteString(row("Graph Edges", value(fmt.Sprintf("%d", m.graph.EdgeCount()))) + "\n")
	sb.WriteString(rule + "\n")

	return sb.String()
}

func (m Model) renderInputArea(mainWidth int) string {
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
		available := max(12, mainWidth-InputPaneStyle.GetHorizontalFrameSize()-18)
		lines = append(lines, HintBorderStyle.Render(prefix)+cmdStr+pad+HintDescStyle.Render(truncateVisible(h.desc, available)))
	}

	glyph, glyphWidth := m.promptGlyph(agName)

	paneWidth := max(8, mainWidth-InputPaneStyle.GetHorizontalFrameSize())
	avail := paneWidth - glyphWidth
	if avail < 8 {
		avail = 8
	}
	m.input.SetWidth(avail)

	taWidth := m.input.Width()
	if taWidth < 1 {
		taWidth = 1
	}
	text := m.input.Value()
	if text == "" {
		m.input.SetHeight(1)
	} else {
		lines := strings.Split(text, "\n")
		totalLines := 0
		for _, line := range lines {
			runeCount := utf8.RuneCountInString(line)
			if runeCount == 0 {
				totalLines++
			} else {
				totalLines += (runeCount + taWidth - 1) / taWidth
			}
		}
		m.input.SetHeight(clamp(totalLines, 1, 3))
	}

	if len(m.promptQueue) > 0 {
		var qItems []string
		for i, q := range m.promptQueue {
			qItems = append(qItems, fmt.Sprintf("%d. %s", i+1, truncate(q, 25)))
			if i >= 2 {
				if len(m.promptQueue) > 3 {
					qItems = append(qItems, fmt.Sprintf("+%d more", len(m.promptQueue)-3))
				}
				break
			}
		}
		lines = append(lines, QueueStyle.Render(fmt.Sprintf(" ⏳ QUEUE (%d): %s", len(m.promptQueue), strings.Join(qItems, " │ "))))
	}

	lines = append(lines, glyph+m.input.View())
	if m.pendingConfirm != "" {
		lines = append(lines, WarningStyle.Render("Action requires exact confirmation: type "+m.confirmationPhrase()+" or Enter to cancel."))
	}

	return InputPaneStyle.Width(paneWidth).Render(strings.Join(lines, "\n"))
}

// promptGlyph returns the current prompt prefix and its visible (unstyled)
// rune width, which callers use to size the input textarea.
func (m Model) promptGlyph(agName string) (string, int) {
	switch {
	case m.pendingConfirm != "":
		g := WarningStyle.Render("CONFIRMATION REQUIRED > ")
		return g, lipgloss.Width(g)
	case m.executing && core.GlobalHitL.HasPending():
		if core.GlobalHitL.PendingKind() == core.ApprovalDuration {
			g := WarningStyle.Render("TOOL APPROVAL (y/n) > ")
			return g, lipgloss.Width(g)
		}
		g := WarningStyle.Render("OPERATOR APPROVAL REQUIRED > ")
		return g, lipgloss.Width(g)
	case m.executing:
		g := PromptGlyphStyle.Render(fmt.Sprintf("%s ❯ ", agName))
		return g, lipgloss.Width(g)
	default:
		g := PromptGlyphStyle.Render(fmt.Sprintf("%s ❯ ", agName))
		return g, lipgloss.Width(g)
	}
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
	m.appendLine(HintDescStyle.Render(fmt.Sprintf("  Operator: %s  ·  Engine: %s/%s  ·  Runtime: %s", opName, provider, model, m.sandbox.RuntimeLabel())))
	if m.recovery != nil {
		tool := m.recovery.CurrentTool
		if tool == "" {
			tool = "planning or execution"
		}
		m.appendLine(WarningStyle.Render(fmt.Sprintf("  [RECOVERY] Previous mission interrupted at checkpoint: %s. Type /resume to continue.", tool)))
	}
	m.appendLine("")
}

func wrapStyledLine(line string, width int) string {
	if width <= 0 {
		return ""
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(line)
	lines := strings.Split(wrapped, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) updateViewportContent() {
	vpWidth, vpHeight := m.viewportDimensions()

	var base string
	if vpWidth > 4 {
		wrapped := make([]string, len(m.lines))
		for i, line := range m.lines {
			wrapped[i] = wrapStyledLine(line, vpWidth)
		}
		base = strings.Join(wrapped, "\n")
		if m.executing && m.currentResponse != "" {
			base += "\n" + wrapStyledLine(m.renderAgentResponseString(m.currentResponse), vpWidth)
		}
	} else {
		base = strings.Join(m.lines, "\n")
		if m.executing && m.currentResponse != "" {
			base += "\n" + m.renderAgentResponseString(m.currentResponse)
		}
	}

	m.viewport.Width = vpWidth
	m.viewport.Height = vpHeight
	m.viewport.SetContent(base)

	// Clamp scroll offset so it never points past the end of content
	// (content can shrink when lines re-wrap to a wider terminal).
	maxOffset := max(0, m.viewport.TotalLineCount()-m.viewport.Height)
	if m.viewport.YOffset > maxOffset {
		m.viewport.YOffset = maxOffset
	}

	if !m.userScrolledUp {
		m.viewport.GotoBottom()
	}
	m.userScrolledUp = !m.viewport.AtBottom()
}

func (m *Model) appendLine(raw string) {
	clean := stripXMLTags(raw)
	if clean == "" {
		m.lines = append(m.lines, "")
		fmt.Println()
		return
	}

	lines := strings.Split(clean, "\n")
	if len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		m.lines = append(m.lines, line)
		fmt.Println(line)
	}
	if len(m.lines) > maxOutputLines {
		m.lines = truncateOutput(m.lines)
	}
}

func (m Model) renderAgentResponseString(content string) string {
	clean := stripXMLTags(content)
	rendered, err := m.mdRenderer.Render(strings.TrimRight(clean, "\n"))
	if err != nil || rendered == "" {
		rendered = strings.TrimRight(clean, "\n")
	}
	
	lines := strings.Split(rendered, "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var processed []string
	for _, line := range lines {
		processed = append(processed, truncateLine(line))
	}
	return strings.Join(processed, "\n")
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
	{"/mode", "Select active workflow methodology"},
	{"/analyze", "Classify target and determine attack path"},
	{"/skills", "List and search available execution modules"},
	{"/status", "Display current session and workspace details"},
	{"/cost", "Show current API token usage and estimated cost"},
	{"/stealth", "Toggle evasive timing policy"},
	{"/auto", "Toggle autonomous execution mode"},
	{"/profile", "Build passive target profile"},
	{"/ctf", "Run local CTF artifact triage"},
	{"/report", "Generate structured pentest report"},
	{"/swarm", "Run a parallel task group"},
	{"/queue", "Show queued prompts waiting to run"},
	{"/sections", "List all previous saved session sections"},
	{"/section", "Switch to a previous saved session section"},
	{"/copy", "Export transcript to clipboard and text file"},
	{"/benchmarks", "Display benchmark statistics and Mermaid charts"},
	{"/sandbox", "Toggle container sandbox execution"},
	{"/persona", "Inject custom agent persona prompt"},
	{"/new", "Clear session memory and start clean"},
	{"/resume", "Resume interrupted execution checkpoint"},
	{"/help", "Show complete command reference"},
	{"/clear", "Clear terminal output screen"},
	{"/exit", "Terminate session gracefully"},
}

func (m Model) renderHints() string {
	if len(m.hints) == 0 {
		return ""
	}
	var lines []string
	for i, h := range m.hints {
		prefix := "  "
		if i == m.selectedHint {
			prefix = " >"
		}
		lines = append(lines, fmt.Sprintf("%s %-12s %s", prefix, HintCmdStyle.Render(h.cmd), HintDescStyle.Render(h.desc)))
	}
	return strings.Join(lines, "\n")
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
			{"/mode", "Select active workflow"},
			{"/analyze", "Classify a target and determine attack path"},
			{"/skills", "List and search available execution modules"},
			{"/profile", "Build passive target profile"},
			{"/ctf", "Run local CTF artifact triage"},
			{"/report", "Generate structured pentest report"},
			{"/swarm", "Run a parallel task group"},
			{"/queue", "Show queued prompts waiting to run"},
			{"/benchmarks", "Display benchmark statistics & Mermaid charts"},
		},
		"CONTROLS": {
			{"/stealth", "Toggle rate limiting policy"},
			{"/auto", "Toggle automatic execution"},
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
			{"/copy", "Copy full transcript to clipboard / file"},
			{"F3", "View output in pager (select & copy any part)"},
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
	sb.WriteString("  " + HintDescStyle.Render("  ↑/↓ scroll output · Alt+↑/↓ history · PgUp/PgDn page · Tab accept suggestion · Ctrl+C abort") + "\n")
	sb.WriteString("  " + HintDescStyle.Render("  While running, type a new objective to queue it · type y/n at an approval prompt to accept/skip") + "\n\n")

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

var xmlTagRegex = regexp.MustCompile(`(?s)<environment_details>.*?</environment_details>|<[^>]+>`)

func stripXMLTags(s string) string {
	return xmlTagRegex.ReplaceAllString(s, "")
}

func (m Model) renderBenchmarks() string {
	content, err := os.ReadFile("BENCHMARKS.md")
	if err != nil {
		content, err = os.ReadFile("../../BENCHMARKS.md")
	}
	if err != nil {
		return ErrorStyle.Render("  [✗] BENCHMARKS.md file not found in workspace root.")
	}

	rendered, err := m.mdRenderer.Render(string(content))
	if err != nil {
		return string(content)
	}
	return rendered
}
