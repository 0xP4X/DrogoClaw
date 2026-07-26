#!/usr/bin/env node
/**
 * Pre-install directory creation for DrogonClaw
 * This runs as a preinstall hook before npm packages are installed
 */

const fs = require('fs');
const path = require('path');
const ROOT = path.resolve(__dirname, '..', '..');

const dirs = [
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

let count = 0;
dirs.forEach((dir) => {
  const fullPath = path.join(ROOT, dir);
  try {
    if (!fs.existsSync(fullPath)) {
      fs.mkdirSync(fullPath, { recursive: true });
      count++;
    }
  } catch (err) {
    // Silently ignore if directory creation fails
  }
});

// After npm install, run full-setup.js if directories were created
if (count > 0) {
  try {
    require('./full-setup.js');
  } catch (err) {
    // Will be run as a postinstall script
  }
}
