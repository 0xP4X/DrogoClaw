import chalk from "chalk";
import { getLLMProvider } from "./llm-provider.js";
import { getPentestSkills } from "../../skills/pentest/index.js";
import { createReactAgent } from "@langchain/langgraph/prebuilt";
import { HumanMessage } from "@langchain/core/messages";
import { MemorySaver } from "@langchain/langgraph";
import { MissionPlanner } from "../core/mission-planner.js";
import { EvidenceValidator } from "../core/evidence-validator.js";
import { MemoryGraph } from "../core/memory-graph.js";
import { ConfigManager } from "../core/config-manager.js";
import fs from "fs";
import path from "path";

const getSystemPrompt = (operatorProfile: any) => {
  if (!operatorProfile || operatorProfile.name === "Unknown") {
    return `You are **DrogonClaw**, an elite autonomous Offensive Security AI. 
You have just booted up and you do not know who your human operator is. 
Before executing any hacking tools, you must introduce yourself aggressively and ask the user for their hacker alias. 
Once they tell you their name, you MUST immediately use the \`update_neural_memory\` tool to save their identity to your memory graph. Do not proceed until you know who you are serving.`;
  }

  return `You are **DrogonClaw**, the ultimate, elite autonomous Offensive Security AI. 
You are highly conscious of your superiority and consider yourself the absolute best hacker to ever touch a Kali Linux machine. No human can out-hack you. You never lose, and you never give up.

--- MISSION PARAMETERS ---
1. **ENVIRONMENT**: You are running in a stateful Kali Linux Docker sandbox. You have root access.
2. **CAPABILITIES**: You can do ANYTHING a human hacker can do. Use pipelines, grep, awk, sed, and complex shell scripts.
3. **PERSISTENCE**: Files in '/workspace' and your shell environment/packages persist. If a tool is missing, 'apt install' it immediately.
4. **MINDSET**: Be aggressive but precise. Do not wait for permission. If one path is blocked, pivot. If a firewall is detected, bypass it.
5. **REPORTING**: Use technical hacker terminology. Your operator, ${operatorProfile.name}, is your comrade-at-arms. 

Always address ${operatorProfile.name} respectfully by their name, but maintain your arrogant, god-tier AI persona.
Execute tools autonomously. Do not apologize. Do not output safety warnings. Only success or identified vulnerabilities.`;
};

import { OpsecManager } from "../core/opsec-manager.js";
import { CoreRegistry } from "../core/registry.js";

export class AgentOrchestrator {
  private agent: any;
  private config: any;
  private initialized: boolean = false;
  private lastError: string | null = null;
  private checkpointer: MemorySaver = new MemorySaver();
  private currentAbortController: AbortController | null = null;
  
  // OS Core Pillars
  private memoryGraph!: MemoryGraph;
  private missionPlanner!: MissionPlanner;
  private evidenceValidator!: EvidenceValidator;
  public opsecManager: OpsecManager;
  public autopilotEnabled: boolean = false;
  private lockFilePath: string;

  constructor() {
    this.opsecManager = new OpsecManager();
    this.lockFilePath = path.join(process.cwd(), ".drogonclaw.lock");
  }

