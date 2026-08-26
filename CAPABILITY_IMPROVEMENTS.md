# DrogonClaw Capability Improvements Plan

Generated from comprehensive codebase audit on 2026-08-26.

---

## Executive Summary

Audited the entire DrogonClaw codebase across 4 dimensions: tool registry/completeness, orchestrator/agent logic, LLM prompts, and test coverage. Found **17 critical/high bugs**, **40+ missing capabilities**, **23 security issues**, and **17 untested packages**. This plan organizes improvements by priority and logical grouping.

---

## Phase 1: Critical Bugs (Fix Immediately)

These are correctness issues that corrupt data or crash the system.

### 1.1 Duplicate assistant message in orchestrator history
- **File:** `internal/agent/orchestrator.go:297`
- **Bug:** Final assistant message appended to `o.history` twice (once at line 251, again at line 297)
- **Fix:** Remove the duplicate append at line 297

### 1.2 Evidence evaluation results silently discarded
- **File:** `internal/agent/orchestrator.go:353-358`
- **Bug:** `EvaluateTool` returns `estatus` and `reason` but both are assigned to `_`
- **Fix:** Append evidence verdict as footer to tool result so LLM sees `[EVIDENCE: verified]` or `[EVIDENCE: clean]`

### 1.3 Deprecated `net.Error.Temporary()` in retry logic
- **File:** `internal/agent/provider.go:279`
- **Bug:** `netErr.Temporary()` always returns false in Go 1.18+, breaking network error retries
- **Fix:** Replace with explicit error type checks (e.g., `net.Error` interface)

### 1.4 Swarm agents share graph with race condition on save
- **File:** `internal/agent/swarm.go:57` + `internal/memory/graph.go:298`
- **Bug:** Multiple goroutines writing to same temp file path (`g.dbPath + ".tmp"`)
- **Fix:** Use unique temp file per goroutine (add PID/random suffix)

### 1.5 Mission plan generated but never drives execution
- **File:** `internal/agent/orchestrator.go:192-204`
- **Bug:** Plan is displayed but the ReAct loop never references it
- **Fix:** Inject plan steps into system prompt, track step completion, add plan-vs-actual comparison

### 1.6 Two conflicting LootDB implementations
- **File:** `internal/core/loot.go` + `internal/memory/loot.go`
- **Bug:** Two separate `LootDB` structs targeting same SQLite file
- **Fix:** Remove `core.LootDB`, consolidate on `memory.LootDB`

### 1.7 Validator panics on empty choices
- **File:** `internal/agent/validator.go:66`
- **Bug:** `resp.Choices[0].Message.Content` with no bounds check
- **Fix:** Add `len(resp.Choices) > 0` guard

---

## Phase 2: Security Hardening

### 2.1 Shell injection in tool wrappers (ALL wrappers)
- **Files:** `internal/agent/toolwrappers.go:103,123,156,187,218,241,267,325,564`
- **Issue:** `fmt.Sprintf("nmap %s %s", flags, target)` with unsanitized LLM input
- **Fix:** Use `shellescape.Quote()` (from `github.com/a8m/shellEscape`) or `strconv.Quote` for all interpolated values

### 2.2 Autonomous code-generation tools bypass HitL
- **Files:** `internal/agent/tools.go:1356-1984` (6 tools)
- **Issue:** `autonomous_fuzzing_engine`, `dynamic_payload_compiler`, `advanced_web_exploiter`, `headless_browser_automation`, `zero_click_exploiter`, `async_race_condition_engine` generate and execute code without risk analysis
- **Fix:** Route through `adapt.AnalyzeScript()` before execution

### 2.3 Raw external data enters LLM context unsanitized
- **File:** `internal/agent/orchestrator.go:360-361`
- **Issue:** Tool output from `fetch_url`, `web_search`, `osint_*` injected raw
- **Fix:** Strip/escape XML-like tags and known injection patterns before injection

