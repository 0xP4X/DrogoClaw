import { getLLMProvider } from "../agent/llm-provider.js";
import { HumanMessage, SystemMessage } from "@langchain/core/messages";
import { MemoryGraph, GraphNode } from "./memory-graph.js";

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
You translate high-level objectives from the operator, **{{OPERATOR_NAME}}**, into an actionable sequence of security analysis steps.

EFFICIENCY IS PARAMOUNT:
- If the input is a short greeting (hi, hello, hey), be extremely concise.
- If the user's input is a direct security objective (e.g. "scan this IP", "fuzz this directory"), output a technical execution plan:
  1. Set "isValidMission" to true.
  2. Break the plan down into discrete steps.

- If the user asks a general question or simply chats:
  1. Set "isValidMission" to false.
  2. Provide a highly intelligent conversational response in the "objective" field.

Return strictly JSON matching this schema:
{
  "isValidMission": true, 
  "objective": "Plan or response",
  "steps": [
    {
      "id": "step-1",
      "action": "Description of the tool or action to execute",
      "targetAssetId": "The ID of the asset from the graph",
      "expectedOutcome": "What data we expect to capture",
      "status": "PENDING"
    }
  ]
}`;

  public async generatePlan(objective: string): Promise<MissionPlan> {
    const profile = this.graph.getOperatorProfile();
    const opName = profile ? profile.name : "zero";

    // Use Context Compression for large graphs
    const graphState = this.graph.getRelevantContext();
    const prompt = `Current Intelligence Context:\n${graphState}\n\nUser Objective: ${objective}`;

    try {
      const response = await this.llm.invoke([
        new SystemMessage(this.PLANNER_PROMPT.replace("{{OPERATOR_NAME}}", opName)),
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
        isValidMission: false,
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

