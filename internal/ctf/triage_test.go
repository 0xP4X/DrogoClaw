package ctf

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunLocalTriageFindsAndVerifiesFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "challenge.txt")
	if err := os.WriteFile(path, []byte("clue\nCTF{evidence_not_vibes}\n"), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := RunLocalTriage(context.Background(), LocalTask{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "verified" || len(result.Candidates) != 1 || result.Candidates[0] != "CTF{evidence_not_vibes}" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRunLocalTriageFindsFlagInsideZipWithProvenance(t *testing.T) {
	dir := t.TempDir()
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	member, err := zw.Create("clues/final.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := member.Write([]byte("CTF{inside_the_archive}")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "challenge.zip")
	if err := os.WriteFile(path, archive.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := RunLocalTriage(context.Background(), LocalTask{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "verified" || len(result.Evidence) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	evidence := result.Evidence[0]
	if evidence.Candidate != "CTF{inside_the_archive}" || evidence.ObservationID == "" || evidence.Source != path+"::clues/final.txt" {
		t.Fatalf("unexpected evidence: %+v", evidence)
	}
}

func TestRunLocalTriageDoesNotClaimSuccessWithoutEvidence(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("nothing to see here"), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := RunLocalTriage(context.Background(), LocalTask{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "needs_analysis" || len(result.Candidates) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}
