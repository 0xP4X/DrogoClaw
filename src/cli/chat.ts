import chalk from "chalk";
import * as readline from "readline";
import { AgentOrchestrator } from "../agent/orchestrator";
import ora from "ora";
import { marked } from "marked";
import TerminalRenderer from "marked-terminal";
import { select, Separator } from "@inquirer/prompts";
import { ConfigManager } from "../core/config-manager";

marked.setOptions({
  // @ts-ignore
  renderer: new TerminalRenderer({
    reflowText: true,
    width: Math.min(80, process.stdout.columns ? process.stdout.columns - 4 : 80),
  })
});

// Helper for typewriter streaming effect
async function streamText(text: string, speedMs: number = 5): Promise<void> {
  const lines = text.split('\n');
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    process.stdout.write("  │  ");
    for (const char of line) {
      process.stdout.write(char);
      if (speedMs > 0) await new Promise(r => setTimeout(r, speedMs));
    }
    process.stdout.write('\n');
  }
}

export async function startChatSession(orchestrator: AgentOrchestrator): Promise<void> {
  const profile = orchestrator.getMemoryGraph().getOperatorProfile();
  const operatorName = profile ? profile.name : "Unknown Operator";
  console.log(chalk.gray("  Welcome, ") + chalk.bold.white(operatorName) + chalk.gray(". Type '/' for commands or 'exit' to quit.\n"));

  const commands = ['/help', '/skills', '/new', '/health', '/swarm ', '/stealth', '/report', '/clear', 'exit', 'quit'];
  
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    completer: (line: string) => {
      const hits = commands.filter((c) => c.startsWith(line));
      return [hits.length ? hits : (line.startsWith('/') ? commands : []), line];
    }
  });

  // Raw mode for live slash command hints
  readline.emitKeypressEvents(process.stdin);
  if (process.stdin.isTTY) process.stdin.setRawMode(true);

  let showingHints = false;
  process.stdin.on('keypress', (str, key) => {
    if (key.ctrl && key.name === 'c') process.exit(0);
    if (key.ctrl && key.name === 'd') process.exit(0);
    
    if (key.name === 'return' || key.name === 'enter') {
      if (showingHints) {
        process.stdout.write(`\x1B[s\x1B[1E\x1B[J\x1B[u`); 
        showingHints = false;
      }
      return;
    }

    const line = rl.line;
    if (line.startsWith('/')) {
      const hits = commands.filter((c) => c.startsWith(line));
      process.stdout.write(`\x1B[s\x1B[1E\x1B[J\x1B[u`); 
      if (hits.length > 0) {
        showingHints = true;
        const hintText = chalk.gray("  ╭─ Suggestions ─────────────────\n") + 
                         chalk.gray("  │  ") + chalk.cyan(hits.join('  ')) + 
                         chalk.gray("\n  ╰───────────────────────────────");
        process.stdout.write(`\x1B[s\x1B[1E${hintText}\x1B[u`);
      } else {
        showingHints = false;
      }
    } else if (showingHints) {
      process.stdout.write(`\x1B[s\x1B[1E\x1B[J\x1B[u`);
      showingHints = false;
    }
  });

  const ask = (query: string): Promise<string> => new Promise((resolve) => rl.question(query, resolve));

  // Auto-Greeting if Operator is Unknown
  if (operatorName === "Unknown Operator") {
    console.log("");
    let spinner = ora({
      text: chalk.gray("Initializing neural pathways..."),
      color: "cyan",
      spinner: "bouncingBar",
    }).start();

    try {
      const result = await orchestrator.execute("The operator has booted up the system but has not provided their name. Introduce yourself aggressively, explain your capabilities briefly, and demand their hacker alias.", (toolName, args) => {
        if (toolName === "thought") {
          let thoughtText = args.thought.substring(0, 85).replace(/\n/g, ' '); 
          if (args.thought.length > 85) thoughtText += "...";
          spinner.text = chalk.gray(`Thinking: `) + chalk.white(thoughtText);
        } else {
          spinner.text = chalk.gray(`Executing: `) + chalk.cyan(toolName);
        }
      });
      spinner.stop();
      console.log(chalk.cyan("\n  ╭─ DrogonClaw ──────────────────────────────────────────"));
      const rendered = await marked.parse(result);
      await streamText(rendered.trimEnd());
      console.log(chalk.cyan("  ╰───────────────────────────────────────────────────────\n"));
    } catch (e) {
      spinner.stop();
    }
  }

  while (true) {
    // Dynamically fetch the operator name in case it was updated by the neural memory tool during the session
    const currentProfile = orchestrator.getMemoryGraph().getOperatorProfile();
    const currentOperatorName = currentProfile ? currentProfile.name : "Unknown Operator";

    const prompt = await ask(chalk.cyan(`╭─ ${currentOperatorName}\n╰─❯ `) + chalk.white(""));

    const command = prompt.trim();
    if (!command) continue;

    if (command.toLowerCase() === "exit" || command.toLowerCase() === "quit") {
      console.log(chalk.gray("\nTerminating session...\n"));
      rl.close();
      process.exit(0);
    }

    if (command.toLowerCase() === "clear" || command.toLowerCase() === "/clear") {
      console.clear();
      console.log(chalk.gray("  Terminal cleared.\n"));
      continue;
    }

    if (command.toLowerCase() === "/help" || command.toLowerCase() === "help") {
      console.log(chalk.gray("\n  ╭─ Commands ──────────────────────────────────────────"));
      console.log(chalk.gray("  │  ") + chalk.cyan("/clear     ") + chalk.gray("Clear the terminal screen"));
      console.log(chalk.gray("  │  ") + chalk.cyan("/skills    ") + chalk.gray("Interactive capability explorer"));
      console.log(chalk.gray("  │  ") + chalk.cyan("/new       ") + chalk.gray("Wipe the agent memory and start fresh"));
      console.log(chalk.gray("  │  ") + chalk.cyan("/health    ") + chalk.gray("Run system diagnostics and verify toolkit"));
      console.log(chalk.gray("  │  ") + chalk.cyan("/swarm     ") + chalk.gray("Split a complex mission into parallel agents"));
      console.log(chalk.gray("  │  ") + chalk.cyan("/report    ") + chalk.gray("Generate a professional Markdown report"));
      console.log(chalk.gray("  │  ") + chalk.cyan("/stealth   ") + chalk.gray("Toggle stealth mode (OPSEC enforcement)"));
      console.log(chalk.gray("  ╰─────────────────────────────────────────────────────\n"));
      continue;
    }

    if (command.toLowerCase() === "/skills") {
      // Interactive Inquirer Menu
      rl.pause();
      const answer = await select({
        message: 'Explore DrogonClaw Capabilities:',
        choices: [
          { name: 'Network Reconnaissance', value: 'recon', description: 'Active port scanning & service discovery (Nmap)' },
          { name: 'Web Enumeration', value: 'web', description: 'Directory fuzzing & vulnerability crawling (Gobuster, Nuclei)' },
          { name: 'Exploitation & Shells', value: 'exploit', description: 'Custom Python & Stateful Bash Execution' },
          { name: 'Heuristic Analysis', value: 'heuristic', description: 'Binary reversing & strings extraction' },
          { name: 'Social Engineering', value: 'phishing', description: 'Spear-Phishing Generation & Evilginx2 AitM' },
          { name: 'Hardware Attacks', value: 'hardware', description: 'Hak5 Rubber Ducky Payload Generation' },
          { name: 'GUI Automation', value: 'gui', description: 'Headless Browser Control (Playwright) for CAPTCHAs & Login Flows' },
          new Separator("--- APT Capabilities ---"),
          { name: 'Zero-Day Fuzzing Engine', value: 'fuzzer', description: 'Autonomous mutational fuzzing for unknown vulnerabilities' },
          { name: 'Dynamic Payload Compiler', value: 'payloads', description: 'AES-encrypted, Syscall C#/Go malware droppers for AV Evasion' },
          { name: 'Autonomous Swarm Pivoting', value: 'pivot', description: 'Dynamic Ligolo/Chisel deployment for internal Active Directory compromise' },
          new Separator(),
          { name: 'Go Back', value: 'back' }
        ]
      });
      rl.resume();
      console.log("");
      continue;
    }

    if (command.toLowerCase() === "/new") {
      orchestrator.newSession();
      console.log(chalk.green("\n  [+] ") + chalk.gray("Memory wiped. New session started.\n"));
      continue;
    }

    if (command.toLowerCase() === "/stealth on" || command.toLowerCase() === "/stealth") {
      if (orchestrator.opsecManager.isStealthModeActive()) {
        orchestrator.opsecManager.disableStealthMode();
        console.log(chalk.yellow("\n  ⬡ ") + chalk.white("Stealth mode ") + chalk.red("disabled") + chalk.gray(" — aggressive scanning active\n"));
      } else {
        orchestrator.opsecManager.enableStealthMode();
        console.log(chalk.yellow("\n  ⬡ ") + chalk.white("Stealth mode ") + chalk.green("enabled") + chalk.gray(" — timing jitter on, noisy scanners suppressed\n"));
      }
      continue;
    }

    if (command.toLowerCase() === "/stealth off") {
      orchestrator.opsecManager.disableStealthMode();
      console.log(chalk.yellow("\n  ⬡ ") + chalk.white("Stealth mode ") + chalk.red("disabled") + chalk.gray(" — aggressive scanning active\n"));
      continue;
    }

    if (command.toLowerCase().startsWith("/swarm ")) {
      const swarmMission = command.substring(7).trim();
      if (!swarmMission) {
        console.log(chalk.red("\n  [x] ") + chalk.gray("Usage: /swarm <objective>\n"));
        continue;
      }
      
      const { SwarmCommander } = await import("../agent/swarm-commander");
      const commander = new SwarmCommander();
      
      const swarmSpinner = ora({ text: chalk.gray("Dispatching parallel agents..."), color: "cyan", spinner: "bouncingBar" }).start();
      try {
        const result = await commander.executeSwarm(swarmMission);
        swarmSpinner.stop();
        console.log(chalk.cyan("\n  ╭─ Swarm Report ─────────────────────────────────────────"));
        console.log(result.split('\n').map((line: string) => chalk.gray("  │ ") + line).join('\n'));
        console.log(chalk.cyan("  ╰───────────────────────────────────────────────────────\n"));
      } catch (error: any) {
        swarmSpinner.stop();
        console.log(chalk.red("\n  [x] ") + chalk.gray(`Swarm error: ${error.message}\n`));
      }
      continue;
    }

    if (command.toLowerCase() === "/report") {
      const reportSpinner = ora({ text: chalk.gray("Generating report..."), color: "cyan", spinner: "bouncingBar" }).start();
      try {
        const { ReportGenerator } = await import("../core/report-generator");
        const generator = new ReportGenerator(orchestrator.getMemoryGraph());
        const { textPath, docPath } = await generator.generateReport();
        reportSpinner.stop();
        console.log(chalk.green("\n  [+] ") + chalk.white("Raw Markdown saved to: ") + chalk.underline.gray(textPath));
        console.log(chalk.green("  [+] ") + chalk.white("Styled Report saved to:  ") + chalk.underline.cyan(docPath) + "\n");
      } catch (error: any) {
        reportSpinner.stop();
        console.log(chalk.red("\n  [x] ") + chalk.gray(`Report error: ${error.message}\n`));
      }
      continue;
    }

    if (command.toLowerCase() === "/health") {
      const { HealthChecker } = await import("../core/health-checker");
      const checker = new HealthChecker();
      await checker.runDiagnostics();
      continue;
    }

    if (command.startsWith("/")) {
      console.log(chalk.yellow("\n  [?] ") + chalk.gray(`Unknown command. Type `) + chalk.cyan("/help") + chalk.gray(" for available commands.\n"));
      continue;
    }

    // ── Agent Execution ──────────────────────────────────────
    console.log("");
    let spinner = ora({
      text: chalk.gray("Initializing neural pathways..."),
      color: "cyan",
      spinner: "bouncingBar",
    }).start();

    let lastTool = "";

    try {
      const result = await orchestrator.execute(command, (toolName, args) => {
        if (toolName === "thought") {
          let thoughtText = args.thought.substring(0, 85).replace(/\n/g, ' '); 
          if (args.thought.length > 85) thoughtText += "...";
          spinner.text = chalk.gray(`Thinking: `) + chalk.white(thoughtText);
        } else {
          spinner.stop();
          if (lastTool) {
            console.log(chalk.cyan(`  └─ `) + chalk.green(`[✓] Execution Complete`));
          }
          lastTool = toolName;
          console.log(chalk.cyan(`  ┌─ `) + chalk.bold.white(`[⚙️] Executing Module: `) + chalk.magenta(toolName));
          if (Object.keys(args).length > 0) {
            let argStr = JSON.stringify(args).substring(0, 100);
            if (argStr.length >= 100) argStr += "...";
            console.log(chalk.cyan(`  ├─ `) + chalk.gray(`Params: ${argStr}`));
          }
          spinner = ora({
            text: chalk.gray(`  ├─ Awaiting output...`),
            color: "cyan",
            spinner: "bouncingBar",
          }).start();
        }
      });
      
      spinner.stop();
      if (lastTool) {
        console.log(chalk.cyan(`  └─ `) + chalk.green(`[✓] Execution Complete`));
      }
      
      console.log(chalk.cyan("\n  ╭─ DrogonClaw ──────────────────────────────────────────"));
      
      const rendered = await marked.parse(result);
      // Stream the output with a typewriter effect
      await streamText(rendered.trimEnd());
      
      console.log(chalk.cyan("  ╰───────────────────────────────────────────────────────\n"));
    } catch (error: any) {
      spinner.stop();
      if (error.name === "HitLPauseError") {
        console.log(chalk.yellow(`\n  [?] Agent execution suspended. Awaiting your input...\n`));
      } else {
        console.log(chalk.red(`\n  [x] Error: ${error.message}\n`));
      }
    }
  }
}
