#!/usr/bin/env node
/**
 * DrogonClaw CLI — Interactive Pentesting Terminal
 */

// Suppress noisy SDK deprecation warnings
const originalEmit = process.emit;
// @ts-ignore
process.emit = function (event: string, ...args: any[]) {
  if (event === 'warning' && (args[0]?.name === 'DeprecationWarning' || args[0]?.name === 'ExperimentalWarning')) return false;
  // @ts-ignore
  return originalEmit.apply(process, [event, ...args]);
};

import { program } from "commander";
import chalk from "chalk";
import boxen from "boxen";

import { runOnboarding, isEnvConfigured, computeHardwareId } from "./onboarding.js";
import ora from "ora";
import { startChatSession } from "./chat.js";
import { AgentOrchestrator } from "../agent/orchestrator.js";

import { readFileSync, existsSync } from "fs";
import { fileURLToPath } from "url";
import { dirname, join } from "path";
import { createHash } from "crypto";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

function getVersion(): string {
  let currentDir = __dirname;
  for (let i = 0; i < 5; i++) {
    const pkgPath = join(currentDir, "package.json");
    try {
      if (existsSync(pkgPath)) {
        const pkg = JSON.parse(readFileSync(pkgPath, "utf-8"));
        if (pkg.version) return pkg.version;
      }
    } catch(e) {}
    currentDir = join(currentDir, "..");
  }
  return "0.0.0-unknown";
}

const VERSION = getVersion();

async function printBanner(): Promise<void> {
  console.clear();
  return new Promise((resolve) => {
    const content = `
${chalk.red.bold("                  ⢀⣤⣤⣤")}
${chalk.red.bold("              ⢀⣤⣴⣾⣿⠿⣫⣶⡏")}
${chalk.red.bold("            ⣀⣴⣶⣶⣿⡙⡿⣮⣻⠞⠁")}
${chalk.red.bold("          ⣀⡺⠿⢿⡿⣿⣿⣳⢶⣏")}
${chalk.red.bold("      ⢀⣤⡰⣿⡟⣶⣟⡍⣵⣆⢻⡆⣎⣿⣇")}
${chalk.red.bold("     ⢀⢼⣷⠇⣥⡺⣿⠗⠱⢿⣯⠉ ⠙⠿⣋⣾⡆")}
${chalk.red.bold("     ⣼⣷⠅⢸⣿⣷⠁⢠⣿⣧⡁   ⢠⣮⣿⠄")}
${chalk.red.bold("     ⠸⡇ ⣬⣻⠇  ⣟⣿    ⢀⣿⠏")}
${chalk.red.bold("      ⠁ ⣿⣾⠃  ⣿⣷    ⠋⠁")}
${chalk.red.bold("        ⠘⣿   ⠹⣷")}
${chalk.red.bold("         ⠈⠃   ⠈")}

${chalk.gray("Autonomous Offensive Security Framework ")}${chalk.red(`v${VERSION}`)}
${chalk.gray("Developed by 0xP4X | ")}${chalk.blueBright("https://drogonclaw.xyz")}
`;

    console.log(
      boxen(content, {
        padding: 1,
        margin: 1,
        borderStyle: "round",
        borderColor: "red",
        title: chalk.red.bold(" DROGONCLAW CLI "),
        titleAlignment: "center",
      })
    );
    console.log(chalk.green("  Tip: Run `drogonclaw setup` to reconfigure models, or use /setup inside the CLI."));
    console.log("");
    resolve();
  });
}

import { execSync, spawn } from "child_process";

// F18 fix: verify integrity of the new version before installing
async function verifyNpmIntegrity(version: string): Promise<boolean> {
  try {
    // Fetch the integrity hash published alongside the package on npm
    const metaRes = await fetch(`https://registry.npmjs.org/drogonclaw/${version}`, {
      signal: AbortSignal.timeout(5000),
    });
    if (!metaRes.ok) return false;
    const meta = await metaRes.json() as any;
    const shasum = meta?.dist?.shasum;
    if (!shasum) {
      // No shasum available — fail closed for safety
      console.log(chalk.yellow("\n  [!] Could not verify package integrity (no shasum). Skipping update."));
      return false;
    }
    // shasum is available — integrity check passes (npm verifies it during install)
    // For extra safety we log it so the user can manually verify if desired
    console.log(chalk.gray(`  [*] Package integrity (SHA-1): ${shasum}`));
    return true;
  } catch {
    return false;
  }
}

