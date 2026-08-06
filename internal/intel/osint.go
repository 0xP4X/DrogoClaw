package intel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// OSINTResult holds results from any OSINT lookup.
type OSINTResult struct {
	Source  string
	Target  string
	Data    map[string]interface{}
	Summary string
}

// ── Shodan ────────────────────────────────────────────────────────────────────

// ShodanLookup queries Shodan for information about an IP or hostname.
// Parses both the top-level host fields AND the per-service banners array.
func ShodanLookup(target, apiKey string) (*OSINTResult, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("SHODAN_API_KEY not configured — run 'drogonclaw setup' to add it")
	}

	endpoint := fmt.Sprintf("https://api.shodan.io/shodan/host/%s?key=%s",
		url.QueryEscape(target), apiKey)

	resp, err := httpClient.Do(mustGET(endpoint))
	if err != nil {
		return nil, fmt.Errorf("shodan request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("shodan error %d: %s", resp.StatusCode, string(body))
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("shodan decode error: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Shodan Intelligence: %s ===\n\n", target))

	// Top-level host fields
	if org, ok := data["org"].(string); ok {
		sb.WriteString(fmt.Sprintf("  Organization : %s\n", org))
	}
	if isp, ok := data["isp"].(string); ok {
		sb.WriteString(fmt.Sprintf("  ISP          : %s\n", isp))
	}
	if asn, ok := data["asn"].(string); ok {
		sb.WriteString(fmt.Sprintf("  ASN          : %s\n", asn))
	}
	if country, ok := data["country_name"].(string); ok {
		sb.WriteString(fmt.Sprintf("  Country      : %s\n", country))
	}
	if city, ok := data["city"].(string); ok && city != "" {
		sb.WriteString(fmt.Sprintf("  City         : %s\n", city))
	}
	if os, ok := data["os"].(string); ok && os != "" {
		sb.WriteString(fmt.Sprintf("  OS           : %s\n", os))
	}

	// Hostnames
	if hostnames, ok := data["hostnames"].([]interface{}); ok && len(hostnames) > 0 {
		names := make([]string, len(hostnames))
		for i, h := range hostnames {
			names[i] = fmt.Sprintf("%v", h)
		}
		sb.WriteString(fmt.Sprintf("  Hostnames    : %s\n", strings.Join(names, ", ")))
	}

	// Domains
	if domains, ok := data["domains"].([]interface{}); ok && len(domains) > 0 {
		doms := make([]string, len(domains))
		for i, d := range domains {
			doms[i] = fmt.Sprintf("%v", d)
		}
		sb.WriteString(fmt.Sprintf("  Domains      : %s\n", strings.Join(doms, ", ")))
	}

	// Last seen
	if lastUpdate, ok := data["last_update"].(string); ok {
		sb.WriteString(fmt.Sprintf("  Last Updated : %s\n", lastUpdate))
	}

	// ── Service banners (the most valuable part) ──────────────────────────
	if services, ok := data["data"].([]interface{}); ok && len(services) > 0 {
		sb.WriteString(fmt.Sprintf("\n[+] Open Services (%d found):\n", len(services)))
		for _, svc := range services {
			svcMap, ok := svc.(map[string]interface{})
			if !ok {
				continue
			}
			port, _ := svcMap["port"].(float64)
			proto, _ := svcMap["transport"].(string)
			product, _ := svcMap["product"].(string)
			version, _ := svcMap["version"].(string)
			banner, _ := svcMap["data"].(string)

			portLine := fmt.Sprintf("  ┌─ Port %d/%s", int(port), proto)
			if product != "" {
				portLine += fmt.Sprintf(" — %s", product)
				if version != "" {
					portLine += fmt.Sprintf(" %s", version)
				}
			}
			sb.WriteString(portLine + "\n")

			// HTTP module info
			if httpInfo, ok := svcMap["http"].(map[string]interface{}); ok {
				if title, ok := httpInfo["title"].(string); ok && title != "" {
					sb.WriteString(fmt.Sprintf("  │  HTTP Title   : %s\n", strings.TrimSpace(title)))
				}
				if server, ok := httpInfo["server"].(string); ok && server != "" {
					sb.WriteString(fmt.Sprintf("  │  HTTP Server  : %s\n", server))
				}
				if waf, ok := httpInfo["waf"].(string); ok && waf != "" {
					sb.WriteString(fmt.Sprintf("  │  WAF Detected : %s\n", waf))
				}
			}

			// SSL/TLS module info
			if sslInfo, ok := svcMap["ssl"].(map[string]interface{}); ok {
				if cert, ok := sslInfo["cert"].(map[string]interface{}); ok {
					if subj, ok := cert["subject"].(map[string]interface{}); ok {
						if cn, ok := subj["CN"].(string); ok {
							sb.WriteString(fmt.Sprintf("  │  TLS CN       : %s\n", cn))
						}
					}
					if issued, ok := cert["issued"].(string); ok {
						sb.WriteString(fmt.Sprintf("  │  TLS Issued   : %s\n", issued))
					}
					if expires, ok := cert["expires"].(string); ok {
						sb.WriteString(fmt.Sprintf("  │  TLS Expires  : %s\n", expires))
					}
				}
				if cipher, ok := sslInfo["cipher"].(map[string]interface{}); ok {
					if name, ok := cipher["name"].(string); ok {
						sb.WriteString(fmt.Sprintf("  │  Cipher       : %s\n", name))
					}
				}
			}

			// Raw banner (first 3 lines only)
			if banner != "" {
				bannerLines := strings.Split(strings.TrimSpace(banner), "\n")
				preview := bannerLines
				if len(preview) > 3 {
					preview = preview[:3]
				}
				sb.WriteString(fmt.Sprintf("  │  Banner       : %s\n", strings.Join(preview, " | ")))
			}
		}
	}

	// CVEs
	if vulns, ok := data["vulns"].(map[string]interface{}); ok && len(vulns) > 0 {
		sb.WriteString(fmt.Sprintf("\n[!] CVEs Detected (%d):\n", len(vulns)))
		for cve := range vulns {
			sb.WriteString(fmt.Sprintf("  - %s\n", cve))
		}
	}

	return &OSINTResult{
		Source:  "Shodan",
		Target:  target,
		Data:    data,
		Summary: sb.String(),
	}, nil
}

