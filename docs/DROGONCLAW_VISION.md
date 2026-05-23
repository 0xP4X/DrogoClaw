# DrogonClaw - Autonomous Pentesting Agent
## True OpenClaw-Style Architecture for Kali Linux

---

## VISION: What DrogonClaw Actually Is

**NOT**: A tool wrapper or command aggregator  
**IS**: A truly autonomous AI agent that:

- 🧠 **THINKS** before acting (extended thinking)
- 🎯 **DECIDES** what to do next (tool selection)
- ⚡ **ACTS** on Kali tools (full bash/process access)
- 💾 **REMEMBERS** across sessions (persistent memory)
- 🔄 **LEARNS** from results (adaptive workflows)
- 📢 **COMMUNICATES** via CLI or Telegram
- 🏗️ **SCALES** to complex multi-phase pentests

**Gateway Pattern** (Like OpenClaw):
```
User (CLI/Telegram)
  ↓
[Gateway] Local control plane (port 18789)
  ├─ Session manager (persistent state)
  ├─ Agent loop (think → decide → act)
  ├─ Skill registry (pluggable tools)
  └─ Tool executor (Kali bash commands)
```

---

## Core Architecture (5 Components)

### 1. GATEWAY (Control Plane)
- Single Node.js process listens on localhost:18789
- Manages all sessions
- Routes messages from channels to agent
- Executes tools, persists state
- Streams progress to user

### 2. AGENT LOOP (Autonomous Decision-Making)
- **Intake**: Message from user
- **Context**: Load session, previous findings, target info
- **Think**: Extended thinking phase
- **Decide**: Choose tools and actions
- **Act**: Execute tools, capture output
- **Adapt**: Update mental model
- **Persist**: Save session state

### 3. SKILL REGISTRY (Tool Instructions)
- YAML + Markdown skill definitions
- ~50 skills covering recon → exploitation → post-ex
- Agent selects relevant skills based on context
- Each skill teaches the agent HOW to use a Kali tool
- Extensible: community can add more skills

### 4. CHANNELS (User Interfaces)
- **CLI**: Interactive terminal prompts + streaming output
- **Telegram**: `/pentest` commands, progress updates, findings
- WebSocket API for future dashboards
- All channels feed same gateway

### 5. PERSISTENCE (Memory & State)
- SQLite database stores:
  - Sessions (target, findings, transcript)
  - Tools used (audit trail)
  - Credentials discovered (encrypted)
  - Timeline of discovery

---

## Why This Is Different From Current Tools

| Aspect | Traditional Tools | DrogonClaw |
|--------|------------------|-----------|
| **Execution** | User runs commands | Agent decides & runs |
| **Intelligence** | No context | Extended thinking |
| **Decision Making** | Manual | Autonomous (tool selection) |
| **Adaptation** | No | Agent adapts based on findings |
| **Persistence** | None | Full session history |
| **Communication** | One interface | CLI + Telegram |
| **Learning** | No | Can improve within session |
| **Autonomy** | Limited | Full autonomous mode |

**Example:**
```
Traditional: "User runs nmap, reads results, runs dirb, reads results, chooses sqlmap"
DrogonClaw: "Agent runs all 3, analyzes together, DECIDES which is best next tool"
```

---

## Autonomous Loop (Real Example)

```
Input: "Pentest example.com"

[1] THINK PHASE
Agent: "I need to:
- Discover what's running (nmap, dns, etc)
- Find web applications (HTTP, HTTPS, web services)
- Look for common vulnerabilities
- Try to exploit them
- Get shell access if possible"

[2] EXECUTION PHASE (with thinking between each step)
→ nmap -sV example.com
  Output: Ports 22,80,443 open
  Agent thinks: "Good, web server. Let's check for tech stack"

→ whatweb example.com
  Output: WordPress 5.1, Apache 2.4.6
  Agent thinks: "WordPress! Known vulnerabilities. Let me check plugins"

→ wpscan --url example.com
  Output: Plugin "xyz" RCE vuln found
  Agent thinks: "BINGO! RCE found. Let me exploit this"

→ metasploit exploit/wordpress/plugin/xyz
  Output: Shell obtained!
  Agent thinks: "Got shell. Now privilege escalation..."

→ sudo -l
  Output: User can run /bin/bash without password
  Agent thinks: "CRITICAL! Full system compromise achieved"

[3] REPORT PHASE
Agent generates:
- Timeline of discovery
- 5 vulnerabilities found
- CVSS scores
- Exploitation proof
- Remediation recommendations
```

The agent THINKS at each step, which is why it's truly autonomous.

---

## Technical Implementation (High-Level)

### Gateway
```
Express.js server
├─ POST /agent (start pentesting)
├─ WS (streaming results)
├─ GET /health (status)
└─ CRUD /sessions (manage tests)
```

### Agent Loop
```
1. Load session (transcript + findings)
2. Build system prompt:
   - Agent role: "You are a pentesting specialist"
   - Available skills: "dns-enum, nmap, sqlmap, etc"
   - Target context: "IP, open ports, tech stack so far"
   - Extended thinking: "Reason through your approach"
3. Call Claude API with tool_use feature
4. Execute returned tools
5. Feed results back to agent
6. Loop until goal achieved or timeout
```

### Skills
```
skills/
├── dns-enum/SKILL.md
│   ├── Teaches: How to use dnsenum, whois, dig
│   ├── When: Reconnaissance phase
│   └── Output parsing: Extract domains
│
├── sql-injection/SKILL.md
│   ├── Teaches: How to use sqlmap effectively
│   ├── When: Exploitation phase
│   └── Evasion: IDS/WAF bypasses
│
└── privilege-escalation/SKILL.md
    ├── Teaches: Linux/Windows privesc techniques
    ├── When: Post-exploitation
    └─ Automation: linprivesc.sh runner
```

