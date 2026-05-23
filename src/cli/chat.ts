import chalk from "chalk";
import * as readline from "readline";
import { AgentOrchestrator } from "../agent/orchestrator";
import ora from "ora";

export async function startChatSession(orchestrator: AgentOrchestrator): Promise<void> {
  console.clear();
  console.log(chalk.cyan("──────────────────────────────────────────────────────────────"));
  console.log(chalk.bold.white("  Autonomous Security Agent") + chalk.gray(" | Terminal Session Active"));
  console.log(chalk.cyan("──────────────────────────────────────────────────────────────"));
  console.log(chalk.gray("  Commands: type '/help'    Autocomplete: press [TAB]"));
  console.log(chalk.gray("  Exit: type 'exit' or 'quit'\n"));

  const commands = ['/help', '/skills', '/new', '/health', '/swarm ', '/stealth', '/report', '/clear', 'exit', 'quit'];
  
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    completer: (line: string) => {
      const hits = commands.filter((c) => c.startsWith(line));
      return [hits.length ? hits : (line.startsWith('/') ? commands : []), line];
    }
  });

  const ask = (query: string): Promise<string> => new Promise((resolve) => rl.question(query, resolve));

  while (true) {
    const prompt = await ask(chalk.cyan("➜ ") + chalk.white(""));

    const command = prompt.trim();
    if (!command) continue;

    if (command.toLowerCase() === "exit" || command.toLowerCase() === "quit") {
      console.log(chalk.gray("\nTerminating session...\n"));
      rl.close();
      process.exit(0);
    }

    if (command.toLowerCase() === "clear" || command.toLowerCase() === "/clear") {
      console.clear();
      console.log(chalk.cyan("──────────────────────────────────────────────────────────────"));
      console.log(chalk.bold.white("  Autonomous Security Agent") + chalk.gray(" | Terminal Session Active"));
      console.log(chalk.cyan("──────────────────────────────────────────────────────────────"));
      continue;
    }

    if (command.toLowerCase() === "/help" || command.toLowerCase() === "help") {
      console.log(chalk.bold.white("\n  COMMANDS"));
      console.log(chalk.cyan("  /clear     ") + chalk.gray("Clear the terminal screen"));
      console.log(chalk.cyan("  /skills    ") + chalk.gray("View loaded capability modules"));
      console.log(chalk.cyan("  /new       ") + chalk.gray("Wipe the agent memory and start fresh"));
      console.log(chalk.cyan("  /health    ") + chalk.gray("Run system diagnostics and verify toolkit"));
      console.log(chalk.cyan("  /swarm     ") + chalk.gray("Split a complex mission into parallel agents"));
      console.log(chalk.cyan("  /report    ") + chalk.gray("Generate a professional Markdown report"));
      console.log(chalk.cyan("  /stealth   ") + chalk.gray("Toggle stealth mode (OPSEC enforcement)"));
      console.log("");
      continue;
    }

    if (command.toLowerCase() === "/skills") {
      console.log(chalk.bold.white("\n  CAPABILITIES"));
      console.log(chalk.cyan("  • Network Recon    ") + chalk.gray("Active port scanning & service discovery"));
      console.log(chalk.cyan("  • Web Enumeration  ") + chalk.gray("Directory fuzzing & vulnerability crawling"));
      console.log(chalk.cyan("  • Exploitation     ") + chalk.gray("Custom Python script generation & execution"));
      console.log(chalk.cyan("  • Memory Graph     ") + chalk.gray("Persistent asset and vulnerability tracking"));
      console.log(chalk.cyan("  • Docker Sandbox   ") + chalk.gray("Isolated safe execution environment"));
      console.log("");
      continue;
    }

    if (command.toLowerCase() === "/new") {
      orchestrator.newSession();
      console.log(chalk.green("  ✓ ") + chalk.gray("Memory wiped. New session started.\n"));
      continue;
    }

    if (command.toLowerCase() === "/stealth on" || command.toLowerCase() === "/stealth") {
      if (orchestrator.opsecManager.isStealthEnabled?.()) {
        orchestrator.opsecManager.disableStealthMode();
        console.log(chalk.yellow("  ⬡ ") + chalk.white("Stealth mode ") + chalk.red("disabled") + chalk.gray(" — aggressive scanning active\n"));
      } else {
        orchestrator.opsecManager.enableStealthMode();
        console.log(chalk.yellow("  ⬡ ") + chalk.white("Stealth mode ") + chalk.green("enabled") + chalk.gray(" — timing jitter on, noisy scanners suppressed\n"));
      }
      continue;
    }

    if (command.toLowerCase() === "/stealth off") {
      orchestrator.opsecManager.disableStealthMode();
      console.log(chalk.yellow("  ⬡ ") + chalk.white("Stealth mode ") + chalk.red("disabled") + chalk.gray(" — aggressive scanning active\n"));
      continue;
    }

    if (command.toLowerCase().startsWith("/swarm ")) {
      const swarmMission = command.substring(7).trim();
      if (!swarmMission) {
        console.log(chalk.red("  ✗ ") + chalk.gray("Usage: /swarm <objective>\n"));
        continue;
      }
      
      const { SwarmCommander } = await import("../agent/swarm-commander");
      const commander = new SwarmCommander();
      
      const swarmSpinner = ora({
        text: chalk.gray("Dispatching parallel agents..."),
        color: "cyan",
        spinner: "dots",
      }).start();

      try {
        const result = await commander.executeSwarm(swarmMission);
        swarmSpinner.stop();
        console.log(chalk.cyan("\n  ┌─ Swarm Report ─────────────────────────────────────────"));
        console.log(result.split('\n').map((line: string) => chalk.gray("  │ ") + line).join('\n'));
        console.log(chalk.cyan("  └───────────────────────────────────────────────────────\n"));
      } catch (error: any) {
        swarmSpinner.stop();
        console.log(chalk.red("  ✗ ") + chalk.gray(`Swarm error: ${error.message}\n`));
      }
      continue;
    }

    if (command.toLowerCase() === "/report") {
      const reportSpinner = ora({
        text: chalk.gray("Generating report..."),
        color: "cyan",
        spinner: "dots",
      }).start();
      try {
        const { ReportGenerator } = await import("../core/report-generator");
        const generator = new ReportGenerator(orchestrator.getMemoryGraph());
        const reportPath = await generator.generateMarkdownReport();
        reportSpinner.stop();
        console.log(chalk.green("  ✓ ") + chalk.white("Report saved to: ") + chalk.underline.gray(reportPath) + "\n");
      } catch (error: any) {
        reportSpinner.stop();
        console.log(chalk.red("  ✗ ") + chalk.gray(`Report error: ${error.message}\n`));
      }
      continue;
    }

    if (command.toLowerCase() === "/health") {
      const { HealthChecker } = await import("../core/health-checker");
      const checker = new HealthChecker();
      await checker.runDiagnostics();
      continue;
    }

    // Guard: catch unrecognized slash commands so typos don't get sent to the agent
    if (command.startsWith("/")) {
      const knownCommands = ['/help', '/skills', '/new', '/health', '/swarm', '/stealth', '/report', '/clear'];
      const typed = command.toLowerCase().split(" ")[0];
      
      // Find closest match for helpful suggestion
      const closest = knownCommands.find(c => c.startsWith(typed.substring(0, 3)));
      
      if (closest) {
        console.log(chalk.yellow("  ? ") + chalk.gray(`Unknown command '${typed}'. Did you mean `) + chalk.cyan(closest) + chalk.gray("?\n"));
      } else {
        console.log(chalk.yellow("  ? ") + chalk.gray(`Unknown command '${typed}'. Type `) + chalk.cyan("/help") + chalk.gray(" for available commands.\n"));
      }
      continue;
    }

    // ── Agent Execution ──────────────────────────────────────
    console.log("");
    const spinner = ora({
      text: chalk.gray("Processing..."),
      color: "cyan",
      spinner: "dots",
    }).start();

    try {
      const result = await orchestrator.execute(command, (toolName, _args) => {
        spinner.text = chalk.gray(`Running: `) + chalk.cyan(toolName);
      });
      
      spinner.stop();
      console.log(chalk.cyan("\n  ┌─ Response ────────────────────────────────────────────"));
      const lines = result.split('\n');
      for (const line of lines) {
        console.log(chalk.cyan("  │ ") + line);
      }
      console.log(chalk.cyan("  └───────────────────────────────────────────────────────\n"));
    } catch (error: any) {
      spinner.stop();
      console.log(chalk.red("  ✗ ") + chalk.gray(`Error: ${error.message}\n`));
    }
  }
}
