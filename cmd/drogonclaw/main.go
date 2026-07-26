package main

import (
	"context"
	"fmt"
	"os"
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

func main() {
	cfg := config.Get()

	core.InitCleanupHandler()
	defer core.PerformCleanup()

	// Handle setup command
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		tui.RunSetup(cfg)
		os.Exit(0)
	}

	// Load skills manifest
	manifest, err := skills.Load("skills_manifest.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [!] Warning: %v\n  [!] Run: node scripts/gen_skill_manifest.mjs\n", err)
		manifest = &skills.Manifest{} // empty manifest — still works for chat
	}

	// Initialize memory graph
	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())
	graph := memory.NewGraph("default") // persist across sessions

	// Initialize sandbox
	sb, err := sandbox.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [x] Docker client init failed: %v\n", err)
		os.Exit(1)
	}

	// Give the sandbox unlimited time to initialize, as pulling the kali image
	// for the first time can take a long time on slow network connections.
	// We do not want to artificially timeout a 2GB+ download.
	initCtx := context.Background()

	if err := sb.Initialize(initCtx, !cfg.IsSandboxEnabled()); err != nil {
		fmt.Fprintf(os.Stderr, "  [x] Sandbox initialization failed: %v\n", err)
		fmt.Fprintln(os.Stderr, "  [!] Refusing to fall back to host command execution automatically. Set USE_SANDBOX=false explicitly only when you intend to use native mode.")
		os.Exit(1)
	}

	// Handle health command
	if len(os.Args) > 1 && os.Args[1] == "health" {
		out := health.RunDiagnostics(context.Background(), sb)
		fmt.Println(out)
		os.Exit(0)
	}

	// OPSEC manager
	opsecMgr := opsec.NewManager()

	// Automated UX Fallback logic for unconfigured clients
	needsSetup := false

	var provider *agent.Provider
	provider = agent.NewProvider(cfg)
	// Use a generous 20s timeout — cold models on NVIDIA/Ollama/OpenRouter can be slow to respond.
	// 10s was too aggressive and triggered false-positive setup wizard launches.
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer pingCancel()
	if err := provider.Ping(pingCtx); err != nil {
		needsSetup = true
	}

	if needsSetup {
		fmt.Fprintf(os.Stderr, "  [!] Configuration missing or invalid. Launching setup wizard...\n")
		tui.RunSetup(cfg)
		// Reload configuration after setup
		cfg = config.Get()

		// Final LLM provider initialization
		provider = agent.NewProvider(cfg)
		// Give the freshly configured provider 45s — Ollama models may need to load from disk.
		pingCtx2, pingCancel2 := context.WithTimeout(context.Background(), 45*time.Second)
		defer pingCancel2()
		if err := provider.Ping(pingCtx2); err != nil {
			fmt.Fprintf(os.Stderr, "  [x] LLM provider unreachable after setup: %v\n", err)
			fmt.Fprintf(os.Stderr, "  [!] Check your API key, model name, and network connection.\n")
			os.Exit(1)
		}
	}

	// Init Loot DB
	lootDb, err := memory.NewLootDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [!] Loot DB init failed: %v\n", err)
	} else {
		defer lootDb.Close()
	}

	// Validator
	validator := agent.NewEvidenceValidator(provider)

	// Tool registry
	tools := agent.NewToolRegistry(manifest, sb, validator, lootDb, cfg, graph, provider)

	// Boot CVE database in background (non-blocking)
	go func() {
		if _, err := intel.LoadCVEDatabase(); err != nil {
			fmt.Fprintf(os.Stderr, "  [!] CVE DB load failed: %v\n", err)
		}
	}()

	// Build initial system prompt
	sysPrompt := agent.BuildSystemPrompt(graph, opsecMgr, "")

	// Orchestrator
	orch := agent.NewOrchestratorWithJournal(provider, tools, sysPrompt, sessionID, graph, memory.NewActionJournal("default"))

	// Telegram C2 Gateway
	tgGateway, err := gateway.NewTelegramGateway(cfg, orch, graph, opsecMgr)
	if err == nil && tgGateway != nil {
		tgGateway.Start()
	}

	// Handle daemon mode
	if len(os.Args) > 1 && os.Args[1] == "daemon" {
		if tgGateway == nil {
			fmt.Fprintf(os.Stderr, "  [x] Cannot run in daemon mode: Telegram gateway is not configured.\n")
			fmt.Fprintf(os.Stderr, "  [!] Run './drogonclaw setup' to configure Telegram.\n")
			os.Exit(1)
		}
		fmt.Println("  [+] DrogonClaw running in Daemon mode. Listening via Telegram...")
		// Block forever
		select {}
	}

	// Build TUI
	model, err := tui.New(orch, graph, opsecMgr, cfg, manifest, sb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [x] TUI init failed: %v\n", err)
		os.Exit(1)
	}

	model.SetPromptRefresher(func() string {
		return agent.BuildSystemPrompt(graph, opsecMgr, "")
	})

	// Launch Bubbletea
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "  [x] TUI crashed: %v\n", err)
		os.Exit(1)
	}
}
