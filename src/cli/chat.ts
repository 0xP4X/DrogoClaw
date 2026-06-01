import chalk from "chalk";
import * as readline from "readline";
import { AgentOrchestrator } from "../agent/orchestrator";
import ora from "ora";
import { marked } from "marked";
import TerminalRenderer from "marked-terminal";
import { select, Separator } from "@inquirer/prompts";
import { ConfigManager } from "../core/config-manager";
import { runOnboarding } from "./onboarding";

marked.setOptions({
  // @ts-ignore
  renderer: new TerminalRenderer({
    reflowText: true,
    width: Math.min(80, process.stdout.columns ? process.stdout.columns - 4 : 80),
  })
});

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

export function colorizeJson(obj: any, indent: string = ""): string {
  if (!obj || typeof obj !== "object") return chalk.gray(String(obj));
  return Object.entries(obj)
    .map(([k, v]) => {
      const key = chalk.cyan(`"${k}"`) + chalk.white(":");
      let val = "";
      if (typeof v === "string") val = chalk.green(`"${v}"`);
      else if (typeof v === "number" || typeof v === "boolean") val = chalk.yellow(String(v));
      else if (typeof v === "object") val = colorizeJson(v, indent + "  ");
      else val = chalk.gray(String(v));
      return `${indent}  ${key} ${val}`;
    })
    .join("\n");
}

export function flashWarning(message: string): void {
  // Clear line, red background, white text, clear background
  process.stdout.write(`\r\x1b[K\x1b[41m\x1b[37m ${message} \x1b[0m\n`);
}

// Helper for cyberpunk typewriter streaming effect
async function streamText(text: string, speedMs: number = 10): Promise<void> {
  const lines = text.split('\n');
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (!line.trim() && i === lines.length - 1) continue;
    
    process.stdout.write(chalk.cyan(" ┃  "));
    
    // Check if the line has ANSI escape codes (e.g. from marked-terminal)
    const hasAnsi = /\u001b\[[0-9;]*m/.test(line);
    
    if (hasAnsi || line.length < 2) {
      process.stdout.write(line + '\n');
    } else {
      // Cyberpunk typewriter with block cursor
      for (let j = 0; j < line.length; j++) {
        const char = line[j];
        process.stdout.write(chalk.white(char) + chalk.cyan('█'));
        
        let delay = speedMs;
        if (['.', ',', ':', ';'].includes(char)) delay = 80;
        
        if (delay > 0) await sleep(delay);
        process.stdout.write('\b \b'); // erase block cursor
      }
      process.stdout.write('\n');
    }
  }
}

