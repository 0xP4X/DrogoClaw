#!/usr/bin/env node
/**
 * Complete setup script for DrogonClaw
 * Creates directory structure and scaffolds the project
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const projectRoot = __dirname;

// Define all directories to create
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

console.log('🐉 DrogonClaw Setup Script\n');
console.log('📁 Creating directory structure...');

// Create directories
directories.forEach((dir) => {
  const fullPath = path.join(projectRoot, dir);
  if (!fs.existsSync(fullPath)) {
    fs.mkdirSync(fullPath, { recursive: true });
    console.log(`  ✓ ${dir}`);
  }
});

console.log('\n✅ Setup complete!');
console.log('\nNext steps:');
console.log('  1. npm install        (install dependencies)');
console.log('  2. npm run build      (compile TypeScript)');
console.log('  3. npm run lint       (verify code style)');
console.log('  4. cp .env.example .env');
console.log('  5. Edit .env with your API keys');
console.log('\nThen:');
console.log('  npm start    - Run CLI');
console.log('  npm run dev  - Development mode');
console.log('  npm test     - Run tests');
