package agent

// ─────────────────────────────────────────────────────────────────────────────
// Phase 2: Dedicated Tool Wrappers
//
// These replace raw shell_execute calls for the top 10 most-used pentest tools.
// Each wrapper hard-codes best-practice flags so the LLM only needs to specify
// the target and mode — NOT remember dozens of flags.
// ─────────────────────────────────────────────────────────────────────────────

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
)

// registerToolWrappers adds all structured tool wrappers to the registry.
// Called from registerBuiltins().
func (r *ToolRegistry) registerToolWrappers() {

	// ── 1. NMAP ──────────────────────────────────────────────────────────────
	r.builtins["run_nmap"] = func(ctx context.Context, args map[string]any) string {
		target, _ := args["target"].(string)
		mode, _ := args["mode"].(string)
		ports, _ := args["ports"].(string)
		if target == "" {
			return "[Error] target is required"
		}
		if ports == "" {
			ports = "-"
		}
		var flags string
		switch strings.ToLower(mode) {
		case "quick":
			flags = "-sV -sC --open -T4 -p 80,443,22,21,25,8080,8443,3306,5432"
		case "udp":
			flags = "-sU --top-ports 200 -T4"
		case "vuln":
			flags = "-sV -sC --script vuln -T4"
		case "stealth":
			flags = "-sS -sV -T2 --randomize-hosts"
		case "full":
			flags = "-sV -sC -O -A -T4 -p-"
		default: // "default" or empty
			flags = "-sV -sC --open -T4"
		}
		if ports != "-" {
			flags = fmt.Sprintf("%s -p %s", flags, ports)
		}
		cmd := fmt.Sprintf("nmap %s %s 2>&1", flags, target)
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[nmap Error] %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[NMAP — %s — %s]\n%s", strings.ToUpper(mode), target, out)
	}

	// ── 2. NUCLEI ─────────────────────────────────────────────────────────────
	r.builtins["run_nuclei"] = func(ctx context.Context, args map[string]any) string {
		target, _ := args["target"].(string)
		severity, _ := args["severity"].(string)
		tags, _ := args["tags"].(string)
		if target == "" {
			return "[Error] target is required"
		}
		if severity == "" {
			severity = "critical,high,medium"
		}
		cmd := fmt.Sprintf("nuclei -u %s -severity %s -silent -timeout 10 -retries 2", target, severity)
		if tags != "" {
			cmd += fmt.Sprintf(" -tags %s", tags)
		}
		cmd += " 2>&1"
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[nuclei Error] %v\nOutput: %s", err, out)
		}
		if strings.TrimSpace(out) == "" {
			return fmt.Sprintf("[nuclei] Scan complete on %s — No vulnerabilities found matching severity: %s", target, severity)
		}
		return fmt.Sprintf("[NUCLEI — %s]\n%s", target, out)
	}

	// ── 3. GOBUSTER ───────────────────────────────────────────────────────────
	r.builtins["run_gobuster"] = func(ctx context.Context, args map[string]any) string {
		target, _ := args["target"].(string)
		mode, _ := args["mode"].(string)
		wordlist, _ := args["wordlist"].(string)
		extensions, _ := args["extensions"].(string)
		if target == "" {
			return "[Error] target is required"
		}
		if mode == "" {
			mode = "dir"
		}
		if wordlist == "" {
			wordlist = "/usr/share/wordlists/dirbuster/directory-list-2.3-medium.txt"
		}
		cmd := fmt.Sprintf("gobuster %s -u %s -w %s -t 40 -q --no-error", mode, target, wordlist)
		if extensions != "" && mode == "dir" {
			cmd += fmt.Sprintf(" -x %s", extensions)
		}
		cmd += " 2>&1"
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[gobuster Error] %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[GOBUSTER — %s — %s]\n%s", strings.ToUpper(mode), target, out)
	}

	// ── 4. FFUF ───────────────────────────────────────────────────────────────
	r.builtins["run_ffuf"] = func(ctx context.Context, args map[string]any) string {
		target, _ := args["target"].(string)
		wordlist, _ := args["wordlist"].(string)
		mode, _ := args["mode"].(string)
		method, _ := args["method"].(string)
		if target == "" {
			return "[Error] target is required (use FUZZ as placeholder in URL)"
		}
		if wordlist == "" {
			wordlist = "/usr/share/wordlists/dirbuster/directory-list-2.3-medium.txt"
		}
		if method == "" {
			method = "GET"
		}
		// Ensure FUZZ is in target
		if !strings.Contains(target, "FUZZ") {
			target += "/FUZZ"
		}
		cmd := fmt.Sprintf("ffuf -u %s -w %s -X %s -mc 200,204,301,302,307,401,403 -t 40 -ac -s 2>&1",
			target, wordlist, method)
		if mode == "parameter" {
			cmd = fmt.Sprintf("ffuf -u %s -w %s -X %s -mc 200,204,301,302 -t 20 -ac -s 2>&1",
				target, wordlist, method)
		}
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[ffuf Error] %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[FFUF — %s]\n%s", target, out)
	}

	// ── 5. SQLMAP ─────────────────────────────────────────────────────────────
	r.builtins["run_sqlmap"] = func(ctx context.Context, args map[string]any) string {
		target, _ := args["target"].(string)
		levelRaw, _ := args["level"].(float64)
		riskRaw, _ := args["risk"].(float64)
		data, _ := args["data"].(string)
		tamper, _ := args["tamper"].(string)
		if target == "" {
			return "[Error] target (URL) is required"
		}
		level := int(levelRaw)
		risk := int(riskRaw)
		if level == 0 {
			level = 2
		}
		if risk == 0 {
			risk = 2
		}
		cmd := fmt.Sprintf("sqlmap -u %s --batch --level %d --risk %d --threads 4 --timeout 30 --output-dir=/workspace/sqlmap_output",
			target, level, risk)
		if data != "" {
			cmd += fmt.Sprintf(" --data=%q", data)
		}
		if tamper != "" {
			cmd += fmt.Sprintf(" --tamper=%s", tamper)
		}
		cmd += " 2>&1"
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[sqlmap Error] %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[SQLMAP — %s]\n%s", target, out)
	}

	// ── 6. SUBFINDER ──────────────────────────────────────────────────────────
	r.builtins["run_subfinder"] = func(ctx context.Context, args map[string]any) string {
		domain, _ := args["domain"].(string)
		allSources, _ := args["all_sources"].(bool)
		if domain == "" {
			return "[Error] domain is required"
		}
		cmd := fmt.Sprintf("subfinder -d %s -silent -timeout 30", domain)
		if allSources {
			cmd += " -all"
		}
		cmd += " 2>&1"
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[subfinder Error] %v\nOutput: %s", err, out)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		return fmt.Sprintf("[SUBFINDER — %s] Found %d subdomains:\n%s", domain, len(lines), out)
	}

	// ── 7. HTTPX ──────────────────────────────────────────────────────────────
	r.builtins["run_httpx"] = func(ctx context.Context, args map[string]any) string {
		target, _ := args["target"].(string)
		if target == "" {
			return "[Error] target is required"
		}
		cmd := fmt.Sprintf("echo %s | httpx -silent -tech-detect -status-code -title -server -content-length -follow-redirects -timeout 10 2>&1",
			target)
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[httpx Error] %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[HTTPX — %s]\n%s", target, out)
	}

	// ── 8. CHECKSEC ───────────────────────────────────────────────────────────
	r.builtins["run_checksec"] = func(ctx context.Context, args map[string]any) string {
		binaryPath, _ := args["binary_path"].(string)
		if binaryPath == "" {
			return "[Error] binary_path is required"
		}
		cmd := fmt.Sprintf(`
file %s
echo "---"
checksec --file=%s 2>/dev/null || checksec --binary=%s 2>/dev/null || echo "[!] checksec not found — installing..." && apt-get install -y checksec -qq && checksec --file=%s
echo "---"
strings -n 8 %s | head -60
echo "---"
ldd %s 2>/dev/null | head -20
`, binaryPath, binaryPath, binaryPath, binaryPath, binaryPath, binaryPath)
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[checksec Error] %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[CHECKSEC — %s]\n%s", binaryPath, out)
	}

	// ── 9. HYDRA ──────────────────────────────────────────────────────────────
	r.builtins["run_hydra"] = func(ctx context.Context, args map[string]any) string {
		target, _ := args["target"].(string)
		service, _ := args["service"].(string)
		userlist, _ := args["userlist"].(string)
		passlist, _ := args["passlist"].(string)
		singleUser, _ := args["username"].(string)
		singlePass, _ := args["password"].(string)
		if target == "" || service == "" {
			return "[Error] target and service are required (services: ssh, ftp, http-post-form, smb, rdp, mysql, postgres)"
		}
		if userlist == "" && singleUser == "" {
			userlist = "/usr/share/wordlists/metasploit/unix_users.txt"
		}
		if passlist == "" && singlePass == "" {
			passlist = "/usr/share/wordlists/rockyou.txt"
		}

		uFlag := fmt.Sprintf("-L %s", userlist)
		if singleUser != "" {
			uFlag = fmt.Sprintf("-l %s", singleUser)
		}
		pFlag := fmt.Sprintf("-P %s", passlist)
		if singlePass != "" {
			pFlag = fmt.Sprintf("-p %s", singlePass)
		}

		cmd := fmt.Sprintf("hydra -t 4 -f -q %s %s %s %s 2>&1", uFlag, pFlag, target, service)
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[hydra Error] %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[HYDRA — %s — %s]\n%s", service, target, out)
	}

	// ── 10. BINWALK / FORENSICS TRIAGE ───────────────────────────────────────
	r.builtins["run_forensics_triage"] = func(ctx context.Context, args map[string]any) string {
		filePath, _ := args["file_path"].(string)
		if filePath == "" {
			return "[Error] file_path is required"
		}
		cmd := fmt.Sprintf(`
echo "=== FILE TYPE ==="
file %s
echo ""
echo "=== EXIFTOOL ==="
exiftool %s 2>/dev/null || echo "[!] exiftool not installed"
echo ""
echo "=== STRINGS (top 40) ==="
strings -n 6 %s | head -40
echo ""
echo "=== BINWALK ==="
binwalk %s 2>/dev/null || echo "[!] binwalk not installed"
echo ""
echo "=== HEX PREVIEW (first 64 bytes) ==="
xxd %s | head -4
`, filePath, filePath, filePath, filePath, filePath)
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[forensics_triage Error] %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[FORENSICS TRIAGE — %s]\n%s", filePath, out)
	}

	// ── 11. ANGR (Symbolic Execution) ─────────────────────────────────────────
	r.builtins["run_angr"] = func(ctx context.Context, args map[string]any) string {
		binaryPath, _ := args["binary_path"].(string)
		findAddr, _ := args["find_addr"].(string)
		avoidAddr, _ := args["avoid_addr"].(string)
		if binaryPath == "" || findAddr == "" {
			return "[Error] binary_path and find_addr are required"
		}
		
		script := fmt.Sprintf(`import angr
import sys

print("[*] Loading project...")
p = angr.Project("%s", auto_load_libs=False)
state = p.factory.entry_state()
simgr = p.factory.simgr(state)

find_addr = int("%s", 16) if "%s".startswith("0x") else "%s"
avoid_addr = int("%s", 16) if "%s".startswith("0x") and "%s" != "" else None

print(f"[*] Exploring... Find: {hex(find_addr) if isinstance(find_addr, int) else find_addr}")
if avoid_addr:
    print(f"[*] Avoiding: {hex(avoid_addr)}")
    simgr.explore(find=find_addr, avoid=avoid_addr)
else:
    simgr.explore(find=find_addr)

if simgr.found:
    print("[+] Found a path!")
    found_state = simgr.found[0]
    print("[+] stdin payload:")
    print(found_state.posix.dumps(0))
else:
    print("[-] No path found.")
`, binaryPath, findAddr, findAddr, findAddr, avoidAddr, avoidAddr, avoidAddr)

		// Write script to sandbox and run
		scriptPath := "/workspace/angr_solve.py"
		err := r.sandbox.WriteFile(ctx, scriptPath, script)
		if err != nil {
			return fmt.Sprintf("[angr Error] Failed to write script: %v", err)
		}
		out, err := r.sandbox.Execute(ctx, "python3 "+scriptPath)
		if err != nil {
			return fmt.Sprintf("[angr Error] %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[ANGR — %s]\n%s", binaryPath, out)
	}

	// ── 12. ROPPER (ROP Gadget Search) ────────────────────────────────────────
	r.builtins["run_ropper"] = func(ctx context.Context, args map[string]any) string {
		binaryPath, _ := args["binary_path"].(string)
		search, _ := args["search"].(string)
		if binaryPath == "" {
			return "[Error] binary_path is required"
		}
		cmd := fmt.Sprintf("ropper --file %s", binaryPath)
		if search != "" {
			cmd += fmt.Sprintf(" --search %q", search)
		} else {
			cmd += " --type rop"
		}
		cmd += " 2>&1"
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[ropper Error] %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[ROPPER — %s]\n%s", binaryPath, out)
	}

	// ── 13. ONE_GADGET (libc Exploitation) ────────────────────────────────────
	r.builtins["run_one_gadget"] = func(ctx context.Context, args map[string]any) string {
		libcPath, _ := args["libc_path"].(string)
		if libcPath == "" {
			return "[Error] libc_path is required"
		}
		cmd := fmt.Sprintf("one_gadget %s 2>&1", libcPath)
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[one_gadget Error] %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[ONE_GADGET — %s]\n%s", libcPath, out)
	}

	// ── 14. PWNTOOLS (Exploit Script Runner) ──────────────────────────────────
	r.builtins["run_pwntools"] = func(ctx context.Context, args map[string]any) string {
		script, _ := args["script"].(string)
		if script == "" {
			return "[Error] script content is required"
		}
		scriptPath := "/workspace/exploit.py"
		err := r.sandbox.WriteFile(ctx, scriptPath, script)
		if err != nil {
			return fmt.Sprintf("[pwntools Error] Failed to write script: %v", err)
		}
		out, err := r.sandbox.Execute(ctx, "python3 "+scriptPath+" 2>&1")
		if err != nil {
			return fmt.Sprintf("[pwntools Error] Execution failed: %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[PWNTOOLS EXPLOIT RUN]\n%s", out)
	}

	// ── 15. VOLATILITY 3 (Memory Forensics) ───────────────────────────────────
	r.builtins["run_volatility3"] = func(ctx context.Context, args map[string]any) string {
		memFile, _ := args["mem_file"].(string)
		plugin, _ := args["plugin"].(string)
		if memFile == "" || plugin == "" {
			return "[Error] mem_file and plugin are required (e.g., windows.pslist, linux.psaux)"
		}
		cmd := fmt.Sprintf("volatility3 -f %s %s 2>&1", memFile, plugin)
		out, err := r.sandbox.Execute(ctx, cmd)
		if err != nil {
			return fmt.Sprintf("[volatility3 Error] %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("[VOLATILITY3 — %s — %s]\n%s", plugin, memFile, out)
	}
}

// toolWrapperDefinitions returns LLM-facing tool schemas for all wrappers.
func toolWrapperDefinitions() []openai.ChatCompletionToolParam {
	return []openai.ChatCompletionToolParam{
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_nmap",
				Description: openai.String("Structured Nmap wrapper with pre-optimised flags. Modes: quick, full, vuln, stealth, udp. Eliminates flag guesswork."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"target": map[string]interface{}{"type": "string", "description": "IP, hostname, CIDR range, or URL"},
						"mode":   map[string]interface{}{"type": "string", "description": "quick | full | vuln | stealth | udp (default: default)"},
						"ports":  map[string]interface{}{"type": "string", "description": "Optional specific port range e.g. '80,443,8080' (default: top-1000)"},
					},
					"required": []string{"target"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_nuclei",
				Description: openai.String("Structured Nuclei vulnerability scanner wrapper. Automatically adds -silent, -timeout, -retries best-practice flags."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"target":   map[string]interface{}{"type": "string", "description": "URL to scan"},
						"severity": map[string]interface{}{"type": "string", "description": "critical,high,medium,low (default: critical,high,medium)"},
						"tags":     map[string]interface{}{"type": "string", "description": "Optional comma-separated tags e.g. 'rce,sqli,xss,api,jwt'"},
					},
					"required": []string{"target"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_gobuster",
				Description: openai.String("Structured Gobuster directory/subdomain brute-forcer. Uses sensible defaults for wordlist and threads."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"target":     map[string]interface{}{"type": "string", "description": "Target URL or domain"},
						"mode":       map[string]interface{}{"type": "string", "description": "dir | dns | vhost (default: dir)"},
						"wordlist":   map[string]interface{}{"type": "string", "description": "Wordlist path (default: dirbuster medium list)"},
						"extensions": map[string]interface{}{"type": "string", "description": "File extensions to check e.g. 'php,html,js,txt'"},
					},
					"required": []string{"target"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_ffuf",
				Description: openai.String("Structured FFUF fuzzer. Automatically adds -mc, -ac, -t flags. Place FUZZ in target URL where you want fuzzing."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"target":   map[string]interface{}{"type": "string", "description": "URL with FUZZ placeholder e.g. http://target.com/FUZZ"},
						"wordlist": map[string]interface{}{"type": "string", "description": "Wordlist path (default: dirbuster medium)"},
						"method":   map[string]interface{}{"type": "string", "description": "HTTP method: GET | POST (default: GET)"},
						"mode":     map[string]interface{}{"type": "string", "description": "directory | parameter (default: directory)"},
					},
					"required": []string{"target"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_sqlmap",
				Description: openai.String("Structured SQLMap wrapper. Always passes --batch to avoid interactive prompts. Never forgets risk/level flags."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"target": map[string]interface{}{"type": "string", "description": "Target URL with injectable parameter e.g. 'http://site.com/page?id=1'"},
						"level":  map[string]interface{}{"type": "integer", "description": "Test level 1-5 (default: 2)"},
						"risk":   map[string]interface{}{"type": "integer", "description": "Risk level 1-3 (default: 2)"},
						"data":   map[string]interface{}{"type": "string", "description": "POST body data if testing POST parameters"},
						"tamper": map[string]interface{}{"type": "string", "description": "Tamper scripts e.g. 'space2comment,between'"},
					},
					"required": []string{"target"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_subfinder",
				Description: openai.String("Structured Subfinder passive subdomain enumeration. Returns a clean list of discovered subdomains."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"domain":      map[string]interface{}{"type": "string", "description": "Root domain e.g. example.com"},
						"all_sources": map[string]interface{}{"type": "boolean", "description": "Use all sources including slow ones (default: false)"},
					},
					"required": []string{"domain"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_httpx",
				Description: openai.String("Structured HTTPX web prober. Detects technologies, status codes, titles, and server headers in one call."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"target": map[string]interface{}{"type": "string", "description": "URL, domain, or IP to probe"},
					},
					"required": []string{"target"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_checksec",
				Description: openai.String("Run complete binary security analysis: file type, checksec protections (NX/PIE/RELRO/canary), strings, and linked libraries. ALWAYS use this before any binary exploitation."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"binary_path": map[string]interface{}{"type": "string", "description": "Full path to binary inside sandbox e.g. /workspace/chall"},
					},
					"required": []string{"binary_path"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_hydra",
				Description: openai.String("Structured Hydra credential brute-forcer. Specify service and target — wordlists default to rockyou.txt/unix_users.txt automatically."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"target":   map[string]interface{}{"type": "string", "description": "Target IP or hostname"},
						"service":  map[string]interface{}{"type": "string", "description": "Protocol: ssh, ftp, http-post-form, smb, rdp, mysql, postgres, telnet"},
						"userlist": map[string]interface{}{"type": "string", "description": "Path to username wordlist (optional, default: unix_users.txt)"},
						"passlist": map[string]interface{}{"type": "string", "description": "Path to password wordlist (optional, default: rockyou.txt)"},
						"username": map[string]interface{}{"type": "string", "description": "Single username to test (overrides userlist)"},
						"password": map[string]interface{}{"type": "string", "description": "Single password to test (overrides passlist)"},
					},
					"required": []string{"target", "service"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_forensics_triage",
				Description: openai.String("Run complete forensics triage on a file: file type, exiftool metadata, strings, binwalk embedded files, and hex preview. Use for any unknown/suspicious file in CTF or IR."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"file_path": map[string]interface{}{"type": "string", "description": "Full path to file in sandbox e.g. /workspace/mystery.png"},
					},
					"required": []string{"file_path"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_angr",
				Description: openai.String("Run symbolic execution on a binary to find a path to a specific address, optionally avoiding another. Good for finding inputs to reach win functions."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"binary_path": map[string]interface{}{"type": "string", "description": "Path to binary"},
						"find_addr":   map[string]interface{}{"type": "string", "description": "Address to find (e.g., '0x401234' or symbol name if resolved)"},
						"avoid_addr":  map[string]interface{}{"type": "string", "description": "Address to avoid (optional)"},
					},
					"required": []string{"binary_path", "find_addr"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_ropper",
				Description: openai.String("Search for ROP gadgets in a binary. Use search parameter for specific gadgets like 'pop rdi; ret'."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"binary_path": map[string]interface{}{"type": "string", "description": "Path to binary"},
						"search":      map[string]interface{}{"type": "string", "description": "Specific gadget to search for (optional)"},
					},
					"required": []string{"binary_path"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_one_gadget",
				Description: openai.String("Find one-gadget RCE shortcuts in a libc.so binary."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"libc_path": map[string]interface{}{"type": "string", "description": "Path to libc.so"},
					},
					"required": []string{"libc_path"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_pwntools",
				Description: openai.String("Write and execute a custom pwntools Python script. Use this to run your final exploits."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"script": map[string]interface{}{"type": "string", "description": "The complete Python script contents"},
					},
					"required": []string{"script"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "run_volatility3",
				Description: openai.String("Run memory forensics on a dump file. Specify the OS-appropriate plugin (windows.pslist, linux.bash, mac.lsof)."),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"mem_file": map[string]interface{}{"type": "string", "description": "Path to memory dump file"},
						"plugin":   map[string]interface{}{"type": "string", "description": "Volatility 3 plugin name"},
					},
					"required": []string{"mem_file", "plugin"},
				},
			},
		},
	}
}
