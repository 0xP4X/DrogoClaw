#!/usr/bin/env node
/**
 * DrogonClaw CLI — Interactive Pentesting Terminal
 */

// Suppress noisy SDK deprecation warnings (Zod `.optional()` without `.nullable()`)
const originalEmit = process.emit;
// @ts-ignore
process.emit = function (event: string, ...args: any[]) {
  if (event === 'warning' && args[0]?.name === 'DeprecationWarning') return false;
  // @ts-ignore
  return originalEmit.apply(process, [event, ...args]);
};
import { program } from "commander";
import chalk from "chalk";
import figlet from "figlet";
import dotenv from "dotenv";
import { runOnboarding, isEnvConfigured } from "./onboarding";
import { startChatSession } from "./chat";
import { AgentOrchestrator } from "../agent/orchestrator";

const VERSION = "0.2.0";

async function printBanner(): Promise<void> {
  console.clear();
  return new Promise((resolve) => {
    figlet.text("DrogonClaw", { font: "Standard", horizontalLayout: "default" }, (err, data) => {
      if (!err && data) {
        console.log(chalk.red(data));
      } else {
        console.log(chalk.red("🐉🔥 DrogonClaw"));
      }
      console.log(chalk.red("══════════════════════════════════════════════════════════════════════════"));
      console.log(chalk.gray(`Autonomous AI Penetration Testing Framework v${VERSION}`));
      console.log(chalk.red("══════════════════════════════════════════════════════════════════════════\n"));
      resolve();
    });
  });
}

async function startInteractiveMode(): Promise<void> {
  try {
    await printBanner();

    if (!isEnvConfigured()) {
      await runOnboarding();
    }

    // Load environment variables after onboarding
    dotenv.config();

    let orchestrator = new AgentOrchestrator();
    await orchestrator.initialize();
    
    while (!orchestrator.isReady()) {
      console.log(chalk.yellow("\n⚠️ Agent Core is offline. Please resolve configuration errors to continue."));
      const { confirm } = await import("@inquirer/prompts");
      const fs = await import("fs");
      const path = await import("path");
      
      const retry = await confirm({ message: "Would you like to run the setup wizard again?", default: true });
      
      if (retry) {
        const envPath = path.join(process.cwd(), ".env");
        if (fs.existsSync(envPath)) {
          fs.unlinkSync(envPath); // Clear the bad config
        }
        await runOnboarding();
        
        // Force manual override of process.env because dotenv.config({override: true}) sometimes fails to mutate cached Node vars
        const newEnvPath = path.join(process.cwd(), ".env");
        if (fs.existsSync(newEnvPath)) {
          const parsed = dotenv.parse(fs.readFileSync(newEnvPath));
          for (const key in parsed) {
            process.env[key] = parsed[key];
          }
        }
        
        console.log(chalk.cyan("\n🔄 Retrying initialization..."));
        orchestrator = new AgentOrchestrator();
        await orchestrator.initialize();
      } else {
        console.log(chalk.red("\n❌ Setup aborted. Please check your credentials manually."));
        process.exit(1);
      }
    }

    await startChatSession(orchestrator);
  } catch (error: any) {
    if (error.name === "ExitPromptError") {
      console.log(chalk.red("\n🐉 Setup cancelled. Stay dangerous. 🔥\n"));
      process.exit(0);
    }
    console.error(chalk.red("\n❌ Fatal Error:"), error);
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
  .command("run")
  .description("Execute a single pentesting instruction")
  .argument("<instruction...>", "Natural language instruction for the agent")
  .action(async (instruction: string[]) => {
    dotenv.config();
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
  .description("Start the DrogonClaw gateway server (HTTP + Telegram)")
  .action(async () => {
    dotenv.config();
    console.log(chalk.cyan("🐉 Starting DrogonClaw Gateway..."));
    const { GatewayServer } = await import("../gateway/server");
    const server = new GatewayServer();
    await server.start();
  });

program.parse(process.argv);