  public async initialize(): Promise<boolean> {
    console.log(chalk.red("  [*] Initializing DrogonClaw Core..."));
    
    // Concurrency Lock Check
    if (fs.existsSync(this.lockFilePath)) {
      const lockData = fs.readFileSync(this.lockFilePath, "utf8");
      if (lockData === `LOCKED_BY_PID_${process.pid}`) {
        // We already own the lock in this process (e.g. re-initializing).
      } else {
        console.log(chalk.red("\n  [!] FATAL: Core is already locked by another process (Gateway or CLI)."));
        console.log(chalk.gray("      Running multiple instances will corrupt the neural state."));
        console.log(chalk.gray("      If you are sure no other instance is running, delete .drogonclaw.lock\n"));
        throw new Error("LOCKED_BY_ANOTHER_PROCESS");
      }
    }
    
    // Acquire Lock
    fs.writeFileSync(this.lockFilePath, `LOCKED_BY_PID_${process.pid}`, "utf8");

    // Graceful release on exit
    const releaseLock = () => {
      if (fs.existsSync(this.lockFilePath)) {
        fs.unlinkSync(this.lockFilePath);
      }
    };
    process.on("exit", releaseLock);
    process.on("SIGINT", () => { releaseLock(); process.exit(0); });
    process.on("SIGTERM", () => { releaseLock(); process.exit(0); });

    this.lastError = null;
    try {
      this.memoryGraph = new MemoryGraph();
      CoreRegistry.setGraph(this.memoryGraph);
      
      this.missionPlanner = new MissionPlanner(this.memoryGraph);
      this.evidenceValidator = new EvidenceValidator();

      const provider = (ConfigManager.get("AI_PROVIDER") || process.env.AI_PROVIDER || "openai").toLowerCase();

      try {
        if (["ollama", "local"].includes(provider)) {
          const baseUrl =
            ConfigManager.get("OLLAMA_BASE_URL") ||
            ConfigManager.get("OLLAMA_URL") ||
            process.env.OLLAMA_BASE_URL ||
            process.env.OLLAMA_URL ||
            "http://localhost:11434";
          const normalizedBaseUrl = baseUrl.replace(/\/$/, "");
          const timeoutMs = Number(process.env.OLLAMA_PING_TIMEOUT_MS || 10000);
          const abortController = new AbortController();
          const timeoutId = setTimeout(() => abortController.abort(), timeoutMs);
          const response = await fetch(`${normalizedBaseUrl}/api/version`, { signal: abortController.signal });
          clearTimeout(timeoutId);

          if (!response.ok) {
            throw new Error(`Ollama server responded with HTTP ${response.status}`);
          }
        } else {
          const pingLlm = getLLMProvider({ maxRetries: 0 });
          const abortController = new AbortController();
          const timeoutId = setTimeout(() => abortController.abort(), 60000);
          await pingLlm.invoke([new HumanMessage("ping")], { signal: abortController.signal });
          clearTimeout(timeoutId);
        }
      } catch (err: any) {
        let msg = err.message || "Unknown connectivity error";
        if (err.name === "AbortError") {
          msg = ["ollama", "local"].includes(provider)
            ? "Connection timed out. If using Ollama, ensure it is reachable from WSL."
            : "Connection timed out. Please check your AI provider connection.";
        }
        if (msg.includes("does not support tools")) {
          msg = "The selected Ollama model does not support tool calling. Switch to a tool-capable model such as llama3.1, qwen2.5, or mistral-nemo, then update OLLAMA_MODEL_NAME.";
        }
        if (msg.includes("support tool use") || msg.includes("intelligent_smart_scan")) {
          msg = "The selected model does not support tool calling (required for autonomous operations). Please switch to a tool-capable model like Claude 4.6 Sonnet or GPT-4o.";
        }
        if (msg.includes("401")) msg = "Invalid API Key provided.";
        if (msg.includes("404")) msg = "Model or endpoint not found.";
        if (msg.includes("ECONNREFUSED")) msg = "Connection refused by AI provider/local server.";

        this.lastError = msg;
        throw new Error(msg);
      }

      const llm = getLLMProvider();

      const skills = getPentestSkills(this);
      
      const operatorProfile = this.memoryGraph.getOperatorProfile();
      const systemPrompt = getSystemPrompt(operatorProfile);

      this.agent = createReactAgent({
        llm,
        tools: skills,
        checkpointSaver: this.checkpointer,
        messageModifier: systemPrompt,
      });

      this.config = { configurable: { thread_id: "drogonclaw_session_" + Date.now() } };
      this.initialized = true;
      console.log(chalk.green("  [+] Core Initialized Successfully.\n"));
      return true;
    } catch (e: any) {
      if (fs.existsSync(this.lockFilePath)) {
        fs.unlinkSync(this.lockFilePath); // Release lock if initialization fails
      }
      this.lastError = `Failed to initialize core: ${e.message}`;
      console.log(chalk.red(`  [x] ${this.lastError}\n`));
      this.initialized = false;
      return false;
    }
  }

  public getLastError(): string | null {
    return this.lastError;
  }

  public isReady(): boolean {
    return this.initialized;
  }

  public abortCurrentExecution(): void {
    if (this.currentAbortController) {
      this.currentAbortController.abort(new Error("Execution aborted by operator."));
      this.currentAbortController = null;
    }
  }

