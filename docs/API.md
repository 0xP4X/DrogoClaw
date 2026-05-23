# DrogonClaw - API Reference

Complete HTTP and WebSocket API documentation.

## Base URL

```
http://localhost:18789
```

## Authentication

Currently no authentication required. Configure firewall rules for production.

## Response Format

All responses follow this format:

```json
{
  "success": true,
  "data": { /* response data */ },
  "error": null,
  "timestamp": 1234567890
}
```

Error responses:

```json
{
  "success": false,
  "data": null,
  "error": "Error message",
  "timestamp": 1234567890
}
```

## Endpoints

### System

#### GET /health

Health check endpoint.

```bash
curl http://localhost:18789/health
```

Response:
```json
{
  "status": "ok",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Sessions

#### GET /api/sessions

List all sessions.

```bash
curl http://localhost:18789/api/sessions
```

Response:
```json
{
  "success": true,
  "data": {
    "sessions": [
      {
        "id": "sess-abc123",
        "target": "example.com",
        "strategy": "thorough",
        "status": "completed",
        "startTime": 1705315800000,
        "endTime": 1705319400000,
        "createdAt": 1705315800000,
        "updatedAt": 1705319400000,
        "findings": [ /* ... */ ]
      }
    ]
  },
  "timestamp": 1705319400000
}
```

#### POST /api/sessions

Create a new session.

```bash
curl -X POST http://localhost:18789/api/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "target": "example.com",
    "strategy": "thorough"
  }'
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "sess-abc123",
    "target": "example.com",
    "strategy": "thorough",
    "status": "pending",
    "startTime": 1705315800000,
    "createdAt": 1705315800000,
    "updatedAt": 1705315800000,
    "findings": []
  },
  "timestamp": 1705315800000
}
```

#### GET /api/sessions/:sessionId

Get session details.

```bash
curl http://localhost:18789/api/sessions/sess-abc123
```

#### PUT /api/sessions/:sessionId

Update session.

```bash
curl -X PUT http://localhost:18789/api/sessions/sess-abc123 \
  -H "Content-Type: application/json" \
  -d '{
    "status": "paused",
    "notes": "Paused for investigation"
  }'
```

#### DELETE /api/sessions/:sessionId

Delete session and all its findings.

```bash
curl -X DELETE http://localhost:18789/api/sessions/sess-abc123
```

### Findings

#### GET /api/findings

List all findings (across all sessions).

```bash
curl http://localhost:18789/api/findings
```

Response:
```json
{
  "success": true,
  "data": {
    "findings": [
      {
        "id": "find-123",
        "sessionId": "sess-abc123",
        "title": "Open SSH Port",
        "description": "SSH service detected on port 22",
        "severity": "low",
        "type": "port_open",
        "target": "192.168.1.1",
        "evidence": ["Port 22 is open"],
        "remediation": "Restrict SSH access to trusted IPs",
        "references": ["https://..."],
        "discoveredAt": 1705315800000,
        "toolUsed": "nmap"
      }
    ]
  },
  "timestamp": 1705319400000
}
```

#### GET /api/findings?sessionId=sess-abc123

List findings for specific session.

```bash
curl http://localhost:18789/api/findings?sessionId=sess-abc123
```

#### GET /api/findings?severity=critical

Filter findings by severity.

```bash
curl http://localhost:18789/api/findings?severity=critical
```

### Tools

#### POST /api/tools/execute

Execute a security tool.

```bash
curl -X POST http://localhost:18789/api/tools/execute \
  -H "Content-Type: application/json" \
  -d '{
    "toolName": "nmap",
    "args": ["-p", "22,80,443", "192.168.1.1"]
  }'
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "exec-123",
    "toolName": "nmap",
    "args": ["-p", "22,80,443", "192.168.1.1"],
    "status": "completed",
    "output": "...",
    "startTime": 1705315800000,
    "endTime": 1705315805000,
    "duration": 5000
  },
  "timestamp": 1705319400000
}
```

### Agents

#### GET /api/agents

List active agents.

```bash
curl http://localhost:18789/api/agents
```

#### POST /api/agents/:agentId/pause

Pause a running agent.

```bash
curl -X POST http://localhost:18789/api/agents/agent-123/pause
```

#### POST /api/agents/:agentId/resume

Resume paused agent.

```bash
curl -X POST http://localhost:18789/api/agents/agent-123/resume
```

#### POST /api/agents/:agentId/stop

Stop agent.

```bash
curl -X POST http://localhost:18789/api/agents/agent-123/stop
```

## WebSocket Events

WebSocket connection: `ws://localhost:18789/ws`

Receive real-time events:

### session_update
```json
{
  "type": "session_update",
  "sessionId": "sess-abc123",
  "payload": {
    "status": "active",
    "findingCount": 5
  },
  "timestamp": 1705315800000
}
```

### finding_discovered
```json
{
  "type": "finding_discovered",
  "sessionId": "sess-abc123",
  "payload": {
    "id": "find-123",
    "title": "Open Port",
    "severity": "high"
  },
  "timestamp": 1705315800000
}
```

### tool_executed
```json
{
  "type": "tool_executed",
  "sessionId": "sess-abc123",
  "payload": {
    "toolName": "nmap",
    "duration": 5000,
    "output": "..."
  },
  "timestamp": 1705315800000
}
```

### agent_thinking
```json
{
  "type": "agent_thinking",
  "sessionId": "sess-abc123",
  "payload": {
    "thoughts": "Analyzing findings...",
    "nextAction": "Execute nmap scan"
  },
  "timestamp": 1705315800000
}
```

### agent_complete
```json
{
  "type": "agent_complete",
  "sessionId": "sess-abc123",
  "payload": {
    "totalFindings": 12,
    "duration": 300000,
    "summary": "Completed thorough assessment"
  },
  "timestamp": 1705315800000
}
```

## Error Responses

### Invalid Request
```json
{
  "success": false,
  "data": null,
  "error": "Invalid request body",
  "timestamp": 1705319400000
}
```

### Not Found
```json
{
  "success": false,
  "data": null,
  "error": "Session not found",
  "timestamp": 1705319400000
}
```

### Server Error
```json
{
  "success": false,
  "data": null,
  "error": "Internal server error",
  "timestamp": 1705319400000
}
```

## Rate Limiting

Currently no rate limiting. Implement based on your deployment needs.

## CORS

WebSocket and HTTP endpoints support CORS from localhost:3000 for development.

## Examples

### JavaScript Client

```javascript
// HTTP
const response = await fetch('http://localhost:18789/api/sessions', {
  method: 'GET',
  headers: { 'Content-Type': 'application/json' }
});
const data = await response.json();
console.log(data.data.sessions);

// WebSocket
const ws = new WebSocket('ws://localhost:18789/ws');
ws.addEventListener('message', (event) => {
  const msg = JSON.parse(event.data);
  console.log(`${msg.type}: ${JSON.stringify(msg.payload)}`);
});
```

### Python Client

```python
import requests
import json
import asyncio
import websockets

# HTTP
response = requests.get('http://localhost:18789/api/sessions')
data = response.json()
print(data['data']['sessions'])

# WebSocket
async def listen():
    async with websockets.connect('ws://localhost:18789/ws') as ws:
        async for message in ws:
            msg = json.loads(message)
            print(f"{msg['type']}: {msg['payload']}")

asyncio.run(listen())
```

### cURL

See examples throughout this guide.

## Support

For issues or questions, see GitHub issues or documentation.