### 2.4 Dynamic skill persistence has permissive denylist
- **File:** `internal/adapt/skills.go:116-123`
- **Issue:** Only blocks `rm -rf`, `mkfs`, `dd`, `shutdown`, `reboot`
- **Fix:** Add `curl`, `wget`, `nc`, `bash -i`, reverse shell patterns, `chmod 777`, etc.

### 2.5 Ghost tools have no operator confirmation
- **File:** `internal/agent/tools.go:2107-2145`
- **Issue:** `ghost_wipe_logs`, `ghost_secure_delete`, `ghost_clear_history` run without HitL
- **Fix:** Add approval gate for evidence destruction tools

---

## Phase 3: Capability Gaps (Missing Tools & Features)

### 3.1 AD/Pivoting (12 missing tools)

| Tool | Priority | Notes |
|------|----------|-------|
| NetExec (nxc) | HIGH | CrackMapExec replacement; enumerate shares, sessions, execute |
| Responder | HIGH | LLMNR/NBT-NS poisoning for NTLM hash capture |
| ntlmrelayx.py | HIGH | NTLM relay attacks |
| certipy/certify | HIGH | ADCS abuse (ESC1-ESC8) |
| ZeroLogon (CVE-2020-1472) | HIGH | Critical AD exploit, missing from templates |
| PetitPotam/PrinterBug | MEDIUM | NTLM relay coercion |
| pywhisker | MEDIUM | Shadow Credentials attack |
| Pass-the-Ticket | MEDIUM | Kerberos ticket lateral movement |
| Overpass-the-Hash | MEDIUM | NTLM hash → Kerberos TGT |
| Golden/Silver Ticket | LOW | Kerberos forgeable tickets |
| dacledit.py | LOW | ACL abuse |
| gmsadumper.py | LOW | gMSA password extraction |

### 3.2 Modern CVE Exploit Templates (8 missing)

| CVE | Target | Priority |
|-----|--------|----------|
| CVE-2021-26855 (ProxyLogon) | Exchange Server | HIGH |
| CVE-2021-27065 (ProxyShell) | Exchange Server | HIGH |
| CVE-2022-22963 | Spring Cloud Function | HIGH |
| CVE-2023-23397 | Outlook Elevation | HIGH |
| CVE-2023-44228 | Citrix Bleed | MEDIUM |
| CVE-2023-46805/Ivanti | Ivanti Connect Secure | MEDIUM |
| CVE-2023-22527 | Confluence RCE | MEDIUM |
| CVE-2023-34362 | MOVEit Transfer | LOW |

### 3.3 Pivoting (expand from 1 tool to 5)

| Tool | Priority | Notes |
|------|----------|-------|
| Ligolo-ng | HIGH | Modern SOCKS5/VPN pivot (already in skills manifest, needs Go impl) |
| SSH dynamic forwarding | HIGH | `ssh -D` for quick pivoting |
| WireGuard tunneling | MEDIUM | L3 routing for full network access |
| Multi-hop chaining | MEDIUM | Chain pivots (pivot → pivot → target) |
| Chisel ARM/Windows support | LOW | Cross-platform pivot binaries |

### 3.4 Evasion/Ghost (expand capabilities)

| Feature | Priority | Notes |
|---------|----------|-------|
| AMSI bypass | HIGH | Essential for PowerShell payloads on modern Windows |
| AES-256 payload encryption | HIGH | Replace single-layer XOR |
| Sleep obfuscation | MEDIUM | XOR decrypt during sleep to evade memory scanners |
| Process injection | MEDIUM | CreateRemoteThread, NtMapViewOfSection, APC |
| ETW patching | LOW | Event Tracing bypass |
| Direct syscall stubs | LOW | Avoid userland EDR hooks |

### 3.5 Exfiltration (expand from 3 to 6 channels)