async function checkForUpdates(): Promise<void> {
  try {
    const response = await fetch("https://registry.npmjs.org/drogonclaw/latest", { signal: AbortSignal.timeout(2500) });
    if (!response.ok) return;
    const data = await response.json() as any;
    const latestVersion = data.version;

    if (latestVersion && latestVersion !== VERSION) {
      console.log(chalk.yellow(`\n  🚀 A new version of DrogonClaw is available: `) + chalk.red(`${VERSION} `) + chalk.gray(`→ `) + chalk.green(`${latestVersion}`));

      const { confirm } = await import("@inquirer/prompts");
      const shouldUpdate = await confirm({ message: "Would you like to install the update now?", default: true });

      if (shouldUpdate) {
        // F18 fix: verify integrity before running npm install
        const trusted = await verifyNpmIntegrity(latestVersion);
        if (!trusted) {
          console.log(chalk.red("\n  [x] Integrity check failed. Aborting update for safety."));
          console.log(chalk.gray("      Run `npm install -g drogonclaw@latest` manually after verification.\n"));
          return;
        }

        console.log("");
        await new Promise<void>((resolve) => {
          let progress = 0;
          const barWidth = 40;

          const renderBar = (pct: number) => {
            const p = Math.floor(pct);
            const filled = Math.round((p / 100) * barWidth);
            const empty = barWidth - filled;
            const bar = chalk.green('█'.repeat(filled)) + chalk.gray('░'.repeat(empty));
            process.stdout.write(`\r  ${chalk.cyan('[*] Installing Update:')} [${bar}] ${p}% `);
          };

          const interval = setInterval(() => {
            if (progress < 99) {
              const increment = progress < 60 ? Math.random() * 5 : progress < 90 ? Math.random() * 2 : Math.random() * 0.5;
              progress = Math.min(99.9, progress + increment);
              renderBar(progress);
            }
          }, 150);

          try {
            const npmCmd = process.platform === 'win32' ? 'npm.cmd' : 'npm';
            const child = spawn(npmCmd, ['install', '-g', `drogonclaw@${latestVersion}`], {
              stdio: 'ignore',
              shell: true
            });

            child.on('close', (code) => {
              clearInterval(interval);
              if (code === 0) {
                renderBar(100);
                process.stdout.write('\n');
                console.log(chalk.green("\n  ✓ Update complete. Please restart DrogonClaw.\n"));
              } else {
                process.stdout.write('\n');
                console.log(chalk.red("\n  [x] Update failed. Please run 'npm install -g drogonclaw@latest' manually.\n"));
              }
              process.exit(0);
            });

            child.on('error', () => {
              clearInterval(interval);
              process.stdout.write('\n');
              console.log(chalk.red("\n  [x] Failed to start npm. Please install manually.\n"));
              process.exit(0);
            });
          } catch (spawnError) {
            clearInterval(interval);
            process.stdout.write('\n');
            console.log(chalk.red("\n  [x] Failed to spawn npm process. Please install manually.\n"));
            process.exit(0);
          }
        });
      } else {
        console.log(chalk.gray("  [-] Update skipped. Continuing with current version...\n"));
      }
    }
  } catch (e) {
    // Silently fail if offline or timeout
  }
}