### Channels
```
CLI:
  $ drogonclaw pentest example.com
  🐉 Starting...
  [→] Reconnaissance
  [→] Enumeration
  [✓] Found RCE vulnerability!

Telegram:
  /pentest example.com
  Bot: 🐉 Starting autonomous pentest...
  [✓] Found 3 open ports
  [✓] WordPress detected
  [✓] RCE vulnerability confirmed!
```

### Persistence
```
SQLite Database
├─ sessions table
│  ├─ session_id
│  ├─ target
│  ├─ status
│  ├─ transcript (full conversation)
│  └─ findings (structured vulnerabilities)
│
├─ findings table
│  ├─ type (sql-injection, xss, rce, etc)
│  ├─ severity (critical, high, medium)
│  ├─ proof (command run, output)
│  └─ remediation
│
└─ tools_used table
   ├─ tool_name
   ├─ command_line
   ├─ timestamp
   └─ output_summary
```

---

## Key Differences From My Previous Plan

### ❌ Old Plan
- "Integrate HexStrike tools directly"
- "Tool executor wrapper"
- "Command aggregator"

### ✅ New Plan
- "Use MCP to help develop the AI/agent logic"
- "HexStrike tools are AVAILABLE but DrogonClaw uses native Kali tools"
- "TRUE autonomous agent (like OpenClaw itself)"
- "Extended thinking drives decision-making"
- "Agent SELECTS which tools to use"

---

## Kali Tools Available (100+)

**Reconnaissance (15+):**
nmap, masscan, dnsenum, fierce, whois, nslookup, dig, amass, subfinder, recon-ng

**Web Enumeration (10+):**
nikto, dirb, dirsearch, feroxbuster, whatweb, httpx, wappalyzer, theharvester

**Exploitation (20+):**
sqlmap, xsser, metasploit, msfvenom, hashcat, john, hydra, aircrack-ng, evil-winrm

**Post-Exploitation (15+):**
linprivesc, winprivesc, mimikatz, impacket, volatility, radare2, ghidra

**Total:** 300+ tools available on Kali Linux

**DrogonClaw accesses them all via:**
```bash
# Agent generates these commands
nmap -sV 192.168.1.1
sqlmap -u "http://example.com" --dbs
metasploit exploit/windows/smb/ms17_010_eternalblue
hashcat -m 1000 hashes.txt wordlist.txt

# Execute in subprocess with streaming
```

---

## Multi-Channel Vision

### Terminal (Interactive)
```bash
$ drogonclaw onboard  # Setup wizard
$ drogonclaw pentest --target example.com --aggressive
$ drogonclaw report findings.md
$ drogonclaw skill install custom-ssl-checker
```

### Telegram (Remote)
```
User: /pentest example.com
Bot: 🐉 Autonomous pentest started on example.com
      
      [Reconnaissance]
      ✓ DNS: 4 subdomains found
      ✓ Ports: 22, 80, 443 open
      ✓ Services: SSH, Apache, Nginx
      
      [Enumeration]
      ✓ WordPress 5.8
      ✓ 12 plugins (3 vulnerable)
      ✓ Outdated jQuery
      
      [Exploitation]
      ✓ Plugin RCE vulnerability
      ✓ Shell obtained (www-data)
      
      [Post-Exploitation]
      ✓ Database access
      ✓ File system access
      ✓ User enumeration
      
      📊 CRITICAL: 5 exploitable vulns found
      💾 Full session saved
```

---

## Why OpenClaw Pattern?

OpenClaw proved:
- ✅ Gateway-based architecture scales
- ✅ Multi-channel support works seamlessly
- ✅ Autonomous agent loops are reliable
- ✅ Persistent sessions enable continuity
- ✅ Skills system is extensible
- ✅ Users prefer natural interfaces (CLI + chat)

DrogonClaw applies same pattern to pentesting domain.

---

## Real Autonomy Through Extended Thinking

```
User: "Pentest example.com, assume nothing is running"

Agent's Thinking:
"OK, I need to:
1. First discover what's there (active reconnaissance)
2. Then identify what technology is in use
3. Then find vulnerabilities based on tech
4. Then try to exploit them
5. Then achieve goal (RCE, data access, etc)

My tools:
- nmap (port discovery) 
- whatweb (tech detection)
- sqlmap (SQL injection)
- hydra (brute force)
- metasploit (exploitation)
- linprivesc (privilege escalation)

My approach:
1. Start with nmap scan
2. Parse results
3. Based on open ports, decide next tools
4. If web server, run whatweb
5. If vulnerable tech, exploit it
6. Once in, privilege escalate
7. Document findings"

This THINKING happens before tool execution.
The agent is autonomous because it REASONS through the approach.
```

---

## Summary: DrogonClaw is...

✅ **OpenClaw-style** autonomous agent  
✅ **Pentesting specialist** trained on security tools  
✅ **Fully autonomous** (extended thinking)  
✅ **Kali-native** (all 300+ tools available)  
✅ **Multi-channel** (CLI + Telegram)  
✅ **Persistent** (remembers findings across sessions)  
✅ **Extensible** (skill registry for community)  
✅ **Learning** (improves within session based on findings)  

**Not**: A simple tool wrapper or command executor  
**Is**: A thinking agent that autonomously conducts pentests

This is what you asked for: OpenClaw applied to pentesting with full autonomy.
