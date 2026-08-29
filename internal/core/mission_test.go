package core

import "testing"

func TestReconTarget(t *testing.T) {
	cases := []struct{ in, want string }{
		{"whois example.com please", "example.com"},
		{"scan 10.0.0.0/24", "10.0.0.0/24"},
		{"enumerate 192.168.1.5", "192.168.1.5"},
		{"find subdomains of acme.io", "acme.io"},
		{"how does DNS work?", ""},
		{"hello there", ""},
		{"fuzz https://app.example.org/login", "app.example.org"},
	}
	for _, c := range cases {
		if got := reconTarget(c.in); got != c.want {
			t.Errorf("reconTarget(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLooksLikeReconMission(t *testing.T) {
	if p := looksLikeReconMission("whois example.com"); p == nil || !p.IsValidMission {
		t.Fatal("whois example.com should be planned as a recon mission")
	} else if len(p.Steps) < 1 || p.Steps[0].TargetAssetID != "example.com" {
		t.Fatalf("expected a passive recon step targeting example.com, got %+v", p.Steps)
	}

	if p := looksLikeReconMission("find all subdomains of example.com"); p == nil || !p.IsValidMission {
		t.Fatal("subdomain enumeration should be a recon mission")
	}

	if p := looksLikeReconMission("scan 10.0.0.0/24 for open ports"); p == nil || !p.IsValidMission {
		t.Fatal("port scan should be a recon mission")
	}

	if p := looksLikeReconMission("what is your favorite color?"); p != nil {
		t.Fatal("plain chit-chat must NOT become a mission")
	}

	if p := looksLikeReconMission("good morning"); p != nil {
		t.Fatal("greeting must NOT become a mission")
	}
}
