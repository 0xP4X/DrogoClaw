package intel

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// TargetClass classifies what kind of target is being engaged.
type TargetClass int

const (
	ClassUnknown      TargetClass = iota
	ClassWeb                      // http/https URL or domain
	ClassAPI                      // REST/GraphQL API endpoint
	ClassNetwork                  // IP address or CIDR range
	ClassBinary                   // ELF/PE/Mach-O binary file
	ClassDomain                   // Bare domain for OSINT
	ClassCloud                    // AWS/GCP/Azure target
	ClassCTFPwn                   // Binary exploitation CTF
	ClassCTFWeb                   // Web challenge CTF
	ClassCTFForensics             // Forensics/stego CTF
	ClassCTFCrypto                // Cryptography CTF
)

func (c TargetClass) String() string {
	switch c {
	case ClassWeb:
		return "WEB"
	case ClassAPI:
		return "API"
	case ClassNetwork:
		return "NETWORK"
	case ClassBinary:
		return "BINARY"
	case ClassDomain:
		return "DOMAIN/OSINT"
	case ClassCloud:
		return "CLOUD"
	case ClassCTFPwn:
		return "CTF/PWN"
	case ClassCTFWeb:
		return "CTF/WEB"
	case ClassCTFForensics:
		return "CTF/FORENSICS"
	case ClassCTFCrypto:
		return "CTF/CRYPTO"
	default:
		return "UNKNOWN"
	}
}

// ToolStep is a single tool invocation in an attack chain.
type ToolStep struct {
	Priority    int
	Tool        string
	Description string
	Flags       string // Recommended flags/parameters
}

// AttackChain is an ordered list of tools to run for a target class.
type AttackChain struct {
	Class       TargetClass
	Name        string
	Description string
	Steps       []ToolStep
}

// TargetProfile is the result of analyzing an input target.
type TargetProfile struct {
	Raw        string
	Class      TargetClass
	Resolved   string // IP if resolved
	Chain      *AttackChain
	Confidence float64
}

// Summarize returns a formatted target profile summary.
func (p *TargetProfile) Summarize() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("┌─ TARGET PROFILE ─────────────────────────────────────┐\n"))
	sb.WriteString(fmt.Sprintf("│  Target    : %-40s│\n", truncate(p.Raw, 40)))
	sb.WriteString(fmt.Sprintf("│  Class     : %-40s│\n", p.Class.String()))
	if p.Resolved != "" {
		sb.WriteString(fmt.Sprintf("│  Resolved  : %-40s│\n", p.Resolved))
	}
	sb.WriteString(fmt.Sprintf("│  Confidence: %-40s│\n", fmt.Sprintf("%.0f%%", p.Confidence*100)))
	if p.Chain != nil {
		sb.WriteString(fmt.Sprintf("│  Chain     : %-40s│\n", p.Chain.Name))
		sb.WriteString("├─ ATTACK CHAIN ───────────────────────────────────────┤\n")
		for _, s := range p.Chain.Steps {
			line := fmt.Sprintf("│  [%02d] %-17s %s", s.Priority, s.Tool, s.Description)
			// Pad to width
			for len(line) < 55 {
				line += " "
			}
			sb.WriteString(line[:55] + "│\n")
		}
	}
	sb.WriteString("└──────────────────────────────────────────────────────┘")
	return sb.String()
}

// ModePromptInjection returns a system prompt injection for this chain.
func (p *TargetProfile) ModePromptInjection() string {
	if p.Chain == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n\n--- ACTIVE OPERATIONAL MODE: %s ---\n", p.Chain.Name))
	sb.WriteString(fmt.Sprintf("Target classified as: %s\n", p.Class.String()))
	sb.WriteString("You MUST execute the following attack chain in order. Do not skip steps.\n\n")
	for _, s := range p.Chain.Steps {
		sb.WriteString(fmt.Sprintf("  STEP %02d: Run %-18s → %s\n", s.Priority, s.Tool, s.Description))
		if s.Flags != "" {
			sb.WriteString(fmt.Sprintf("           Recommended flags: %s\n", s.Flags))
		}
	}
	sb.WriteString("\nAfter completing each step, synthesize findings before proceeding to the next.\n")
	sb.WriteString("Do NOT proceed to exploitation before completing all recon steps.\n")
	return sb.String()
}

