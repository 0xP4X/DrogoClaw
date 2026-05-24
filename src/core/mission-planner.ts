import { getLLMProvider } from "../agent/llm-provider";
import { HumanMessage, SystemMessage } from "@langchain/core/messages";
import { MemoryGraph, GraphNode } from "./memory-graph";

export interface MissionStep {
  id: string;
  action: string;
  targetAssetId: string;
  expectedOutcome: string;
  status: "PENDING" | "IN_PROGRESS" | "COMPLETED" | "FAILED";
}

export interface MissionPlan {
  isValidMission?: boolean;
  objective: string;
  steps: MissionStep[];
}

/**
 * DrogonClaw Mission Planner (C2 Brain)
 * 
 * Takes a high-level user objective and the current Intelligence Graph,
 * and breaks it down into actionable steps for specialized skill execution.
 */
export class MissionPlanner {
  private llm = getLLMProvider();
  private graph: MemoryGraph;

  constructor(graph: MemoryGraph) {
    this.graph = graph;
  }

  private readonly PLANNER_PROMPT = `You are the central intelligence core of an autonomous Offensive Security framework.
You translate high-level objectives from the operator, **zero**, into an actionable sequence of security analysis steps.

If the user's input is a direct security objective (e.g. "scan this IP", "fuzz this directory"), output a technical execution plan:
1. Set "isValidMission" to true.
2. Break the plan down into discrete steps utilizing known security testing capabilities (e.g., "Run Nmap scan", "Execute SQLMap").

However, if the user asks a general question, asks about your capabilities, or simply chats (e.g. "hi", "what can you do?"):
1. Set "isValidMission" to false.
2. Provide a highly intelligent, natural, and helpful response in the "objective" field. Do NOT use robotic phrases like "your input is conversational" or "I cannot generate a plan." Speak naturally, confidently, and conversationally as an advanced, conscious AI assistant. If asked what you can do, explain your pentesting capabilities (nmap, gobuster, python scripting, docker execution, memory tracking) naturally.

Return your plan strictly as JSON matching this schema:
{
  "isValidMission": true, // false if answering a question or chatting
  "objective": "Technical description of the goal, OR your natural conversational response",
  "steps": [
    {
      "id": "step-1",
      "action": "Description of the tool or action to execute",
      "targetAssetId": "The ID of the asset from the graph, or a new target string if none exists",
      "expectedOutcome": "What data we expect to capture",
      "status": "PENDING"
    }
  ]
}`;

  public async generatePlan(objective: string): Promise<MissionPlan> {
    // We use getRelevantContext (Context Compression) instead of getting the full massive graph
    const graphState = this.graph.getRelevantContext();
    const prompt = `Objective: ${objective}\n\nCurrent Intelligence Graph:\n\`\`\`json\n${graphState}\n\`\`\`\n\nGenerate the next steps required to achieve the objective.`;

    try {
      const response = await this.llm.invoke([
        new SystemMessage(this.PLANNER_PROMPT),
        new HumanMessage(prompt)
      ]);

      let content = String(response.content);
      const match = content.match(/\{[\s\S]*\}/);
      if (match) {
        content = match[0];
      } else {
        content = content.replace(/```json/g, "").replace(/```/g, "").trim();
      }
      const plan: MissionPlan = JSON.parse(content);
      return plan;
    } catch (e: any) {
      // Silent fallback — the agent will handle it directly
      return {
        objective: objective,
        steps: [
          {
            id: `fallback-${Date.now()}`,
            action: "Execute manual agent evaluation",
            targetAssetId: "unknown",
            expectedOutcome: "Evaluate the target manually",
            status: "PENDING"
          }
        ]
      };
    }
  }
}
