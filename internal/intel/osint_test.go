package intel

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCertTransparencyFallback(t *testing.T) {
	orig := httpClient
	defer func() { httpClient = orig }()
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var status int
		var body string
		switch {
		case strings.Contains(r.URL.String(), "crt.sh"):
			status = 404
			body = "not found"
		case strings.Contains(r.URL.String(), "certspotter"):
			status = 200
			body = `[{"dns_names":["lyrie.ai","www.lyrie.ai","api.lyrie.ai","*.lyrie.ai"]}]`
		default:
			status = 404
			body = "n/a"
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	res, err := CertTransparencyLookup("lyrie.ai")
	if err != nil {
		t.Fatalf("CertTransparencyLookup error: %v", err)
	}
	if res.Source != "certspotter" {
		t.Errorf("Source = %q, want certspotter (crt.sh 404 should fall back)", res.Source)
	}
	subs, _ := res.Data["subdomains"].([]string)
	if len(subs) < 3 {
		t.Errorf("expected >=3 subdomains from certspotter, got %v", subs)
	}
}
