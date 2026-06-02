#!/usr/bin/env node
/**
 * DrogonClaw CLI — Interactive Pentesting Terminal
 */

// Suppress noisy SDK deprecation warnings (Zod `.optional()` without `.nullable()`) and ExperimentalWarnings
const originalEmit = process.emit;
// @ts-ignore
process.emit = function (event: string, ...args: any[]) {
  if (event === 'warning' && (args[0]?.name === 'DeprecationWarning' || args[0]?.name === 'ExperimentalWarning')) return false;
  // @ts-ignore
  return originalEmit.apply(process, [event, ...args]);
};
import { program } from "commander";
import chalk from "chalk";
import figlet from "figlet";
import { runOnboarding, isEnvConfigured } from "./onboarding.js";
import ora from "ora";
import { startChatSession } from "./chat.js";
import { AgentOrchestrator } from "../agent/orchestrator.js";

const VERSION = "0.3.0";

async function printBanner(): Promise<void> {
  console.clear();
  return new Promise((resolve) => {
    figlet.text("DROGONCLAW", { font: "ANSI Shadow", horizontalLayout: "fitted" }, (err, data) => {
      console.log("");
      if (!err && data) {
        // Apply a subtle red-to-dark gradient effect
        const lines = data.split('\n');
        lines.forEach((line, i) => {
          const intensity = Math.max(50, 255 - (i * 20));
          console.log(chalk.rgb(intensity, 0, 0).bold(line));
        });
      } else {
        console.log(chalk.red.bold("  [*] DROGONCLAW"));
      }
      console.log(chalk.red.bold("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"));
      console.log(chalk.gray(`  Autonomous Offensive Security Framework`) + chalk.red(` v${VERSION}`) + chalk.gray(` | Root: `) + chalk.green(`Active`));
      console.log(chalk.gray(`  Developed by 0xP4X`));
      console.log(chalk.red.bold("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"));
      console.log(chalk.green("  Tip: Run `drogonclaw setup` to reconfigure models, or use /setup inside the CLI."));
      console.log("");
      resolve();
    });
  });
}

import { execSync, spawn } from "child_process";

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


          const npmCmd = process.platform === 'win32' ? 'npm.cmd' : 'npm';
          const child = spawn(npmCmd, ['install', '-g', 'drogonclaw@latest'], { stdio: 'ignore' });

          child.on('close', () => {
            clearInterval(interval);
            renderBar(100);
            process.stdout.write('\n');
            console.log(chalk.green("\n  ✓ Update complete. Please restart DrogonClaw.\n"));
            process.exit(0);
          });
        });
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
      } catch (e) {
        spinner.stop();
        return { completed: false };
      }
    };

    while (true) {
      const initResult = await initializeWithTimeout(orchestrator, Number(process.env.DROGON_INIT_TIMEOUT_MS || 10000));
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
        // wait without timeout
        await orchestrator.initialize();
        if (orchestrator.isReady()) break;
        // otherwise loop again
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

program
  .command("setup")
  .description("Run the setup wizard to reconfigure the agent")
  .action(async () => {
    await runOnboarding();
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