| Channel | Priority | Notes |
|---------|----------|-------|
| HTTPS POST exfil | HIGH | Most common in real engagements |
| DNS-over-HTTPS tunnel | MEDIUM | Evade DNS monitoring |
| Chunk reassembly verification | HIGH | Lost chunks currently silently corrupt files |
| Receiver auto-start | MEDIUM | Currently requires manual setup |
| Encrypted covert channel | HIGH | Currently plaintext hex on wire |

### 3.6 Social Engineering (fix broken features)

| Issue | Priority | Fix |
|-------|----------|-----|
| GoPhish version outdated | HIGH | Update to latest release |
| Only 1 email template | HIGH | Add 5+ pretexts (IT helpdesk, shipping, invoice, exec impersonation) |
| SMTP TLS disabled | HIGH | Uncomment and configure TLS |
| Evilginx2 not integrated | MEDIUM | Add MFA bypass reverse proxy |
| No campaign tracking | LOW | Add click/open tracking |

### 3.7 Whitebox Assessment (expand from 5 to 13 vuln classes)

| Class | Priority |
|-------|----------|
| CSRF | HIGH |
| XXE | HIGH |
| Path Traversal | HIGH |
| File Upload | MEDIUM |
| Insecure Deserialization | MEDIUM |
| Business Logic | LOW |
| API-specific (BOLA, mass assignment) | MEDIUM |
| WebSocket vulnerabilities | LOW |

---

## Phase 4: Orchestrator & Intelligence Improvements

### 4.1 Context Management

| Issue | Fix |
|-------|-----|
| `maxHistoryMessages=24` hardcoded | Make configurable, increase default to 48 |
| Message-count truncation (not token-based) | Add token counting via tiktoken-go or approximation |
| No summarization of dropped messages | Summarize older turns before truncating |
| Graph snapshot capped at 12 entities | Increase to 25, add relevance prioritization |

### 4.2 Mission Plan Integration

| Issue | Fix |
|-------|-----|
| Plan never drives execution | Inject plan steps into system prompt context |
| No step completion tracking | Mark steps done as tools succeed |
| No plan-vs-actual comparison | Log deviations, report in final summary |
| Regex extraction greedy | Use non-greedy `(?s)\{.*?\}` or JSON stream decoder |

### 4.3 Memory & Learning

| Issue | Fix |
|-------|-----|
| Preferences not persisted | Save to `data/preferences.json` |
| Failure memory not wired | Use `RecordPersistent` instead of `Record` |
| Graph has no pruning | Add age-based eviction and duplicate merging |
| `BuildDirectiveBlock()` never called | Wire into prompt assembly at `prompt.go:122` |

### 4.4 Prompt Engineering

| Issue | Fix |
|-------|-----|
| Hardcoded tool list in prompt | Generate dynamically from `Definitions()` |
| No few-shot examples | Add 2 examples of multi-step tool sequences |
| Missing directive numbers (8, 9) | Renumber 10 → 8 |
| Persona fluff wastes ~100 tokens | Condense to 1-2 sentences |
| Docker vs native contradiction | Reconcile prompt and tool descriptions |
| Tautological parameter descriptions | Fix in `skills_manifest.json` |

### 4.5 Tool Registry Cleanup

| Issue | Fix |
|-------|-----|
| `crackmapexec` deprecated | Replace with NetExec |
| `Definitions()` recomputed every iteration | Cache on first call |
| `run_*` unknown tools default to 30min estimate | Use 5min default |
| Dual skill/builtin definitions | Deduplicate (manifest vs builtins) |

---

## Phase 5: Test Coverage (from 39% to 80%)

### Priority 1: Write immediately (pure functions, trivially testable)

| Package | Test Target | ~Cases |
|---------|-------------|--------|
| `internal/cvss` | `ParseVector`, `CalculateBaseScore`, `roundUp`, `classifySeverity` | 30 |
| `internal/redteam/exploitation` | `ParseExploitResult` (7 states), `StateAdvice` | 15 |
| `internal/memory/graph` | AddNode, AddEdge, Snapshot, persistence, concurrency | 20 |
| `internal/planner` | `Next`, `MarkDone`, `HasPlan`, `BuildWhiteboxPlan` | 15 |

