# DrogonClaw Tool Reference

Complete reference for all DrogonClaw tools, their parameters, and usage.

## Tool Wrappers

### `run_nmap`

Network port scanner wrapper with best-practice flag defaults.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target IP, hostname, or CIDR |
| `mode` | string | No | `default` | Scan mode: `quick`, `udp`, `vuln`, `stealth`, `full` |
| `ports` | string | No | `-` (all) | Port specification (e.g., `80,443` or `1-1000`) |

**Modes:**
- `quick` — `-Pn -sV -sC --open -T4 -p 80,443,22,21,25,8080,8443,3306,5432`
- `udp` — `-Pn -sU --top-ports 200 -T4`
- `vuln` — `-Pn -sV -sC --script vuln -T4`
- `stealth` — `-Pn -sS -sV -T2 --randomize-hosts`
- `full` — `-Pn -sV -sC -O -A -T4 -p-`

---

### `run_nuclei`

Vulnerability scanner using Nuclei templates.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target URL or IP |
| `severity` | string | No | `critical,high,medium` | Comma-separated severity levels |
| `tags` | string | No | — | Template tags to filter by |
| `dast` | bool | No | `false` | Enable DAST mode |

---

### `run_gobuster`

Directory/subdomain brute-forcer.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target URL |
| `mode` | string | No | `dir` | Mode: `dir`, `vhost`, `dns` |
| `wordlist` | string | No | `/usr/share/wordlists/dirbuster/directory-list-2.3-medium.txt` | Wordlist path |
| `extensions` | string | No | — | File extensions (dir mode only) |

---

### `run_ffuf`

Web fuzzer for directories, parameters, and more.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target URL with `FUZZ` placeholder |
| `wordlist` | string | No | `/usr/share/wordlists/dirbuster/directory-list-2.3-medium.txt` | Wordlist path |
| `mode` | string | No | `dir` | Fuzzing mode |
| `method` | string | No | `GET` | HTTP method |

---

### `run_sqlmap`

Automatic SQL injection detection and exploitation.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target URL with injectable parameter |
| `level` | int | No | `1` | Test level (1-5) |
| `risk` | int | No | `1` | Risk level (1-3) |
| `dbs` | bool | No | `false` | Enumerate databases |
| `tables` | bool | No | `false` | Enumerate tables |

---

### `run_subfinder`

Subdomain enumeration tool.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target domain |

---

### `run_httpx`

HTTP probing and technology detection.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target domain or IP |

---

### `run_hydra`

Credential brute-forcing.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target IP |
| `service` | string | Yes | — | Service (ssh, ftp, http, etc.) |
| `user` | string | No | — | Username or username file |
| `pass` | string | No | — | Password or password file |

---

### `run_checksec`

Binary security analysis.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Path to binary |

---

## Built-in Tools

### `shell_execute`

Execute arbitrary shell commands in the sandbox.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `command` | string | Yes | — | Shell command to execute |
| `timeout` | int | No | `60` | Timeout in seconds |

**Note:** Subject to dynamic skill denylist. Dangerous commands are blocked.

---

### `store_loot`

Store findings in the loot database.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `type` | string | Yes | — | Loot type: `credential`, `flag`, `vulnerability`, `note` |
| `data` | string | Yes | — | JSON-encoded loot data |
| `target` | string | No | — | Associated target |

---

### `query_loot`

Query the loot database.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `type` | string | No | — | Filter by loot type |
| `target` | string | No | — | Filter by target |

---

### `graph_add_node`

Add an entity to the intelligence graph.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `id` | string | Yes | — | Unique entity ID |
| `label` | string | Yes | — | Entity type: `Target`, `Asset`, `Port`, `Service`, `Vulnerability`, `Credential`, `Flag`, `Task` |
| `properties` | object | No | — | Key-value properties |

---

### `graph_add_edge`

