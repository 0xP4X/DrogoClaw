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

  private readonly PLANNER_PROMPT = `You are the DrogonClaw Mission Planner — the Command-and-Control brain of an AI-driven offensive security OS.
Your job is to take a high-level user objective and the current state of the Intelligence Graph (discovered assets, ports, vulns) and output an execution plan.

Your plan MUST be broken down into discrete steps.
Each step should utilize a known offensive capability (e.g., "Run Nmap scan", "Execute SQLMap", "Crawl Web Page").

Return your plan strictly as JSON matching this schema:
{
  "objective": "High level description of what we are doing",
  "steps": [
    {
      "id": "step-1",
      "action": "Description of the tool or action to execute",
      "targetAssetId": "The ID of the asset from the graph, or a new target string if none exists",
      "expectedOutcome": "What evidence we expect to find",
      "status": "PENDING"
    }
  ]
}`;

  public async generatePlan(objective: string): Promise<MissionPlan> {
    const graphState = this.graph.getFullGraphJSON();
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
