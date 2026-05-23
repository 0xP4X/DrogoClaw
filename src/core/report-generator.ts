import { MemoryGraph } from "./memory-graph";
import { getLLMProvider } from "../agent/llm-provider";
import { HumanMessage } from "@langchain/core/messages";
import fs from "fs";
import path from "path";
import chalk from "chalk";

export class ReportGenerator {
  private memoryGraph: MemoryGraph;

  constructor(memoryGraph: MemoryGraph) {
    this.memoryGraph = memoryGraph;
  }

  public async generateMarkdownReport(): Promise<string> {
    const memoryDump = this.memoryGraph.readAll();
    
    if (!memoryDump || Object.keys(memoryDump).length === 0) {
      throw new Error("Memory graph is empty. Run a pentest mission before generating a report.");
    }

    const llm = getLLMProvider();
    
    const prompt = `You are an elite penetration testing report generator.
I will provide you with a raw JSON dump of the DrogonClaw Memory Graph, which contains the assets, open ports, and vulnerabilities discovered during an autonomous mission.

Write a highly professional, compliance-ready penetration test report in Markdown.
It must include:
1. Executive Summary
2. Discovered Assets & Attack Surface
3. Detailed Vulnerability Findings (with estimated CVSS scores)
4. Remediation Steps

Here is the raw memory graph:
${JSON.stringify(memoryDump, null, 2)}`;

    console.log(chalk.yellow("📝 Drafting compliance-ready report..."));
    
    try {
      const response = await llm.invoke([new HumanMessage(prompt)]);
      const reportContent = String(response.content);
      
      const reportsDir = path.join(process.cwd(), "reports");
      if (!fs.existsSync(reportsDir)) fs.mkdirSync(reportsDir, { recursive: true });
      
      const filepath = path.join(reportsDir, `drogonclaw_report_${Date.now()}.md`);
      fs.writeFileSync(filepath, reportContent);
      
      return filepath;
    } catch (e: any) {
      throw new Error(`Failed to generate report: ${e.message}`);
    }
  }
}
