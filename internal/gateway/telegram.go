package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/0xP4X/drogonclaw-go/internal/agent"
	"github.com/0xP4X/drogonclaw-go/internal/config"
	"github.com/0xP4X/drogonclaw-go/internal/core"
	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/0xP4X/drogonclaw-go/internal/opsec"
	tele "gopkg.in/telebot.v3"
)

const (
	cbApprove = "dcc:approve"
	cbSkip    = "dcc:skip"
	cbCancel  = "dcc:cancel"

	progressWidth = 10
	panelMaxRunes = 1800
	streamCap     = 4000
	activityDepth = 4
	editDebounce  = 220 * time.Millisecond
	typingBeat    = 4 * time.Second
	clockBeat     = 3 * time.Second
)

var (
	secretScrub = regexp.MustCompile(`(?i)(api[_-]?key|token|secret|bearer|password|authorization|auth)["']?\s*[:=]\s*["']?([A-Za-z0-9._\-+/]{4,})`)
	bearerScrub = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._\-+/=]{10,}`)

	argsKeyOrder = []string{"target", "url", "domain", "host", "ip", "ports", "path", "wordlist", "method"}

	signalMarkers = []string{"found", "open", "vuln", "cve-", "credential", "exposed", "leaked", "200 ok", "banner", "misconfig", "shell", "unauthor"}
	signalNegate  = []string{"error", "failed", "refused", "timeout", "no result", "no open", "not open", "no ports", "empty", "nothing", "not found", "is closed"}
)

type TelegramGateway struct {
	bot      *tele.Bot
	chatID   int64
	orch     *agent.Orchestrator
	graph    *memory.Graph
	opsecMgr *opsec.Manager
	loot     *memory.LootDB

	mu      sync.Mutex
	session *missionSession
}

func NewTelegramGateway(cfg *config.Manager, orch *agent.Orchestrator, graph *memory.Graph, opsecMgr *opsec.Manager, loot *memory.LootDB) (*TelegramGateway, error) {
	token := cfg.GetString("TELEGRAM_TOKEN")
	chatIDStr := cfg.GetString("TELEGRAM_CHAT_ID")

	if token == "" || chatIDStr == "" {
		return nil, nil // Disabled
	}

	var chatID int64
	_, _ = fmt.Sscanf(chatIDStr, "%d", &chatID)

	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to start Telegram bot: %w", err)
	}

	tg := &TelegramGateway{
		bot:      b,
		chatID:   chatID,
		orch:     orch,
		graph:    graph,
		opsecMgr: opsecMgr,
		loot:     loot,
	}

	tg.setupHandlers()
	return tg, nil
}

func (tg *TelegramGateway) Start() {
	go tg.bot.Start()
}

func (tg *TelegramGateway) setupHandlers() {
	tg.bot.Handle(tele.OnText, func(c tele.Context) error {
		if c.Chat().ID != tg.chatID {
			return nil
		}

		text := c.Text()

		// Slash commands always win, even while an approval is pending — the
		// operator may need /status or /cancel at the exact moment the agent
		// is asking for a go/no-go. Plain (non-slash) text is the answer.
		if strings.HasPrefix(text, "/") {
			return tg.handleCommand(c, text)
		}

		if core.GlobalHitL.HasPending() {
			core.GlobalHitL.Resolve(text)
			if s := tg.activeSession(); s != nil {
				s.cleared()
			}
			return c.Send("✅ Approval received. Resuming execution.")
		}

		if s := tg.activeSession(); s != nil && !s.isFinished() {
			return c.Send("⏳ A mission is already running — send /status to peek or /cancel to stop it.")
		}

		return tg.executeMission(c, text)
	})

	tg.bot.Handle(tele.OnCallback, func(c tele.Context) error {
		if c.Chat().ID != tg.chatID {
			return nil
		}
		switch c.Data() {
		case cbApprove:
			core.GlobalHitL.Resolve("y")
			if s := tg.activeSession(); s != nil {
				s.cleared()
			}
			return c.Respond(&tele.CallbackResponse{Text: "Approved — resuming."})
		case cbSkip:
			core.GlobalHitL.Resolve("n")
			if s := tg.activeSession(); s != nil {
				s.cleared()
			}
			return c.Respond(&tele.CallbackResponse{Text: "Skipped — continuing without it."})
		case cbCancel:
			if s := tg.activeSession(); s != nil {
				_ = c.Respond(&tele.CallbackResponse{Text: "Sending cancel signal…"})
				s.requestCancel()
				return nil
			}
		}
		return c.Respond()
	})
}

func (tg *TelegramGateway) activeSession() *missionSession {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	return tg.session
}

func (tg *TelegramGateway) handleCommand(c tele.Context, text string) error {
	parts := strings.Fields(text)
	cmd := parts[0]

	switch cmd {
	case "/report":
		_ = c.Send("Drafting compliance-ready penetration test report...")
		reporter := core.NewReportGenerator(tg.orch.GetProvider(), tg.graph)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		path, err := reporter.GenerateMarkdownReport(ctx)
		if err != nil {
			return c.Send(fmt.Sprintf("❌ Report generation failed: %v", err))
		}

		doc := &tele.Document{File: tele.FromDisk(path)}
		return c.Send(doc)

	case "/swarm":
		if len(parts) < 2 {
			return c.Send("❌ Usage: /swarm <mission>")
		}
		mission := strings.Join(parts[1:], " ")
		return tg.executeSwarm(c, mission)

	case "/status":
		s := tg.activeSession()
		if s == nil || s.isFinished() {
			return c.Send(tg.dashboardHTML(), tele.ModeHTML)
		}
		body, _ := s.render()
		_, err := tg.bot.Send(c.Chat(), body, tele.ModeHTML)
		return err

	case "/findings":
		return tg.sendFindings(c)

	case "/autopilot":
		return tg.sendAutopilot(c, parts)

	case "/whoami":
		return c.Send(tg.whoamiHTML(), tele.ModeHTML)

	case "/cancel":
		s := tg.activeSession()
		if s == nil || s.isFinished() {
			return c.Send("No mission is currently running.")
		}
		s.requestCancel()
		return c.Send("✋ Cancel signal sent.")

	case "/help", "/start":
		return c.Send(strings.Join([]string{
			"<b>🐉 DrogonClaw — Telegram C2</b>",
			"",
			"• <code>&lt;free text&gt;</code> — run a mission",
			"• <code>/swarm &lt;mission&gt;</code> — parallel execution vectors",
			"• <code>/findings</code> — what the agent has collected so far",
			"• <code>/autopilot [on|off]</code> — auto-accept low-risk approvals",
			"• <code>/status</code> — live mission snapshot / daemon dashboard",
			"• <code>/cancel</code> — abort the running mission",
			"• <code>/report</code> — generate the pentest report",
			"• <code>/whoami</code> — operator / agent identity",
			"• <code>/help</code> — this list",
			"",
			"When the agent needs a go/no-go the mission panel turns into buttons — tap <b>✓ Approve</b> or <b>✗ Skip</b>, or answer with your own text.",
		}, "\n"), tele.ModeHTML)

	default:
		return c.Send("Unknown command. Send /help for the supported commands.")
	}
}

// sendFindings renders the loot ledger to the chat.
func (tg *TelegramGateway) sendFindings(c tele.Context) error {
	if tg.loot == nil {
		return c.Send("No findings ledger available (Loot DB not initialised).")
	}
	rep, err := tg.loot.Findings(8)
	if err != nil {
		return c.Send("❌ Could not read the findings ledger: " + err.Error())
	}
	return c.Send(findingsHTML(rep), tele.ModeHTML)
}

// sendAutopilot reads or toggles the delegating "autopilot" mode. When enabled,
// long-running low-risk tools no longer pause for approval.
func (tg *TelegramGateway) sendAutopilot(c tele.Context, parts []string) error {
	want, set := parseAutopilotArg(parts)
	if set {
		tg.orch.Autopilot = want
	}
	state := tg.orch.Autopilot
	var hint string
	if state {
		hint = "Long-running low-risk tools auto-accept — the agent will not pause for ⏸ approvals."
	} else {
		hint = "The agent pauses and asks before long-running low-risk tools; you can ✓ Approve or ✗ Skip."
	}
	return c.Send(autopilotHTML(state, hint), tele.ModeHTML)
}

// sendWhoami renders the operator and agent identity from the intelligence
// graph.
func (tg *TelegramGateway) whoamiHTML() string {
	var b strings.Builder
	b.WriteString("<b>🐉 Operator & Agent</b>\n")
	op, agentID := "—", "—"
	if p := tg.graph.GetOperatorProfile(); p != nil && p.Name != "" {
		op = p.Name
	}
	if p := tg.graph.GetAgentProfile(); p != nil && p.Name != "" {
		agentID = p.Name
	}
	b.WriteString("👤 Operator: <code>" + htmlEscape(op) + "</code>\n")
	b.WriteString("🤖 Agent:    <code>" + htmlEscape(agentID) + "</code>\n")
	return b.String()
}

// dashboardHTML is the idle dashboard shown by /status when nothing is running:
// session identity, graph size and the findings ledger totals.
func (tg *TelegramGateway) dashboardHTML() string {
	var b strings.Builder
	b.WriteString("✅ <b>Standing by — no mission running.</b>\n")
	b.WriteString("🧩 Session <code>" + htmlEscape(tg.orch.SessionID) + "</code>\n")
	b.WriteString("🗺 " + fmt.Sprint(tg.graph.NodeCount()) + " nodes · " + fmt.Sprint(tg.graph.EdgeCount()) + " edges in the intelligence graph\n")
	if tg.loot != nil {
		if rep, err := tg.loot.Findings(1); err == nil {
			b.WriteString("🔎 " + fmt.Sprint(rep.Ports) + " ports · " + fmt.Sprint(rep.Credentials) + " credentials · " + fmt.Sprint(rep.Vulnerabilities) + " vulnerabilities\n")
		}
	}
	autopilot := "off"
	if tg.orch.Autopilot {
		autopilot = "on"
	}
	b.WriteString("⚡ Autopilot: " + autopilot + "\n\n")
	b.WriteString("Send a mission, or /help for commands.")
	return b.String()
}

func (tg *TelegramGateway) executeMission(c tele.Context, mission string) error {
	s, err := tg.beginSession(c, "mission", mission, 30*time.Minute)
	if err != nil {
		return c.Send("⏳ A mission is already in progress — send /status to peek or /cancel to stop it.")
	}

	events := make(chan agent.Event, 64)
	go func() {
		defer s.cancel()
		_ = tg.orch.Execute(s.ctx, mission, events)
	}()

	for ev := range events {
		s.apply(ev)
	}

	tg.finalize(s)
	return nil
}

func (tg *TelegramGateway) executeSwarm(c tele.Context, mission string) error {
	s, err := tg.beginSession(c, "swarm", mission, 30*time.Minute)
	if err != nil {
		return c.Send("⏳ A mission is already in progress — send /status to peek or /cancel to stop it.")
	}

	events := make(chan agent.Event, 64)
	go func() {
		defer close(events)
		defer s.cancel()
		commander := agent.NewSwarmCommander(tg.orch.GetProvider(), tg.orch.GetTools(), "", tg.orch.SessionID, tg.graph)
		res, err := commander.ExecuteSwarm(s.ctx, mission, events)
		if err != nil {
			s.setFinal("❌ Swarm failed: " + err.Error())
			return
		}
		s.setFinal(res)
	}()

	for ev := range events {
		s.apply(ev)
	}

	tg.finalize(s)
	return nil
}

func (tg *TelegramGateway) finalize(s *missionSession) {
	if s.isCancelled() {
		tg.paint(s)
		tg.teardown(s)
		return
	}
	s.setFinished()
	tg.paint(s)
	tg.deliverFinal(s)
	tg.teardown(s)
}

func (tg *TelegramGateway) deliverFinal(s *missionSession) {
	text := s.finalBody()
	chat := &tele.Chat{ID: s.chatID}
	if text == "" {
		_, _ = tg.bot.Send(chat, "✅ Mission complete — no further output.")
		return
	}
	tg.chunkSend(chat, text, tele.ModeHTML)
}

func (tg *TelegramGateway) beginSession(c tele.Context, kind, objective string, timeout time.Duration) (*missionSession, error) {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	if tg.session != nil && !tg.session.isFinished() {
		return nil, fmt.Errorf("session busy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	s := &missionSession{
		chatID:    c.Chat().ID,
		ctx:       ctx,
		cancel:    cancel,
		kind:      kind,
		objective: objective,
		started:   time.Now(),
		sessionID: tg.orch.SessionID,
		pump:      make(chan struct{}, 1),
	}

	msg, err := tg.bot.Send(c.Chat(), s.spinnerHTML(), tele.ModeHTML, cancelMarkup())
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to open mission panel: %w", err)
	}
	s.panel = msg

	tg.session = s
	go tg.painter(s)
	go tg.typingLoop(s)
	go tg.clockLoop(s)
	return s, nil
}

func (tg *TelegramGateway) teardown(s *missionSession) {
	tg.mu.Lock()
	if tg.session == s {
		tg.session = nil
	}
	tg.mu.Unlock()
}

func (tg *TelegramGateway) painter(s *missionSession) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.pump:
			for {
				select {
				case <-s.pump:
				default:
					goto drain
				}
			}
		drain:
			time.Sleep(editDebounce)
			tg.paint(s)
		}
	}
}

func (tg *TelegramGateway) paint(s *missionSession) {
	body, markup := s.render()
	if body == "" || s.panel == nil {
		return
	}
	if _, err := tg.bot.Edit(s.panel, body, tele.ModeHTML, markup); err != nil {
		s.mu.Lock()
		tg.collapse(s)
		s.mu.Unlock()
	}
}

func (tg *TelegramGateway) collapse(s *missionSession) {
	if s.panel == nil || s.collapsed {
		return
	}
	s.collapsed = true
	s.panel = nil
}

func (tg *TelegramGateway) typingLoop(s *missionSession) {
	t := time.NewTicker(typingBeat)
	defer t.Stop()
	chat := &tele.Chat{ID: s.chatID}
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-t.C:
			s.mu.Lock()
			active := !s.finished && !s.cancelled && !s.awaiting
			s.mu.Unlock()
			if active {
				_ = tg.bot.Notify(chat, tele.Typing)
			}
		}
	}
}

func (tg *TelegramGateway) clockLoop(s *missionSession) {
	t := time.NewTicker(clockBeat)
	defer t.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-t.C:
			s.wake()
		}
	}
}

func (tg *TelegramGateway) chunkSend(chat *tele.Chat, text string, parseMode string) {
	for _, chunk := range chunkText(text, 4000) {
		_, _ = tg.bot.Send(chat, chunk, parseMode)
	}
}

// chunkText splits text into rune-aligned chunks of at most size runes so
// multi-byte characters are never torn apart mid-sequence.
func chunkText(text string, size int) []string {
	text = strings.TrimLeft(text, "\n")
	if text == "" {
		return nil
	}
	if utf8.RuneCountInString(text) <= size {
		return []string{text}
	}
	runes := []rune(text)
	var chunks []string
	for len(runes) > size {
		chunks = append(chunks, string(runes[:size]))
		runes = runes[size:]
	}
	if len(runes) > 0 {
		chunks = append(chunks, string(runes))
	}
	return chunks
}

// missionSession is the live state behind the single mission panel message.
type missionSession struct {
	chatID    int64
	ctx       context.Context
	cancel    context.CancelFunc
	kind      string
	sessionID string
	started   time.Time

	pump  chan struct{}
	panel *tele.Message

	mu        sync.Mutex
	objective string
	plan      []string
	planDone  int
	current   string
	currentOf string
	activity  []string
	signals   int
	tools     int
	awaiting  bool
	awaitNote string
	streaming strings.Builder
	final     string
	errLine   string
	finished  bool
	cancelled bool
	collapsed bool
}

func (s *missionSession) wake() {
	select {
	case s.pump <- struct{}{}:
	default:
	}
}

func (s *missionSession) isFinished() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.finished || s.cancelled
}

func (s *missionSession) isCancelled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancelled
}

func (s *missionSession) setFinished() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.final == "" && s.errLine != "" {
		s.final = "❌ " + s.errLine
	}
	s.finished = true
}

func (s *missionSession) setFinal(text string) {
	s.mu.Lock()
	s.final = text
	s.mu.Unlock()
	s.wake()
}

func (s *missionSession) cleared() {
	s.mu.Lock()
	s.awaiting = false
	s.awaitNote = ""
	s.mu.Unlock()
	s.wake()
}

func (s *missionSession) requestCancel() {
	s.mu.Lock()
	if s.finished || s.cancelled {
		s.mu.Unlock()
		return
	}
	s.cancelled = true
	s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	s.wake()
}

func (s *missionSession) apply(ev agent.Event) {
	s.mu.Lock()
	switch ev.Type {
	case agent.EvPlan:
		s.applyPlan(ev)
	case agent.EvToolStart:
		s.applyToolStart(ev)
	case agent.EvToolDone:
		s.applyToolDone(ev)
	case agent.EvStatus:
		s.applyStatus(ev)
	case agent.EvError:
		s.applyError(ev)
	case agent.EvToken:
		s.applyToken(ev)
	case agent.EvDone:
		s.applyDone(ev)
	}
	s.mu.Unlock()
	s.wake()
}

func (s *missionSession) applyPlan(ev agent.Event) {
	if ev.Plan == nil {
		return
	}
	if ev.Plan.Objective != "" {
		s.objective = ev.Plan.Objective
	}
	s.plan = make([]string, 0, len(ev.Plan.Steps))
	for _, st := range ev.Plan.Steps {
		s.plan = append(s.plan, st.Action)
	}
}

func (s *missionSession) applyToolStart(ev agent.Event) {
	s.current = toolLabel(ev.Tool)
	s.currentOf = shortArgs(ev.Args)
}

func (s *missionSession) applyToolDone(ev agent.Event) {
	s.current, s.currentOf = "", ""
	s.tools++
	if s.planDone < len(s.plan) {
		s.planDone++
	}
	if isSignal(ev.Result) {
		s.signals++
	}
	s.pushLine("✅ " + toolLabel(ev.Tool) + friendly(scrubText(ev.Result)))
}

func (s *missionSession) applyStatus(ev agent.Event) {
	low := strings.ToLower(ev.Content)
	switch {
	case strings.Contains(low, "awaiting") || strings.Contains(low, "requires approval") || strings.Contains(low, "suspended"):
		s.awaiting = true
		s.awaitNote = scrubText(ev.Content)
	case strings.Contains(low, "resuming") || strings.Contains(low, "approval received"):
		s.awaiting = false
		s.awaitNote = ""
	default:
		s.pushLine("· " + scrubText(ev.Content))
	}
}

func (s *missionSession) applyError(ev agent.Event) {
	s.errLine = ev.Content
	s.pushLine("❌ " + scrubText(ev.Content))
}

func (s *missionSession) applyToken(ev agent.Event) {
	s.streaming.WriteString(scrubText(ev.Content))
	s.trimStream()
}

func (s *missionSession) applyDone(ev agent.Event) {
	s.final = ev.Content
}

func (s *missionSession) pushLine(line string) {
	line = oneLine(line)
	if len(s.activity) > 0 && s.activity[len(s.activity)-1] == line {
		return
	}
	s.activity = append(s.activity, line)
	if len(s.activity) > activityDepth {
		s.activity = append([]string(nil), s.activity[len(s.activity)-activityDepth:]...)
	}
}

func (s *missionSession) trimStream() {
	if utf8.RuneCountInString(s.streaming.String()) <= streamCap {
		return
	}
	tail := string([]rune(s.streaming.String())[len([]rune(s.streaming.String()))-streamCap/2:])
	s.streaming.Reset()
	s.streaming.WriteString("[…streaming…] " + tail)
}

func (s *missionSession) finalBody() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	text := strings.TrimSpace(s.final)
	if text == "" {
		return ""
	}
	return "✅ <b>Mission complete.</b>  " + footerLine(s) + "\n\n" + htmlEscape(text)
}

func (s *missionSession) spinnerHTML() string {
	return "⚡ <b>Initializing mission vector…</b>\n" + "<i>" + htmlEscape(truncate(s.objective, 90)) + "</i>"
}

func (s *missionSession) render() (string, *tele.ReplyMarkup) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case s.cancelled:
		return s.cancelHTML(), &tele.ReplyMarkup{}
	case s.awaiting:
		return s.approvalHTML(), approvalMarkup()
	case s.finished:
		return s.completeHTML(), &tele.ReplyMarkup{}
	case s.streaming.Len() > 0:
		return truncate(s.streamingHTML(), panelMaxRunes), cancelMarkup()
	default:
		return truncate(s.tickerHTML(), panelMaxRunes), cancelMarkup()
	}
}

func (s *missionSession) headerHTML(symbol string) string {
	sys := truncate(s.sessionID, 6)
	return symbol + " <b>MISSION</b> <code>" + sys + "</code>\n" +
		"<i>" + htmlEscape(truncate(s.objective, 90)) + "</i>"
}

func (s *missionSession) tickerHTML() string {
	var b strings.Builder
	b.WriteString(s.headerHTML("⚡"))
	b.WriteString("\n")
	for i := 0; i < 21; i++ {
		b.WriteRune('─')
	}
	b.WriteString("\n")
	if len(s.plan) > 0 {
		b.WriteString(progressInline(s.planDone, len(s.plan)))
		b.WriteString("\n")
		for i, st := range s.plan {
			if i >= 6 {
				b.WriteString("+ " + fmt.Sprint(len(s.plan)-6) + " more\n")
				break
			}
			if i < s.planDone {
				b.WriteString("✓ " + htmlEscape(truncate(st, 58)) + "\n")
			} else {
				b.WriteString("☐ " + htmlEscape(truncate(st, 58)) + "\n")
			}
		}
		b.WriteString("─────────────────────\n")
	}
	if s.current != "" {
		b.WriteString("🛠 <b>" + htmlEscape(s.current) + "</b> " + codeSpan(htmlEscape(s.currentOf)) + "\n")
	}
	for _, line := range s.activity {
		b.WriteString(htmlEscape(line) + "\n")
	}
	b.WriteString("─────────────────────\n")
	b.WriteString(footerLine(s))
	return b.String()
}

func (s *missionSession) streamingHTML() string {
	var b strings.Builder
	b.WriteString(s.headerHTML("⚡"))
	b.WriteString("\n\n")
	b.WriteString(htmlEscape(truncate(s.streaming.String(), 900)))
	b.WriteString("\n\n")
	b.WriteString(footerLine(s))
	return b.String()
}

func (s *missionSession) completeHTML() string {
	var b strings.Builder
	b.WriteString(s.headerHTML("✅"))
	b.WriteString("\n<i>Mission concluded.</i>\n")
	b.WriteString("─────────────────────\n")
	b.WriteString(footerLine(s))
	return b.String()
}

func (s *missionSession) cancelHTML() string {
	var b strings.Builder
	b.WriteString(s.headerHTML("✋"))
	b.WriteString("\n<i>Stopped at operator request.</i>\n")
	b.WriteString("─────────────────────\n")
	b.WriteString(footerLine(s))
	return b.String()
}

func (s *missionSession) approvalHTML() string {
	var b strings.Builder
	b.WriteString("⚠️ <b>APPROVAL REQUIRED</b>\n")
	if s.awaitNote != "" {
		b.WriteString("<i>" + htmlEscape(truncate(s.awaitNote, 160)) + "</i>\n")
	} else {
		b.WriteString("<i>The agent suspended execution and needs your go/no-go.</i>\n")
	}
	b.WriteString("─────────────────────\n")
	b.WriteString(footerLine(s))
	b.WriteString("\n\nTap a button, or reply with your own text answer.")
	return b.String()
}

func footerLine(s *missionSession) string {
	sinceStr := since(s.started)
	if s.signals == 0 && s.tools == 0 {
		return "⏱ " + sinceStr
	}
	return "🔍 " + fmt.Sprint(s.signals) + " signals · 🛠 " + fmt.Sprint(s.tools) + " tools · ⏱ " + sinceStr
}

func findingsHTML(rep memory.FindingsReport) string {
	var b strings.Builder
	b.WriteString("🔎 <b>FINDINGS LEDGER</b>\n")
	b.WriteString("🛰 " + fmt.Sprint(rep.Ports) + " ports · 🔑 " + fmt.Sprint(rep.Credentials) + " credentials · ⚠ " + fmt.Sprint(rep.Vulnerabilities) + " vulnerabilities\n")
	if len(rep.Items) == 0 {
		b.WriteString("\n<i>Nothing collected yet — send a mission.</i>")
		return b.String()
	}
	b.WriteString("─────────────────\n")
	for _, it := range rep.Items {
		switch it.Category {
		case "port":
			b.WriteString("🛰 " + htmlEscape(it.Detail) + "\n")
		case "vuln":
			sev := it.Severity
			sevText := strings.ToUpper(sev)
			switch {
			case strings.Contains(sevText, "CRITI") || strings.Contains(sevText, "HIGH"):
				b.WriteString("⚠ " + htmlEscape(it.Target) + " — <b>HIGH</b> " + htmlEscape(it.Detail) + "\n")
			default:
				b.WriteString("⚠ " + htmlEscape(it.Target) + " — " + htmlEscape(it.Detail) + "\n")
			}
		case "cred":
			b.WriteString("🔑 " + htmlEscape(it.Target) + " — " + htmlEscape(it.Detail) + "\n")
		}
	}
	return b.String()
}

func autopilotHTML(on bool, hint string) string {
	state := "off"
	glyph := "◻"
	if on {
		state = "on"
		glyph = "◼"
	}
	return "⚡ <b>Autopilot: " + state + "</b> " + glyph + "\n<i>" + htmlEscape(hint) + "</i>\n\n" +
		"Use <code>/autopilot on</code> or <code>/autopilot off</code> to change it."
}

// parseAutopilotArg interprets the optional argument of /autopilot. Returns the
// desired state and whether an explicit change was requested.
func parseAutopilotArg(parts []string) (want, set bool) {
	if len(parts) < 2 {
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(parts[1])) {
	case "on", "enable", "yes", "1", "true":
		return true, true
	case "off", "disable", "no", "0", "false":
		return false, true
	}
	return false, false
}

func cancelMarkup() *tele.ReplyMarkup {
	return &tele.ReplyMarkup{InlineKeyboard: [][]tele.InlineButton{{{
		Text: "✕ Cancel mission", Data: cbCancel,
	}}}}
}

func approvalMarkup() *tele.ReplyMarkup {
	return &tele.ReplyMarkup{InlineKeyboard: [][]tele.InlineButton{
		{{
			Text: "✓ Approve", Data: cbApprove,
		}, {
			Text: "✗ Skip", Data: cbSkip,
		}},
		{{
			Text: "✕ Cancel mission", Data: cbCancel,
		}},
	}}
}

func progressInline(done, total int) string {
	if total <= 0 {
		return ""
	}
	fill := done * progressWidth / total
	if fill > progressWidth {
		fill = progressWidth
	}
	return strings.Repeat("▰", fill) + strings.Repeat("▱", progressWidth-fill) + "  " +
		fmt.Sprintf("%d/%d", clamp(done, 0, total), total)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func since(t time.Time) string {
	d := time.Since(t)
	if h := int(d.Hours()); h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, int(d.Minutes())%60, int(d.Seconds())%60)
	}
	return fmt.Sprintf("%02d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}

func toolLabel(t string) string {
	t = strings.ReplaceAll(strings.TrimPrefix(t, "run_"), "_", " ")
	fields := strings.Fields(t)
	for i, w := range fields {
		if len(w) == 0 {
			continue
		}
		r, size := utf8.DecodeRuneInString(w)
		fields[i] = string(unicode.ToUpper(r)) + w[size:]
	}
	return strings.Join(fields, " ")
}

func shortArgs(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	m := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return truncate(scrubText(raw), 60)
	}
	var pairs []string
	for _, k := range argsKeyOrder {
		if v, ok := m[k]; ok {
			pairs = append(pairs, k+"="+truncate(scrubText(fmt.Sprint(v)), 48))
			if len(pairs) == 2 {
				break
			}
		}
	}
	if len(pairs) == 0 {
		for k := range m {
			pairs = append(pairs, k+"=…")
			break
		}
	}
	return strings.Join(pairs, " · ")
}

func friendly(res string) string {
	line := oneLine(res)
	if line == "" {
		return ""
	}
	return " — " + line
}

func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return truncate(s, 100)
}

func isSignal(res string) bool {
	low := strings.ToLower(res)
	for _, n := range signalNegate {
		if strings.Contains(low, n) {
			return false
		}
	}
	for _, m := range signalMarkers {
		if strings.Contains(low, m) {
			return true
		}
	}
	return false
}

func scrubText(s string) string {
	s = bearerScrub.ReplaceAllString(s, "Bearer ••••")
	s = secretScrub.ReplaceAllString(s, "$1=••••")
	return s
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

func codeSpan(s string) string {
	if s == "" {
		return ""
	}
	return "<code>" + s + "</code>"
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}
