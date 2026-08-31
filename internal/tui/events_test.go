package tui

import (
	"testing"

	"github.com/0xP4X/drogonclaw-go/internal/agent"
)

func TestDetectFindings(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		tool       string
		wantType   string
		wantCount  int
	}{
		{
			name:      "SQL injection vulnerability",
			output:    "[+] Vulnerability found: SQL injection in parameter 'id'",
			tool:      "sqlmap",
			wantType:  "vulnerability",
			wantCount: 1,
		},
		{
			name:      "XSS vulnerability",
			output:    "Cross-site scripting vulnerability detected in search field",
			tool:      "nuclei",
			wantType:  "vulnerability",
			wantCount: 1,
		},
		{
			name:      "CVE reference",
			output:    "Target is vulnerable to CVE-2024-1234",
			tool:      "nmap",
			wantType:  "vulnerability",
			wantCount: 1,
		},
		{
			name:      "Password credential",
			output:    "Found credential: password=admin123",
			tool:      "hydra",
			wantType:  "credential",
			wantCount: 1,
		},
		{
			name:      "API key credential",
			output:    "api_key: sk-1234567890abcdef",
			tool:      "nuclei",
			wantType:  "credential",
			wantCount: 1,
		},
		{
			name:      "CTF flag",
			output:    "flag{this_is_a_test_flag}",
			tool:      "custom",
			wantType:  "flag",
			wantCount: 1,
		},
		{
			name:      "HTB flag format",
			output:    "HTB{s0m3_h4sh_h3r3}",
			tool:      "custom",
			wantType:  "flag",
			wantCount: 1,
		},
		{
			name:      "Open port info",
			output:    "port 22 open",
			tool:      "nmap",
			wantType:  "info",
			wantCount: 1,
		},
		{
			name:      "No findings",
			output:    "Scan completed successfully with no issues found",
			tool:      "nmap",
			wantType:  "",
			wantCount: 0,
		},
		{
			name:      "Multiple findings",
			output:    "vulnerability detected: XSS\npassword=test123\nflag{found}",
			tool:      "multi",
			wantType:  "multiple",
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := detectFindings(tt.output, tt.tool)

			if len(findings) != tt.wantCount {
				t.Errorf("detectFindings() count = %d, want %d", len(findings), tt.wantCount)
			}

			if tt.wantCount > 0 && tt.wantType != "multiple" {
				if findings[0].Type != tt.wantType {
					t.Errorf("detectFindings() type = %s, want %s", findings[0].Type, tt.wantType)
				}
			}
		})
	}
}

func TestColorizeOutputLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
	}{
		{
			name:     "Error line",
			input:    "[ERROR] Connection refused",
			wantType: "error",
		},
		{
			name:     "Warning line",
			input:    "[WARNING] Rate limit approaching",
			wantType: "warning",
		},
		{
			name:     "Success line",
			input:    "[+] Vulnerability found",
			wantType: "success",
		},
		{
			name:     "Info line",
			input:    "Scanning target 192.168.1.1",
			wantType: "info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorizeOutputLine(tt.input)
			// Just verify it doesn't panic and returns non-empty
			if result == "" {
				t.Error("colorizeOutputLine() returned empty string")
			}
		})
	}
}

func TestSanitizeToolOutputLines(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLines int
		wantTrunc bool
	}{
		{
			name:      "Short output",
			input:     "Line 1\nLine 2\nLine 3",
			wantLines: 3,
			wantTrunc: false,
		},
		{
			name:      "HTML font CSS dump",
			input:     "@font-face { font-family: --mat-sys-display; }",
			wantLines: 1,
			wantTrunc: false,
		},
		{
			name:      "Express 500 error",
			input:     "Unexpected path: /test #stacktrace Error: Cannot find module",
			wantLines: 1,
			wantTrunc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeToolOutputLines(tt.input)
			if len(result) != tt.wantLines {
				t.Errorf("sanitizeToolOutputLines() lines = %d, want %d", len(result), tt.wantLines)
			}
		})
	}
}

func TestRedactCredential(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Password redaction",
			input: "password=admin123",
			want:  "password=[REDACTED]",
		},
		{
			name:  "API key redaction",
			input: "api_key=sk-1234567890",
			want:  "api_key=[REDACTED]",
		},
		{
			name:  "Token redaction",
			input: "token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			want:  "token=[REDACTED]",
		},
		{
			name:  "No sensitive data",
			input: "username=admin",
			want:  "username=admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactCredential(tt.input)
			if result != tt.want {
				t.Errorf("redactCredential() = %s, want %s", result, tt.want)
			}
		})
	}
}

func TestWaitForEvent(t *testing.T) {
	events := make(chan agent.Event, 1)
	events <- agent.Event{Type: agent.EvDone, Content: "test"}

	cmd := waitForEvent(events)
	msg := cmd()

	agentMsg, ok := msg.(AgentEventMsg)
	if !ok {
		t.Fatal("waitForEvent() did not return AgentEventMsg")
	}

	if agentMsg.Event.Type != agent.EvDone {
		t.Errorf("waitForEvent() event type = %v, want %v", agentMsg.Event.Type, agent.EvDone)
	}
}

func TestWaitForEventClosed(t *testing.T) {
	events := make(chan agent.Event)
	close(events)

	cmd := waitForEvent(events)
	msg := cmd()

	agentMsg, ok := msg.(AgentEventMsg)
	if !ok {
		t.Fatal("waitForEvent() did not return AgentEventMsg")
	}

	if agentMsg.Event.Type != agent.EvDone {
		t.Errorf("waitForEvent() should return EvDone when channel closed")
	}
}

func BenchmarkDetectFindings(b *testing.B) {
	output := `
[+] Vulnerability found: SQL injection in parameter 'id'
password=admin123
flag{test_flag}
port 22 open
service version 2.4.1
`
	tool := "nmap"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detectFindings(output, tool)
	}
}

func BenchmarkColorizeOutputLine(b *testing.B) {
	line := "[ERROR] Connection refused on port 443"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = colorizeOutputLine(line)
	}
}
