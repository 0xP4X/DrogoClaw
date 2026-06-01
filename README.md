# DrogonClaw 🐉🔥

<div align="center">
  <img src="assets/logo.png" alt="DrogonClaw Logo" width="300" />
</div>

```text
      /\      /\      /\
     /  \    /  \    /  \
    |    |  |    |  |    |
    |    |  |    |  |    |
    |  /\|  |/\  |  |/\  |
     \ \  \  /  /   / / /
      \ \  \/  /   / / /
       \ \    /   / / /
        \      \ / / /
         \      / / /
          |    | | |

    ____                               ________              
   / __ \_________  ____ _____  ____  / ____/ /___ __      __
  / / / / ___/ __ \/ __ `/ __ \/ __ \/ /   / / __ `/ | /| / /
 / /_/ / /  / /_/ / /_/ / /_/ / / / / /___/ / /_/ /| |/ |/ / 
/_____/_/   \____/\__, /\____/_/ /_/\____/_/\__,_/ |__/|__/  
                 /____/                                      
```

> **AI-Driven Offensive Security Framework**
> *Developed by 0day (0xP4X)*

DrogonClaw is a next-generation cyber operations platform. Rather than acting as a simple wrapper for Kali tools, DrogonClaw operates as a **Command-and-Control (C2) Brain**. It understands objectives, plans attack workflows, adapts to new discoveries, and orchestrates a swarm of specialized autonomous agents through a unified intelligence core.

DrogonClaw focuses on **high-confidence autonomous workflows**, explainable findings, and reproducible evidence, avoiding the hallucinations common in early AI security tools.

## 🏛️ Architectural Pillars

The platform revolves around three major pillars:

### 1. The Orchestration Core
- **Mission Planner**: Breaks down objectives, reasons about paths, and delegates to specialized agents.
- **Intelligence Graph**: A persistent, graph-based memory system that maps out discovered assets, vulnerabilities, and context across engagements.
- **Evidence Validator**: An AI validation layer that demands reproducible evidence, scoring confidence from 0-100% and rejecting hallucinations.

### 2. The Skill Ecosystem
A modular plugin architecture allowing seamless integration of:
- OSINT modules
- Network reconnaissance scanners
- Browser automation packs
- Exploit validators

### 3. Autonomous Execution Layer
DrogonClaw isolates operational risk through:
- **Sandboxed Tool Execution**: Running command-line tools (Nmap, Metasploit, etc.) in isolated Docker environments.
- **Safety Monitors**: Enforcing rate limits, scope boundaries, and timeout constraints.

## 🚀 Quick Start & Setup Guide

DrogonClaw operates through multiple interconnected modules. You can run it locally from source, or install it globally as a standalone CLI tool.

### 1. Global Installation (Recommended)

DrogonClaw is published on NPM and can be installed globally:

```bash
npm install -g drogonclaw
```

Once installed, simply run `drogonclaw` from anywhere on your system to launch the setup wizard and enter the AI.

### 2. Local Installation (For Developers)

```bash
git clone https://github.com/0xP4X/DrogoClaw.git
cd drogonclaw
npm install
npm run build
npm run cli
```

### 3. Initialization Wizard

Upon the first launch of `drogonclaw`, the **DrogonClaw Configuration Wizard** will guide you through setting up your neural pathways:
- You will be prompted to select an AI Provider (OpenAI, Anthropic, OpenRouter, or local Ollama).
- You will securely enter your API keys.
- You can optionally configure a **Telegram Gateway** for remote mobile C2 operations.

If you ever need to reconfigure your setup, run `drogonclaw setup` or type `/setup` inside the interactive terminal.

### 4. Interactive Terminal & Dynamic Execution

Inside the `drogon>` prompt, you can converse with the AI naturally or use specific slash commands:
* `/skills` - List all loaded penetration testing modules
* `/setup` - Relaunch the configuration wizard
* `/clear` - Wipe the terminal screen

**Graceful Action Abortion:** If DrogonClaw is running a long scan or executing an exploit and you want to steer it in a different direction, simply press `Ctrl+C`. This will instantly sever the active thread, halt all sandboxed executions, and drop you back to the prompt, preserving the session memory so you can inject new instructions.

#### 📱 Telegram Gateway
Allows you to text instructions to your agent from your phone:
```bash
npm run gateway
```
*Security Note: You must provide your `TELEGRAM_CHAT_ID` during initialization to whitelist your account, otherwise the agent will reject all commands.*

## 🛠️ Modularity & Swarm Intelligence

DrogonClaw is designed to scale into collaborative agent swarms. You can inject new specialized agents (e.g., a "Web Fuzzer Agent" or an "Active Directory Hound") without modifying the core orchestrator.

## 👨‍💻 Author

**0day (0xP4X)**
- GitHub: [@0xP4X](https://github.com/0xP4X)

## ⚠️ Disclaimer

DrogonClaw is designed for **authorized security testing only**. Always ensure you have explicit permission before testing any system. Unauthorized access to computer systems is illegal.

## 📄 License

MIT
