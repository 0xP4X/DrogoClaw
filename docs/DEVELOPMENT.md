# DrogonClaw - Development Guide

For contributors and developers extending DrogonClaw.

## Architecture Overview

```
DrogonClaw
├── Gateway (Express HTTP server)
├── Agent (AI-powered pentesting loop)
├── Skills (YAML-based pentesting modules)
├── Storage (SQLite database)
├── Channels (CLI, Telegram, WebSocket)
└── Tools (Kali Linux tool integration)
```

### Data Flow

1. **User** inputs target and strategy via CLI
2. **Gateway** creates session and loads skill registry
3. **Agent Loop** starts:
   - Loads session context
   - Builds system prompt
   - Calls Claude/OpenAI with tools
   - Parses tool calls
   - Executes tools
   - Stores findings
   - Repeats until complete
4. **Channels** deliver results (CLI display, Telegram, WebSocket)

## Development Setup

### Install Development Dependencies

```bash
npm install
npm run build
npm run lint
```

### Watch Mode

```bash
npm run dev
```

Automatically recompiles on file changes.

### Running Tests

```bash
npm test
npm test:watch
```

### Code Style

Check and fix automatically:

```bash
npm run lint       # Check code style
npm run format     # Auto-format code
```

## Project Structure

```
src/
├── types/           # TypeScript interfaces
│   └── index.ts     # All type definitions
├── config/          # Configuration
│   └── loader.ts    # Load from .env
├── gateway/         # HTTP server
│   ├── index.ts     # Express setup
│   ├── routes/      # API endpoints
│   └── websocket.ts # WebSocket handling
├── agent/           # AI agent loop
│   ├── loop.ts      # Main agent loop
│   ├── model-client.ts  # Claude/OpenAI/Ollama
│   ├── tool-executor.ts # Execute system tools
│   └── strategies/   # Pentesting strategies
├── skills/          # YAML skill definitions
│   ├── registry.ts  # Load and manage skills
│   ├── recon/       # Reconnaissance skills
│   └── exploitation/ # Exploitation skills
├── storage/         # Database layer
│   └── sqlite.ts    # SQLite implementation
├── channels/        # Output channels
│   ├── cli/         # Terminal interface
│   └── telegram/    # Telegram bot
├── utils/           # Utility functions
│   └── logger.ts    # Logging system
└── cli/             # CLI entry point
    └── index.ts     # Command parser

tests/
├── unit/            # Unit tests
├── integration/     # Integration tests
└── fixtures/        # Test data
```

## Key Modules

### 1. Config Loader (src/config/loader.ts)

Loads configuration from `.env`:

```typescript
import { loadConfig } from '@/config/loader.js';

const config = loadConfig();
console.log(config.aiProvider); // 'claude'
```

### 2. Logger (src/utils/logger.ts)

Log messages throughout the app:

```typescript
import { logger } from '@/utils/logger.js';

logger.info('Starting agent loop');
logger.error({ error }, 'Tool execution failed');
logger.debug({ context }, 'Detailed debug info');
```

### 3. Storage (src/storage/sqlite.ts)

Save and load sessions:

```typescript
import { storage } from '@/storage/sqlite.js';

await storage.saveSession(session);
const loaded = await storage.loadSession(sessionId);
```

### 4. Agent Loop (src/agent/loop.ts)

Main pentesting orchestration:

```typescript
import { AgentLoop } from '@/agent/loop.js';

const agent = new AgentLoop();
await agent.run(context);
```

### 5. Model Client (src/agent/model-client.ts)

AI model integration:

```typescript
import { ModelClient } from '@/agent/model-client.js';

const client = new ModelClient(config);
const response = await client.call({
  systemPrompt: '...',
  messages: [...],
});
```

## Adding Features

### 1. Add New API Endpoint

In `src/gateway/routes/`:

```typescript
// new-route.ts
import { Router } from 'express';
import { logger } from '@/utils/logger.js';

const router = Router();

router.get('/my-endpoint', (req, res) => {
  logger.info('API call to /my-endpoint');
  res.json({
    success: true,
    data: { /* response */ },
    timestamp: Date.now(),
  });
});

export default router;
```

