import { getLLMProvider } from "../agent/llm-provider.js";
import { HumanMessage, SystemMessage } from "@langchain/core/messages";

export interface ValidationResult {
  isValid: boolean;
  confidenceScore: number; // 0-100
  reasoning: string;
  extractedEntities: {
    assets?: string[];
    ports?: number[];
    vulnerabilities?: string[];
  };
}

/**
 * DrogonClaw Evidence Validator
 * 
 * Analyzes raw tool output and reasoning to ensure the agent is not hallucinating.
 * Scores the confidence of findings and extracts structured entities for the Intelligence Graph.
 */
export class EvidenceValidator {
  private llm = getLLMProvider();

  private readonly VALIDATOR_PROMPT = `You are the DrogonClaw Evidence Validator — a critical component of an AI-driven offensive security OS.
Your job is to review raw tool outputs (from nmap, curl, sqlmap, etc.) alongside the agent's claim about what that output means.
You must prevent hallucinations. If the agent claims a vulnerability exists but the raw tool output does NOT explicitly prove it, you must REJECT it.

Analyze the provided data and return a JSON object with the following schema:
{
  "isValid": boolean, // True only if the evidence directly supports the claim
  "confidenceScore": number, // 0-100. 100 = Absolute mathematical proof. 0 = Pure hallucination.
  "reasoning": string, // Briefly explain why you accepted or rejected the claim based strictly on the evidence
  "extractedEntities": { // Extract verified entities to populate the Intelligence Graph
    "assets": string[], // e.g., ["10.10.10.1", "target.com"]
    "ports": number[], // e.g., [80, 443, 22]
    "vulnerabilities": string[] // e.g., ["CVE-2021-44228", "SQL Injection in /login"]
  }
}

Respond ONLY with valid JSON.`;

  public async validateEvidence(toolName: string, rawOutput: string, agentClaim: string): Promise<ValidationResult> {
    const prompt = `Tool Executed: ${toolName}\n\nRaw Output:\n\`\`\`\n${rawOutput.substring(0, 5000)}\n\`\`\`\n\nAgent's Claim/Finding: ${agentClaim}`;
    
    try {
      const response = await this.llm.invoke([
        new SystemMessage(this.VALIDATOR_PROMPT),
        new HumanMessage(prompt)
      ]);

      const content = String(response.content).replace(/```json/g, "").replace(/```/g, "").trim();
      const result: ValidationResult = JSON.parse(content);
      return result;
    } catch (e: any) {
      console.error("[Evidence Validator] Failed to parse LLM response. Defaulting to safe rejection.", e.message);
      return {
        isValid: false,
        confidenceScore: 0,
        reasoning: "Validation failed due to parsing error. Rejected for safety.",
        extractedEntities: {}
      };
    }
  }
}
