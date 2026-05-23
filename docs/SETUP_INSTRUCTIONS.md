# DrogonClaw Project Setup Instructions

Due to environment limitations, the automated setup scripts cannot execute. Please follow these manual steps to complete the project setup.

## Step 1: Create Directory Structure

Open Command Prompt (cmd.exe) and run these commands:

```batch
cd C:\Users\0day\Desktop\drogon

REM Create all directories
mkdir src\gateway
mkdir src\agent\strategies
mkdir src\skills\recon
mkdir src\skills\exploitation
mkdir src\channels\cli
mkdir src\channels\telegram
mkdir src\storage
mkdir src\reporting
mkdir src\cli\commands
mkdir src\types
mkdir config
mkdir tests\unit
mkdir tests\integration
mkdir tests\fixtures
mkdir docs
```

**OR** - Run the batch file:
```
create-structure.bat
```

**OR** - Run the Python script:
```
python setup_project.py
```

**OR** - Run the Node.js script:
```
node setup-project.js
```

## Step 2: Move Source Files into Correct Directories

After creating the directories, move these files:

1. Move `cli-index.ts` to `src\cli\index.ts`
2. Move `gateway-server.ts` to `src\gateway\server.ts`
3. Move `agent-orchestrator.ts` to `src\agent\orchestrator.ts`

## Step 3: Create .gitkeep Files (Optional, for git)

Create empty `.gitkeep` files in these directories to preserve them in git:

- `src\skills\recon\.gitkeep`
- `src\skills\exploitation\.gitkeep`
- `src\channels\cli\.gitkeep`
- `src\channels\telegram\.gitkeep`
- `src\storage\.gitkeep`
- `src\reporting\.gitkeep`
- `src\types\.gitkeep`
- `src\agent\strategies\.gitkeep`
- `src\cli\commands\.gitkeep`
- `config\.gitkeep`
- `tests\unit\.gitkeep`
- `tests\integration\.gitkeep`
- `tests\fixtures\.gitkeep`
- `docs\.gitkeep`

## Step 4: Install Dependencies

```bash
npm install
```

## Step 5: Verify Build

```bash
npm run build
```

## Step 6: Run Linter

```bash
npm run lint
```

## Completion Checklist

- [ ] Directory structure created
- [ ] TypeScript files moved to correct locations
- [ ] `.gitkeep` files created (optional)
- [ ] `npm install` completed successfully
- [ ] `npm run build` compiled without errors
- [ ] `npm run lint` passed checks

## Project Structure Expected After Setup

```
drogon/
├── .env.example                    ✓ Created
├── .eslintrc.json                  ✓ Created
├── .gitignore                      ✓ Created
├── .prettierrc                     ✓ Created
├── README.md                       ✓ Created
├── package.json                    ✓ Created
├── tsconfig.json                   ✓ Created
├── src/
│   ├── gateway/
│   │   └── server.ts               (move gateway-server.ts here)
│   ├── agent/
│   │   ├── orchestrator.ts         (move agent-orchestrator.ts here)
│   │   └── strategies/             ✓ Create directory
│   ├── skills/
│   │   ├── recon/                  ✓ Create directory
│   │   └── exploitation/           ✓ Create directory
│   ├── channels/
│   │   ├── cli/                    ✓ Create directory
│   │   └── telegram/               ✓ Create directory
│   ├── storage/                    ✓ Create directory
│   ├── reporting/                  ✓ Create directory
│   ├── cli/
│   │   ├── index.ts                (move cli-index.ts here)
│   │   └── commands/               ✓ Create directory
│   └── types/                      ✓ Create directory
├── config/                         ✓ Create directory
├── tests/
│   ├── unit/                       ✓ Create directory
│   ├── integration/                ✓ Create directory
│   └── fixtures/                   ✓ Create directory
└── docs/                           ✓ Create directory
```

## Automated Scripts Available

Three scripts have been provided for your convenience:

1. **setup-project.js** - Node.js script (requires node.js 22+)
2. **setup_project.py** - Python script (requires python 3.7+)
3. **create-structure.bat** - Batch script (Windows native)

Choose any one and run it from the project directory.

## Files Created Successfully

✓ package.json
✓ tsconfig.json
✓ .eslintrc.json
✓ .prettierrc
✓ .gitignore
✓ .env.example
✓ README.md
✓ cli-index.ts (source code for src/cli/index.ts)
✓ gateway-server.ts (source code for src/gateway/server.ts)
✓ agent-orchestrator.ts (source code for src/agent/orchestrator.ts)
✓ setup-project.js
✓ setup_project.py
✓ create-structure.bat

Total: 12 configuration/setup files + 3 source files

## Next Steps After Setup

After completing the steps above, you can:

```bash
npm run dev          # Start development with hot reload
npm run build        # Build TypeScript to JavaScript
npm start            # Run the compiled CLI
npm run lint         # Check code style
npm run format       # Auto-format code
npm test             # Run tests (when added)
```

Good luck with DrogonClaw! 🐉
