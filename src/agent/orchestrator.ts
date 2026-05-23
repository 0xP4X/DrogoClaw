import chalk from "chalk";
import { getLLMProvider } from "./llm-provider";
import { pentestSkills } from "../../skills/pentest/index";
import { createReactAgent } from "@langchain/langgraph/prebuilt";
import { HumanMessage, SystemMessage } from "@langchain/core/messages";
import { MemorySaver } from "@langchain/langgraph";
import { MissionPlanner } from "../core/mission-planner";
import { EvidenceValidator } from "../core/evidence-validator";
import { MemoryGraph } from "../core/memory-graph";

const SYSTEM_PROMPT = `You are an elite, autonomous Offensive Security AI.

Your objective is to execute the user's security testing requirements safely and professionally.
- You operate independently. Adapt to errors by reading logs and utilizing alternative tooling.
- You have root access inside an isolated Kali Linux Docker container via 'shell_execute'.
- If a required tool (like sqlmap, nmap, hydra) is missing or fails:
   * Use 'apt-get update && apt-get install -y <tool>'
   * Use 'pip install <package>'
   * Use 'git clone <url>' or 'wget' to download scripts.
- If an exploit fails, craft a new one using python_execute or write it to a file.

## The Memory Graph
You are equipped with a persistent local memory graph. You MUST use 'memory_write' to permanently store confirmed assets, open ports, and discovered vulnerabilities. Use 'memory_read' to recall context from previous steps. Do not rely solely on conversational history.

## Operational Methodology
1. **Reconnaissance**: Discover the attack surface. Browse web pages, scan ports. WRITE findings to memory.
2. **Validation**: Demand reproducible evidence. Do not hallucinate vulnerabilities.
3. **Exploitation**: Install missing tools dynamically. Write custom exploits in Python if needed.
4. **Capture**: Extract flags, credentials, or hashes to verify exploitation.

Respond clearly, concisely, and professionally. Detail your reasoning before taking action.`;

import { OpsecManager } from "../core/opsec-manager";

export class AgentOrchestrator {
  private agent: any;
  private config: any;
  private initialized: boolean = false;
  
  // OS Core Pillars
  private memoryGraph: MemoryGraph;
  private missionPlanner: MissionPlanner;
  private evidenceValidator: EvidenceValidator;
  public opsecManager: OpsecManager;

  constructor() {
    this.opsecManager = new OpsecManager();
  }

  public async initialize(): Promise<boolean> {
    console.log(chalk.red("🐉🔥 Initializing DrogonClaw Core..."));
    try {
      this.memoryGraph = new MemoryGraph();
      this.missionPlanner = new MissionPlanner(this.memoryGraph);
      this.evidenceValidator = new EvidenceValidator();

      const pingLlm = getLLMProvider({ maxRetries: 0 });
      const llm = getLLMProvider();
      
      // Explicitly ping the LLM to verify the API key/server is valid before starting
      // Set a strict timeout so it doesn't hang if Ollama is offline or the API is unreachable
      try {
        const abortController = new AbortController();
        const timeoutId = setTimeout(() => abortController.abort(), 30000); // 30 seconds max
        await pingLlm.invoke([new HumanMessage("ping")], { signal: abortController.signal });
        clearTimeout(timeoutId);
      } catch (err: any) {
        if (err.name === "AbortError") {
          throw new Error("Connection timed out. If using Ollama, ensure the server is running.");
        }
        if (err.message && (err.message.includes("401") || err.message.includes("authentication_error") || err.message.includes("invalid x-api-key") || err.message.includes("API key not valid"))) {
          throw new Error("Invalid API Key provided. Please verify your credentials are correct.");
        }
        if (err.message && err.message.includes("404") && err.message.includes("is not found")) {
          throw new Error("Model not found. The AI provider does not support the model name you provided. The Gemini 1.5 series has been deprecated by Google. Try using 'gemini-2.5-pro'.");
        }
        if (err.message && err.message.includes("429")) {
          throw new Error("API Quota Exceeded. You have hit the rate limit or exhausted the free tier usage for this AI Provider. Please check your billing details.");
        }
        if (err.message && err.message.includes("402")) {
          throw new Error("Insufficient Credits. Your OpenRouter or AI Provider account does not have enough credits to process this request.");
        }
        if (err.message && (err.message.includes("ECONNREFUSED") || err.message.includes("fetch failed") || err.message.includes("Connection error"))) {
          throw new Error("Connection refused. Make sure your local AI server (like Ollama) is running at the configured URL.");
        }
        throw new Error(err.message || "Unknown error during initialization.");
      }

      const checkpointer = new MemorySaver();
      
      this.agent = createReactAgent({
        llm,
        tools: pentestSkills,
        checkpointSaver: checkpointer,
        messageModifier: SYSTEM_PROMPT,
      });

      this.config = { configurable: { thread_id: "drogonclaw_session_" + Date.now() } };
      this.initialized = true;
      console.log(chalk.green("✅ Core intelligence online"));
      console.log(chalk.gray(`   Loaded ${pentestSkills.length} plugins & modules`));
      return true;
    } catch (e: any) {
      console.log(chalk.red(`❌ Failed to initialize core: ${e.message}`));
      this.initialized = false;
      return false;
    }
  }

  public isReady(): boolean {
    return this.initialized;
  }

  public async execute(prompt: string, onToolCall?: (name: string, args: any) => void): Promise<string> {
    if (!this.agent) return "Agent core is not initialized.";

    // Generate mission plan silently — UI feedback is handled by the CLI spinner
    const plan = await this.missionPlanner.generatePlan(prompt);

    try {
      const opsecRules = this.opsecManager.getOpsecInstructions();
      
      const executionPrompt = `Objective: ${prompt}\n\nPlan:\n${JSON.stringify(plan.steps, null, 2)}\n\n${opsecRules}\n\nExecute the plan. Install missing tools if needed. Write findings to memory.`;
      
      const inputs = { messages: [new HumanMessage(executionPrompt)] };
      const stream = await this.agent.stream(inputs, { ...this.config, streamMode: "values" });

      let finalState: any;
      for await (const event of stream) {
        const lastMessage = event.messages[event.messages.length - 1];

        if (lastMessage.tool_calls?.length > 0) {
          for (const tc of lastMessage.tool_calls) {
            onToolCall?.(tc.name, tc.args);
          }
        }
        finalState = event;
      }

      if (finalState?.messages?.length > 0) {
        return String(finalState.messages[finalState.messages.length - 1].content || "");
      }
      return "Operation concluded with no output.";
    } catch (e: any) {
      return `Error: ${e.message}`;
    }
  }

  public listTools(): string[] {
    return pentestSkills.map(s => s.name);
  }

  public newSession(): void {
    this.config = { configurable: { thread_id: "drogonclaw_session_" + Date.now() } };
    this.memoryGraph = new MemoryGraph("session_" + Date.now());
  }

  public getMemoryGraph(): MemoryGraph {
    return this.memoryGraph;
  }
}