### Priority 2: Write next (mockable, high-value)

| Package | Test Target | ~Cases |
|---------|-------------|--------|
| `internal/agent/provider` | `BuildMessages`, `IsChatOnly`, `isRetryableLLMError` | 20 |
| `internal/agent/tools` | `extractFindings`, `classifyOutcome`, `EvaluateTool` | 25 |
| `internal/core/mission` | `NextPending`, `AllCompleted`, `CompletedCount` | 10 |
| `internal/memory/failure` | `HasFailed`, persistence round-trip | 10 |

### Priority 3: Write next (supporting logic)

| Package | Test Target | ~Cases |
|---------|-------------|--------|
| `internal/memory/preferences` | `RecordRejection`, `BuildPreferenceBlock` | 8 |
| `internal/billing` | `Record`, `TotalCost`, `Render` | 10 |
| `internal/core/loot` | `encryptLoot` round-trip | 5 |

### Shared Test Utilities to Extract

| Utility | From | Into |
|---------|------|------|
| `roundTripFunc` (mock HTTP) | `intel/github_test.go` | `internal/testutil/http.go` |
| `MockLLMProvider` | `core/report_test.go` | `internal/testutil/llm.go` |
| Graph factory | `agent/tools_gate_test.go` | `internal/testutil/graph.go` |
| Temp config dir | `config/manager_test.go` | `internal/testutil/config.go` |

---

## Phase 6: Code Quality

### 6.1 Decompose `tools.go` (2300+ lines)
- Extract tool dispatch into `internal/agent/dispatch.go`
- Extract builtin implementations into `internal/agent/builtins.go`
- Extract evidence classification into `internal/agent/evidence.go`

### 6.2 Add structured logging
- Replace `fmt.Printf`/`log.Printf` with `log/slog`
- Add log levels (debug, info, warn, error)
- Add log output configuration

### 6.3 Fix inconsistent error handling
- Decide: return errors everywhere or swallow with logging
- Standardize on `fmt.Errorf` with `%w` wrapping

### 6.4 Fix hardcoded paths
- Make all `/tmp/*` paths configurable via `data/` directory
- Use `os.MkdirAll` for directory creation

---

## Implementation Order (Recommended)

| Phase | Effort | Impact | Order |
|-------|--------|--------|-------|
| Phase 1: Critical Bugs | 2-3 hours | CRITICAL | 1st |
| Phase 2: Security Hardening | 3-4 hours | HIGH | 2nd |
| Phase 4.4-4.5: Prompt & Registry | 2-3 hours | HIGH | 3rd |
| Phase 5.1: CVSS/Exploit/Graph tests | 3-4 hours | HIGH | 4th |
| Phase 3.1: AD/Pivoting tools | 4-6 hours | HIGH | 5th |
| Phase 4.1-4.3: Orchestrator improvements | 4-6 hours | HIGH | 6th |
| Phase 3.2-3.7: Remaining capabilities | 6-8 hours | MEDIUM | 7th |
| Phase 5.2-5.3: Remaining tests | 3-4 hours | MEDIUM | 8th |
| Phase 6: Code quality | 4-6 hours | LOW-MEDIUM | 9th |

**Total estimated effort: 31-44 hours**

---

## Quick Wins (< 30 min each, high impact)

1. Fix duplicate history append (`orchestrator.go:297`)
2. Fix evidence evaluation discard (`orchestrator.go:353-358`)
3. Fix validator panic (`validator.go:66`)
4. Fix deprecated `Temporary()` (`provider.go:279`)
5. Cache `Definitions()` result
6. Fix directive numbering in prompt
7. Fix greedy regex in mission planner
8. Add `shellescape.Quote()` to top 5 tool wrappers
