# DrogonClaw 🐉🔥

> **AI-Driven Offensive Security Framework**

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

DrogonClaw operates through multiple interconnected modules. You can run it entirely from the CLI, or spin up the Web Dashboard and Telegram Gateway for a full Command & Control (C2) experience.

### 1. Installation

```bash
git clone https://github.com/your-org/drogonclaw.git
cd drogonclaw
npm install
```

### 2. Initialization & Diagnostics

Before running any web servers, you must configure the core intelligence engine:

```bash
npm run cli
```

This will launch the **DrogonClaw Initialization Wizard**.
- You will be prompted to enter your AI Provider API Keys (OpenAI, Anthropic, Gemini, or Ollama).
- You will optionally be asked for a **Telegram Bot Token** and your **Telegram Chat ID**.

Once inside the terminal, type `/health`. The agent will perform a system diagnostic and optionally install any missing pentesting tools (Nmap, Gobuster, Go, Metasploit) on your machine.

### 3. Launching the OS Components

DrogonClaw can be run in several modes depending on your operational needs. In separate terminal windows, run the following:

#### 💻 Interactive CLI
The core interactive terminal session:
```bash
npm run cli
```
*Useful commands: `/swarm <task>`, `/stealth on`, `/report`, `/skills`*

#### 📱 Telegram Gateway
Allows you to text instructions to your agent from your phone:
```bash
npm run gateway
```
*Security Note: You must provide your `TELEGRAM_CHAT_ID` during initialization to whitelist your account, otherwise the agent will reject all commands.*

#### 🌐 C2 Web Dashboard
A premium Cyberpunk UI to visualize the attack tree and memory graph:
```bash
# Terminal 1: Start the backend WebSocket API
npm run api

# Terminal 2: Start the Next.js Frontend
npm run web
```
Then navigate to `http://localhost:3000` in your browser.

## 🛠️ Modularity & Swarm Intelligence

DrogonClaw is designed to scale into collaborative agent swarms. You can inject new specialized agents (e.g., a "Web Fuzzer Agent" or an "Active Directory Hound") without modifying the core orchestrator.

## ⚠️ Disclaimer

DrogonClaw is designed for **authorized security testing only**. Always ensure you have explicit permission before testing any system. Unauthorized access to computer systems is illegal.

## 📄 License

MIT
