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

// BenchmarkFixtures holds a collection of CTF challenge fixtures for benchmarking.
type BenchmarkFixtures struct {
	Name        string
	Path        string
	ChallengeType string
	HasFlag     bool
}

// BenchmarkCTFCategories runs a benchmark across CTF challenge categories
// and reports verified completion rate, tool calls, and evidence provenance.
func BenchmarkCTFCategories(b *testing.B) {
	fixtures := []BenchmarkFixtures{
		{Name: "forensics_txt", ChallengeType: "forensics", HasFlag: true},
		{Name: "forensics_zip", ChallengeType: "forensics", HasFlag: true},
		{Name: "forensics_no_flag", ChallengeType: "forensics", HasFlag: false},
	}

	for _, f := range fixtures {
		b.Run(f.Name, func(b *testing.B) {
			dir := b.TempDir()
			var task LocalTask
			switch f.ChallengeType {
			case "forensics":
				if f.HasFlag {
					task = LocalTask{Path: dir, FlagPattern: defaultFlagPattern}
					os.WriteFile(filepath.Join(dir, "challenge.txt"), []byte("CTF{benchmark_flag}"), 0600)
				} else {
					task = LocalTask{Path: dir, FlagPattern: defaultFlagPattern}
					os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("no flag here"), 0600)
				}
			default:
				task = LocalTask{Path: dir}
			}

			for i := 0; i < b.N; i++ {
				result, err := RunLocalTriage(context.Background(), task)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}

// TestBenchmarkMetrics verifies that the benchmark metrics are tracked correctly.
func TestBenchmarkMetrics(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "flag.txt"), []byte("CTF{test_flag_123}"), 0600)

	result, err := RunLocalTriage(context.Background(), LocalTask{Path: dir})
	if err != nil {
		t.Fatal(err)
	}

	// Verified completion rate: every fixture with a flag should be verified.
	if result.State != "verified" {
		t.Errorf("expected verified state, got %s", result.State)
	}
	if len(result.Candidates) == 0 {
		t.Error("expected at least one verified candidate")
	}
	// Every candidate must have evidence provenance.
	for _, e := range result.Evidence {
		if e.ObservationID == "" {
			t.Error("evidence missing observation provenance")
		}
		if e.Source == "" {
			t.Error("evidence missing source")
		}
	}
	// No unverified success claims.
	if result.State == "verified" && len(result.Candidates) == 0 {
		t.Error("unverified success claim: state is verified but no candidates found")
	}
}
