package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/agent"
	"github.com/0xP4X/drogonclaw-go/internal/benchmark"
	"github.com/0xP4X/drogonclaw-go/internal/config"
	"github.com/0xP4X/drogonclaw-go/internal/core"
	"github.com/0xP4X/drogonclaw-go/internal/gateway"
	"github.com/0xP4X/drogonclaw-go/internal/health"
	"github.com/0xP4X/drogonclaw-go/internal/intel"
	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/0xP4X/drogonclaw-go/internal/opsec"
	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
	"github.com/0xP4X/drogonclaw-go/internal/skills"
	"github.com/0xP4X/drogonclaw-go/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func runStartup(cfg *config.Manager) (*agent.Provider, *sandbox.Docker, *skills.Manifest, error) {
	var manifest *skills.Manifest
	var sb *sandbox.Docker
	var provider *agent.Provider

	steps := []string{
		"Loading skills manifest",
		"Initializing sandbox runtime",
		"Connecting to model provider",
		"Preparing workspace",
	}

	err := tui.RunLoadingSteps(steps, func(i int) error {
		switch i {
		case 0:
			var loadErr error
			manifest, loadErr = skills.Load("skills_manifest.json")
			if loadErr != nil {
				manifest = &skills.Manifest{}
			}
			return nil
		case 1:
			var initErr error
			sb, initErr = sandbox.New()
			if initErr != nil {
				return initErr
			}
			initCtx := context.Background()
			return sb.Initialize(initCtx, !cfg.IsSandboxEnabled())
		case 2:
			provider = agent.NewProvider(cfg)
			pingCtx, pingCancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer pingCancel()
			return provider.Ping(pingCtx)
		case 3:
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return provider, sb, manifest, nil
}

func main() {
	cfg := config.Get()
	rand.Seed(time.Now().UnixNano())

	core.InitCleanupHandler()
	defer core.PerformCleanup()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "sandbox":
			os.Setenv("USE_SANDBOX", "true")
			fmt.Println("  [+] Launching in SANDBOX mode (Docker/Kali)")
		case "native":
			os.Setenv("USE_SANDBOX", "false")
			fmt.Println("  [+] Launching in NATIVE mode (host OS)")
			if len(os.Args) > 2 {
				details := strings.Join(os.Args[2:], " ")
				fmt.Printf("  [*] Environment details: %s\n", details)
			}
		}
	}

	if len(os.Args) > 1 && os.Args[1] == "setup" {
		tui.RunSetup(cfg)
		os.Exit(0)
	}

	if len(os.Args) > 1 && os.Args[1] == "bench" {
		runBenchmark(cfg, os.Args[2:])
		os.Exit(0)
	}

	if len(os.Args) > 1 && os.Args[1] == "bench" {
		runBenchmark(cfg, os.Args[2:])
		os.Exit(0)
	}

	provider, sb, manifest, err := runStartup(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [x] Startup failed: %v\n", err)
		os.Exit(1)
	}

	if sb == nil {
		fmt.Fprintf(os.Stderr, "  [x] Sandbox initialization failed\n")
		os.Exit(1)
	}

	sessionID := fmt.Sprintf("ss%010d", rand.Intn(9000000000)+1000000000)
	graph := memory.NewGraph(sessionID)

	if len(os.Args) > 1 && os.Args[1] == "health" {
		out := health.RunDiagnostics(context.Background(), sb)
		fmt.Println(out)
		os.Exit(0)
	}

	lootDb, err := memory.NewLootDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [!] Loot DB init failed: %v\n", err)
	} else {
		defer lootDb.Close()
	}

	validator := agent.NewEvidenceValidator(provider)
	tools := agent.NewToolRegistry(manifest, sb, validator, lootDb, cfg, graph, provider)

	go func() {
		if _, err := intel.LoadCVEDatabase(); err != nil {
			fmt.Fprintf(os.Stderr, "  [!] CVE DB load failed: %v\n", err)
		}
	}()

	opsecMgr := opsec.NewManager()
	sysPrompt := agent.BuildSystemPrompt(graph, opsecMgr, "", sb.RuntimeLabel())
	orch := agent.NewOrchestratorWithJournal(provider, tools, sysPrompt, sessionID, graph, memory.NewActionJournal(sessionID), cfg.GetMaxIterations())

	tgGateway, err := gateway.NewTelegramGateway(cfg, orch, graph, opsecMgr)
	if err == nil && tgGateway != nil {
		tgGateway.Start()
	}

	if len(os.Args) > 1 && os.Args[1] == "daemon" {
		if tgGateway == nil {
			fmt.Fprintf(os.Stderr, "  [x] Cannot run in daemon mode: Telegram gateway is not configured.\n")
			fmt.Fprintf(os.Stderr, "  [!] Run './drogonclaw setup' to configure Telegram.\n")
			os.Exit(1)
		}
		fmt.Println("  [+] DrogonClaw running in Daemon mode. Listening via Telegram...")
		select {}
	}

	model, err := tui.New(orch, graph, opsecMgr, cfg, manifest, sb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [x] TUI init failed: %v\n", err)
		os.Exit(1)
	}

	model.SetPromptRefresher(func() string {
		return agent.BuildSystemPrompt(graph, opsecMgr, "", sb.RuntimeLabel())
	})

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "  [x] TUI crashed: %v\n", err)
		os.Exit(1)
	}
}

// runBenchmark handles the `drogonclaw bench` subcommand.
//
//	./drogonclaw bench --set benchmarks/xben/set.json --out benchmark_runs
//
// It loads a challenge set, runs each through the agent's headless ReAct loop,
// and writes an XBEN-style report (report.md + results.json) to the output dir.
func runBenchmark(cfg *config.Manager, args []string) {
	var setPath, outDir string
	timeout := 15 * time.Minute

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--set", "-set":
			if i+1 < len(args) {
				setPath = args[i+1]
				i++
			}
		case "--out", "-out":
			if i+1 < len(args) {
				outDir = args[i+1]
				i++
			}
		case "--timeout", "-timeout":
			if i+1 < len(args) {
				if d, err := time.ParseDuration(args[i+1]); err == nil {
					timeout = d
				}
				i++
			}
		default:
			if setPath == "" && !strings.HasPrefix(args[i], "-") {
				setPath = args[i]
			}
		}
	}

	if setPath == "" {
		setPath = "benchmarks/sample/set.json"
	}
	if outDir == "" {
		outDir = "benchmark_runs"
	}

	fmt.Printf("  [*] Loading benchmark set: %s\n", setPath)
	set, err := benchmark.LoadSet(setPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [x] %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  [*] %d challenges loaded. Running (timeout %s each)...\n", len(set.Challenges), timeout)

	summary, err := benchmark.Run(context.Background(), set, cfg, benchmark.RunMode{ChallengeTimeout: timeout})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [x] benchmark run failed: %v\n", err)
		os.Exit(1)
	}

	reportPath, err := summary.Write(outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [x] writing report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n  ✔ Benchmark complete: %d/%d solved (%.1f%%)\n", summary.Solved, summary.Total, summary.SuccessRate)
	fmt.Printf("  ✔ Report: %s\n", reportPath)
}
