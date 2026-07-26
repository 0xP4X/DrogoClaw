package intel

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildPublicProfilePreservesEvidenceAndGaps(t *testing.T) {
	deps := ProfileDependencies{
		DNS: func(string) (*OSINTResult, error) {
			return &OSINTResult{Data: map[string]interface{}{"A": []string{"203.0.113.10"}}}, nil
		},
		RDAP: func(string) (*OSINTResult, error) { return nil, errors.New("rate limited") },
		Certs: func(string) (*OSINTResult, error) {
			return &OSINTResult{Data: map[string]interface{}{"subdomains": []string{"www.example.test"}, "count": 1}}, nil
		},
	}
	profile, err := BuildPublicProfile("https://Example.Test/login", "", "", deps)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Host != "example.test" || profile.Kind != "domain" {
		t.Fatalf("unexpected target normalization: %+v", profile)
	}
	formatted := FormatPublicProfile(profile)
	if !strings.Contains(formatted, "203.0.113.10") || !strings.Contains(formatted, "rate limited") || !strings.Contains(formatted, "Coverage gaps: RDAP, Shodan, VirusTotal") {
		t.Fatalf("missing evidence or gap: %s", formatted)
	}
}

func TestBuildPublicProfileDoesNotUseDomainCollectorsForIP(t *testing.T) {
	deps := ProfileDependencies{
		DNS:   func(string) (*OSINTResult, error) { t.Fatal("DNS should not run for an IP"); return nil, nil },
		Certs: func(string) (*OSINTResult, error) { t.Fatal("certs should not run for an IP"); return nil, nil },
		RDAP:  func(string) (*OSINTResult, error) { return &OSINTResult{}, nil },
	}
	profile, err := BuildPublicProfile("203.0.113.5", "", "", deps)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Kind != "ip" || profile.Evidence[0].Status != "not_applicable" {
		t.Fatalf("unexpected IP profile: %+v", profile)
	}
}

func TestEmailHarvestDoesNotFallBackToSearchDorks(t *testing.T) {
	if _, err := EmailHarvest("example.test", ""); err == nil || !strings.Contains(err.Error(), "search-engine dorks") {
		t.Fatalf("expected the dork fallback to be disabled, got %v", err)
	}
}
