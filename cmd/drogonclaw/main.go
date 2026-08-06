package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/agent"
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
	sysPrompt := agent.BuildSystemPrompt(graph, opsecMgr, "")
	orch := agent.NewOrchestratorWithJournal(provider, tools, sysPrompt, sessionID, graph, memory.NewActionJournal(sessionID))

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
		return agent.BuildSystemPrompt(graph, opsecMgr, "")
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