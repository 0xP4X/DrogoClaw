// Package c2 holds implant/command-and-control backends for DrogonClaw. It sits
// beside the Telegram gateway (internal/gateway) as an alternative operator
// control plane for authorized engagements.
//
// Backends orchestrate external C2 frameworks (e.g. Sliver) — they never fork or
// reimplement them. Execution is injected via Executor so the backend stays
// decoupled from the agent/sandbox packages and avoids an import cycle.
package c2

import (
	"context"
	"fmt"
	"strings"
)

// Executor runs a shell command. In the agent this is bound to the sandbox.
type Executor func(ctx context.Context, cmd string) (string, error)

// SliverBackend wraps the operator's own Sliver server (orchestration only).
// Sliver must be installed in the sandbox; this backend drives its CLI rather
// than embedding any Sliver code.
type SliverBackend struct {
	exec Executor
}

// NewSliverBackend constructs a Sliver C2 backend with the given command runner.
func NewSliverBackend(exec Executor) *SliverBackend {
	return &SliverBackend{exec: exec}
}

// Run executes an operator-supplied sliver subcommand, e.g.
//
//	Run(ctx, "generate", "--os windows --http 10.0.0.5 --save /tmp/implant")
//	Run(ctx, "listener", "create --http 10.0.0.5:8443")
//	Run(ctx, "sessions", "")
func (s *SliverBackend) Run(ctx context.Context, subcommand, args string) (string, error) {
	if strings.TrimSpace(subcommand) == "" {
		return "", fmt.Errorf("sliver: subcommand required (e.g. generate, listener, sessions)")
	}
	cmd := "sliver " + subcommand
	if args != "" {
		cmd += " " + args
	}
	out, err := s.exec(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("sliver run failed: %w", err)
	}
	return out, nil
}