// ─────────────────────────────────────────────────────────
// Pre-built Attack Chains
// ─────────────────────────────────────────────────────────

var chainWebApp = &AttackChain{
	Class:       ClassWeb,
	Name:        "WEB-PENTEST",
	Description: "Full web application penetration test",
	Steps: []ToolStep{
		{1, "nmap", "Port scan & service fingerprint", "-sV -sC -p 80,443,8080,8443"},
		{2, "whatweb", "Technology stack detection", "--color=never -a 3"},
		{3, "gobuster", "Directory & file bruteforce", "dir -x php,html,js,txt -t 50"},
		{4, "ffuf", "Parameter & endpoint fuzzing", "-mc 200,301,302,401,403"},
		{5, "nuclei", "Automated vulnerability scanning", "-severity critical,high -tags tech"},
		{6, "nikto", "Web server misconfiguration scan", "-C all"},
		{7, "sqlmap", "SQL injection detection", "--batch --level 2 --risk 2"},
		{8, "dalfox", "XSS vulnerability scanning", "--mining-dom --mining-dict"},
	},
}

var chainAPI = &AttackChain{
	Class:       ClassAPI,
	Name:        "API-PENTEST",
	Description: "REST/GraphQL API security assessment",
	Steps: []ToolStep{
		{1, "httpx", "API endpoint probing & headers", "-td -sc -title -server"},
		{2, "arjun", "Hidden parameter discovery", "-m GET,POST --stable"},
		{3, "ffuf", "Endpoint enumeration & fuzzing", "-mc 200,201,204,301,302,401,403"},
		{4, "nuclei", "API-specific vulnerability templates", "-tags api,graphql,jwt -severity high,critical"},
		{5, "sqlmap", "SQL injection in API params", "--batch --level 3 --risk 2"},
		{6, "dalfox", "XSS in API response context", "--mining-dom"},
	},
}

var chainNetwork = &AttackChain{
	Class:       ClassNetwork,
	Name:        "NETWORK-PENTEST",
	Description: "Internal/external network penetration test",
	Steps: []ToolStep{
		{1, "rustscan", "Ultra-fast full port discovery", "--ulimit 5000 -b 1500"},
		{2, "nmap", "Deep service & OS fingerprinting", "-sV -sC -O -A -p-"},
		{3, "enum4linux-ng", "SMB/NetBIOS/RPC enumeration", "-A -C"},
		{4, "smbmap", "SMB share enumeration & access", "-H <target> --depth 5"},
		{5, "netexec", "Credential validation & spraying", "smb <target> -u users.txt -p pass.txt"},
		{6, "nuclei", "Network service CVE scanning", "-tags network,ssh,ftp -severity critical,high"},
		{7, "hydra", "Credential brute-force on open services", "-L users.txt -P rockyou.txt"},
	},
}

var chainOSINT = &AttackChain{
	Class:       ClassDomain,
	Name:        "OSINT-PROFILE",
	Description: "Evidence-led passive target profiling",
	Steps: []ToolStep{
		{1, "profile_target", "Collect DNS, RDAP, certificate, and configured passive-source evidence", ""},
		{2, "web_search", "Answer a specific operator question only after identifying a coverage gap", "focused query, max 5 results"},
	},
}

var chainCTFPwn = &AttackChain{
	Class:       ClassCTFPwn,
	Name:        "CTF-PWN",
	Description: "Binary exploitation CTF challenge",
	Steps: []ToolStep{
		{1, "checksec", "Binary protection analysis (NX, PIE, RELRO, canary)", "--file=<binary>"},
		{2, "strings", "Extract readable strings & flag hints", "-n 8 <binary>"},
		{3, "file", "Architecture & binary format identification", "<binary>"},
		{4, "shell_execute", "Disassemble with objdump or radare2", "r2 -A <binary> || objdump -d <binary>"},
		{5, "shell_execute", "Decompile with ghidra headless or Cutter", "ghidra_headless or cutter"},
		{6, "shell_execute", "Find ROP gadgets with ropper", "ropper -f <binary> --type rop"},
		{7, "shell_execute", "Symbolic execution with angr", "python3 angr_solve.py"},
		{8, "shell_execute", "Find one-gadget for libc exploits", "one_gadget libc.so.6"},
		{9, "shell_execute", "Write & run pwntools exploit script", "python3 exploit.py"},
	},
}

