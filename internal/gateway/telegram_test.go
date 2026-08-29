package gateway

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/agent"
	"github.com/0xP4X/drogonclaw-go/internal/memory"
)

func agentEvent(t agent.EventType, tool, args string) agent.Event {
	return agent.Event{Type: t, Tool: tool, Args: args}
}

func agentEventWithResult(t agent.EventType, tool, result string) agent.Event {
	return agent.Event{Type: t, Tool: tool, Result: result}
}

func agentEventWithContent(t agent.EventType, content string) agent.Event {
	return agent.Event{Type: t, Content: content}
}

func TestToolLabel(t *testing.T) {
	cases := map[string]string{
		"run_nmap":            "Nmap",
		"run_subfinder":       "Subfinder",
		"run_http_request":    "Http Request",
		"run_curl":            "Curl",
		"ask_operator":        "Ask Operator",
		"run_wpscan":          "Wpscan",
		"runcrack":            "Runcrack",
		"run_":                "",
		"domain_enum":         "Domain Enum",
		"run_dns_enumeration": "Dns Enumeration",
	}
	for in, want := range cases {
		if got := toolLabel(in); got != want {
			t.Errorf("toolLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScrubText(t *testing.T) {
	cases := []string{
		`{"api_key":"sk-1234567890abcdef"}`,
		`Authorization: Bearer abcdef1234567890abcdef`,
		`password=supersecret1 admin`,
		`token=deadbeef` + " " + `plain text`,
	}
	for _, in := range cases {
		got := scrubText(in)
		for _, secret := range []string{"sk-1234567890abcdef", "abcdef1234567890abcdef", "supersecret1", "deadbeef"} {
			if strings.Contains(got, secret) {
				t.Errorf("scrubText(%q) leaked secret %q: %q", in, secret, got)
			}
		}
	}
	if got := scrubText("no secrets here"); got != "no secrets here" {
		t.Errorf("scrubText should be idempotent on clean text, got %q", got)
	}
}

func TestShortArgs(t *testing.T) {
	got := shortArgs(`{"target":"10.10.10.5","ports":"80,443","extra":"zzz"}`)
	if !strings.Contains(got, "target=10.10.10.5") || !strings.Contains(got, "ports=80,443") || strings.Contains(got, "extra") {
		t.Errorf("shortArgs = %q, want target+ports pairs only", got)
	}
	if got := shortArgs(`{"key":"sk-abc1234567890"}`); strings.Contains(got, "sk-abc1234567890") {
		t.Errorf("shortArgs leaked unknown value: %q", got)
	}
	if got := shortArgs("not json at all"); got == "" {
		t.Errorf("shortArgs fallback should return something for non-JSON")
	}
	if got := shortArgs(""); got != "" {
		t.Errorf("shortArgs empty input = %q, want empty", got)
	}
}

func TestIsSignal(t *testing.T) {
	if !isSignal("Found open port 22 on 10.0.0.1") {
		t.Errorf("isSignal should flag found port")
	}
	if !isSignal("CVE-2021-44228 detected") {
		t.Errorf("isSignal should flag CVE")
	}
	if isSignal("Error: connection refused") {
		t.Errorf("isSignal must not flag errors")
	}
	if isSignal("No open ports in range") {
		t.Errorf("isSignal must not flag negated results")
	}
}

func TestPushLineBounded(t *testing.T) {
	s := &missionSession{pump: make(chan struct{}, 1)}
	for i := 0; i < 10; i++ {
		s.pushLine("line" + string(rune('a'+i)))
	}
	if len(s.activity) != activityDepth {
		t.Fatalf("pushLine bounded to %d, got %d", activityDepth, len(s.activity))
	}
	want := "line" + string(rune('a'+10-activityDepth))
	if s.activity[0] != want || s.activity[activityDepth-1] != "linej" {
		t.Errorf("pushLine should keep newest, got %v", s.activity)
	}
}

func TestProgressInline(t *testing.T) {
	if got := progressInline(0, 4); got != "▱▱▱▱▱▱▱▱▱▱  0/4" {
		t.Errorf("progressInline 0/4 = %q", got)
	}
	if got := progressInline(2, 4); got != "▰▰▰▰▰▱▱▱▱▱  2/4" {
		t.Errorf("progressInline 2/4 = %q", got)
	}
	if got := progressInline(4, 4); got != "▰▰▰▰▰▰▰▰▰▰  4/4" {
		t.Errorf("progressInline 4/4 = %q", got)
	}
	if got := progressInline(9, 4); got != "▰▰▰▰▰▰▰▰▰▰  4/4" {
		t.Errorf("progressInline should clamp, got %q", got)
	}
}

func TestSince(t *testing.T) {
	if got := since(time.Now().Add(-90 * time.Second)); got != "01:30" {
		t.Errorf("since 90s = %q, want 01:30", got)
	}
	if got := since(time.Now().Add(-3661 * time.Second)); got != "1:01:01" {
		t.Errorf("since 3661s = %q, want 1:01:01", got)
	}
}

func TestOneLineAndTruncate(t *testing.T) {
	if got := oneLine("  hello\nworld  "); got != "hello" {
		t.Errorf("oneLine = %q", got)
	}
	r := truncate("héllo wörld", 6)
	if len([]rune(r)) != 7 || []rune(r)[0] != 'h' || !strings.HasSuffix(r, "…") {
		t.Errorf("truncate should keep n runes plus ellipsis, got %q", r)
	}
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate should not touch short strings, got %q", got)
	}
}

func TestTickerHTMLShowsObjectivePlanAndFooter(t *testing.T) {
	s := &missionSession{
		ctx:       context.Background(),
		objective: "scan 10.0.0.5 for exposed ports",
		plan:      []string{"Enumerate", "Probe ports", "Check services"},
		planDone:  1,
		activity:  []string{"✅ Nmap — 22,80 open", "· Analyzing results…"},
		signals:   2,
		tools:     3,
		started:   time.Now(),
	}
	out := s.tickerHTML()
	for _, want := range []string{"scan 10.0.0.5 for exposed ports", "1/3", "✓ Enumerate", "☐ Probe ports", "✅ Nmap", "🔍 2 signals", "🛠 3 tools"} {
		if !strings.Contains(out, want) {
			t.Errorf("tickerHTML missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "&amp;") || strings.Contains(out, "MISSION") == false {
		t.Errorf("tickerHTML escaping/symbol unexpected")
	}
}

func TestApplyTracksPipeline(t *testing.T) {
	s := &missionSession{ctx: context.Background(), started: time.Now(), pump: make(chan struct{}, 1)}
	s.apply(agentEvent(agent.EvToolStart, "run_nmap", `{"target":"scanme.in","ports":"80"}`))
	if s.current != "Nmap" || !strings.Contains(s.currentOf, "scanme.in") {
		t.Fatalf("tool start not tracked: %v %v", s.current, s.currentOf)
	}
	s.apply(agentEventWithResult(agent.EvToolDone, "run_nmap", "Port 22 is open"))
	if s.tools != 1 || s.signals != 1 || s.current != "" {
		t.Errorf("tool done not tracked: tools=%d signals=%d current=%q", s.tools, s.signals, s.current)
	}
	s.apply(agentEventWithResult(agent.EvToolDone, "run_curl", "Error: timeout"))
	if s.signals != 1 {
		t.Errorf("errors must not count as signals, got %d", s.signals)
	}
}

func TestAwaitingLifecycle(t *testing.T) {
	s := &missionSession{ctx: context.Background(), started: time.Now(), pump: make(chan struct{}, 1)}
	s.apply(agentEventWithContent(agent.EvStatus, "Awaiting operator acceptance to run gobuster..."))
	if !s.awaiting || !strings.Contains(strings.ToLower(s.awaitNote), "awaiting") {
		t.Fatalf("awaiting not set: %v %q", s.awaiting, s.awaitNote)
	}
	s.cleared()
	if s.awaiting {
		t.Errorf("cleared() should lift awaiting")
	}
}

func TestScrubAppliedToToolInfo(t *testing.T) {
	s := &missionSession{ctx: context.Background(), started: time.Now(), pump: make(chan struct{}, 1)}
	s.apply(agentEventWithResult(agent.EvToolDone, "run_curl", `Authorization: Bearer abcdef12345678901234567890`))
	if strings.Contains(s.activity[len(s.activity)-1], "abcdef12345678901234567890") {
		t.Errorf("activity leaked bearer token: %v", s.activity)
	}
}

func TestCancelFlagsAndFinalize(t *testing.T) {
	s := &missionSession{ctx: context.Background(), started: time.Now(), pump: make(chan struct{}, 1)}
	s.requestCancel()
	if !s.isCancelled() {
		t.Fatalf("cancelled flag not set")
	}
	s.requestCancel() // idempotent
	if s.cancelled != true {
		t.Errorf("double cancel should stay cancelled")
	}

	s2 := &missionSession{ctx: context.Background(), started: time.Now(), pump: make(chan struct{}, 1)}
	s2.setFinal("probe done")
	s2.setFinished()
	if !s2.isFinished() || s2.final != "probe done" {
		t.Errorf("finalize failed: %v %q", s2.isFinished(), s2.final)
	}
	body := s2.finalBody()
	if !strings.Contains(body, "probe done") || !strings.Contains(body, "Mission complete") {
		t.Errorf("finalBody = %q", body)
	}
}

func TestFinalBodyFromError(t *testing.T) {
	s := &missionSession{ctx: context.Background(), started: time.Now()}
	s.errLine = "LLM error: 429 too many requests"
	s.setFinished()
	if !strings.Contains(s.finalBody(), "LLM error: 429 too many requests") {
		t.Errorf("setFinished should fold errLine into final: %q", s.finalBody())
	}
}

func TestChunkTextRuneAligned(t *testing.T) {
	long := strings.Repeat("é", 9000)
	chunks := chunkText(long, 4000)
	if len(chunks) != 3 {
		t.Fatalf("chunkText = %d chunks, want 3", len(chunks))
	}
	for _, c := range chunks {
		if len([]rune(c)) > 4000 {
			t.Errorf("chunk exceeds 4000 runes: %d", len([]rune(c)))
		}
		if !strings.HasSuffix(c, "é") {
			t.Errorf("chunk torn mid-rune: %q", c[len(c)-2:])
		}
	}
	if got := chunkText("", 10); got != nil {
		t.Errorf("chunkText empty = %v", got)
	}
	if got := chunkText("hello", 10); len(got) != 1 || got[0] != "hello" {
		t.Errorf("chunkText short = %v", got)
	}
}

func TestParseAutopilotArg(t *testing.T) {
	cases := []struct {
		args string
		want bool
		set  bool
	}{
		{"/autopilot", false, false},
		{"/autopilot on", true, true},
		{"/autopilot off", false, true},
		{"/autopilot enable", true, true},
		{"/autopilot disable", false, true},
		{"/autopilot yes", true, true},
		{"/autopilot nope", false, false},
	}
	for _, tc := range cases {
		want, set := parseAutopilotArg(strings.Fields(tc.args))
		if want != tc.want || set != tc.set {
			t.Errorf("parseAutopilotArg(%q) = (%v,%v), want (%v,%v)", tc.args, want, set, tc.want, tc.set)
		}
	}
}

func TestFindingsHTML(t *testing.T) {
	rep := memory.FindingsReport{
		Ports: 3, Credentials: 1, Vulnerabilities: 2,
		Items: []memory.FindingItem{
			{Category: "port", Target: "10.0.0.7", Detail: "10.0.0.7/443 https"},
			{Category: "vuln", Target: "10.0.0.7", Detail: "CVE-2026-0001 — RCE", Severity: "high"},
			{Category: "cred", Target: "10.0.0.7", Detail: "admin / secret123"},
		},
	}
	out := findingsHTML(rep)
	for _, want := range []string{"3 ports", "1 credentials", "2 vulnerabilities", "CVE-2026-0001", "admin / secret123", "HIGH"} {
		if !strings.Contains(out, want) {
			t.Errorf("findingsHTML missing %q:\n%s", want, out)
		}
	}
	empty := findingsHTML(memory.FindingsReport{})
	if !strings.Contains(empty, "Nothing collected") {
		t.Errorf("empty findings should say nothing collected: %q", empty)
	}
	if strings.Contains(out, "&") && !strings.Contains(out, "&amp;") && !strings.Contains(out, "&lt;") {
		t.Errorf("findingsHTML should escape HTML")
	}
}

func TestAutopilotHTML(t *testing.T) {
	on := autopilotHTML(true, "hint")
	off := autopilotHTML(false, "hint")
	if !strings.Contains(on, "on") || strings.Contains(on, "/autopilot on") == false {
		t.Errorf("autopilot on render wrong: %q", on)
	}
	if !strings.Contains(off, "off") {
		t.Errorf("autopilot off render wrong: %q", off)
	}
}
