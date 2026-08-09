# DrogonClaw Enterprise Platform

DrogonClaw is an autonomous Red Team and Blue Team platform designed to deliver continuous security testing, vulnerability management, and threat hunting across enterprise environments.

## Architecture

The DrogonClaw engine operates in two primary modes:

### Red Team (Offensive Operations)
*   **ReAct Loop Engine**: Uses LLMs to dynamically plan, execute, and adapt to the environment.
*   **Memory Graph**: Local JSON-backed graph memory storing targets, assets, ports, services, vulnerabilities, credentials, flags, and their relationships as the agent records verified findings.
*   **Docker Sandbox Execution**: Tools and exploits run through the DrogonClaw sandbox runtime, using a persistent Kali container when sandbox mode is enabled or the host shell when native mode is selected.
*   **Payload Generation**: Generates Fully Undetectable (FUD) payloads using polymorphic crypters.
*   **Enterprise Integration**: Findings are mapped to the MITRE ATT&CK framework and scored using CVSS v3.1.

### Blue Team (Defensive Operations)
*   **CIS Benchmark Scanner**: Deterministic shell checks to validate OS hardening on Linux/Windows.
*   **Threat Hunting**: YARA rule matching and IOC scanning (hashes, malicious IPs, persistence mechanisms).
*   **Compliance Mapping**: Security findings mapped directly to PCI-DSS, SOC 2, and HIPAA.
*   **Incident Response**: Automated playbooks and procedural guidance for active intrusions (e.g., Ransomware).
*   **Vulnerability Lifecycle Management**: Asset-based CVSS patch prioritization.

## Getting Started

DrogonClaw relies on a Go-based architecture and Docker for sandbox execution.

### Requirements
*   Go 1.26+
*   Docker (Daemon must be running)

### Installation
1. Compile the binary:
   ```bash
   make build
   ```
2. (Optional) Start the supporting services (e.g. a local Ollama instance):
   ```bash
   make docker-compose
   ```
3. Run the interactive setup wizard, which writes your configuration to
   `~/.drogonclaw/config.json`:
   ```bash
   ./drogonclaw setup
   ```

### Local API

The REST API (`internal/api/server.go`) is an optional, local control-plane
component. It is not started by the CLI or the Docker image, so it must be
wired up separately (e.g. by your own process) if you need it. It authenticates
with a static Bearer token from `DROGONCLAW_API_KEY` and does not implement JWT
or role-based access control. Use the explicit TLS server entrypoint
(`StartTLSServerAt`) and a deliberate host binding for any non-loopback
deployment.
*   **Authentication**: Static Bearer token via `DROGONCLAW_API_KEY` environment variable.
*   **Limitations**: No JWT, no RBAC, no TLS by default. The API binds to loopback only unless `StartServerAt` or `StartTLSServerAt` is used with an explicit host.

### TUI Operational Commands
*   `/health` runs real toolkit diagnostics against the active sandbox runtime and reports installed/missing tooling.
*   `/status` prints policy state plus memory graph entity and relationship counts by type.
*   `/skills` summarizes loaded executable modules by category.
*   `/skills <term>` searches module names, descriptions, parameters, and inferred categories.
*   `/skills <exact_name>` shows required parameters and the execution backend for that module.

## Legal Disclaimer
DrogonClaw is built for authorized security testing only. Do not deploy offensive capabilities against networks without explicit written consent.
