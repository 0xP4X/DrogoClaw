# Skill Learning

DrogonClaw learns from successful attacks and reuses techniques on similar targets. Inspired by Hermes Agent's autonomous skill creation system.

## The Problem

Without skill learning, every new engagement starts from zero. The agent re-discovers the same SQL injection techniques on WordPress targets, the same privilege escalation paths on Linux systems, and the same enumeration patterns on every network.

## The Solution

The `SkillLearner` observes tool executions, detects verified successes, and saves the technique as a reusable skill. Future targets are automatically matched against learned patterns, and the most reliable techniques are injected into the LLM context.

## How It Works

```
Tool Execution → Evidence Verification → Skill Learner → Learned Skills
                                                            ↓
                                              Next Target → LLM Context
```

### Step 1: Observe Execution

After every tool execution, the orchestrator records the outcome:

```go
skillLearner.ObserveExecution("run_nuclei", args, output, verified)
```

### Step 2: Classify Target

The learner identifies what kind of system was attacked:

| Output Pattern | Target Classification |
|----------------|----------------------|
| `WordPress`, `wp-content`, `wp-admin` | `wordpress` |
| `Apache/2.4`, `apache` | `apache` |
| `nginx/1.18` | `nginx` |
| `OpenSSH_8.2` | `ssh` |
| `MySQL 8.0` | `mysql` |
| `PostgreSQL` | `postgres` |
| `Microsoft-IIS` | `iis` |
| `SMB`, `Samba` | `smb` |
| `Docker` | `docker` |
| `Kubernetes` | `kubernetes` |

### Step 3: Extract Findings

The learner extracts structured findings from the output:

```
CVE-2024-1234 → cve:CVE-2024-1234
80/open       → port:80/open
status: 200   → http:200
SQL injection → vuln:sqli
XSS           → vuln:xss
RCE           → vuln:rce
```

### Step 4: Save or Reinforce Skill

- **New technique** → Creates a new skill with success count = 1
- **Existing technique** → Increments success count (reinforcement)

### Step 5: Inject into Context

When attacking a new target, the learner finds relevant skills and formats them for the LLM:

```
LEARNED ATTACK PATTERNS (from previous successes):
- [exploit] nuclei on wordpress — WordPress vulnerability found (success rate: 100%, used 3 times)
  Tool: run_nuclei
  How: Successfully used run_nuclei against wordpress targets, finding: vuln:confirmed
- [web] httpx on apache — Apache detection (success rate: 100%, used 5 times)
  Tool: run_httpx
  How: Successfully used run_httpx against apache targets, finding: http:200
```

## Skill Structure

```json
{
  "id": "run_nuclei::wordpress::cve:CVE-2024-1234",
  "name": "nuclei on wordpress — cve:CVE-2024-1234",
  "category": "exploit",
  "tool": "run_nuclei",
  "args": {"target": "http://wordpress.com"},
  "target_pattern": "wordpress",
  "findings": ["cve:CVE-2024-1234", "vuln:confirmed"],
  "description": "Successfully used run_nuclei against wordpress targets, finding: cve:CVE-2024-1234",
  "tags": ["wordpress", "exploit", "cve:CVE-2024-1234"],
  "success_count": 3,
  "fail_count": 0,
  "last_used": "2026-08-26T10:00:00Z",
  "created_at": "2026-08-26T09:00:00Z"
}
```

## Categories

| Category | Description | Example Tools |
|----------|-------------|---------------|
| `recon` | Reconnaissance and enumeration | `run_nmap`, `run_subfinder`, `run_httpx` |
| `exploit` | Vulnerability exploitation | `run_nuclei`, `run_sqlmap` |
| `enum` | Directory/parameter enumeration | `run_gobuster`, `run_ffuf` |
| `web` | Web application testing | HTTP tools, curl, requests |
| `privesc` | Privilege escalation | sudo, SUID, kernel exploits |
| `lateral` | Lateral movement | SMB, SSH, WinRM |

## Skill Matching

When looking up skills for a new target:

1. **Exact match** — Same target pattern (e.g., `wordpress`)
2. **Partial match** — Target contains query or vice versa
3. **Sort by success rate** — Most reliable skills first
4. **Limit to top 5** — Don't overwhelm the LLM

## Integration with Orchestrator

The skill learner is automatically integrated:

```go
// In orchestrator.go:
learnedCtx := o.tools.GetLearnedContext(target)
messages = BuildMessages(o.sysPrompt, combinedCtx, o.history, userMsg)

// In tools.go (after verified success):
r.skillLearner.ObserveExecution(tr.ToolName, nil, tr.Stdout, true)
```

## Persistence

Skills are stored as JSON files in `data/learned_skills/`:

```
data/learned_skills/
├── run_nuclei::wordpress::cve_CVE-2024-1234.json
├── run_nmap::ssh::port_22_open.json
├── run_sqlmap::php::vuln_sqli.json
└── ...
```

## API Reference

### SkillLearner
- `NewSkillLearner(dataDir string) *SkillLearner`
- `ObserveExecution(tool string, args map[string]any, result string, verified bool)`
- `RecordFailure(tool string, args map[string]any, result string)`
- `FindRelevantSkills(targetPattern string, category string) []*LearnedSkill`
- `GetSkill(id string) *LearnedSkill`
- `ListSkills() []*LearnedSkill`
- `FormatForLLM(target string) string`
- `SaveHistory(tool string, args map[string]any, result string, duration time.Duration)`
