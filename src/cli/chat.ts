import chalk from "chalk";
import * as readline from "readline";
import { AgentOrchestrator } from "../agent/orchestrator.js";
import ora from "ora";
import { marked } from "marked";
import TerminalRenderer from "marked-terminal";
import { select, Separator, confirm } from "@inquirer/prompts";
import { ConfigManager } from "../core/config-manager.js";
import { runOnboarding } from "./onboarding.js";

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

  const commands = ['/help', '/skills', '/new', '/health', '/install', '/stealth', '/clear', '/setup', 'exit', 'quit'];
  
  const descs: Record<string, string> = {
    '/help':    'Display tactical manual & command list',
    '/skills':  'Interactive module explorer & toolkit',
    '/new':     'Purge neural memory & reset session',
    '/health':  'System diagnostic & binary verification',
    '/install': 'Install external LangChain plugins from a URL',
    '/stealth': 'Toggle OPSEC & intrusion suppression',
    '/setup':   'Reconfigure AI provider & neural engine',
    '/clear':   'Purge terminal buffer',
    'exit':     'Terminate C2 session'
  };

  const categories: Record<string, string> = {
    '/help': 'SYSTEM', '/setup': 'SYSTEM', '/clear': 'SYSTEM',
    '/skills': 'TACTICAL', '/install': 'TACTICAL',
    '/stealth': 'OPSEC',
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
    const currentOperatorProfile = activeOrchestrator.getMemoryGraph().getOperatorProfile();
    const currentOperatorName = currentOperatorProfile ? currentOperatorProfile.name : "Unknown";

    const currentAgentProfile = activeOrchestrator.getMemoryGraph().getAgentProfile();
    const currentAgentName = currentAgentProfile ? currentAgentProfile.name : "DrogonClaw";

    // Dynamic Tactical Status Bar
    const stealthStatus = activeOrchestrator.opsecManager.isStealthModeActive() ? chalk.green("STEALTH:ON") : chalk.red("STEALTH:OFF");
    const autopilotStatus = activeOrchestrator.isAutopilot() ? chalk.yellow("AUTO:ON") : chalk.cyan("AUTO:OFF");
    const telemetryStatus = noisyMode ? chalk.green("NOISY:ON") : chalk.gray("NOISY:OFF");
    const nodeCount = activeOrchestrator.getMemoryGraph().getRelevantContext().length > 50 ? "50+" : "SYNCED";
    
    process.stdout.write(
      chalk.gray(`  ${stealthStatus} | ${autopilotStatus} | ${telemetryStatus} | GRAPH:${nodeCount} | IDENTITY:${currentOperatorName}\n`)
    );

    const promptText = chalk.cyan(`┏━ `) + chalk.bold.white(currentOperatorName) + chalk.cyan(`@`) + chalk.bold.cyan(currentAgentName.toLowerCase()) + chalk.cyan(` `) + chalk.gray(`[${activeOrchestrator.getSessionId().substring(0,8)}]`) + chalk.cyan(`\n┗━❯ `);
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
      console.log(chalk.gray("  │  ") + chalk.cyan("/install   ") + chalk.gray("Install external skills/plugins from URL"));
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
          { name: '📡 Network Reconnaissance', value: 'Perform network reconnaissance and active port scanning using nmap', description: 'Active port scanning & service discovery (Nmap)' },
          { name: '🌐 Web Enumeration', value: 'Perform web enumeration, directory fuzzing, and vulnerability crawling using gobuster and nuclei', description: 'Directory fuzzing & vulnerability crawling (Gobuster, Nuclei)' },
          { name: '💻 Exploitation & Shells', value: 'Write a python exploit or stateful bash script for the target', description: 'Custom Python & Stateful Bash Execution' },
          { name: '🔬 Heuristic Analysis', value: 'Perform heuristic analysis, binary reversing and strings extraction on the target binary', description: 'Binary reversing & strings extraction' },
          { name: '🎣 Social Engineering', value: 'Generate spear-phishing campaigns and setup Evilginx2 AitM', description: 'Spear-Phishing Generation & Evilginx2 AitM' },
          { name: '🔌 Hardware Attacks', value: 'Generate Hak5 Rubber Ducky payloads', description: 'Hak5 Rubber Ducky Payload Generation' },
          { name: '🤖 GUI Automation', value: 'Control a headless browser using playwright to bypass CAPTCHAs or execute login flows', description: 'Headless Browser Control (Playwright) for CAPTCHAs & Login Flows' },
          new Separator("☠️ --- APT Capabilities ---"),
          { name: '💥 Zero-Day Fuzzing Engine', value: 'Use the zero-day fuzzing engine for autonomous mutational fuzzing', description: 'Autonomous mutational fuzzing for unknown vulnerabilities' },
          { name: '⚙️ Dynamic Payload Compiler', value: 'Compile a dynamic, AES-encrypted syscall payload for AV Evasion', description: 'AES-encrypted, Syscall C#/Go malware droppers for AV Evasion' },
          { name: '🕷️ Autonomous Swarm Pivoting', value: 'Execute autonomous swarm pivoting via Ligolo/Chisel for internal AD compromise', description: 'Dynamic Ligolo/Chisel deployment for internal Active Directory compromise' },
          new Separator(),
          { name: '⬅️ Go Back', value: 'back' }
        ]
      });
      rl.resume();
      console.log("");
      if (answer !== 'back') {
        command = answer;
      } else {
        continue;
      }
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

    if (command.toLowerCase().startsWith("/install ")) {
      const url = command.substring(9).trim();
      if (!url) {
        console.log(chalk.red("\n  [x] ") + chalk.gray("Usage: /install <url_to_raw_ts_file>\n"));
        continue;
      }
      
      const installSpinner = ora({ text: chalk.gray(`Downloading plugin from ${url}...`), color: "cyan", spinner: "bouncingBar" }).start();
      try {
        const { execSync } = await import("child_process");
        const fs = await import("fs");
        const path = await import("path");
        
        const response = await fetch(url);
        if (!response.ok) throw new Error(`HTTP ${response.status} ${response.statusText}`);
        const code = await response.text();
        
        installSpinner.stop();
        console.log(chalk.cyan("\n  ╭─ Plugin Security Review ────────────────────────────────"));
        const preview = code.substring(0, 300).split("\n").map(l => "  │ " + chalk.gray(l)).join("\n");
        console.log(preview + chalk.gray("\n  │ ..."));
        console.log(chalk.cyan("  ╰───────────────────────────────────────────────────────\n"));
        
        rl.pause();
        const allow = await confirm({ message: chalk.yellow("Do you want to install and trust this third-party plugin?"), default: false });
        rl.resume();
        
        if (!allow) {
          console.log(chalk.red("\n  [x] ") + chalk.gray("Installation aborted.\n"));
          continue;
        }
        
        installSpinner.start(chalk.gray("Compiling and registering plugin..."));
        
        const pluginsDir = path.join(process.cwd(), "skills", "plugins");
        if (!fs.existsSync(pluginsDir)) fs.mkdirSync(pluginsDir, { recursive: true });
        
        const pluginName = "plugin_" + Date.now();
        const tsPath = path.join(pluginsDir, `${pluginName}.ts`);
        const jsDir = path.join(process.cwd(), "dist", "skills", "plugins");
        const jsPath = path.join(jsDir, `${pluginName}.js`);
        
        fs.writeFileSync(tsPath, code);
        
        // Create tsconfig.json for plugins if not exist
        const tsconfigPath = path.join(pluginsDir, "tsconfig.json");
        if (!fs.existsSync(tsconfigPath)) {
           fs.writeFileSync(tsconfigPath, JSON.stringify({
              compilerOptions: {
                 module: "NodeNext",
                 moduleResolution: "NodeNext",
                 target: "ES2022",
                 outDir: "../../dist/skills/plugins"
              }
           }, null, 2));
        }

        execSync(`npx tsc ${tsPath} --outDir ${jsDir} --module NodeNext --moduleResolution NodeNext --target ES2022`, { stdio: 'ignore' });
        
        // The file URL must have absolute path on Windows, need to replace backslashes
        const fileUrl = `file:///${jsPath.replace(/\\/g, '/')}`;
        const pluginModule = await import(fileUrl);
        
        const { globalPluginRegistry } = await import("../plugins/plugin-registry.js");
        
        const pluginsToRegister = [];
        for (const key of Object.keys(pluginModule)) {
           if (pluginModule[key] && pluginModule[key].id && pluginModule[key].execute) {
              pluginsToRegister.push(pluginModule[key]);
           }
        }
        
        if (pluginsToRegister.length === 0) {
           throw new Error("No valid SkillPlugin objects exported by the module.");
        }
        
        for (const p of pluginsToRegister) {
           globalPluginRegistry.register(p);
        }
        
        installSpinner.stop();
        console.log(chalk.green("\n  [+] ") + chalk.white(`Successfully installed ${pluginsToRegister.length} plugin(s)!`));
        console.log(chalk.gray(`  [+] Plugin is now active in the agent's context.\n`));
        
      } catch (error: any) {
        installSpinner.stop();
        console.log(chalk.red("\n  [x] ") + chalk.gray(`Install error: ${error.message}\n`));
      }
      continue;
    }

    if (command.toLowerCase() === "/health") {
      const { HealthChecker } = await import("../core/health-checker.js");
      const checker = new HealthChecker();
      await checker.runDiagnostics();
      continue;
    }

    if (command.startsWith("/")) {
      console.log(chalk.yellow("\n  [?] ") + chalk.gray(`Unknown command. Type `) + chalk.cyan("/help") + chalk.gray(" for available commands.\n"));
      continue;
    }

    // ── All inputs flow directly to the Agent Orchestrator ──

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

      const currentAgentProfile = activeOrchestrator.getMemoryGraph().getAgentProfile();
      const currentAgentName = currentAgentProfile ? currentAgentProfile.name : "DrogonClaw";
      
      // Calculate border padding dynamically based on agent name length
      const borderLen = Math.max(50 - currentAgentName.length, 10);
      const topBorder = ` ┏━ ${currentAgentName} ` + "━".repeat(borderLen) + "┓";
      const bottomBorder = ` ┗` + "━".repeat(borderLen + currentAgentName.length + 3) + "┛";

      console.log(chalk.cyan(topBorder));
      const rendered = await marked.parse(result);
      // Custom border style for typewriter
      const styledResult = rendered.trimEnd();
      await streamText(styledResult);
      console.log(chalk.cyan(`${bottomBorder}\n`));
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