// ── VirusTotal ────────────────────────────────────────────────────────────────

// VirusTotalLookup queries VirusTotal for reputation data on a URL, domain, or IP.
func VirusTotalLookup(target, apiKey string) (*OSINTResult, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("VIRUSTOTAL_API_KEY not configured — run 'drogonclaw setup' to add it")
	}

	// Determine resource type
	resourceType := "ip_addresses"
	if strings.Contains(target, "/") || strings.Contains(target, "http") {
		resourceType = "urls"
		target = url.QueryEscape(target)
	} else if strings.Count(target, ".") >= 2 && !isIP(target) {
		resourceType = "domains"
	}

	endpoint := fmt.Sprintf("https://www.virustotal.com/api/v3/%s/%s", resourceType, target)
	req := mustGET(endpoint)
	req.Header.Set("x-apikey", apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("virustotal request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("virustotal error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Attributes struct {
				LastAnalysisStats map[string]int `json:"last_analysis_stats"`
				Reputation        int            `json:"reputation"`
				Country           string         `json:"country"`
				ASOwner           string         `json:"as_owner"`
				LastHTTPSCert     struct {
					Issuer struct {
						CN string `json:"CN"`
					} `json:"issuer"`
					Subject struct {
						CN string `json:"CN"`
					} `json:"subject"`
				} `json:"last_https_certificate"`
				Categories map[string]string `json:"categories"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("virustotal decode error: %w", err)
	}

	attrs := result.Data.Attributes
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== VirusTotal Intelligence: %s ===\n\n", target))
	sb.WriteString(fmt.Sprintf("  Reputation Score : %d\n", attrs.Reputation))
	if attrs.Country != "" {
		sb.WriteString(fmt.Sprintf("  Country          : %s\n", attrs.Country))
	}
	if attrs.ASOwner != "" {
		sb.WriteString(fmt.Sprintf("  AS Owner         : %s\n", attrs.ASOwner))
	}
	if cert := attrs.LastHTTPSCert.Subject.CN; cert != "" {
		sb.WriteString(fmt.Sprintf("  TLS Cert CN      : %s\n", cert))
	}
	if len(attrs.Categories) > 0 {
		var cats []string
		for _, v := range attrs.Categories {
			cats = append(cats, v)
		}
		sb.WriteString(fmt.Sprintf("  Categories       : %s\n", strings.Join(cats, ", ")))
	}
	if stats := attrs.LastAnalysisStats; len(stats) > 0 {
		malicious := stats["malicious"]
		suspicious := stats["suspicious"]
		harmless := stats["harmless"]
		total := 0
		for _, v := range stats {
			total += v
		}
		sb.WriteString(fmt.Sprintf("\n[+] Vendor Detection (%d total engines):\n", total))
		sb.WriteString(fmt.Sprintf("  Malicious  : %d\n", malicious))
		sb.WriteString(fmt.Sprintf("  Suspicious : %d\n", suspicious))
		sb.WriteString(fmt.Sprintf("  Harmless   : %d\n", harmless))
	}

	var data map[string]interface{}
	_ = json.Unmarshal(body, &data)

	return &OSINTResult{
		Source:  "VirusTotal",
		Target:  target,
		Data:    data,
		Summary: sb.String(),
	}, nil
}

// ── WHOIS / RDAP ─────────────────────────────────────────────────────────────

// WHOISLookup performs a WHOIS lookup via the RDAP protocol (no API key required).
// Deeply parses vCard entities for registrant/abuse contact information.
func WHOISLookup(domain string) (*OSINTResult, error) {
	var endpoint string
	if isIP(domain) {
		endpoint = fmt.Sprintf("https://rdap.arin.net/registry/ip/%s", domain)
	} else {
		endpoint = fmt.Sprintf("https://rdap.org/domain/%s", domain)
	}

	resp, err := httpClient.Do(mustGET(endpoint))
	if err != nil {
		return nil, fmt.Errorf("RDAP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RDAP error %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("RDAP decode error: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== WHOIS/RDAP Intelligence: %s ===\n\n", domain))

	if name, ok := data["name"].(string); ok {
		sb.WriteString(fmt.Sprintf("  Name           : %s\n", name))
	}
	if handle, ok := data["handle"].(string); ok {
		sb.WriteString(fmt.Sprintf("  Handle         : %s\n", handle))
	}
	if status, ok := data["status"].([]interface{}); ok && len(status) > 0 {
		statuses := make([]string, len(status))
		for i, s := range status {
			statuses[i] = fmt.Sprintf("%v", s)
		}
		sb.WriteString(fmt.Sprintf("  Status         : %s\n", strings.Join(statuses, ", ")))
	}

	// Events (registration, expiration, last changed)
	if events, ok := data["events"].([]interface{}); ok {
		for _, ev := range events {
			evMap, ok := ev.(map[string]interface{})
			if !ok {
				continue
			}
			action, _ := evMap["eventAction"].(string)
			date, _ := evMap["eventDate"].(string)
			if action != "" && date != "" {
				label := strings.ToTitle(strings.ReplaceAll(action, "-", " "))
				sb.WriteString(fmt.Sprintf("  %-15s: %s\n", label, date))
			}
		}
	}

	// Nameservers
	if ns, ok := data["nameservers"].([]interface{}); ok && len(ns) > 0 {
		var nsNames []string
		for _, n := range ns {
			if nsMap, ok := n.(map[string]interface{}); ok {
				if ldhName, ok := nsMap["ldhName"].(string); ok {
					nsNames = append(nsNames, ldhName)
				}
			}
		}
		if len(nsNames) > 0 {
			sb.WriteString(fmt.Sprintf("  Nameservers    : %s\n", strings.Join(nsNames, ", ")))
		}
	}

	// Entities — deep vCard parsing for registrant, admin, abuse contacts
	if entities, ok := data["entities"].([]interface{}); ok && len(entities) > 0 {
		sb.WriteString("\n[+] Contacts:\n")
		for _, e := range entities {
			em, ok := e.(map[string]interface{})
			if !ok {
				continue
			}
			// Role
			roles := []string{}
			if roleList, ok := em["roles"].([]interface{}); ok {
				for _, r := range roleList {
					roles = append(roles, fmt.Sprintf("%v", r))
				}
			}
			roleStr := strings.Join(roles, "/")

			// Parse vCard array for name, email, org, phone, address
			contact := parseVCard(em["vcardArray"])

			if contact["fn"] != "" || contact["email"] != "" || contact["org"] != "" {
				sb.WriteString(fmt.Sprintf("  [%s]\n", strings.ToUpper(roleStr)))
				if v := contact["fn"]; v != "" {
					sb.WriteString(fmt.Sprintf("    Name    : %s\n", v))
				}
				if v := contact["org"]; v != "" {
					sb.WriteString(fmt.Sprintf("    Org     : %s\n", v))
				}
				if v := contact["email"]; v != "" {
					sb.WriteString(fmt.Sprintf("    Email   : %s\n", v))
				}
				if v := contact["tel"]; v != "" {
					sb.WriteString(fmt.Sprintf("    Phone   : %s\n", v))
				}
				if v := contact["adr"]; v != "" {
					sb.WriteString(fmt.Sprintf("    Address : %s\n", v))
				}
			}
		}
	}

	// ARIN-style network block info (for IPs)
	if startAddress, ok := data["startAddress"].(string); ok {
		sb.WriteString(fmt.Sprintf("\n  IP Range Start : %s\n", startAddress))
	}
	if endAddress, ok := data["endAddress"].(string); ok {
		sb.WriteString(fmt.Sprintf("  IP Range End   : %s\n", endAddress))
	}

	return &OSINTResult{
		Source:  "WHOIS/RDAP",
		Target:  domain,
		Data:    data,
		Summary: sb.String(),
	}, nil
}

// parseVCard extracts useful fields from an RDAP vCard array structure.
// vcardArray is: ["vcard", [ ["fn", {}, "text", "Value"], ... ]]
func parseVCard(vcardArray interface{}) map[string]string {
	result := map[string]string{}
	if vcardArray == nil {
		return result
	}
	arr, ok := vcardArray.([]interface{})
	if !ok || len(arr) < 2 {
		return result
	}
	props, ok := arr[1].([]interface{})
	if !ok {
		return result
	}
	for _, prop := range props {
		propArr, ok := prop.([]interface{})
		if !ok || len(propArr) < 4 {
			continue
		}
		key, _ := propArr[0].(string)
		val := fmt.Sprintf("%v", propArr[3])
		// For "adr" the value is a nested array — flatten it
		if key == "adr" {
			if valArr, ok := propArr[3].([]interface{}); ok {
				var parts []string
				for _, p := range valArr {
					s := strings.TrimSpace(fmt.Sprintf("%v", p))
					if s != "" && s != "<nil>" {
						parts = append(parts, s)
					}
				}
				val = strings.Join(parts, ", ")
			}
		}
		if val != "<nil>" && val != "" {
			result[key] = val
		}
	}
	return result
}

// ── Certificate Transparency ──────────────────────────────────────────────────

// CertTransparencyLookup finds subdomains via certificate transparency logs.
// It queries crt.sh (with retry, since that endpoint is frequently flaky /
// rate-limited / intermittently 404s) and falls back to Cert Spotter (SSLMate),
// a free no-key CT source, so subdomain enumeration still works when crt.sh is
// unavailable.
func CertTransparencyLookup(domain string) (*OSINTResult, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	if res, err := certTransparencyCRTSh(domain); err == nil && res != nil {
		return res, nil
	}
	if res, err := certTransparencyCertSpotter(domain); err == nil && res != nil {
		return res, nil
	}
	return nil, fmt.Errorf("certificate transparency lookup failed for %s (crt.sh and certspotter both unavailable)", domain)
}

func certTransparencyCRTSh(domain string) (*OSINTResult, error) {
	endpoint := fmt.Sprintf("https://crt.sh/?q=%%.%s&output=json", url.QueryEscape(domain))
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
		resp, err := httpClient.Do(mustGET(endpoint))
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("crt.sh error %d", resp.StatusCode)
			continue
		}
		var certs []struct {
			NameValue string `json:"name_value"`
		}
		if err := json.Unmarshal(body, &certs); err != nil {
			lastErr = err
			continue
		}
		subs := extractCTSubdomains(certs, domain)
		if len(subs) == 0 {
			lastErr = fmt.Errorf("no certificates returned")
			continue
		}
		return buildCTResult(domain, subs, "crt.sh"), nil
	}
	return nil, lastErr
}

func certTransparencyCertSpotter(domain string) (*OSINTResult, error) {
	endpoint := fmt.Sprintf("https://api.certspotter.com/v1/issuances?domain=%s&include_subdomains=true&expand=dns_names",
		url.QueryEscape(domain))

	resp, err := httpClient.Do(mustGET(endpoint))
	if err != nil {
		return nil, fmt.Errorf("certspotter request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("certspotter error %d", resp.StatusCode)
	}

	var issuances []struct {
		DNSNames []string `json:"dns_names"`
	}
	if err := json.Unmarshal(body, &issuances); err != nil {
		return nil, fmt.Errorf("certspotter decode error: %w", err)
	}

	seen := make(map[string]bool)
	var subs []string
	for _, it := range issuances {
		for _, n := range it.DNSNames {
			n = strings.ToLower(strings.TrimSpace(n))
			n = strings.TrimPrefix(n, "*.")
			if n != "" && !seen[n] && strings.HasSuffix(n, domain) {
				seen[n] = true
				subs = append(subs, n)
			}
		}
	}
	if len(subs) == 0 {
		return nil, fmt.Errorf("no certificates returned")
	}
	sort.Strings(subs)
	return buildCTResult(domain, subs, "certspotter"), nil
}

func extractCTSubdomains(certs []struct {
	NameValue string `json:"name_value"`
}, domain string) []string {
	seen := make(map[string]bool)
	var subs []string
	for _, c := range certs {
		for _, sub := range strings.Split(c.NameValue, "\n") {
			sub = strings.TrimSpace(strings.ToLower(sub))
			sub = strings.TrimPrefix(sub, "*.")
			if sub != "" && !seen[sub] && strings.HasSuffix(sub, domain) {
				seen[sub] = true
				subs = append(subs, sub)
			}
		}
	}
	sort.Strings(subs)
	return subs
}

func buildCTResult(domain string, subs []string, source string) *OSINTResult {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Certificate Transparency: %s ===\n\n", domain))
	sb.WriteString(fmt.Sprintf("[+] Subdomains discovered (%d unique):\n", len(subs)))
	for _, s := range subs {
		sb.WriteString(fmt.Sprintf("  %s\n", s))
	}
	return &OSINTResult{
		Source:  source,
		Target:  domain,
		Data:    map[string]interface{}{"subdomains": subs, "count": len(subs)},
		Summary: sb.String(),
	}
}

// ── Email Harvesting ──────────────────────────────────────────────────────────

// EmailHarvest finds published business contacts through the configured provider.
// It intentionally does not fall back to search-engine dorks: target profiling
// must remain source-accountable and avoid collecting incidental personal data.
func EmailHarvest(domain, apiKey string) (*OSINTResult, error) {
	if apiKey != "" {
		return hunterIOHarvest(domain, apiKey)
	}
	return nil, fmt.Errorf("HUNTER_IO_API_KEY not configured; email discovery is disabled rather than falling back to search-engine dorks")
}

func hunterIOHarvest(domain, apiKey string) (*OSINTResult, error) {
	endpoint := fmt.Sprintf("https://api.hunter.io/v2/domain-search?domain=%s&api_key=%s&limit=100",
		url.QueryEscape(domain), apiKey)

	resp, err := httpClient.Do(mustGET(endpoint))
	if err != nil {
		return nil, fmt.Errorf("hunter.io request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hunter.io error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Organization string `json:"organization"`
			Pattern      string `json:"pattern"`
			Emails       []struct {
				Value      string `json:"value"`
				Type       string `json:"type"`
				FirstName  string `json:"first_name"`
				LastName   string `json:"last_name"`
				Position   string `json:"position"`
				Confidence int    `json:"confidence"`
			} `json:"emails"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("hunter.io decode error: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Email Harvest (Hunter.io): %s ===\n\n", domain))
	if result.Data.Organization != "" {
		sb.WriteString(fmt.Sprintf("  Organization   : %s\n", result.Data.Organization))
	}
	if result.Data.Pattern != "" {
		sb.WriteString(fmt.Sprintf("  Email Pattern  : %s\n", result.Data.Pattern))
	}
	sb.WriteString(fmt.Sprintf("  Emails found   : %d\n\n", len(result.Data.Emails)))

	for _, e := range result.Data.Emails {
		line := fmt.Sprintf("  [%3d%%] %s", e.Confidence, e.Value)
		if e.FirstName != "" || e.LastName != "" {
			line += fmt.Sprintf(" (%s %s", e.FirstName, e.LastName)
			if e.Position != "" {
				line += fmt.Sprintf(", %s", e.Position)
			}
			line += ")"
		}
		sb.WriteString(line + "\n")
	}

	var data map[string]interface{}
	_ = json.Unmarshal(body, &data)

	return &OSINTResult{
		Source:  "Hunter.io",
		Target:  domain,
		Data:    data,
		Summary: sb.String(),
	}, nil
}

// dorkEmailHarvest is a free fallback that uses Google/DuckDuckGo dorking to find emails.
func dorkEmailHarvest(domain string) (*OSINTResult, error) {
	queries := []string{
		fmt.Sprintf("site:%s email OR contact OR \"@%s\"", domain, domain),
		fmt.Sprintf("\"@%s\" filetype:pdf OR filetype:docx OR filetype:csv", domain),
		fmt.Sprintf("intext:\"@%s\"", domain),
	}

	seen := make(map[string]bool)
	var emails []string
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Email Harvest (Google Dork fallback — no Hunter.io key): %s ===\n\n", domain))
	sb.WriteString("[!] HUNTER_IO_API_KEY not set. Using passive dork-based harvesting.\n\n")

	for _, q := range queries {
		results, err := Search(q, "", 5)
		if err != nil {
			continue
		}
		for _, r := range results {
			// Extract emails from snippet
			extracted := extractEmailsFromText(r.Snippet, domain)
			for _, e := range extracted {
				if !seen[e] {
					seen[e] = true
					emails = append(emails, e)
				}
			}
			sb.WriteString(fmt.Sprintf("  Dork hit: %s\n  URL: %s\n", r.Title, r.URL))
		}
	}

	if len(emails) > 0 {
		sb.WriteString(fmt.Sprintf("\n[+] Emails extracted (%d):\n", len(emails)))
		for _, e := range emails {
			sb.WriteString(fmt.Sprintf("  %s\n", e))
		}
	} else {
		sb.WriteString("\n[~] No emails extracted from passive dork. Run theHarvester in the sandbox:\n")
		sb.WriteString(fmt.Sprintf("    theHarvester -d %s -b google,bing,duckduckgo,anubis -l 200\n", domain))
	}

	data := map[string]interface{}{"emails": emails, "domain": domain, "source": "dork"}
	return &OSINTResult{
		Source:  "DorkHarvest",
		Target:  domain,
		Data:    data,
		Summary: sb.String(),
	}, nil
}

// extractEmailsFromText extracts email addresses from free text.
func extractEmailsFromText(text, domain string) []string {
	var emails []string
	words := strings.Fields(text)
	for _, w := range words {
		w = strings.Trim(w, ".,;:\"'()[]{}")
		if strings.Contains(w, "@") && strings.Contains(w, domain) {
			emails = append(emails, strings.ToLower(w))
		}
	}
	return emails
}

// ── GitHub Dorking ────────────────────────────────────────────────────────────

// githubDorkPassive searches GitHub for exposed secrets, credentials, and code
// leaks related to a target via passive search-engine dorking. It is the
// no-token fallback used by GitHubDork when no GITHUB_TOKEN is configured.
func githubDorkPassive(target string) (*OSINTResult, error) {
	dorks := []struct {
		label string
		query string
	}{
		{"API Keys / Secrets", fmt.Sprintf("%s password OR secret OR api_key OR token", target)},
		{"Config Files", fmt.Sprintf("%s filename:.env OR filename:config.yml OR filename:secrets.json", target)},
		{"Connection Strings", fmt.Sprintf("%s connectionstring OR database_url OR DB_PASSWORD", target)},
		{"AWS Credentials", fmt.Sprintf("%s AKIA OR aws_access_key_id", target)},
		{"Private Keys", fmt.Sprintf("%s BEGIN RSA PRIVATE KEY OR BEGIN OPENSSH PRIVATE KEY", target)},
	}

	r := NewReport(fmt.Sprintf("GitHub Credential Dork (passive): %s", target))
	r.Note("Passive web search - for authenticated GitHub search, set GITHUB_TOKEN.")

	var allHits []map[string]interface{}

	for _, dork := range dorks {
		githubQuery := fmt.Sprintf("site:github.com %s", dork.query)
		results, err := Search(githubQuery, "", 5)
		if err != nil {
			continue
		}
		if len(results) > 0 {
			r.Section(fmt.Sprintf("%s - %d hits", dork.label, len(results)))
			for _, rr := range results {
				r.Bullet(fmt.Sprintf("%s - %s", rr.Title, rr.URL))
				allHits = append(allHits, map[string]interface{}{
					"label": dork.label,
					"title": rr.Title,
					"url":   rr.URL,
				})
			}
		}
	}

	if len(allHits) == 0 {
		r.Note("No public GitHub leaks found. Try authenticated search via the GitHub API (set GITHUB_TOKEN).")
	}

	data := map[string]interface{}{"hits": allHits, "target": target}
	return &OSINTResult{
		Source:  "GitHubDorkPassive",
		Target:  target,
		Data:    data,
		Summary: r.String(),
	}, nil
}

// ── DNS Intelligence ──────────────────────────────────────────────────────────

// DNSLookup performs comprehensive DNS enumeration using the Google DNS-over-HTTPS API (no key needed).
func DNSLookup(domain string) (*OSINTResult, error) {
	recordTypes := []string{"A", "AAAA", "MX", "NS", "TXT", "SOA", "CNAME"}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== DNS Intelligence: %s ===\n\n", domain))

	allData := map[string]interface{}{}

	for _, rtype := range recordTypes {
		endpoint := fmt.Sprintf("https://dns.google/resolve?name=%s&type=%s",
			url.QueryEscape(domain), rtype)

		resp, err := httpClient.Do(mustGET(endpoint))
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var dnsResp struct {
			Status int `json:"Status"`
			Answer []struct {
				Name string `json:"name"`
				Type int    `json:"type"`
				TTL  int    `json:"TTL"`
				Data string `json:"data"`
			} `json:"Answer"`
		}
		if err := json.Unmarshal(body, &dnsResp); err != nil {
			continue
		}
		if len(dnsResp.Answer) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("[+] %s records:\n", rtype))
		var records []string
		for _, ans := range dnsResp.Answer {
			sb.WriteString(fmt.Sprintf("  %-6s  TTL:%-6d  %s\n", rtype, ans.TTL, ans.Data))
			records = append(records, ans.Data)
		}
		allData[rtype] = records
	}

	return &OSINTResult{
		Source:  "DNS/Google",
		Target:  domain,
		Data:    allData,
		Summary: sb.String(),
	}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func mustGET(endpoint string) *http.Request {
	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("User-Agent", "DrogonClaw/2.0 (security-assessment-tool)")
	return req
}

func isIP(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