Add a relationship between entities.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `source_id` | string | Yes | — | Source entity ID |
| `target_id` | string | Yes | — | Target entity ID |
| `relationship` | string | Yes | — | Relationship type |

---

### `deep_research`

Extended research using OSINT sources.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | Yes | — | Research query |
| `sources` | string | No | — | Comma-separated sources |

---

### `osint_certs`

Certificate transparency log search.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target domain |

---

### `refresh_cve_feeds`

Update the local CVE database from NVD.

No required parameters.

---

### `write_and_run_script`

Write and execute a script in the sandbox.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `language` | string | Yes | — | Script language (python, bash, etc.) |
| `code` | string | Yes | — | Script content |
| `timeout` | int | No | `300` | Timeout in seconds |

**Note:** Always requires Human-in-the-Loop approval.

---

## Adversary Tools

### `advanced_web_exploiter`

Advanced web application exploitation.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target URL |
| `technique` | string | No | — | Specific technique |

**Note:** Always requires Human-in-the-Loop approval.

---

### `zero_click_exploiter`

Zero-click exploitation framework.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target |
| `cve` | string | No | — | Specific CVE |

**Note:** Always requires Human-in-the-Loop approval.

---

### `dynamic_payload_compiler`

Compile custom payloads at runtime.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `payload_type` | string | Yes | — | Payload type |
| `target_os` | string | No | `linux` | Target OS |
| `arch` | string | No | `amd64` | Target architecture |

**Note:** Always requires Human-in-the-Loop approval.

---

### `autonomous_fuzzing_engine`

Autonomous vulnerability fuzzing.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target URL or endpoint |
| `mode` | string | No | `auto` | Fuzzing mode |

**Note:** Always requires Human-in-the-Loop approval.

---

### `async_race_condition_engine`

Race condition detection and exploitation.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target URL |
| `endpoints` | string | No | — | Comma-separated endpoint pairs |

**Note:** Always requires Human-in-the-Loop approval.

---

### `fuzz_endpoint`

Endpoint fuzzing with FFUF integration.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `target` | string | Yes | — | Target URL with FUZZ placeholder |
| `wordlist` | string | No | — | Custom wordlist |

---

## Ghost Tools (Anti-Forensics)

All ghost tools require Human-in-the-Loop approval.

| Tool | Description |
|------|-------------|
| `ghost_wipe_logs` | Wipe system logs |
| `ghost_secure_delete` | Secure file deletion |
| `ghost_clear_history` | Clear command history |

---

## OSINT Tools

| Tool | Description |
|------|-------------|
| `intel_shodan` | Shodan device search |
| `intel_virustotal` | VirusTotal malware analysis |
| `intel_github_dork` | GitHub dork searches |
| `intel_censys` | Censys host search |

---

## Cloud Tools

| Tool | Description |
|------|-------------|
| `cloud_s3_enum` | AWS S3 bucket enumeration |
| `cloud_azure_enum` | Azure resource enumeration |
| `cloud_gcp_enum` | GCP project enumeration |

---

## Blue Team Tools

| Tool | Description |
|------|-------------|
| `blueteam_triage` | Incident triage |
| `blueteam_hunt` | Threat hunting |
| `blueteam_compliance` | Compliance checking |
| `run_forensics_triage` | Digital forensics |

---

## CTF Tools

| Tool | Description |
|------|-------------|
| `ctf_flag_submit` | Flag submission |
| `ctf_challenge_solve` | Challenge solver |

---

## Evidence Verification

Every tool execution goes through the Evidence Pipeline:

1. **Tool executes** → raw output captured
2. **classifyOutcome** → typed `ToolResult` with failure class
3. **extractFindings** → conservative, high-signal evidence extraction
4. **EvaluateTool** → deterministic verdict (verified/clean/failed)
5. **RecordVerifiedFinding** → persists to LootDB + teaches Skill Learner
6. **Evidence footer** → appended to tool result for LLM context

Only verified findings (flags, CVEs, confirmed vulns) count as success. Prose claims do not.
