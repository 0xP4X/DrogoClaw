# 🐉 DrogonClaw - OpenClaw-Style Autonomous Pentesting Agent
## Complete Architecture & Implementation Plan

**Status:** Architecture Research Complete + Vision Defined  
**Based On:** OpenClaw (proven personal AI architecture)  
**Domain:** Autonomous Pentesting on Kali Linux  
**Delivery:** CLI + Telegram Bot  

---

## Core Concept (What Makes It Different)

### Traditional Pentesting Tools
```
Pentester runs:
  nmap → reads output → runs dirb → reads output → runs sqlmap → reads output...
(User drives every decision)
```

### DrogonClaw (Autonomous Agent)
```
User: "Pentest example.com"
Agent:
  [THINKS] "What should I do?"
  [DECIDES] "Run nmap first"
  → Analyzes nmap output
  [THINKS] "What next?"
  [DECIDES] "Run whatweb"
  → Analyzes whatweb output
  [THINKS] "Found WordPress"
  [DECIDES] "Run wpscan"
  → Found RCE vulnerability
  [THINKS] "Should exploit"
  [DECIDES] "Use metasploit"
  → Got shell!
(Agent drives every decision using extended thinking)
```

---

## Architecture (OpenClaw Pattern)

```
                         🐉 DrogonClaw Gateway
                    (Node.js on Kali Linux port 18789)
                                   |
                ┌──────────────────┼──────────────────┐
                ↓                  ↓                  ↓
            [CLI Channel]   [Telegram Channel]  [WebSocket API]
           (Interactive)      (Remote Bot)      (Future Web UI)
                |                 |                  |
                └──────────────────┼──────────────────┘
                                   ↓
                         ┌─ Session Manager
                         ├─ Agent Loop (think→decide→act)
                         ├─ Skill Registry (50+ skills)
                         ├─ Kali Tool Executor (bash/process)
                         └─ SQLite Persistence
                                   ↓
                    🔧 All Kali Tools (300+)
                    ├─ nmap, masscan, dnsenum
                    ├─ nikto, dirb, whatweb
                    ├─ sqlmap, xsser, nuclei
                    ├─ metasploit, hashcat, john
                    ├─ linprivesc, privilege escalation
                    └─ Everything else on Kali
```

---

## 5 Core Components

### 1. GATEWAY (Control Plane)
- Single Node.js process
- Listens on localhost:18789
- Routes messages from CLI/Telegram to agent
- Manages sessions (load/save)
- Executes tools and streams results
- Enforces timeouts and safety

```typescript
class DrogonClawGateway {
  sessionManager: SessionManager;
  agentLoop: AgentLoop;
  skillRegistry: SkillRegistry;
  kaliExecutor: KaliToolExecutor;
  channels: Map<string, Channel>;
  
  // Main entry point
  async handleUserMessage(msg: string, source: 'cli'|'telegram') {
    const session = await sessionManager.loadOrCreate();
    const response = await agentLoop.run(msg, session);
    await sessionManager.save(session);
    channels.get(source).send(response);
  }
}
```

### 2. AGENT LOOP (Autonomous Decision-Making)
The "agentic loop" - inspired by OpenClaw's proven pattern:

```
[1] INTAKE: User message
[2] CONTEXT: Load session (what we know so far)
[3] SYSTEM PROMPT: Build full context
    ├─ Agent role: "You are a pentesting specialist"
    ├─ Available skills: List of 50+ tools
    ├─ Target info: What we've discovered
    ├─ Findings: Vulnerabilities so far
    └─ Extended thinking: "Reason about approach"
[4] MODEL CALL: Claude with extended thinking + tool_use
    ├─ Think through next steps
    ├─ Decide which tools to use
    └─ Return action plan
[5] TOOL EXECUTION: Execute returned tools
    ├─ Run in subprocess with timeout
    ├─ Capture output
    ├─ Parse for findings
    └─ Stream progress to user
[6] ANALYSIS: Agent analyzes tool output
    ├─ Extracts findings
    ├─ Updates findings DB
    ├─ Decides: continue? pivot? exploit?
    └─ Returns to step 1 if continuing
[7] PERSISTENCE: Save session state
    ├─ Transcript (full conversation)
    ├─ Findings (structured vulns)
    ├─ Tools used (audit trail)
    └─ Session metadata
```

### 3. SKILL REGISTRY (Tool Instructions)
50+ YAML-defined skills teach agent HOW to use Kali tools:

