const fs = require('fs');
const path = require('path');

const baseDir = process.cwd();

// Define all directories to create
const directories = [
  'src',
  'src/gateway',
  'src/agent',
  'src/agent/strategies',
  'src/skills',
  'src/skills/recon',
  'src/skills/exploitation',
  'src/channels',
  'src/channels/cli',
  'src/channels/telegram',
  'src/storage',
  'src/reporting',
  'src/cli',
  'src/cli/commands',
  'src/types',
  'config',
  'tests',
  'tests/unit',
  'tests/integration',
  'tests/fixtures',
  'docs'
];

// Create directories
console.log('Creating directories...');
directories.forEach(dir => {
  const fullPath = path.join(baseDir, dir);
  if (!fs.existsSync(fullPath)) {
    fs.mkdirSync(fullPath, { recursive: true });
    console.log(`✓ Created: ${dir}`);
  }
});

// Create .gitkeep files in specific directories
const gitkeepDirs = [
  'config',
  'tests/unit',
  'tests/integration',
  'tests/fixtures',
  'docs',
  'src/skills/recon',
  'src/skills/exploitation',
  'src/channels/telegram',
  'src/storage',
  'src/reporting',
  'src/types',
  'src/agent/strategies',
  'src/channels/cli',
  'src/cli/commands'
];

console.log('\nCreating .gitkeep files...');
gitkeepDirs.forEach(dir => {
  const gitkeepPath = path.join(baseDir, dir, '.gitkeep');
  if (!fs.existsSync(gitkeepPath)) {
    fs.writeFileSync(gitkeepPath, '');
    console.log(`✓ Created: ${dir}/.gitkeep`);
  }
});

console.log('\n✅ Directory structure setup complete!');