var chainCTFWeb = &AttackChain{
	Class:       ClassCTFWeb,
	Name:        "CTF-WEB",
	Description: "Web-based CTF challenge",
	Steps: []ToolStep{
		{1, "shell_execute", "Technology fingerprinting", "curl -I <target> && whatweb <target>"},
		{2, "gobuster", "Directory & file discovery", "dir -w /usr/share/wordlists/dirbuster/directory-list-2.3-medium.txt"},
		{3, "shell_execute", "Source code inspection & JS analysis", "curl <target> | grep -E '(flag|secret|key|token|admin)'"},
		{4, "ffuf", "Parameter fuzzing (GET/POST)", "-w params.txt -u <target>?FUZZ=test"},
		{5, "sqlmap", "SQL injection with tamper scripts", "--batch --level 3 --risk 3 --tamper=space2comment"},
		{6, "shell_execute", "SSTI template injection testing", "manually test {{7*7}} Jinja2/Twig patterns"},
		{7, "shell_execute", "JWT decode & manipulation", "python3 -c 'import jwt; ...'"},
		{8, "shell_execute", "SSRF, LFI, path traversal testing", "curl <target>?file=../../../../etc/passwd"},
	},
}

var chainCTFForensics = &AttackChain{
	Class:       ClassCTFForensics,
	Name:        "CTF-FORENSICS",
	Description: "Digital forensics & steganography CTF",
	Steps: []ToolStep{
		{1, "file", "File type & magic byte identification", "<file>"},
		{2, "exiftool", "Metadata extraction", "<file>"},
		{3, "strings", "Readable string extraction", "-n 4 <file>"},
		{4, "binwalk", "Embedded file & firmware extraction", "-e <file>"},
		{5, "shell_execute", "Steghide extraction attempt", "steghide extract -sf <file>"},
		{6, "shell_execute", "Volatility memory analysis (if .mem/.vmem)", "volatility3 -f <file> windows.pslist"},
		{7, "shell_execute", "Foremost file carving", "foremost -i <file> -o ./output"},
		{8, "shell_execute", "Hex analysis", "xxd <file> | head -100"},
	},
}

var chainCTFCrypto = &AttackChain{
	Class:       ClassCTFCrypto,
	Name:        "CTF-CRYPTO",
	Description: "Cryptography CTF challenge",
	Steps: []ToolStep{
		{1, "shell_execute", "Identify cipher/encoding type", "python3 -c 'import base64,binascii; ...'"},
		{2, "shell_execute", "Classical cipher analysis (Caesar, Vigenere)", "use CyberChef or python cryptanalysis"},
		{3, "shell_execute", "RSA parameter analysis (n, e, c)", "python3 factordb lookup or RsaCtfTool"},
		{4, "shell_execute", "Hash identification & cracking", "hashid <hash> && john --wordlist=rockyou.txt"},
		{5, "shell_execute", "XOR analysis & frequency analysis", "python3 xor_crack.py"},
		{6, "shell_execute", "Padding oracle / CBC bit-flip attack", "padbuster or custom python"},
	},
}

var chainCloud = &AttackChain{
	Class:       ClassCloud,
	Name:        "CLOUD-SECURITY",
	Description: "Cloud infrastructure security assessment",
	Steps: []ToolStep{
		{1, "shell_execute", "AWS credential enumeration", "aws sts get-caller-identity && aws iam list-users"},
		{2, "shell_execute", "S3 bucket enumeration & misconfig", "aws s3 ls && s3scanner scan --buckets-file list.txt"},
		{3, "nuclei", "Cloud-specific vulnerability templates", "-tags aws,azure,gcp -severity critical,high"},
		{4, "shell_execute", "IAM privilege escalation check", "python3 enumerate_iam.py"},
		{5, "shell_execute", "Lambda & EC2 metadata SSRF check", "curl http://169.254.169.254/latest/meta-data/"},
	},
}

// chainMap maps TargetClass → AttackChain
var chainMap = map[TargetClass]*AttackChain{
	ClassWeb:          chainWebApp,
	ClassAPI:          chainAPI,
	ClassNetwork:      chainNetwork,
	ClassDomain:       chainOSINT,
	ClassCTFPwn:       chainCTFPwn,
	ClassCTFWeb:       chainCTFWeb,
	ClassCTFForensics: chainCTFForensics,
	ClassCTFCrypto:    chainCTFCrypto,
	ClassCloud:        chainCloud,
}