```
skills/
├── reconnaissance/
│   ├── dns-enum/SKILL.md              # dnsenum, dig, whois
│   ├── port-scanning/SKILL.md         # nmap -sV, masscan
│   ├── web-probing/SKILL.md           # curl, wget, httpx
│   └── subdomain-discovery/SKILL.md   # subfinder, amass, fierce
│
├── enumeration/
│   ├── web-tech-detection/SKILL.md    # whatweb, wappalyzer
│   ├── directory-brute/SKILL.md       # dirb, dirsearch
│   ├── service-enumeration/SKILL.md   # nmap scripts, smbmap
│   └── api-discovery/SKILL.md         # arjun, paramspider
│
├── exploitation/
│   ├── sql-injection/SKILL.md         # sqlmap command generation
│   ├── xss-detection/SKILL.md         # xsser, dalfox
│   ├── rce-exploitation/SKILL.md      # metasploit, msfvenom
│   ├── authentication-bypass/SKILL.md # hydra, medusa, hashcat
│   └── web-shell-upload/SKILL.md      # file upload exploitation
│
├── post-exploitation/
│   ├── privilege-escalation/SKILL.md  # linprivesc.sh, winprivesc
│   ├── persistence/SKILL.md           # cron, backdoors, sudoers
│   ├── lateral-movement/SKILL.md      # psexec, ssh pivoting
│   └── data-exfiltration/SKILL.md     # scp, http upload
│
└── reporting/
    ├── finding-generator/SKILL.md     # Format vulnerabilities
    ├── cvss-calculator/SKILL.md       # Score findings
    └── report-html/SKILL.md           # Generate HTML report
```

Each skill: YAML frontmatter + markdown instructions for the agent

### 4. CHANNELS (User Interfaces)

#### CLI Channel (Interactive Terminal)
```bash
$ drogonclaw pentest --target example.com --strategy aggressive

🐉 DrogonClaw Pentesting Agent
Target: example.com
Strategy: Aggressive (exploit-focused)

[→] Reconnaissance Phase
    └─ Running: nmap -sV, dnsenum...
       ✓ Found 8 subdomains
       ✓ Found 3 open ports (22, 80, 443)
       ✓ Services: SSH, Apache, Nginx

[→] Enumeration Phase
    └─ Running: whatweb, dirb...
       ✓ WordPress 5.8
       ✓ 15 plugins (3 with known vulns)

[→] Exploitation Phase
    └─ Running: wpscan, sqlmap...
       🔴 CRITICAL: Plugin RCE Found!
       🟠 HIGH: SQL Injection in search
       🟡 MEDIUM: Outdated jQuery

[→] Post-Exploitation Phase
    └─ Running: privilege escalation...
       ✓ www-data shell obtained
       ✓ Escalated to root!

[✓] Session Complete
    Vulnerabilities Found: 5
    Severity: 2 CRITICAL, 1 HIGH, 2 MEDIUM
    
    Full session saved to:
    ~/.drogonclaw/sessions/example.com_2026_05_23.json
```

#### Telegram Channel (Remote Bot)
```
User: /pentest example.com
Bot: 🐉 Starting autonomous pentest on example.com

    [Reconnaissance]
    ✓ DNS enumeration: 4 subdomains found
    ✓ Port scan: 22, 80, 443 open
    ✓ Service detection: SSH, Apache, Nginx
    
    [Enumeration]
    ✓ Web tech: WordPress 5.8
    ✓ Plugins: 12 total, 3 with known vulns
    ✓ Directory brute: 23 dirs/files found
    
    [Exploitation]
    🔴 Plugin RCE - CRITICAL
    ✓ Shell obtained as www-data
    
    [Post-Exploitation]
    ✓ Privilege escalation: www-data → root
    
    📊 Final Report:
    5 vulnerabilities found
    2 CRITICAL, 1 HIGH, 2 MEDIUM
    
    Session: example.com_20260523_143922
```

### 5. PERSISTENCE (SQLite Memory)

```
Database: ~/.drogonclaw/sessions.db

sessions table:
├─ session_id (UUID)
├─ target (domain/IP)
├─ strategy (aggressive|thorough|stealthy)
├─ status (pending|running|paused|complete)
├─ created_at, updated_at
├─ transcript (full conversation JSON)
├─ findings (discovered vulnerabilities)
├─ tools_used (list of executed tools)
└─ metadata (custom tags, client info)

findings table:
├─ finding_id (UUID)
├─ session_id (foreign key)
├─ type (sql-injection|xss|rce|privesc)
├─ severity (critical|high|medium|low)
├─ description
├─ proof (command run, output, screenshot)
├─ cvss_score
└─ remediation

Sessions can be:
✓ Paused and resumed
✓ Reviewed for findings
✓ Exported for reporting
✓ Searched for previous discoveries
```

---

## What Makes It Autonomous

### Extended Thinking
Agent uses thinking tokens at critical decision points:

```
Before each major step:
  "What do I know about the target?"
  "What gaps exist in my knowledge?"
  "Which tools should I use next?"
  "Why those tools?"
  "What could go wrong?"
  "How do I handle errors?"

During tool execution:
  "What does this output mean?"
  "Is this a real vulnerability?"
  "How critical is it?"
  "Should I try to exploit it?"
  "What's my next step?"

Before exploitation:
  "What's the weakest point?"
  "Which exploit has highest success rate?"
  "What's the blast radius?"
  "Do I have enough access?"
```

