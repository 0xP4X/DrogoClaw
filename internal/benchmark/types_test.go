package benchmark

import (
	"os"
	"testing"
)

func TestChallengeFindFlag(t *testing.T) {
	cases := []struct {
		name     string
		regex    string
		input    string
		wantFlag string
	}{
		{"default_flag", "", "The flag is flag{drogon_test_2026} end.", "flag{drogon_test_2026}"},
		{"custom_regex", "MYFLAG\\{[^}]+\\}", "Result: MYFLAG{custom-123}", "MYFLAG{custom-123}"},
		{"no_match", "", "no flag here sorry", ""},
		{"case_insensitive", "", "FLAG{CaseInsensitive}", "FLAG{CaseInsensitive}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ch := Challenge{FlagRegex: tc.regex}
			got := ch.findFlag(tc.input)
			if got != tc.wantFlag {
				t.Errorf("findFlag(%q) = %q, want %q", tc.input, got, tc.wantFlag)
			}
		})
	}
}

func TestLoadSet(t *testing.T) {
	json := `{
		"name": "test-set",
		"challenges": [
			{"id":"c1","class":"web","goal":"find flag","url":"http://localhost:1","flagRegex":"flag\\{[^}]+\\}"}
		]
	}`
	tmp := t.TempDir() + "/set.json"
	if err := os.WriteFile(tmp, []byte(json), 0644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSet(tmp)
	if err != nil {
		t.Fatalf("LoadSet: %v", err)
	}
	if s.Name != "test-set" {
		t.Errorf("Name = %q, want test-set", s.Name)
	}
	if len(s.Challenges) != 1 || s.Challenges[0].ID != "c1" {
		t.Errorf("Challenges = %v", s.Challenges)
	}
}

func TestSummaryWrite(t *testing.T) {
	s := &Summary{
		Set:     "test",
		Solved:  1,
		Total:   2,
		Outcomes: []Outcome{
			{ID: "c1", Class: "web", Solved: true, Flag: "flag{abc}", Duration: "1s"},
			{ID: "c2", Class: "web", Solved: false, Duration: "2s", Err: "timeout"},
		},
	}
	dir := t.TempDir()
	_, err := s.Write(dir)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	// report.md and results.json must exist
	for _, name := range []string{"report.md", "results.json"} {
		if _, err := os.Stat(dir + "/" + name); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
}
