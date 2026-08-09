package core

import (
	"context"
	"strings"
	"sync"
)

// ApprovalKind distinguishes why execution is suspended awaiting a human.
type ApprovalKind string

const (
	// ApprovalDanger is the classic Human-in-the-Loop gate (dangerous command,
	// script, or explicit ask_operator that needs a free-form operator reply).
	ApprovalDanger ApprovalKind = "danger"
	// ApprovalDuration is a low-risk but time-consuming tool (e.g. gobuster,
	// ffuf, large nmap) that the operator may simply accept or skip.
	ApprovalDuration ApprovalKind = "duration"
)

// HitLManager manages the Human-in-the-Loop state.
type HitLManager struct {
	mu            sync.Mutex
	pending       bool
	pendingAnswer string
	pendingKind   ApprovalKind
	pendingDetail string
	waitCh        chan struct{}
}

// GlobalHitL is the singleton instance used across the app (TUI, Telegram, Orchestrator).
var GlobalHitL = &HitLManager{}

// RequestApproval sets the HitL state to pending (danger kind) and returns the
// suspension string.
func (h *HitLManager) RequestApproval() string {
	return h.RequestApprovalWithDetail(ApprovalDanger, "")
}

// RequestApprovalWithDetail sets the HitL state to pending with a kind and a
// human-readable detail string (e.g. the estimated runtime). It returns the
// suspension token that callers embed in their result so the orchestrator can
// detect a suspended execution.
func (h *HitLManager) RequestApprovalWithDetail(kind ApprovalKind, detail string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pending {
		return "[HitL_SUSPENDED]"
	}
	h.pending = true
	h.pendingAnswer = ""
	h.pendingKind = kind
	h.pendingDetail = detail
	h.waitCh = make(chan struct{})
	return "[HitL_SUSPENDED]"
}

// HasPending returns true if an approval is currently awaited.
func (h *HitLManager) HasPending() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pending
}

// PendingKind returns the kind of the current pending approval (empty if none).
func (h *HitLManager) PendingKind() ApprovalKind {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pendingKind
}

// PendingDetail returns the human-readable detail for the current pending approval.
func (h *HitLManager) PendingDetail() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pendingDetail
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

// ConsumeApproved reads the answer and reports whether the operator accepted.
// Any empty/affirmative reply (yes/y/approve/run/ok) is treated as accepted;
// explicit rejections (no/n/skip/cancel/deny/reject) are declined. Use this for
// low-risk duration approvals where the reply is not fed back to the model.
func (h *HitLManager) ConsumeApproved() bool {
	ans := strings.ToLower(strings.TrimSpace(h.ConsumeAnswer()))
	switch ans {
	case "", "y", "yes", "approve", "approved", "run", "ok", "go", "accept", "confirmed":
		return true
	}
	return false
}
