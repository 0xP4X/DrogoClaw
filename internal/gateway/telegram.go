package gateway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/agent"
	"github.com/0xP4X/drogonclaw-go/internal/config"
	"github.com/0xP4X/drogonclaw-go/internal/core"
	"github.com/0xP4X/drogonclaw-go/internal/memory"
	"github.com/0xP4X/drogonclaw-go/internal/opsec"
	tele "gopkg.in/telebot.v3"
)

type TelegramGateway struct {
	bot      *tele.Bot
	chatID   int64
	orch     *agent.Orchestrator
	graph    *memory.Graph
	opsecMgr *opsec.Manager
	manifest any
}

func NewTelegramGateway(cfg *config.Manager, orch *agent.Orchestrator, graph *memory.Graph, opsecMgr *opsec.Manager) (*TelegramGateway, error) {
	token := cfg.GetString("TELEGRAM_TOKEN")
	chatIDStr := cfg.GetString("TELEGRAM_CHAT_ID")

	if token == "" || chatIDStr == "" {
		return nil, nil // Disabled
	}

	var chatID int64
	fmt.Sscanf(chatIDStr, "%d", &chatID)

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
	}

	tg.setupHandlers()
	return tg, nil
}

func (tg *TelegramGateway) Start() {
	go tg.bot.Start()
}

func (tg *TelegramGateway) setupHandlers() {
	tg.bot.Handle(tele.OnText, func(c tele.Context) error {
		// Strict security block
		if c.Chat().ID != tg.chatID {
			return nil
		}

		text := c.Text()

		// Handle HitL responses
		if core.GlobalHitL.HasPending() {
			core.GlobalHitL.Resolve(text)
			return c.Send("✅ Approval received. Resuming execution.")
		}

		if strings.HasPrefix(text, "/") {
			return tg.handleCommand(c, text)
		}

		// Otherwise, it's a mission
		return tg.executeMission(c, text)
	})
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
		_ = c.Send("🐝 Engaging Swarm Command...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		events := make(chan agent.Event, 32)
		go func() {
			defer close(events)
			commander := agent.NewSwarmCommander(tg.orch.GetProvider(), tg.orch.GetTools(), "", tg.orch.SessionID, tg.graph)
			res, err := commander.ExecuteSwarm(ctx, mission, events)
			if err != nil {
				_, _ = tg.bot.Send(c.Chat(), fmt.Sprintf("Swarm failed: %v", err))
			} else {
				// Chunk and send
				tg.chunkSend(c.Chat(), res)
			}
		}()

		// Drain events to avoid blocking
		go func() {
			for range events {
			}
		}()

		return nil

	default:
		return c.Send("Unknown command. Supported: /report, /swarm")
	}
}

func (tg *TelegramGateway) executeMission(c tele.Context, mission string) error {
	msg, err := tg.bot.Send(c.Chat(), "⚡ Initializing mission vector...")
	if err != nil {
		return err
	}

	events := make(chan agent.Event, 32)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)

	go func() {
		defer cancel()
		tg.orch.Execute(ctx, mission, events)
	}()

	var fullResponse strings.Builder
	var lastUpdate time.Time

	for ev := range events {
		switch ev.Type {
		case agent.EvToolStart:
			_ = tg.editMessage(msg, fmt.Sprintf("🛠 *Executing Tool:* `%s`\n\n```json\n%s\n```", ev.Tool, ev.Args))
		case agent.EvStatus:
			// Catch HitL / duration-approval prompts
			if strings.Contains(strings.ToLower(ev.Content), "awaiting") {
				_ = tg.editMessage(msg, "⚠️ *AGENT REQUIRES APPROVAL*\n\nThe agent has suspended execution and requires human input. Reply `y` to accept or `n` to skip, otherwise your message is taken as approval.")
			} else {
				if time.Since(lastUpdate) > 2*time.Second {
					_ = tg.editMessage(msg, fmt.Sprintf("⚡ %s", ev.Content))
					lastUpdate = time.Now()
				}
			}
		case agent.EvToken:
			fullResponse.WriteString(ev.Content)
			if time.Since(lastUpdate) > 2*time.Second {
				_ = tg.editMessage(msg, fullResponse.String())
				lastUpdate = time.Now()
			}
		}
	}

	// Final update
	if fullResponse.Len() > 0 {
		_ = tg.editMessage(msg, fullResponse.String())
	} else {
		_ = tg.editMessage(msg, "✅ Mission complete.")
	}

	return nil
}

func (tg *TelegramGateway) editMessage(msg *tele.Message, text string) error {
	if len(text) > 4000 {
		text = text[:4000] + "...\n[TRUNCATED]"
	}
	_, err := tg.bot.Edit(msg, text, tele.ModeMarkdown)
	return err
}

func (tg *TelegramGateway) chunkSend(chat *tele.Chat, text string) {
	for len(text) > 4000 {
		_, _ = tg.bot.Send(chat, text[:4000])
		text = text[4000:]
	}
	if len(text) > 0 {
		_, _ = tg.bot.Send(chat, text)
	}
}
