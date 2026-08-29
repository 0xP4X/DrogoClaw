package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/config"
	"github.com/0xP4X/drogonclaw-go/internal/tui"
	"github.com/charmbracelet/lipgloss"
)

// Build metadata, injected at link time via the Makefile:
//
//	go build -ldflags "-X main.version=... -X main.buildTime=..."
var (
	version   = "dev"
	buildTime = "unknown"
)

// cliAction enumerates the possible outcomes of parsing the command line.
type cliAction int

const (
	actionTUI cliAction = iota
	actionHelp
	actionVersion
	actionSetup
	actionBench
	actionWhitebox
	actionHealth
	actionDaemon
)

// cliOptions is the parsed description of the operator's intent.
type cliOptions struct {
	action       cliAction
	forceSandbox *bool
	extraArgs    []string
}

// cliEntry describes one subcommand for the reference screen.
type cliEntry struct {
	cmd     string
	usage   string
	summary string
}

// cliEntries is the single source of truth for `./drogonclaw help` and for
// unknown-command hints, so the dispatcher and its documentation cannot drift.
var cliEntries = []cliEntry{
	{"", "", "(no command)  Launch the interactive TUI"},
	{"sandbox", "", "Launch the TUI with the Docker/Kali sandbox forced on"},
	{"native", "[<env details>]", "Launch the TUI in native host mode"},
	{"setup", "", "Run the interactive configuration wizard"},
	{"health", "", "Run runtime diagnostics and dependency checks"},
	{"bench", "[--set FILE] [--out DIR] [-c N] [--timeout D]", "Run the autonomous benchmark suite"},
	{"whitebox", "-u URL [-r REPO] [-o OUT] [-s SESSION] [--no-verify]", "Run the autonomous white-box web/API pipeline"},
	{"daemon", "", "Run headless as a Telegram daemon"},
	{"version", "", "Print build version and metadata"},
	{"help", "", "Show this reference"},
}

var (
	cliTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#58a6ff")).
			Bold(true)
	cliDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6e7681"))
	cliMutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b949e"))
	cliRuleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#30363d"))
	cliCmdStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3fb950")).
			Bold(true)
)

// runStandaloneCommand executes the subcommands that finish before the runtime
// boots. It reports whether a subcommand was handled; the caller exits with the
// returned code immediately when handled is true, so every terminal action
// shares one exit path.
func runStandaloneCommand(opts cliOptions, cfg *config.Manager) (handled bool, code int) {
	switch opts.action {
	case actionHelp:
		printCLIHelp(os.Stdout)
		return true, 0
	case actionVersion:
		printVersion(os.Stdout)
		return true, 0
	case actionSetup:
		tui.RunSetup(cfg)
		return true, 0
	case actionBench:
		runBenchmark(cfg, opts.extraArgs)
		return true, 0
	case actionWhitebox:
		runWhitebox(cfg, opts.extraArgs)
		return true, 0
	}
	return false, 0
}

// parseCLI inspects the raw arguments and classifies the requested action.
// Unknown commands return a descriptive error so the operator is never silently
// dropped into the TUI on a typo.
func parseCLI(args []string) (cliOptions, error) {
	opts := cliOptions{action: actionTUI}
	if len(args) == 0 {
		return opts, nil
	}
	switch args[0] {
	case "help", "-h", "--help":
		opts.action = actionHelp
	case "version", "-v", "--version":
		opts.action = actionVersion
	case "setup":
		opts.action = actionSetup
	case "bench":
		opts.action = actionBench
		opts.extraArgs = args[1:]
	case "whitebox":
		opts.action = actionWhitebox
		opts.extraArgs = args[1:]
	case "health":
		opts.action = actionHealth
	case "daemon":
		opts.action = actionDaemon
	case "sandbox":
		t := true
		opts.forceSandbox = &t
	case "native":
		t := false
		opts.forceSandbox = &t
		if len(args) > 1 {
			opts.extraArgs = args[1:]
		}
	default:
		return opts, fmt.Errorf("unknown command: %s", args[0])
	}
	return opts, nil
}

// applyRunMode applies the forced sandbox/native mode as an environment hint
// and prints a launch banner. It is a no-op when no mode was forced.
func applyRunMode(opts cliOptions) {
	if opts.forceSandbox == nil {
		return
	}
	if *opts.forceSandbox {
		os.Setenv("USE_SANDBOX", "true")
		fmt.Println("  [+] Launching in SANDBOX mode (Docker/Kali)")
		return
	}
	os.Setenv("USE_SANDBOX", "false")
	fmt.Println("  [+] Launching in NATIVE mode (host OS)")
	if len(opts.extraArgs) > 0 {
		fmt.Printf("  [*] Environment details: %s\n", strings.Join(opts.extraArgs, " "))
	}
}

// printVersion prints build identity and runtime metadata.
func printVersion(w io.Writer) {
	fmt.Fprintln(w, cliTitleStyle.Render("DrogonClaw "+version))
	fmt.Fprintf(w, "  %s %s\n", cliDimStyle.Render("build   "), cliMutedStyle.Render(buildTime))
	fmt.Fprintf(w, "  %s %s\n", cliDimStyle.Render("go      "), cliMutedStyle.Render(runtime.Version()))
}

// printCLIHelp renders the graphical sub-command reference.
func printCLIHelp(w io.Writer) {
	fmt.Fprintln(w, cliTitleStyle.Render("DrogonClaw — Autonomous Offensive Security AI"))
	fmt.Fprintln(w, cliDimStyle.Render("usage: drogonclaw [command] [flags]"))
	fmt.Fprintln(w, cliRuleStyle.Render(strings.Repeat("─", 60)))

	maxCmd := 0
	for _, e := range cliEntries {
		if len(e.cmd) > maxCmd {
			maxCmd = len(e.cmd)
		}
	}

	for _, e := range cliEntries {
		left := e.cmd
		if left == "" {
			left = "(none)"
		}
		pad := strings.Repeat(" ", max(1, maxCmd+2-len(left)))
		line := "  " + cliCmdStyle.Render(left) + pad
		if e.usage != "" {
			line += cliMutedStyle.Render(e.usage) + "   "
		}
		fmt.Fprintf(w, "%s%s\n", line, cliDimStyle.Render(e.summary))
	}

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, cliMutedStyle.Render("  Run 'drogonclaw' with no arguments, or 'drogonclaw sandbox' / 'drogonclaw native' to force an execution mode."))
	fmt.Fprintln(w, cliMutedStyle.Render("  Inside the TUI, type /help or press Ctrl+P for the command palette."))
}

// printCLIUnknownError renders the one-line + hint used on a bad subcommand.
func printCLIUnknownError(w io.Writer) {
	fmt.Fprintln(w, cliDimStyle.Render("  Run 'drogonclaw help' for a full list of commands."))
}
