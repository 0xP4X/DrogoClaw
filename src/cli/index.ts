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
import { runOnboarding, isEnvConfigured } from "./onboarding";
import ora from "ora";
import { startChatSession } from "./chat";
import { AgentOrchestrator } from "../agent/orchestrator";

const VERSION = "0.2.0";

const DROGONCLAW_BANNER = \`
      /\\      /\\      /\\
     /  \\    /  \\    /  \\
    |    |  |    |  |    |
    |    |  |    |  |    |
    |  /\\|  |/\\  |  |/\\  |
     \\ \\  \\  /  /   / / /
      \\ \\  \\/  /   / / /
       \\ \\    /   / / /
        \\      \\ / / /
         \\      / / /
          |    | | |

    ____                               ________              
   / __ \\_________  ____ _____  ____  / ____/ /___ __      __
  / / / / ___/ __ \\/ __ \`/ __ \\/ __ \\/ /   / / __ \`/ | /| / /
 / /_/ / /  / /_/ / /_/ / /_/ / / / / /___/ / /_/ /| |/ |/ / 
/_____/_/   \\____/\\__, /\\____/_/ /_/\\____/_/\\__,_/ |__/|__/  
                 /____/                                      
\`;

async function printBanner(): Promise<void> {
  console.clear();
  return new Promise((resolve) => {
    console.log("");
    const lines = DROGONCLAW_BANNER.split('\\n');
    lines.forEach((line, i) => {
      // Glow effect: red to dark red
      const intensity = Math.max(80, 255 - (i * 8));
      const r = intensity;
      const g = Math.max(0, intensity - 180); // Slight orange tint at the top
      console.log(chalk.rgb(r, g, 0).bold(line));
    });

    console.log(chalk.red.bold("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"));
    console.log(chalk.gray(\`  Autonomous Offensive Security Framework\`) + chalk.red(\` v\${VERSION}\`) + chalk.gray(\` | Root: \`) + chalk.green(\`Active\`));
    console.log(chalk.red.bold("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\\n"));
    console.log(chalk.green("  Tip: Run \`drogonclaw setup\` to reconfigure models, or use /setup inside the CLI."));
    console.log("");
    resolve();
  });
}

async function startInteractiveMode(): Promise<void> {
  try {
    await printBanner();

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
    const { GatewayServer } = await import("../gateway/telegram");
    const server = new GatewayServer();
    await server.start();
  });

program.parse(process.argv);