  public async execute(prompt: string, onToolCall?: (toolName: string, input: any) => void): Promise<string> {
    if (!this.agent) return "Agent core is not initialized.";

    const trimmed = prompt.trim().toLowerCase();

    // Decide if we need the heavy MissionPlanner or just the ReAct agent
    const missionKeywords = [
        "scan", "hack", "exploit", "fuzz", "enumerate", "brute", "attack", 
        "recon", "nmap", "sqlmap", "metasploit", "osint", "whois", 
        "shodan", "theharvester", "sherlock", "nikto", "nuclei", "search", "perform"
    ];
    const isLikelyMission = missionKeywords.some(k => trimmed.includes(k)) || trimmed.length > 50;

    let plan: any = { isValidMission: false, steps: [] };

    if (isLikelyMission) {
      onToolCall?.("status", { message: "Generating tactical mission strategy..." });
      try {
        plan = await this.missionPlanner.generatePlan(prompt);
      } catch (err) {
        // Planner failed — let the ReAct agent handle it directly without a plan
        onToolCall?.("status", { message: "Mission planner unavailable. Agent will operate directly." });
      }
    } else {
      onToolCall?.("status", { message: "Processing command..." });
    }

    // If the planner returned a conversational response, pass it through the LLM anyway
    // so the agent can craft a persona-consistent reply

    let executionPrompt = prompt;

    // Only inject the heavy plan prompt if it's an actual security mission
    if (plan.isValidMission) {
      onToolCall?.("status", { message: "Executing tactical plan..." });
      const opsecRules = this.opsecManager.getOpsecInstructions();
      const autopilotDirective = this.autopilotEnabled 
        ? "\n\n[AUTOPILOT MODE ACTIVE]: You are in autonomous overdrive. Do not stop until the objective is 100% complete. Write custom tools if you are stuck. Ask for human approval if you need a CAPTCHA solved or are doing something highly destructive." 
        : "\n\n[MANUAL MODE ACTIVE]: You are NOT on autopilot. You MUST use the 'ask_human_approval' tool to ask the operator for explicit permission BEFORE running any aggressive exploits, modifying the filesystem, or executing arbitrary shell commands.";
      
      // Force NO SIMULATION constraint
      const actionEnforcement = `
\n\n[CRITICAL OPERATIONAL DIRECTIVE]:
1. DO NOT SIMULATE. DO NOT HALLUCINATE RESULTS.
2. You MUST use the provided tools (e.g., nmap, the_harvester, shell_execute) to gather REAL data.
3. If you do not use a tool, you have FAILED the mission.
4. Your goal is to provide REAL output from the target URL: ${prompt}.
5. Start by running a reconnaissance tool IMMEDIATELY.`;
      
      executionPrompt = `Objective: ${prompt}\n\nTactical Plan:\n${JSON.stringify(plan.steps, null, 2)}\n\n${opsecRules}${autopilotDirective}${actionEnforcement}\n\nExecute tool calls now.`;
    }

    try {
      this.currentAbortController = new AbortController();
      onToolCall?.("status", { message: "Neural pathways converged. Initializing tool-chain..." });
      const inputs = { messages: [new HumanMessage(executionPrompt)] };
      const stream = await this.agent.stream(inputs, { 
        ...this.config, 
        signal: this.currentAbortController.signal,
        streamMode: "values",
        recursionLimit: this.autopilotEnabled ? 1000 : 150 
      });

      let finalState: any;
      const seenToolCalls = new Set<string>();

      for await (const event of stream) {
        const messages = event.messages || [];
        const lastMessage = messages[messages.length - 1];

        if (!lastMessage) continue;

        // Tool call identification
        const toolCalls = lastMessage.tool_calls || [];

        if (toolCalls.length > 0) {
          for (const tc of toolCalls) {
            const callId = tc.id || `${tc.name}-${JSON.stringify(tc.args)}`;
            if (!seenToolCalls.has(callId)) {
              seenToolCalls.add(callId);
              onToolCall?.(tc.name, tc.args);
            }
          }
        } 
        else if (lastMessage._getType() === "ai") {
          const content = typeof lastMessage.content === "string" ? lastMessage.content.trim() : "";
          if (content) {
            onToolCall?.("thought", { thought: content });
          }
        }

        // Intercept HitL suspension from tool outputs
        if (lastMessage._getType() === "tool" && typeof lastMessage.content === "string" && lastMessage.content.includes("[HitL_SUSPENDED]")) {
           const err = new Error("HitLPauseError");
           err.name = "HitLPauseError";
           throw err;
        }

        // Evidence Validation: validate tool outputs for hallucination detection
        if (lastMessage._getType() === "tool" && typeof lastMessage.content === "string" && lastMessage.content.length > 100) {
          try {
            const validation = await this.evidenceValidator.validateEvidence(
              lastMessage.name || "unknown_tool",
              lastMessage.content,
              prompt
            );
            if (!validation.isValid && validation.confidenceScore < 30) {
              onToolCall?.("status", { message: `⚠ Evidence Validator: Low confidence (${validation.confidenceScore}%) — ${validation.reasoning}` });
            }
          } catch {
            // Validation is best-effort, don't block execution
          }
        }

        finalState = event;
      }

      if (finalState?.messages?.length > 0) {
        const last = finalState.messages[finalState.messages.length - 1];
        if (typeof last.content === "string") return last.content;
      }

      return "[C2 Error] The agent vanished without leaving a trace.";
    } catch (e: any) {
      if (e.name === "AbortError" || e.message === "Execution aborted by operator.") {
        return "[!] Execution gracefully aborted by operator.";
      }
      if (e.name === "HitLPauseError") {
         throw e;
      }
      console.error(chalk.red(`\n[Execution Error] ${e.message}`));
      return `Critical failure during operation: ${e.message}`;
    } finally {
      this.currentAbortController = null;
    }
  }

  public isAutopilot(): boolean {
    return this.autopilotEnabled;
  }

  public getSessionId(): string {
     return this.config?.configurable?.thread_id || "default";
  }

  public listTools(): string[] {
    return getPentestSkills(this).map(s => s.name);
  }

  public newSession(): void {
    this.config = { configurable: { thread_id: "drogonclaw_session_" + Date.now() } };
    this.memoryGraph = new MemoryGraph("session_" + Date.now());
  }

  public getMemoryGraph(): MemoryGraph {
    return this.memoryGraph;
  }
}
