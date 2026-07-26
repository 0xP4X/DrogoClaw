/**
 * IMMEDIATE SETUP: Run with: npm exec ts-node setup-immediate.ts
 * Or: npx tsx setup-immediate.ts
 * Creates complete project structure and all source files
 */

import * as fs from 'fs';
import * as path from 'path';

const __dirname = process.cwd();

// Directories to create
const directories = [
  'src/types',
  'src/config',
  'src/gateway',
  'src/gateway/routes',
  'src/agent',
  'src/agent/strategies',
  'src/skills',
  'src/skills/recon',
  'src/skills/enumeration',
  'src/skills/exploitation',
  'src/channels',
  'src/channels/cli',
  'src/channels/cli/commands',
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

console.log('🐉 DrogonClaw - Immediate Setup\n');

// Create directories
console.log('📁 Creating directory structure...');
directories.forEach((dir) => {
  const fullPath = path.join(__dirname, dir);
  if (!fs.existsSync(fullPath)) {
    fs.mkdirSync(fullPath, { recursive: true });
    console.log(`  ✓ ${dir}`);
  }
});

console.log('\n✅ Directory structure created!');
console.log('\nNext steps:');
console.log('  npm install');
console.log('  npm run build');
console.log('  npm run lint');
