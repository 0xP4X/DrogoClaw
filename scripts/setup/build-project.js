#!/usr/bin/env node
/**
 * DrogonClaw Complete Setup Script
 * Creates all directories and source files
 * Run with: node build-project.js
 */

const fs = require('fs');
const path = require('path');

const ROOT = __dirname;

// Define all source files to create with their content
const sourceFiles = {
  // Core types
  'src/types/index.ts': require('./types-index.ts'),
  
  // Configuration loader
  'src/config/loader.ts': `import { ConfigSchema, Config } from '@/types/index.js';
import pino from 'pino';
import * as dotenv from 'dotenv';

const logger = pino();

export function loadConfig(): Config {
  dotenv.config();

  const config = {
    aiProvider: process.env.AI_PROVIDER || 'claude',
    anthropicApiKey: process.env.ANTHROPIC_API_KEY,
    openaiApiKey: process.env.OPENAI_API_KEY,
    ollamaUrl: process.env.OLLAMA_URL || 'http://localhost:11434',
    ollamaModel: process.env.OLLAMA_MODEL || 'mistral',
    gatewayPort: parseInt(process.env.GATEWAY_PORT || '18789', 10),
    gatewayHost: process.env.GATEWAY_HOST || 'localhost',
    databasePath: process.env.DATABASE_PATH || './data/drogonclaw.db',
    logLevel: process.env.LOG_LEVEL || 'info',
    sessionTimeout: parseInt(process.env.SESSION_TIMEOUT || '3600000', 10),
    maxToolTimeout: parseInt(process.env.MAX_TOOL_TIMEOUT || '300000', 10),
    toolWhitelist: (process.env.TOOL_WHITELIST || '').split(',').filter(Boolean),
    telegramToken: process.env.TELEGRAM_TOKEN,
    telegramChatId: process.env.TELEGRAM_CHAT_ID,
    enableWebSocket: process.env.ENABLE_WEBSOCKET !== 'false',
    enableTelegram: process.env.ENABLE_TELEGRAM === 'true',
  };

  const result = ConfigSchema.safeParse(config);
  if (!result.success) {
    logger.error({ errors: result.error.errors }, 'Invalid configuration');
    throw new Error('Configuration validation failed');
  }

  return result.data;
}

export const config = loadConfig();
`,
  
  // Logger utility
  'src/utils/logger.ts': `import pino from 'pino';
import { LogEntry } from '@/types/index.js';

export function createLogger(name: string) {
  return pino({ name });
}

export const logger = createLogger('DrogonClaw');
`,

  // Placeholder files for other modules (to be expanded)
  'src/gateway/index.ts': `import express from 'express';
import http from 'http';
import { config } from '@/config/loader.js';
import { logger } from '@/utils/logger.js';

const app = express();
const server = http.createServer(app);

app.use(express.json());

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// API routes
app.get('/api/sessions', (req, res) => {
  res.json({ sessions: [] });
});

app.get('/api/findings', (req, res) => {
  res.json({ findings: [] });
});

server.listen(config.gatewayPort, config.gatewayHost, () => {
  logger.info(\`Gateway listening on \${config.gatewayHost}:\${config.gatewayPort}\`);
});
`,

  'src/agent/loop.ts': `import { Session, Finding, AgentContext } from '@/types/index.js';
import { logger } from '@/utils/logger.js';

export class AgentLoop {
  async run(context: AgentContext): Promise<void> {
    logger.info({ target: context.session.target }, 'Starting agent loop');
    
    // TODO: Implement full agent loop
    // 1. Load session and context
    // 2. Build system prompt
    // 3. Call AI model
    // 4. Execute tool calls
    // 5. Update session
    // 6. Emit events
  }
}
`,

  'src/storage/sqlite.ts': `import sqlite3 from 'sqlite3';
import { Session, Finding } from '@/types/index.js';
import { config } from '@/config/loader.js';
import { logger } from '@/utils/logger.js';

export class SQLiteStorage {
  private db: sqlite3.Database;

  constructor() {
    this.db = new sqlite3.Database(config.databasePath);
    this.initialize();
  }

  private initialize() {
    this.db.serialize(() => {
      this.db.run(\`
        CREATE TABLE IF NOT EXISTS sessions (
          id TEXT PRIMARY KEY,
          target TEXT NOT NULL,
          strategy TEXT NOT NULL,
          status TEXT NOT NULL,
          findings TEXT,
          metadata TEXT,
          created_at INTEGER,
          updated_at INTEGER
        )
      \`);

      this.db.run(\`
        CREATE TABLE IF NOT EXISTS findings (
          id TEXT PRIMARY KEY,
          session_id TEXT NOT NULL,
          title TEXT NOT NULL,
          description TEXT,
          severity TEXT,
          type TEXT,
          target TEXT,
          evidence TEXT,
          discovered_at INTEGER,
          FOREIGN KEY (session_id) REFERENCES sessions(id)
        )
      \`);

      logger.info('Database initialized');
    });
  }

  async saveSession(session: Session): Promise<void> {
    return new Promise((resolve, reject) => {
      this.db.run(
        \`INSERT OR REPLACE INTO sessions (id, target, strategy, status, findings, metadata, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)\`,
        [
          session.id,
          session.target,
          session.strategy,
          session.status,
          JSON.stringify(session.findings),
          JSON.stringify(session.metadata),
          session.createdAt,
          session.updatedAt,
        ],
        (err) => {
          if (err) reject(err);
          else resolve();
        },
      );
    });
  }

  async loadSession(sessionId: string): Promise<Session | null> {
    return new Promise((resolve, reject) => {
      this.db.get('SELECT * FROM sessions WHERE id = ?', [sessionId], (err, row: any) => {
        if (err) reject(err);
        else if (!row) resolve(null);
        else {
          resolve({
            id: row.id,
            target: row.target,
            strategy: row.strategy,
            status: row.status,
            findings: JSON.parse(row.findings || '[]'),
            metadata: JSON.parse(row.metadata || '{}'),
            notes: '',
            startTime: row.created_at,
            createdAt: row.created_at,
            updatedAt: row.updated_at,
          });
        }
      });
    });
  }
}

export const storage = new SQLiteStorage();
`,

  'src/channels/cli/index.ts': `import inquirer from 'inquirer';
import chalk from 'chalk';
import { logger } from '@/utils/logger.js';

export async function startCLI() {
  logger.info('Starting DrogonClaw CLI');

  const answers = await inquirer.prompt([
    {
      type: 'input',
      name: 'target',
      message: 'Enter target (domain or IP):',
    },
    {
      type: 'list',
      name: 'strategy',
      message: 'Select strategy:',
      choices: ['light', 'thorough', 'deep', 'aggressive'],
    },
  ]);

  console.log(chalk.green(\`\\n🐉 Starting pentest of \${answers.target} with \${answers.strategy} strategy\\n\`));

  // TODO: Implement CLI channel
}
`,

  'src/cli/index.ts': `#!/usr/bin/env node

import { config } from '@/config/loader.js';
import { logger } from '@/utils/logger.js';
import { startCLI } from '@/channels/cli/index.js';

async function main() {
  logger.info({ config }, 'DrogonClaw starting');
  await startCLI();
}

main().catch((err) => {
  logger.error(err, 'Fatal error');
  process.exit(1);
});
`,
};

// Create directories
const directories = [
  'src/types',
  'src/config',
  'src/gateway',
  'src/gateway/routes',
  'src/agent',
  'src/agent/strategies',
  'src/skills',
  'src/channels',
  'src/channels/cli',
  'src/channels/telegram',
  'src/storage',
  'src/utils',
  'src/models',
  'src/reporting',
  'tests/unit',
  'tests/integration',
  'tests/fixtures',
  'docs',
  'config',
  'data',
];

console.log('🐉 DrogonClaw Setup\n');
console.log('Creating directories...');

directories.forEach((dir) => {
  const fullPath = path.join(ROOT, dir);
  if (!fs.existsSync(fullPath)) {
    fs.mkdirSync(fullPath, { recursive: true });
    console.log(\`  ✓ \${dir}\`);
  }
});

console.log('\\nDirectory structure created!');
