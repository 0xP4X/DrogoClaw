package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestHitLWaitResolvesWithoutPolling(t *testing.T) {
	h := &HitLManager{}
	h.RequestApproval()

	done := make(chan error, 1)
	go func() {
		done <- h.Wait(context.Background())
	}()

	select {
	case err := <-done:
		t.Fatalf("wait returned before approval: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	h.Resolve("approved")
	if err := <-done; err != nil {
		t.Fatalf("wait returned an error after approval: %v", err)
	}
	if answer := h.ConsumeAnswer(); answer != "approved" {
		t.Fatalf("expected approval answer, got %q", answer)
	}
}

func TestHitLWaitHonorsCancellation(t *testing.T) {
	h := &HitLManager{}
	h.RequestApproval()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := h.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