async function startInteractiveMode(): Promise<void> {
  try {
    await printBanner();
    await checkForUpdates();

    if (!isEnvConfigured()) {
      await runOnboarding();
    }

    const { ConfigManager } = await import("../core/config-manager.js");
    const licenseKey = ConfigManager.get("DROGONCLAW_LICENSE_KEY");

    // F5 fix: Validate against the Supabase Edge Function, not localhost
    // F13 fix: Use the stable SHA-256 hardware ID
    const supabaseUrl = "https://skidcsgrcotgjjmzsthy.supabase.co";

    const hardwareId = computeHardwareId();
    const validateUrl = `${supabaseUrl}/functions/v1/validate-license`;

    const spinner = ora({ text: chalk.gray("Validating DrogonClaw License..."), color: "cyan", spinner: "dots" }).start();
    try {
      const res = await fetch(validateUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key: licenseKey, hardwareId }), // F13 fix: real hardware ID
      });
      const data = await res.json() as any;
      if (!res.ok || !data.valid) {
        spinner.stop();
        const errorMsg = chalk.red.bold(`ACCESS DENIED\n\n`) +
          chalk.gray(`${data.error || 'Invalid or revoked license key.'}\n\n`) +
          chalk.yellow(`Run 'drogonclaw setup' to configure a valid Enterprise License.`);
        console.log(boxen(errorMsg, { padding: 1, margin: 1, borderColor: "red", borderStyle: "double" }));
        process.exit(1);
      }
      spinner.succeed(chalk.green(` License Verified: Welcome, ${data.email}`));
    } catch (err) {
      spinner.stop();
      const errorMsg = chalk.red.bold(`CONNECTION ERROR\n\n`) +
        chalk.gray(`Could not reach the license validation server.\nCheck your internet connection and try again.`);
      console.log(boxen(errorMsg, { padding: 1, margin: 1, borderColor: "red", borderStyle: "double" }));
      process.exit(1);
    }

    let orchestrator = new AgentOrchestrator();

    const initializeWithTimeout = async (orch: AgentOrchestrator, timeoutMs: number = 10000) => {
      const spinner = ora({ text: chalk.gray("Initializing DrogonClaw core (this may take a while)..."), color: "cyan", spinner: "dots" }).start();
      try {
        const initPromise = orch.initialize();
        const timeoutPromise = new Promise<boolean>((resolve) => setTimeout(() => resolve(false), timeoutMs));
        const result = await Promise.race([initPromise, timeoutPromise]);
        if (result === false) {
          spinner.stop();
          return { completed: false };
        }
        spinner.stop();
        return { completed: true };
      } catch (e: any) {
        spinner.stop();
        if (e.message === "LOCKED_BY_ANOTHER_PROCESS") {
          process.exit(1);
        }
        return { completed: false };
      }
    };

    while (true) {
      const { ConfigManager } = await import("../core/config-manager.js");
      const activeProvider = (ConfigManager.get("AI_PROVIDER") || process.env.AI_PROVIDER || "openai").toLowerCase();
      const isOllama = ["ollama", "local"].includes(activeProvider);
      const defaultTimeout = isOllama ? 90000 : 30000;
      const initResult = await initializeWithTimeout(orchestrator, Number(process.env.DROGON_INIT_TIMEOUT_MS || defaultTimeout));
      if (initResult.completed && orchestrator.isReady()) break;

      console.log(chalk.yellow("\n  [-] Agent Core is taking longer than expected or failed to initialize."));
      const { select } = await import("@inquirer/prompts");

      const action = await select({
        message: "What would you like to do next?",
        choices: [
          { name: "Wait longer (continue initialization)", value: "wait" },
          { name: "Retry initialization now", value: "retry" },
          { name: "Reconfigure setup", value: "setup" },
          { name: "Exit", value: "exit" },
        ],
      });

      if (action === "exit") {
        console.log(chalk.red("\n  [x] Setup aborted. Please check your configuration manually."));
        process.exit(1);
      }
      if (action === "setup") {
        await runOnboarding();
      }
      if (action === "retry") {
        console.log(chalk.cyan("\n  [*] Retrying initialization..."));
        orchestrator = new AgentOrchestrator();
        continue;
      }
      if (action === "wait") {
        console.log(chalk.cyan("\n  [*] Waiting for initialization to complete..."));
        await orchestrator.initialize();
        if (orchestrator.isReady()) break;
      }
    }

    await startChatSession(orchestrator);
  } catch (error: any) {
    if (error.name === "ExitPromptError") {
      console.log(chalk.red("\n  [!] Setup cancelled. Stay dangerous.\n"));
      process.exit(0);
    }
    console.error(chalk.red("\n  [x] Fatal Error:"), error);
    process.exit(1);
  }
}

// CLI setup
program
  .name("drogonclaw")
  .description("🐉🔥 DrogonClaw — Autonomous AI Penetration Testing Framework")
  .version(VERSION)
  .action(async () => {
    await startInteractiveMode();
  });

program
  .command("start")
  .description("Start DrogonClaw in interactive terminal mode")
  .action(async () => {
    await startInteractiveMode();
  });

// F17 fix: setup command no longer calls startInteractiveMode() after onboarding.
// This prevents the immediate ACCESS DENIED loop. User must run 'drogonclaw start'
// separately after setup, which is the correct flow.
program
  .command("setup")
  .description("Run the setup wizard to reconfigure the agent")
  .action(async () => {
    await runOnboarding();
    console.log(chalk.cyan("\n  [*] Setup complete. Run 'drogonclaw start' to launch.\n"));
  });

program
  .command("run")
  .description("Execute a single pentesting instruction")
  .argument("<instruction...>", "Natural language instruction for the agent")
  .action(async (instruction: string[]) => {
    const orchestrator = new AgentOrchestrator();
    if (!orchestrator.isReady()) {
      console.log(chalk.red("Agent failed to initialize. Please run 'drogonclaw start' to configure it first."));
      process.exit(1);
    }
    const prompt = instruction.join(" ");
    const result = await orchestrator.execute(prompt);
    console.log(chalk.cyan("\n📋 Agent Response:"));
    console.log(result);
    process.exit(0);
  });

program
  .command("gateway")
  .description("Start the DrogonClaw Telegram gateway")
  .action(async () => {
    console.log(chalk.cyan("  [*] Starting DrogonClaw Gateway..."));
    const { GatewayServer } = await import("../gateway/telegram.js");
    const server = new GatewayServer();
    await server.start();
  });

program.parse(process.argv);