// ─────────────────────────────────────────────────────────
// TargetAnalyzer
// ─────────────────────────────────────────────────────────

// TargetAnalyzer classifies any target string and returns the
// optimal attack chain + a full profile.
type TargetAnalyzer struct{}

var (
	reIPv4   = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}(/\d{1,2})?$`)
	reDomain = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z]{2,})+$`)
)

// Analyze classifies the target and builds a profile.
func (a *TargetAnalyzer) Analyze(target string) *TargetProfile {
	p := &TargetProfile{Raw: target}
	target = strings.TrimSpace(target)

	class, confidence := a.classify(target)
	p.Class = class
	p.Confidence = confidence
	p.Chain = chainMap[class]

	// Try to resolve domain to IP
	if class == ClassDomain || class == ClassWeb || class == ClassAPI {
		host := target
		if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
			// strip scheme & path
			host = strings.TrimPrefix(host, "https://")
			host = strings.TrimPrefix(host, "http://")
			if idx := strings.Index(host, "/"); idx != -1 {
				host = host[:idx]
			}
		}
		if addrs, err := net.LookupHost(host); err == nil && len(addrs) > 0 {
			p.Resolved = addrs[0]
		}
	}

	return p
}

// classify determines the best TargetClass for a raw input string.
func (a *TargetAnalyzer) classify(t string) (TargetClass, float64) {
	lower := strings.ToLower(t)

	// ── Binary file extensions
	binaryExts := []string{".elf", ".exe", ".bin", ".so", ".dll", ".o", ".out", ".ko"}
	for _, ext := range binaryExts {
		if strings.HasSuffix(lower, ext) {
			return ClassCTFPwn, 0.95
		}
	}

	// ── Forensics file extensions
	forensicsExts := []string{".pcap", ".pcapng", ".mem", ".vmem", ".img", ".iso", ".jpg", ".png", ".wav", ".mp3", ".zip", ".tar"}
	for _, ext := range forensicsExts {
		if strings.HasSuffix(lower, ext) {
			return ClassCTFForensics, 0.90
		}
	}

	// ── IP address or CIDR
	if reIPv4.MatchString(t) {
		return ClassNetwork, 0.98
	}

	// ── Cloud services
	cloudKeywords := []string{"amazonaws.com", "azure.com", "googleapis.com", "s3.", "ec2.", "lambda."}
	for _, kw := range cloudKeywords {
		if strings.Contains(lower, kw) {
			return ClassCloud, 0.95
		}
	}

	// ── API endpoint indicators
	apiKeywords := []string{"/api/", "/v1/", "/v2/", "/graphql", "/rest/", "/swagger", "/openapi"}
	for _, kw := range apiKeywords {
		if strings.Contains(lower, kw) {
			return ClassAPI, 0.92
		}
	}

	// ── Web URL (has http scheme)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return ClassWeb, 0.96
	}

	// ── Bare domain (OSINT/recon)
	if reDomain.MatchString(t) {
		return ClassDomain, 0.88
	}

	return ClassUnknown, 0.0
}

// GetChainByName returns a chain by mode name (for /mode command).
func GetChainByName(name string) (*AttackChain, bool) {
	lookup := map[string]*AttackChain{
		"web":           chainWebApp,
		"web-pentest":   chainWebApp,
		"api":           chainAPI,
		"network":       chainNetwork,
		"net":           chainNetwork,
		"osint":         chainOSINT,
		"domain":        chainOSINT,
		"recon":         chainOSINT,
		"ctf-pwn":       chainCTFPwn,
		"pwn":           chainCTFPwn,
		"binary":        chainCTFPwn,
		"ctf-web":       chainCTFWeb,
		"ctf-forensics": chainCTFForensics,
		"forensics":     chainCTFForensics,
		"stego":         chainCTFForensics,
		"ctf-crypto":    chainCTFCrypto,
		"crypto":        chainCTFCrypto,
		"cloud":         chainCloud,
		"aws":           chainCloud,
	}
	chain, ok := lookup[strings.ToLower(name)]
	return chain, ok
}

// ListModes returns all available mode names.
func ListModes() []string {
	return []string{
		"web", "api", "network", "osint", "recon",
		"ctf-pwn", "ctf-web", "ctf-forensics", "ctf-crypto",
		"cloud",
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