This THINKING is what makes it truly autonomous, not just a script runner.

### Adaptive Loop
Agent adapts based on findings:

```
Scenario: Found WordPress

Agent thinks:
  "WordPress found on port 80"
  "This narrows my toolkit significantly"
  "I should check for:
    - Plugin vulnerabilities
    - Theme vulnerabilities
    - Outdated WordPress core
    - Default credentials"
  
Agent selects tools:
  wpscan (WordPress vulnerability scanner)
  hydra (brute force default creds)
  
Agent analyzes results:
  "Plugin XYZ has known RCE"
  "This is exploitable!"
  
Agent decides:
  "Exploit this plugin"
```

The agent ADAPTS its approach based on what it discovers.

---

## Implementation Timeline

### Phase 1: Foundation (Week 1-2)
- Express.js gateway
- Session manager (SQLite)
- Kali tool executor (bash subprocess)
- CLI channel (readline prompts)
- Basic agent loop (no extended thinking yet)

### Phase 2: Autonomy (Week 3-4)
- Extended thinking integration
- Skill registry (20+ skills)
- Smarter decision-making
- Finding aggregation
- Telegram bot channel

### Phase 3: Polish (Week 5-6)
- 50+ skills
- Auto-reporting (HTML/PDF)
- Plugin architecture
- Performance optimization
- Security hardening

### Phase 4: Scale (Week 7+)
- Multi-target support
- Team collaboration
- Web dashboard
- Advanced analytics
- Custom skill marketplace

---

## How It Uses Kali Tools

DrogonClaw doesn't "integrate" Kali tools like a plugin system.

Instead: **It executes them like any user would:**

```typescript
class KaliToolExecutor {
  async execute(tool: string, args: string[]): Promise<string> {
    // Simple subprocess execution
    const proc = spawn(tool, args, {
      cwd: process.cwd(),
      env: process.env,
      timeout: 30000
    });
    
    let output = '';
    proc.stdout.on('data', (data) => {
      output += data.toString();
      // Stream to user in real-time
      this.emit('output', data.toString());
    });
    
    return new Promise((resolve, reject) => {
      proc.on('close', (code) => {
        resolve(output);
      });
      proc.on('error', reject);
    });
  }
}

// Agent calls it like:
const result = await executor.execute('nmap', ['-sV', '192.168.1.1']);
const result = await executor.execute('sqlmap', ['-u', 'http://example.com', '--dbs']);
const result = await executor.execute('metasploit', [...exploit_args]);
```

All 300+ Kali tools are available. The agent chooses which to use.

---

## Comparison: DrogonClaw vs Current Approach

| Aspect | Current Plan | New DrogonClaw |
|--------|--------|---------|
| **Architecture** | Hexstrike MCP wrapper | OpenClaw gateway pattern |
| **Decision-Making** | Tool executor | Autonomous agent with thinking |
| **Tool Access** | HexStrike API (69 tools) | Native Kali (300+ tools) |
| **Intelligence** | Routing | Extended thinking + adaptation |
| **Session State** | Per-tool | Persistent across entire pentest |
| **Autonomy Level** | Command executor | True autonomous agent |
| **Learning** | No | Within-session adaptation |
| **Scaling** | Limited | Multi-session, multi-target |
| **User Experience** | "Run this tool" | "Pentest this target" |

---

## Getting Started (Next Steps)

1. **Update the Plan**: Use new OpenClaw-inspired architecture
2. **Create new project structure**: Gateway-centric, not tool-centric
3. **Phase 1 implementation**: Gateway + basic agent loop
4. **Integration testing**: Verify Kali tool execution
5. **Skill development**: Create 20+ skills in YAML
6. **Autonomous testing**: Test true autonomy with extended thinking

---

## Success Criteria

✅ Agent runs autonomously (no prompting between steps)  
✅ Discovers vulnerabilities without user intervention  
✅ Makes intelligent decisions about which tools to use  
✅ Persists findings across sessions  
✅ Works via CLI and Telegram  
✅ Handles complex multi-phase pentests  
✅ Generates professional reports  
✅ Extensible via skill system  

---

## Summary

**DrogonClaw is not:**
- A tool aggregator
- A script executor
- A command wrapper
- An API gateway

**DrogonClaw IS:**
- A true autonomous pentesting agent
- Inspired by OpenClaw's proven architecture
- Using extended thinking for intelligence
- With full access to Kali's toolset
- Accessible via CLI and Telegram
- Capable of real, autonomous pentesting

This is what OpenClaw is to personal assistance,
DrogonClaw is to pentesting.

Ready to implement? 🐉
