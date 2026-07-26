package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/memory"
)

// Phase represents a stage in the Red Team lifecycle.
type Phase string

const (
	PhaseRecon           Phase = "RECONNAISSANCE"
	PhasePendingApproval Phase = "PENDING_APPROVAL"
	PhaseExploitation    Phase = "EXPLOITATION"
	PhasePostExploiting  Phase = "POST_EXPLOITATION"
	PhaseComplete        Phase = "COMPLETE"
	// PhaseNeedsHuman is terminal. It explicitly records that no live
	// exploitation capability is available instead of fabricating a finding.
	PhaseNeedsHuman Phase = "NEEDS_HUMAN"
)

// Engagement defines a targeted Red Team operation.
type Engagement struct {
	mu         sync.RWMutex
	approvalCh chan struct{}
	ID         string    `json:"id"`
	Target     string    `json:"target"`
	Status     Phase     `json:"status"`
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	Graph      *memory.Graph
}

// Orchestrator manages Red Team engagements.
type Orchestrator struct {
	mu                sync.RWMutex
	ActiveEngagements map[string]*Engagement
}

func New() *Orchestrator {
	return &Orchestrator{
		ActiveEngagements: make(map[string]*Engagement),
	}
}

// StartEngagement initializes a new pentest operation.
func (o *Orchestrator) StartEngagement(target string) *Engagement {
	eng := &Engagement{
		ID:         fmt.Sprintf("RT-%d", time.Now().UnixNano()),
		approvalCh: make(chan struct{}),
		Target:     target,
		Status:     PhaseRecon,
		StartTime:  time.Now(),
		Graph:      memory.NewGraph(fmt.Sprintf("rt_graph_%d", time.Now().Unix())),
	}
	o.mu.Lock()
	o.ActiveEngagements[eng.ID] = eng
	o.mu.Unlock()
	return eng
}

// ApproveEngagement resumes execution of a paused engagement.
func (o *Orchestrator) ApproveEngagement(id string) error {
	eng, ok := o.engagement(id)
	if !ok {
		return fmt.Errorf("engagement %s not found", id)
	}
	eng.mu.Lock()
	defer eng.mu.Unlock()
	if eng.Status != PhasePendingApproval {
		return fmt.Errorf("engagement %s is not pending approval", id)
	}
	eng.Status = PhaseExploitation
	close(eng.approvalCh)
	return nil
}

// Execute triggers the main Red Team loop for an engagement.
func (o *Orchestrator) Execute(ctx context.Context, id string) error {
	eng, ok := o.engagement(id)
	if !ok {
		return fmt.Errorf("engagement %s not found", id)
	}

	log.Printf("[Orchestrator] Starting engagement %s against %s", eng.ID, eng.Target)

	for !isTerminal(eng.status()) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			switch eng.status() {
			case PhaseRecon:
				// The API has no scope authorization or typed workflow yet. Do not
				// send network requests merely because a caller supplied a target.
				log.Printf("[Orchestrator] [%s] Awaiting approval; no remote actions have run.", eng.ID)
				eng.setStatus(PhasePendingApproval)

			case PhasePendingApproval:
				// Wait for the explicit approval event instead of polling or sleeping.
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-eng.approvalCh:
				}

			case PhaseExploitation:
				eng.needsHuman("live engagement execution is not implemented; no finding was created")
				log.Printf("[Orchestrator] [%s] Live execution unavailable; no findings created.", eng.ID)

			case PhasePostExploiting:
				log.Printf("[Orchestrator] [%s] Starting Post-Exploitation phase...", eng.ID)
				eng.complete()
				log.Printf("[Orchestrator] [%s] Engagement Complete.", eng.ID)
			}
		}
	}
	return nil
}

func (e *Engagement) status() Phase {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Status
}

func (e *Engagement) setStatus(status Phase) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Status = status
}

func (e *Engagement) complete() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Status = PhaseComplete
	e.EndTime = time.Now()
}

func (e *Engagement) needsHuman(reason string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Status = PhaseNeedsHuman
	e.Reason = reason
	e.EndTime = time.Now()
}

func (o *Orchestrator) engagement(id string) (*Engagement, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	eng, ok := o.ActiveEngagements[id]
	return eng, ok
}

func isTerminal(phase Phase) bool {
	return phase == PhaseComplete || phase == PhaseNeedsHuman
}
