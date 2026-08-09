package ctf

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func TestSolveDecodesHiddenFlag(t *testing.T) {
	dir := t.TempDir()

	// Literal flag — must still be found (regression of triage behavior).
	lit := filepath.Join(dir, "lit.txt")
	if err := os.WriteFile(lit, []byte("intro CTF{plain_flag} outro"), 0600); err != nil {
		t.Fatal(err)
	}

	// Hidden flag: base64 of "CTF{encoded_flag}" — the old regex triage would
	// never find this; the solver must decode it.
	enc := filepath.Join(dir, "enc.txt")
	hidden := base64.StdEncoding.EncodeToString([]byte("CTF{encoded_flag}"))
	if err := os.WriteFile(enc, []byte("data="+hidden+" end"), 0600); err != nil {
		t.Fatal(err)
	}

	rs, err := Solve(context.Background(), LocalTask{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !rs.Verified {
		t.Fatalf("expected verified solve, got state=%s note=%s facts=%+v", rs.State, rs.VerifierNote, rs.Facts)
	}
	if !contains(rs.Candidates, "CTF{plain_flag}") || !contains(rs.Candidates, "CTF{encoded_flag}") {
		t.Fatalf("missing candidates: %+v", rs.Candidates)
	}
}
