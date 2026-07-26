package intel

import (
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
)

// ProfileEvidence records both a result and the source that produced it. A
// failed source is retained so the operator can distinguish a gap from a fact.
type ProfileEvidence struct {
	Source  string
	Status  string
	Summary string
	Facts   map[string][]string
}

// PublicProfile is an evidence-led, passive profile of a user-supplied domain,
// IP, or URL. It deliberately excludes people-search, credential searches, and
// broad web crawling.
type PublicProfile struct {
	Target   string
	Host     string
	Kind     string
	Evidence []ProfileEvidence
	Gaps     []string
}

// ProfileDependencies makes the collection policy testable and lets callers
// choose which authenticated passive sources are configured.
type ProfileDependencies struct {
	DNS    func(string) (*OSINTResult, error)
	RDAP   func(string) (*OSINTResult, error)
	Certs  func(string) (*OSINTResult, error)
	Shodan func(string, string) (*OSINTResult, error)
	VT     func(string, string) (*OSINTResult, error)
}

func DefaultProfileDependencies() ProfileDependencies {
	return ProfileDependencies{
		DNS:    DNSLookup,
		RDAP:   WHOISLookup,
		Certs:  CertTransparencyLookup,
		Shodan: ShodanLookup,
		VT:     VirusTotalLookup,
	}
}

// BuildPublicProfile aggregates independent passive sources. Search engines are
// intentionally not part of the baseline: a search or crawl should answer a
// specific operator question rather than create unauditable noise.
func BuildPublicProfile(target, shodanKey, virusTotalKey string, deps ProfileDependencies) (*PublicProfile, error) {
	host, kind, err := normalizeProfileTarget(target)
	if err != nil {
		return nil, err
	}
	if deps.DNS == nil || deps.RDAP == nil || deps.Certs == nil {
		return nil, fmt.Errorf("profile dependencies must include DNS, RDAP, and certificate transparency collectors")
	}

	profile := &PublicProfile{Target: target, Host: host, Kind: kind}
	add := func(source string, result *OSINTResult, err error) {
		if err != nil {
			profile.Evidence = append(profile.Evidence, ProfileEvidence{Source: source, Status: "unavailable", Summary: err.Error()})
			profile.Gaps = append(profile.Gaps, source)
			return
		}
		if result == nil {
			profile.Evidence = append(profile.Evidence, ProfileEvidence{Source: source, Status: "unavailable", Summary: "source returned no result"})
			profile.Gaps = append(profile.Gaps, source)
			return
		}
		profile.Evidence = append(profile.Evidence, ProfileEvidence{Source: source, Status: "observed", Facts: compactFacts(result.Data)})
	}

	if kind == "domain" {
		result, err := deps.DNS(host)
		add("DNS", result, err)
		result, err = deps.Certs(host)
		add("Certificate transparency", result, err)
	} else {
		profile.Evidence = append(profile.Evidence, ProfileEvidence{Source: "DNS", Status: "not_applicable", Summary: "IP target"})
		profile.Evidence = append(profile.Evidence, ProfileEvidence{Source: "Certificate transparency", Status: "not_applicable", Summary: "IP target"})
	}

	result, err := deps.RDAP(host)
	add("RDAP", result, err)
	if shodanKey == "" {
		profile.Evidence = append(profile.Evidence, ProfileEvidence{Source: "Shodan", Status: "not_configured", Summary: "SHODAN_API_KEY is not configured"})
		profile.Gaps = append(profile.Gaps, "Shodan")
	} else if deps.Shodan != nil {
		result, err = deps.Shodan(host, shodanKey)
		add("Shodan", result, err)
	}
	if virusTotalKey == "" {
		profile.Evidence = append(profile.Evidence, ProfileEvidence{Source: "VirusTotal", Status: "not_configured", Summary: "VIRUSTOTAL_API_KEY is not configured"})
		profile.Gaps = append(profile.Gaps, "VirusTotal")
	} else if deps.VT != nil {
		result, err = deps.VT(host, virusTotalKey)
		add("VirusTotal", result, err)
	}
	return profile, nil
}

func normalizeProfileTarget(target string) (string, string, error) {
	value := strings.TrimSpace(target)
	if value == "" {
		return "", "", fmt.Errorf("target is required")
	}
	if strings.Contains(value, "://") {
		u, err := url.Parse(value)
		if err != nil || u.Hostname() == "" {
			return "", "", fmt.Errorf("invalid target URL")
		}
		value = u.Hostname()
	}
	value = strings.TrimSuffix(strings.ToLower(value), ".")
	if ip := net.ParseIP(value); ip != nil {
		return value, "ip", nil
	}
	if strings.ContainsAny(value, "/\\@ ") || !strings.Contains(value, ".") {
		return "", "", fmt.Errorf("target must be a public domain, IP, or URL")
	}
	return value, "domain", nil
}

func compactFacts(data map[string]interface{}) map[string][]string {
	facts := map[string][]string{}
	for _, key := range []string{"A", "AAAA", "MX", "NS", "TXT", "CNAME", "subdomains", "hostnames", "domains", "status", "name", "handle", "org", "isp", "asn", "country_name", "country", "as_owner"} {
		if values := stringValues(data[key]); len(values) > 0 {
			facts[key] = values
		}
	}
	if count, ok := data["count"].(int); ok {
		facts["count"] = []string{fmt.Sprintf("%d", count)}
	}
	return facts
}

func stringValues(value interface{}) []string {
	if value == nil {
		return nil
	}
	var values []string
	switch typed := value.(type) {
	case string:
		values = []string{typed}
	case []string:
		values = append(values, typed...)
	case []interface{}:
		for _, item := range typed {
			values = append(values, fmt.Sprint(item))
		}
	}
	sort.Strings(values)
	if len(values) > 10 {
		values = append(values[:10], fmt.Sprintf("… %d more", len(values)-10))
	}
	return values
}

func FormatPublicProfile(profile *PublicProfile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[PASSIVE TARGET PROFILE]\nTarget: %s\nType: %s\n", profile.Host, strings.ToUpper(profile.Kind))
	for _, evidence := range profile.Evidence {
		fmt.Fprintf(&b, "\n[%s] %s", strings.ToUpper(evidence.Status), evidence.Source)
		if evidence.Summary != "" {
			fmt.Fprintf(&b, " — %s", evidence.Summary)
		}
		b.WriteString("\n")
		keys := make([]string, 0, len(evidence.Facts))
		for key := range evidence.Facts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&b, "  %s: %s\n", key, strings.Join(evidence.Facts[key], ", "))
		}
	}
	if len(profile.Gaps) > 0 {
		fmt.Fprintf(&b, "\nCoverage gaps: %s\n", strings.Join(profile.Gaps, ", "))
	}
	b.WriteString("\nNext step: ask a focused question before searching or fetching pages; do not run broad dorks by default.")
	return b.String()
}
