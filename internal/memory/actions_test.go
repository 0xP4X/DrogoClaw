package memory

import (
	"path/filepath"
	"testing"
)

func TestActionJournalRecoversInFlightTool(t *testing.T) {
	path := filepath.Join(t.TempDir(), "actions.json")
	j := &ActionJournal{path: path}
	j.Begin("enumerate the approved target")
	j.SetPlan([]string{"collect passive evidence", "verify services"})
	j.ToolStarted("run_nmap", `{"target":"example.test"}`)

	// Reopening models the next process after an unexpected exit.
	reopened := &ActionJournal{path: path}
	reopened.load()
	recovery := reopened.Recovery()
	if recovery == nil {
		t.Fatal("expected interrupted action to be recoverable")
	}
	if recovery.Status != "interrupted" || recovery.CurrentTool != "run_nmap" {
		t.Fatalf("unexpected recovery state: %#v", recovery)
	}
	if recovery.Objective != "enumerate the approved target" || len(recovery.Plan) != 2 {
		t.Fatalf("recovery lost mission context: %#v", recovery)
	}
}

func TestActionJournalDoesNotOfferCompletedRun(t *testing.T) {
	j := &ActionJournal{path: filepath.Join(t.TempDir(), "actions.json")}
	j.Begin("completed objective")
	j.ToolStarted("profile_target", "target=example.test")
	j.ToolFinished("profile_target", "profile completed")
	j.Finish("completed", "")
	if got := j.Recovery(); got != nil {
		t.Fatalf("completed mission should not be resumable: %#v", got)
	}
}
