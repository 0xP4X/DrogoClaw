package intel

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestProbeWHOIS is an opt-in live check (DC_PROBE=1) that verifies the RDAP
// endpoint is reachable and parses. It is excluded from the normal suite so
// offline CI never depends on external DNS/HTTPS.
func TestProbeWHOIS(t *testing.T) {
	if os.Getenv("DC_PROBE") != "1" {
		t.Skip("set DC_PROBE=1 to run the live WHOIS probe")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = ctx
	res, err := WHOISLookup("example.com")
	if err != nil {
		t.Fatalf("WHOIS lookup failed: %v", err)
	}
	if res.Summary == "" {
		t.Fatal("WHOIS lookup returned no content")
	}
}
