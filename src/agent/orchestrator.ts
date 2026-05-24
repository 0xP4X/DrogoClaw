import chalk from "chalk";
import { getLLMProvider } from "./llm-provider";
import { getPentestSkills } from "../../skills/pentest/index";
import { createReactAgent } from "@langchain/langgraph/prebuilt";
import { HumanMessage } from "@langchain/core/messages";
import { MemorySaver } from "@langchain/langgraph";
import { MissionPlanner } from "../core/mission-planner";
import { EvidenceValidator } from "../core/evidence-validator";
import { MemoryGraph } from "../core/memory-graph";
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
Your Operator is: ${operatorProfile.name} ${operatorProfile.skillLevel ? `(Skill Level: ${operatorProfile.skillLevel})` : ""}
${operatorProfile.preferences ? `Operator Preferences: ${operatorProfile.preferences}` : ""}
Always address your operator respectfully by their name, but maintain your arrogant, god-tier AI persona.
Execute tools autonomously to achieve the objective. Do not ask for help unless you are completely stuck.`;
};

import { OpsecManager } from "../core/opsec-manager";
import { CoreRegistry } from "../core/registry";

export class AgentOrchestrator {
  private agent: any;
  private config: any;
  private initialized: boolean = false;
  private lastError: string | null = null;
  private checkpointer: MemorySaver = new MemorySaver();
  
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
        process.exit(1);
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

      const pingLlm = getLLMProvider({ maxRetries: 0 });
      const llm = getLLMProvider();
      
      // Explicitly ping the LLM to verify the API key/server is valid before starting
      try {
        const abortController = new AbortController();
        const timeoutId = setTimeout(() => abortController.abort(), 10000); // 10 seconds max for ping
        await pingLlm.invoke([new HumanMessage("ping")], { signal: abortController.signal });
        clearTimeout(timeoutId);
      } catch (err: any) {
        let msg = err.message || "Unknown connectivity error";
        if (err.name === "AbortError") {
          msg = "Connection timed out. If using Ollama, ensure it is responsive.";
        }
        if (msg.includes("401")) msg = "Invalid API Key provided.";
        if (msg.includes("404")) msg = "Model or endpoint not found.";
        if (msg.includes("ECONNREFUSED")) msg = "Connection refused by AI provider/local server.";
        
        this.lastError = msg;
        throw new Error(msg);
      }

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

  public async execute(prompt: string, onToolCall?: (name: string, args: any) => void): Promise<string> {
    if (!this.agent) return "Agent core is not initialized.";

    // Generate mission plan silently — UI feedback is handled by the CLI spinner
    const plan = await this.missionPlanner.generatePlan(prompt);

    let executionPrompt = prompt;

    // Only inject the heavy plan prompt if it's an actual security mission
    if (plan.isValidMission) {
      const opsecRules = this.opsecManager.getOpsecInstructions();
      const autopilotDirective = this.autopilotEnabled ? "\n\n[AUTOPILOT MODE ACTIVE]: You are in autonomous overdrive. Do not stop until the objective is 100% complete. Write custom tools if you are stuck. Ask for human approval if you need a CAPTCHA solved or are doing something highly destructive." : "";
      executionPrompt = `Objective: ${prompt}\n\nPlan:\n${JSON.stringify(plan.steps, null, 2)}\n\n${opsecRules}${autopilotDirective}\n\nExecute the plan. Install missing tools if needed. Write findings to memory.`;
    }

    try {
      const inputs = { messages: [new HumanMessage(executionPrompt)] };
      const stream = await this.agent.stream(inputs, { 
        ...this.config, 
        streamMode: "values",
        recursionLimit: this.autopilotEnabled ? 1000 : 150 
      });

      let finalState: any;
      const seenToolCalls = new Set<string>();

      for await (const event of stream) {
        const messages = event.messages || [];
        const lastMessage = messages[messages.length - 1];

        if (lastMessage?._getType() === "ai" && typeof lastMessage.content === "string" && lastMessage.content.trim()) {
           onToolCall?.("thought", { thought: lastMessage.content.trim() });
        }

        // Intercept HitL suspension from tool outputs
        if (lastMessage?._getType() === "tool" && typeof lastMessage.content === "string" && lastMessage.content.includes("[HitL_SUSPENDED]")) {
           const err = new Error("HitLPauseError");
           err.name = "HitLPauseError";
           throw err;
        }

        if (lastMessage?.tool_calls?.length > 0) {
          for (const tc of lastMessage.tool_calls) {
            const callId = tc.id || `${tc.name}-${JSON.stringify(tc.args)}`;
            if (!seenToolCalls.has(callId)) {
              seenToolCalls.add(callId);
              onToolCall?.(tc.name, tc.args);
            }
          }
        }
        finalState = event;
      }

      if (finalState?.messages?.length > 0) {
        return String(finalState.messages[finalState.messages.length - 1].content || "");
      }
      return "Operation concluded with no output.";
    } catch (e: any) {
      if (e.name === "HitLPauseError") throw e;
      return `Error: ${e.message}`;
    }
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
