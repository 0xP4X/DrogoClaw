package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseCLIDefaultStartsTUI(t *testing.T) {
	opts, err := parseCLI(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.action != actionTUI {
		t.Errorf("default action = %v, want actionTUI", opts.action)
	}
	if opts.forceSandbox != nil {
		t.Errorf("default should not force a run mode")
	}
}

func TestParseCLIRetrievalCommands(t *testing.T) {
	cases := map[string]cliAction{
		"help":      actionHelp,
		"-h":        actionHelp,
		"--help":    actionHelp,
		"version":   actionVersion,
		"-v":        actionVersion,
		"--version": actionVersion,
		"setup":     actionSetup,
		"health":    actionHealth,
		"daemon":    actionDaemon,
	}
	for arg, want := range cases {
		opts, err := parseCLI([]string{arg})
		if err != nil {
			t.Fatalf("parseCLI(%q): %v", arg, err)
		}
		if opts.action != want {
			t.Errorf("parseCLI(%q) action = %v, want %v", arg, opts.action, want)
		}
	}
}

func TestParseCLIRunModes(t *testing.T) {
	opts, err := parseCLI([]string{"sandbox"})
	if err != nil || opts.forceSandbox == nil || !*opts.forceSandbox {
		t.Errorf("sandbox should force sandbox mode, got ok=%t", err == nil)
	}

	opts, err = parseCLI([]string{"native", "Kali 2026, 8 GB RAM"})
	if err != nil || opts.forceSandbox == nil || *opts.forceSandbox {
		t.Errorf("native should force native mode, got ok=%t", err == nil)
	}
	if len(opts.extraArgs) != 1 || opts.extraArgs[0] != "Kali 2026, 8 GB RAM" {
		t.Errorf("native should preserve env details, got %v", opts.extraArgs)
	}
}

func TestParseCLICarriesBenchAndWhiteboxArgs(t *testing.T) {
	for _, cmd := range []string{"bench", "whitebox"} {
		opts, err := parseCLI([]string{cmd, "--flag", "value"})
		if err != nil {
			t.Fatalf("parseCLI(%s): %v", cmd, err)
		}
		if len(opts.extraArgs) != 2 || opts.extraArgs[0] != "--flag" {
			t.Errorf("%s should forward its flags, got %v", cmd, opts.extraArgs)
		}
	}
}

func TestParseCLIRejectsUnknownCommand(t *testing.T) {
	if _, err := parseCLI([]string{"frobnicate"}); err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestCLIHelpListsEveryCommand(t *testing.T) {
	var buf bytes.Buffer
	printCLIHelp(&buf)
	out := buf.String()
	if !strings.Contains(out, "DrogonClaw — Autonomous Offensive Security AI") {
		t.Error("help missing banner")
	}
	if !strings.Contains(out, "usage: drogonclaw") {
		t.Error("help missing usage line")
	}
	for _, e := range cliEntries {
		if e.cmd == "" {
			continue
		}
		if !strings.Contains(out, e.cmd) {
			t.Errorf("help missing command %q", e.cmd)
		}
	}
}

func TestPrintVersion(t *testing.T) {
	version = "test-version"
	buildTime = "test-build"
	var buf bytes.Buffer
	printVersion(&buf)
	out := buf.String()
	if !strings.Contains(out, "DrogonClaw test-version") {
		t.Errorf("version output missing version string: %q", out)
	}
	if !strings.Contains(out, "test-build") {
		t.Errorf("version output missing build time: %q", out)
	}
}