Then import in `gateway/index.ts`:

```typescript
import myRoute from './routes/new-route.js';
app.use('/api', myRoute);
```

### 2. Create New Skill

YAML file in `skills/`:

```yaml
# skills/my-skill.yaml
id: my-skill
name: My Skill
# ... rest of skill definition
```

Load automatically when gateway starts.

### 3. Add New Tool Support

In `src/agent/tool-executor.ts`:

```typescript
private validateTool(toolName: string): boolean {
  const whitelist = this.config.toolWhitelist;
  return whitelist.includes(toolName);
}
```

Add tool to `TOOL_WHITELIST` in `.env`.

### 4. Add AI Provider

In `src/agent/model-client.ts`:

```typescript
async callGemini(): Promise<ModelResponse> {
  // Implement Google Gemini API
}
```

Add provider selection:

```typescript
switch (this.config.aiProvider) {
  case 'claude':
    return this.callClaude();
  case 'gemini':
    return this.callGemini();
  // ...
}
```

## Testing

### Unit Tests

Test individual functions:

```typescript
// tests/unit/config.test.ts
import { loadConfig } from '@/config/loader.js';

describe('Config Loader', () => {
  it('should load defaults', () => {
    const config = loadConfig();
    expect(config.gatewayPort).toBe(18789);
  });
});
```

Run: `npm test`

### Integration Tests

Test modules working together:

```typescript
// tests/integration/gateway.test.ts
import { app, server } from '@/gateway/index.js';

describe('Gateway', () => {
  it('should have health endpoint', async () => {
    const response = await request(app).get('/health');
    expect(response.status).toBe(200);
  });
});
```

## Debugging

### Enable Debug Logging

```bash
LOG_LEVEL=debug npm start
```

Shows detailed execution trace.

### Node Debugger

```bash
node --inspect dist/cli/index.js
```

Open `chrome://inspect` in Chrome DevTools.

### Log specific module

```typescript
import { logger } from '@/utils/logger.js';

logger.debug({ variable }, 'Debugging info');
```

## Performance Optimization

### Connection Pooling

SQLite already handles this.

### Tool Execution

Use timeouts to prevent hanging:

```typescript
setTimeout(() => {
  childProcess.kill();
}, config.maxToolTimeout);
```

### Async Operations

Use async/await properly:

```typescript
const findings = await Promise.all([
  runNmap(target),
  runCurl(target),
  runDig(target),
]);
```

## Code Style

Follow ESLint rules:

```bash
npm run lint --fix
```

Key principles:
- TypeScript strict mode enabled
- No `any` types without reason
- Error handling for all async operations
- JSDoc comments for public functions
- Descriptive variable names

## Pull Request Process

1. Fork repository
2. Create feature branch: `git checkout -b feature/my-feature`
3. Make changes
4. Run tests: `npm test`
5. Fix lint: `npm run lint`
6. Commit: `git commit -m "Add feature description"`
7. Push: `git push origin feature/my-feature`
8. Create pull request

### PR Requirements

- [ ] Tests added/updated
- [ ] Linting passes (`npm run lint`)
- [ ] No console.log (use logger)
- [ ] TypeScript strict mode
- [ ] Documentation updated
- [ ] No secrets in code

## Deployment

### Local Development

```bash
npm run dev
```

### Production Build

```bash
npm run build
npm run lint
npm test
NODE_ENV=production npm start
```

### Docker (Recommended)

```dockerfile
FROM node:22-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --omit=dev
COPY dist ./dist
ENV NODE_ENV=production
CMD ["node", "dist/cli/index.js"]
```

## Troubleshooting Development

### Build fails
```bash
rm -rf dist node_modules
npm install
npm run build
```

### Tests fail
```bash
npm test -- --no-coverage
```

### TypeScript errors
```bash
npm run build -- --noEmit
```

## Contributing

See `CONTRIBUTING.md` for contribution guidelines.

## Support

- **Issues**: GitHub Issues
- **Discussions**: GitHub Discussions
- **Questions**: Email support@drogonclaw.dev

## Next Steps

- Read API documentation: `API.md`
- Learn about skills: `SKILLS.md`
- Check existing code for examples
- Create a feature branch and start coding!

