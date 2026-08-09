package benchmark

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// targetHandle tracks a locally-spawned challenge server.
type targetHandle struct {
	cmd *exec.Cmd
}

// startTarget launches a challenge's local server (Cmd) if provided and waits
// briefly for it to come up. It returns a handle whose Close kills the process.
func startTarget(ch Challenge) (*targetHandle, error) {
	if strings.TrimSpace(ch.Cmd) == "" {
		return nil, nil
	}
	cmd := exec.Command("sh", "-c", ch.Cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting target %q: %w", ch.ID, err)
	}
	// Give the server a moment to bind.
	time.Sleep(2 * time.Second)
	return &targetHandle{cmd: cmd}, nil
}

// Close terminates a spawned target server.
func (h *targetHandle) Close() {
	if h == nil || h.cmd == nil || h.cmd.Process == nil {
		return
	}
	_ = h.cmd.Process.Kill()
	_, _ = h.cmd.Process.Wait()
}