export async function startChatSession(orchestrator: AgentOrchestrator): Promise<void> {
  let activeOrchestrator = orchestrator;
  const profile = activeOrchestrator.getMemoryGraph().getOperatorProfile();
  const operatorName = profile ? profile.name : "Unknown Operator";

  // Dynamic status summary (quick glance)
  const drawBox = (lines: string[], title?: string) => {
    const pad = 2;
    const width = Math.max(...lines.map((l) => l.length), (title?.length || 0) + 4);
    
    // Use heavy border characters for "unique" design
    const top = title 
      ? `┏━ ${chalk.bold(title)} ${'━'.repeat(width - title.length + pad * 2 - 4)}┓`
      : `┏${'━'.repeat(width + pad * 2)}┓`;
    const bottom = `┗${'━'.repeat(width + pad * 2)}┛`;
    const middle = lines.map((l) => `┃${' '.repeat(pad)}${l.padEnd(width)}${' '.repeat(pad)}┃`).join('\n');
    return chalk.cyan(top + '\n' + middle + '\n' + bottom);
  };

  const renderStatus = (orch: AgentOrchestrator) => {
    const provider = (ConfigManager.get("AI_PROVIDER") || process.env.AI_PROVIDER || "unset").toString();
    const model = (ConfigManager.get("OLLAMA_MODEL_NAME") || ConfigManager.get("OPENAI_MODEL_NAME") || process.env.OLLAMA_MODEL_NAME || process.env.OPENAI_MODEL_NAME || "unset").toString();
    const stealth = orch.opsecManager.isStealthModeActive() ? chalk.green('ACTIVE') : chalk.gray('SILENT');
    const autopilot = orch.isAutopilot() ? chalk.yellow('ENABLED') : chalk.gray('MANUAL');
    const nodes = orch.getMemoryGraph().getNodesCount();

    const lines = [];
    lines.push(`${chalk.gray('CORE   ')} ❯ ${chalk.white(provider)}:${chalk.cyan(model)}`);
    lines.push(`${chalk.gray('OPSEC  ')} ❯ Stealth:${stealth} | Autopilot:${autopilot}`);
    lines.push(`${chalk.gray('INTEL  ')} ❯ ${chalk.magenta(nodes)} entries in neural graph`);

    process.stdout.write("\n" + drawBox(lines, "DROGONCLAW C2 MONITOR") + "\n\n");
  };

  renderStatus(activeOrchestrator);
  console.log(chalk.gray("  Welcome, ") + chalk.bold.white(operatorName) + chalk.gray(". Type '/' for commands or 'exit' to quit.\n"));

  const commands = ['/help', '/skills', '/new', '/health', '/swarm', '/stealth', '/report', '/clear', '/setup', 'exit', 'quit'];
  
  const descs: Record<string, string> = {
    '/help':    'Display tactical manual & command list',
    '/skills':  'Interactive module explorer & toolkit',
    '/new':     'Purge neural memory & reset session',
    '/health':  'System diagnostic & binary verification',
    '/swarm':   'Initialize multi-agent neural swarm',
    '/stealth': 'Toggle OPSEC & intrusion suppression',
    '/report':  'Generate tactical operations report',
    '/setup':   'Reconfigure AI provider & neural engine',
    '/clear':   'Purge terminal buffer',
    'exit':     'Terminate C2 session'
  };

  const categories: Record<string, string> = {
    '/help': 'SYSTEM', '/setup': 'SYSTEM', '/clear': 'SYSTEM',
    '/skills': 'TACTICAL', '/swarm': 'TACTICAL',
    '/stealth': 'OPSEC',
    '/report': 'INTEL',
    '/new': 'CORE', '/health': 'CORE',
    'exit': 'SYSTEM'
  };

  const commandPalette = async (): Promise<string | null> => {
    rl.pause();
    try {
      const ranked = commands
        .filter(c => c !== 'quit')
        .sort((a,b) => {
          const catA = categories[a] || 'Z';
          const catB = categories[b] || 'Z';
          return catA.localeCompare(catB) || a.localeCompare(b);
        });

      inPalette = true;
      const hideCursor = () => process.stdout.write('\x1B[?25l');
      const showCursor = () => process.stdout.write('\x1B[?25h');
      hideCursor();

      let selected = 0;
      let lastCategory = '';
      
      const render = () => {
        lastCategory = '';
        const lines: string[] = [];
        lines.push(chalk.cyan(' ┏━ COMMAND LAUNCHER ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓'));
        
        ranked.forEach((c, i) => {
          const cat = categories[c] || 'MISC';
          if (cat !== lastCategory) {
            lines.push(chalk.gray(` ┃  [ ${cat} ]`));
            lastCategory = cat;
          }

          const isSelected = i === selected;
          const prefix = isSelected ? chalk.cyan(' ┃  ❯ ') : chalk.gray(' ┃    ');
          const accel = i < 9 ? chalk.gray(`${i+1}. `) : '   ';
          const cmdCol = isSelected ? chalk.bold.white(c.padEnd(10)) : chalk.cyan(c.padEnd(10));
          const descCol = isSelected ? chalk.white(descs[c]) : chalk.gray(descs[c]);
          
          lines.push(`${prefix}${accel}${cmdCol} ${chalk.gray('·')} ${descCol}`);
        });
        
        lines.push(chalk.gray(' ┃'));
        lines.push(chalk.gray(` ┃  ${chalk.cyan('↑/↓')} navigate  ${chalk.cyan('↵')} select  ${chalk.cyan('esc')} cancel`));
        lines.push(chalk.cyan(' ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛'));
        
        const output = lines.join('\n') + '\n';
        process.stdout.write(output);
        return lines.length;
      };

      const printed = render();

      const selectionPromise = new Promise<string | null>((resolve) => {
        const keyHandler = (s: string, k: any) => {
          if (!k) return;
          if (k.name === 'up' || k.name === 'k') {
            selected = (selected - 1 + ranked.length) % ranked.length;
          } else if (k.name === 'down' || k.name === 'j') {
            selected = (selected + 1) % ranked.length;
          } else if (k.name === 'return' || k.name === 'enter') {
             cleanup();
             return resolve(ranked[selected]);
          } else if (k.name === 'escape' || k.name === 'q') {
            cleanup();
            return resolve(null);
          } else if (/^[1-9]$/.test(s)) {
            const n = Number(s);
            if (n >= 1 && n <= ranked.length) {
              cleanup();
              return resolve(ranked[n-1]);
            }
          }

          process.stdout.write(`\x1B[${printed}A`);
          render();
        };

        const cleanup = () => {
          process.stdin.off('keypress', keyHandler);
          showCursor();
          inPalette = false;
        };

        process.stdin.on('keypress', keyHandler as any);
      });

      const selection = await selectionPromise;
      return selection;
    } finally {
      rl.resume();
    }
  };
  
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
  let lastHintCount = 0;
  let inPalette = false;
  let noisyMode = false;
  let isExecuting = false;

  const clearHints = () => {
    if (showingHints && lastHintCount > 0) {
      let clearStr = '\x1B[s';
      for (let i = 0; i < lastHintCount; i++) {
        clearStr += '\x1B[1E\x1B[K';
      }
      clearStr += '\x1B[u';
      process.stdout.write(clearStr);
      showingHints = false;
      lastHintCount = 0;
    }
  };

  process.stdin.on('keypress', (str, key) => {
    if (inPalette) return;
    if (!key) return; // Guard for undefined keys

    if (key.ctrl && key.name === 'c') {
      if (isExecuting) {
        process.stdout.write(chalk.yellow("\n[!] Gracefully aborting execution sequence...\n"));
        activeOrchestrator.abortCurrentExecution();
        return;
      }
      process.exit(0);
    }
    
    // Clear hints on enter or backspace to avoid ghosts
    if (key.name === 'return' || key.name === 'enter' || key.name === 'backspace') {
      clearHints();
    }

    // Use a small delay to let readline update rl.line
    setTimeout(() => {
      // Check if rl is available and has line property
      if (!rl) return;
      const line = rl.line || "";
      
      if (line.startsWith('/') && line.length > 0) {
        const hits = commands.filter((c) => c.startsWith(line.split(' ')[0]));
        
        if (hits.length > 0 && hits.length < commands.length) {
          clearHints();
          showingHints = true;
          lastHintCount = hits.length;
          
          let hintOutput = '\x1B[s'; // Save cursor
          hits.forEach((h, idx) => {
            const isFirst = idx === 0;
            const glyph = isFirst ? '╭' : (idx === hits.length - 1 ? '╰' : '├');
            const desc = descs[h] || '';
            const padding = " ".repeat(Math.max(2, 12 - h.length));
            hintOutput += `\x1B[1E\x1B[K  ${chalk.cyan(glyph)} ${chalk.white(h)}${padding}${chalk.gray(desc)}`;
          });
          hintOutput += '\x1B[u'; // Restore cursor
          process.stdout.write(hintOutput);
        } else {
          clearHints();
        }
      } else {
        clearHints();
      }
    }, 5);
  });

  const ask = (query: string): Promise<string> => new Promise((resolve) => rl.question(query, (ans) => {
    clearHints();
    resolve(ans);
  }));

  // Auto-Greeting if Operator is Unknown
  if (operatorName === "Unknown Operator") {
    console.log("");
    let spinner = ora({
      text: chalk.gray("Initializing neural pathways..."),
      color: "cyan",
      spinner: "bouncingBar",
    }).start();

    try {
      const result = await activeOrchestrator.execute("The operator has booted up the system but has not provided their name. Introduce yourself aggressively, explain your capabilities briefly, and demand their hacker alias.", (toolName, args) => {
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
    const currentProfile = activeOrchestrator.getMemoryGraph().getOperatorProfile();
    const currentOperatorName = currentProfile ? currentProfile.name : "Unknown Operator";

    // Dynamic Tactical Status Bar
    const stealthStatus = activeOrchestrator.opsecManager.isStealthModeActive() ? chalk.green("STEALTH:ON") : chalk.red("STEALTH:OFF");
    const autopilotStatus = activeOrchestrator.isAutopilot() ? chalk.yellow("AUTO:ON") : chalk.cyan("AUTO:OFF");
    const telemetryStatus = noisyMode ? chalk.green("NOISY:ON") : chalk.gray("NOISY:OFF");
    const nodeCount = activeOrchestrator.getMemoryGraph().getRelevantContext().length > 50 ? "50+" : "SYNCED";
    
    process.stdout.write(
      chalk.gray(`  ${stealthStatus} | ${autopilotStatus} | ${telemetryStatus} | GRAPH:${nodeCount} | IDENTITY:${currentOperatorName}\n`)
    );

    const promptText = chalk.cyan(`┏━ `) + chalk.bold.white(currentOperatorName) + chalk.cyan(`@drogonclaw `) + chalk.gray(`[${activeOrchestrator.getSessionId().substring(0,8)}]`) + chalk.cyan(`\n┗━❯ `);
    const prompt = await ask(promptText + chalk.white(""));

    let command = prompt.trim();
    if (command === '/') {
      const picked = await commandPalette();
      if (!picked) continue;
      command = picked;
    }
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
      activeOrchestrator.newSession();
      console.log(chalk.green("\n  [+] ") + chalk.gray("Memory wiped. New session started.\n"));
      continue;
    }

    if (command.toLowerCase() === "/setup") {
      rl.pause();
      console.log(chalk.cyan("\n  [*] Launching setup wizard...\n"));
      try {
        await runOnboarding();
        ConfigManager.load();

        const refreshedOrchestrator = new AgentOrchestrator();
        const spinner = ora({ text: chalk.gray("Reinitializing agent core..."), color: "cyan", spinner: "bouncingBar" }).start();
        const initialized = await refreshedOrchestrator.initialize();
        spinner.stop();

        if (!initialized || !refreshedOrchestrator.isReady()) {
          console.log(chalk.red("\n  [x] Reconfiguration failed. Keeping the current session active.\n"));
        } else {
          activeOrchestrator = refreshedOrchestrator;
          console.log(chalk.green("\n  [+] Reconfiguration complete. The new model is active.\n"));
        }
      } finally {
        rl.resume();
      }
      continue;
    }

    if (command.toLowerCase() === "/stealth on" || command.toLowerCase() === "/stealth") {
      if (activeOrchestrator.opsecManager.isStealthModeActive()) {
        activeOrchestrator.opsecManager.disableStealthMode();
        flashWarning(" OPSEC STEALTH DISABLED ");
        console.log(chalk.yellow("\n  ⬡ ") + chalk.white("Stealth mode ") + chalk.red("disabled") + chalk.gray(" — aggressive scanning active\n"));
      } else {
        activeOrchestrator.opsecManager.enableStealthMode();
        console.log(chalk.yellow("\n  ⬡ ") + chalk.white("Stealth mode ") + chalk.green("enabled") + chalk.gray(" — timing jitter on, noisy scanners suppressed\n"));
      }
      continue;
    }

    if (command.toLowerCase() === "/stealth off") {
      activeOrchestrator.opsecManager.disableStealthMode();
      flashWarning(" OPSEC STEALTH DISABLED ");
      console.log(chalk.yellow("\n  ⬡ ") + chalk.white("Stealth mode ") + chalk.red("disabled") + chalk.gray(" — aggressive scanning active\n"));
      continue;
    }

    if (command.toLowerCase() === "/noisy") {
      noisyMode = !noisyMode;
      const status = noisyMode ? chalk.green("ENABLED") : chalk.red("DISABLED");
      console.log(chalk.yellow("\n  ⚡ ") + chalk.white("Noisy mode ") + status + chalk.gray(" — live neural telemetry stream\n"));
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
        const generator = new ReportGenerator(activeOrchestrator.getMemoryGraph());
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

    // ── Pre-Execution Fast Paths (Zero Latency) ──────────
    const fastCommand = command.trim().toLowerCase();
    const commonGreetings = ["hi", "hey", "hello", "yo", "hola", "help"];
    const identityQuestions = ["who are you", "what are you", "what is your name", "what's my name", "who am i", "what is my name"];
    
    if (commonGreetings.includes(fastCommand) || fastCommand.length < 3) {
       console.log(chalk.cyan("\n  ╭─ DrogonClaw ──────────────────────────────────────────"));
       console.log("  Greetings, operator. I am synchronized and ready for deployment.");
       console.log(chalk.cyan("  ╰───────────────────────────────────────────────────────\n"));
       continue;
    }

    if (identityQuestions.some(q => fastCommand.includes(q))) {
       const operator = activeOrchestrator.getMemoryGraph().getOperatorProfile();
       const name = operator?.name || "seed";
       console.log(chalk.cyan("\n  ╭─ Identity Protocol ───────────────────────────────────"));
       if (fastCommand.includes("my name") || fastCommand.includes("who am i")) {
         console.log(`  Target Operator Alias: ${chalk.bold.green(name)}`);
       } else {
         console.log("  Framework: DrogonClaw v0.2.0 (High-Precision Autonomous C2)");
       }
       console.log(chalk.cyan("  ╰───────────────────────────────────────────────────────\n"));
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
    isExecuting = true;

    try {
      const result = await activeOrchestrator.execute(command, (toolName, args) => {
        if (toolName === "thought") {
          if (noisyMode) {
            spinner.stop();
            console.log(chalk.gray(`  ┃  ┌─ `) + chalk.yellow(`[🧠] Neural Processing`));
            const thoughtLines = args.thought.split('\n');
            thoughtLines.forEach((l: string) => {
               if (l.trim()) console.log(chalk.gray(`  ┃  │ `) + chalk.italic.white(l));
            });
            spinner.start(chalk.gray("Processing..."));
          } else {
            let thoughtText = args.thought.substring(0, 85).replace(/\n/g, ' '); 
            if (args.thought.length > 85) thoughtText += "...";
            spinner.text = chalk.gray(`Thinking: `) + chalk.white(thoughtText);
          }
        } else if (toolName === "status") {
          if (noisyMode) {
            spinner.stop();
            console.log(chalk.cyan(`  ┃  ⚡ `) + chalk.white(args.message));
            spinner.start(chalk.gray("Executing..."));
          } else {
            const icon = args.message.includes("success") ? "✔" : "⚡";
            spinner.text = chalk.cyan(`[${icon}] `) + chalk.white(args.message);
          }
        } else {
          spinner.stop();
          if (lastTool) {
            console.log(chalk.cyan(`  ┃  └─ `) + chalk.green(`[✓] Execution Complete`));
          }
          lastTool = toolName;
          console.log(chalk.cyan(`  ┃  ┌─ `) + chalk.bold.white(`[⚙️] Executing Module: `) + chalk.magenta(toolName));
          if (Object.keys(args).length > 0) {
            console.log(chalk.cyan(`  ┃  ├─ `) + chalk.gray(`Params:`));
            const coloredArgs = colorizeJson(args, "  ┃    ");
            console.log(coloredArgs);
          }
          
          let spinnerText = chalk.gray(`  ┃  ├─ `) + chalk.cyan(`Executing ${toolName}...`);
          if (toolName === "nmap_scan") spinnerText = chalk.gray(`  ┃  ├─ `) + chalk.red(`Blasting target ports...`);
          if (toolName.includes("payload") || toolName.includes("exploit") || toolName.includes("deserialization")) spinnerText = chalk.gray(`  ┃  ├─ `) + chalk.red(`Compiling weaponized payload...`);
          if (toolName.includes("crypto")) spinnerText = chalk.gray(`  ┃  ├─ `) + chalk.magenta(`Crunching crypto mathematics...`);
          
          spinner = ora({
            text: spinnerText,
            color: "cyan",
            spinner: "dots",
          }).start();
        }
      });
      
      spinner.stop();
      if (lastTool) {
        console.log(chalk.cyan(`  ┃  └─ `) + chalk.green(`[✓] Execution Complete`));
      }

      console.log(chalk.cyan(" ┏━ DrogonClaw ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓"));
      const rendered = await marked.parse(result);
      // Custom border style for typewriter
      const styledResult = rendered.trimEnd();
      await streamText(styledResult);
      console.log(chalk.cyan(" ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n"));
    } catch (error: any) {
      spinner.stop();
      if (error.name === "HitLPauseError") {
        flashWarning(" HUMAN-IN-THE-LOOP SUSPENSION ");
        console.log(chalk.yellow(`\n  [?] Agent execution suspended. Awaiting your input...\n`));
      } else {
        console.log(chalk.red(`\n  [x] Error: ${error.message}\n`));
      }
    } finally {
      isExecuting = false;
    }
  }
}
