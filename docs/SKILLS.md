# DrogonClaw - Skill Development Guide

Learn how to create custom pentesting skills for DrogonClaw.

## Skill Anatomy

Skills are YAML files that define reusable pentesting procedures.

### Minimal Skill

```yaml
# skills/my-skill.yaml
id: my-skill
name: My Custom Skill
description: Describe what this skill does
category: recon
priority: 50
author: Your Name
version: 1.0.0

tools:
  - curl
  - dig

preconditions:
  - target_is_valid

instructions: |
  1. Perform DNS lookup
  2. Check HTTP headers
  3. Extract metadata

expected_outputs:
  - DNS records
  - Server information
```

### Complete Skill Example

```yaml
# skills/web-reconnaissance.yaml
id: web-recon
name: Web Server Reconnaissance
description: Comprehensive web server discovery and fingerprinting
category: recon
priority: 80
author: Security Team
version: 1.0.0

tools:
  - curl
  - dig
  - nmap

preconditions:
  - target_is_domain_or_ip
  - port_80_or_443_open

instructions: |
  ## Phase 1: DNS Resolution
  1. Resolve target domain to IP
  2. Perform reverse DNS lookup
  3. Check for CNAME records
  
  ## Phase 2: HTTP Headers
  4. GET request to identify server
  5. Extract X-Powered-By header
  6. Check for security headers
  
  ## Phase 3: SSL/TLS Analysis
  7. Check certificate validity
  8. Extract certificate details
  9. Identify weak ciphers
  
  ## Phase 4: Technology Stack
  10. Identify web framework
  11. Detect CMS platform
  12. Find JavaScript frameworks

expected_outputs:
  - Server IP address
  - Reverse DNS name
  - DNS records (A, AAAA, CNAME, MX)
  - Server type and version
  - Security headers
  - SSL certificate details
  - Technology fingerprint

remediation: |
  1. Update web server to latest version
  2. Remove version info from headers
  3. Implement security headers
  4. Use modern TLS versions
  5. Keep CMS updated
```

## Skill Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique skill identifier (lowercase, no spaces) |
| `name` | string | Human-readable skill name |
| `description` | string | What the skill does |
| `category` | string | Skill category: recon, enumeration, exploitation, reporting |
| `author` | string | Author name |
| `version` | string | Semantic version (1.0.0) |
| `tools` | array | List of required tools |
| `instructions` | string | Step-by-step instructions |
| `expected_outputs` | array | Types of findings to expect |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `priority` | number | Execution priority (0-100, default 50) |
| `preconditions` | array | Requirements before execution |
| `remediation` | string | How to fix the issues found |
| `tags` | array | Skill tags for filtering |
| `timeout` | number | Execution timeout in seconds |
| `parallelizable` | boolean | Can run in parallel with other skills |

## Skill Categories

### Recon
Discovery and passive reconnaissance

Examples:
- DNS enumeration
- WHOIS lookups
- Public database searches
- Search engine queries

### Enumeration
Active service discovery

Examples:
- Port scanning
- Service enumeration
- Version detection
- Technology fingerprinting

### Exploitation
Active vulnerability testing

Examples:
- Vulnerability scanning
- Exploitation attempts
- Payload delivery
- Post-exploitation

### Reporting
Analysis and reporting

Examples:
- Finding aggregation
- CVSS scoring
- Report generation
- Recommendation synthesis

## Writing Instructions

Instructions should be clear, numbered steps:

```yaml
instructions: |
  1. Initial reconnaissance
     - Resolve target to IP
     - Perform whois lookup
  2. Identify services
     - Scan common ports
     - Enumerate services
  3. Analyze findings
     - Check for known vulnerabilities
     - Assess risk level
```

## Expected Outputs

List types of findings this skill may produce:

```yaml
expected_outputs:
  - open_port
  - service_detected
  - version_detected
  - vulnerability_found
  - weak_configuration
```

## Preconditions

Define when this skill can be executed:

```yaml
preconditions:
  - target_is_ip          # Target must be IP address
  - port_443_open        # Port 443 must be open
  - https_enabled        # HTTPS must be available
  - not_localhost        # Don't run against localhost
```

Available preconditions:
- `target_is_ip`
- `target_is_domain`
- `target_is_ip_or_domain`
- `port_80_open`
- `port_443_open`
- `http_enabled`
- `https_enabled`
- `ssh_enabled`
- `ftp_enabled`
- Any custom precondition

## Tools Reference

Available tools (customize with `TOOL_WHITELIST`):

```
Standard:
  - nmap         # Network scanning
  - curl         # HTTP requests
  - dig          # DNS queries
  - whois        # Domain information
  - traceroute   # Network path tracing
  - netstat      # Network statistics
  - ping         # Host reachability
  - ifconfig     # Network configuration
  - ipconfig     # Windows network config
  
Custom:
  - Any executable in PATH
  - Custom scripts
```

## Creating a Skill File

1. Create YAML file in `skills/` directory:

```bash
mkdir -p skills
vim skills/my-skill.yaml
```

2. Add skill definition

3. Load in system:

```bash
npm run setup
```

4. Test skill:

```bash
npm start
# Select your skill during pentesting
```

## Testing Skills

### Manual Testing

```bash
# Start CLI
npm start

# Select your skill and target
# Verify output and findings
```

### Automated Testing

```bash
# Create test in tests/unit/skills.test.ts
npm test
```

## Skill Examples

### Example 1: DNS Enumeration

```yaml
id: dns-enum
name: DNS Enumeration
description: Enumerate DNS records for domain
category: recon
priority: 90
author: DrogonClaw Team
version: 1.0.0
tools:
  - dig
preconditions:
  - target_is_domain
instructions: |
  1. Query A records: dig @8.8.8.8 example.com
  2. Query MX records: dig @8.8.8.8 example.com MX
  3. Query NS records: dig @8.8.8.8 example.com NS
  4. Query TXT records: dig @8.8.8.8 example.com TXT
expected_outputs:
  - dns_a_records
  - dns_mx_records
  - dns_ns_records
  - dns_txt_records
```

### Example 2: Port Scanning

```yaml
id: port-scan
name: Port Scanning
description: Scan common ports
category: enumeration
priority: 85
author: DrogonClaw Team
version: 1.0.0
tools:
  - nmap
preconditions:
  - target_is_ip_or_domain
instructions: |
  1. Scan top 100 ports: nmap -F target
  2. Identify open ports
  3. Determine services
  4. Check versions
expected_outputs:
  - open_port
  - closed_port
  - service_detected
```

### Example 3: Web Server Detection

```yaml
id: web-detect
name: Web Server Detection
description: Identify and fingerprint web servers
category: enumeration
priority: 75
author: DrogonClaw Team
version: 1.0.0
tools:
  - curl
  - dig
preconditions:
  - port_80_open or port_443_open
instructions: |
  1. Request HTTP headers: curl -I http://target
  2. Extract Server header
  3. Check for X-Powered-By
  4. Identify framework
  5. Detect CMS
expected_outputs:
  - web_server_detected
  - cms_detected
  - framework_detected
```

## Publishing Skills

To share your skill with the community:

1. Create `.yaml` file in `skills/` directory
2. Add documentation
3. Test thoroughly
4. Submit pull request to DrogonClaw repository
5. Share on GitHub Discussions

## Next Steps

- Create your first skill
- Test with `npm start`
- Contribute to DrogonClaw
- See `DEVELOPMENT.md` for code examples

## Support

- Questions: GitHub Discussions
- Issues: GitHub Issues
- Examples: Browse existing skills in `skills/` directory

