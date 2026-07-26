package core

import (
	"context"
	"sync"
)

// HitLManager manages the Human-in-the-Loop state.
type HitLManager struct {
	mu            sync.Mutex
	pending       bool
	pendingAnswer string
	waitCh        chan struct{}
}

// GlobalHitL is the singleton instance used across the app (TUI, Telegram, Orchestrator).
var GlobalHitL = &HitLManager{}

// RequestApproval sets the HitL state to pending and returns the suspension string.
func (h *HitLManager) RequestApproval() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pending {
		return "[HitL_SUSPENDED]"
	}
	h.pending = true
	h.pendingAnswer = ""
	h.waitCh = make(chan struct{})
	return "[HitL_SUSPENDED]"
}

// HasPending returns true if an approval is currently awaited.
func (h *HitLManager) HasPending() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pending
}

// Resolve provides the human's answer to the pending request.
func (h *HitLManager) Resolve(ans string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.pending {
		return
	}
	h.pending = false
	h.pendingAnswer = ans
	close(h.waitCh)
	h.waitCh = nil
}

// Wait blocks without polling until the pending approval is resolved or the
// run context is cancelled. Call ConsumeAnswer after Wait returns successfully.
func (h *HitLManager) Wait(ctx context.Context) error {
	h.mu.Lock()
	if !h.pending {
		h.mu.Unlock()
		return nil
	}
	ch := h.waitCh
	h.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
}

// ConsumeAnswer reads the answer and clears it.
func (h *HitLManager) ConsumeAnswer() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	ans := h.pendingAnswer
	h.pendingAnswer = ""
	return ans
}
