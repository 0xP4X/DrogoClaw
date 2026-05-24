import { MemoryGraph } from "./memory-graph";
import { getLLMProvider } from "../agent/llm-provider";
import { HumanMessage } from "@langchain/core/messages";
import fs from "fs";
import path from "path";
import chalk from "chalk";
import { marked } from "marked";
import { exec } from "child_process";

export class ReportGenerator {
  private memoryGraph: MemoryGraph;

  constructor(memoryGraph: MemoryGraph) {
    this.memoryGraph = memoryGraph;
  }

  public async generateMarkdownReport(): Promise<{ mdPath: string, htmlPath: string }> {
    const memoryDumpJSON = this.memoryGraph.getFullGraphJSON();
    const memoryDump = JSON.parse(memoryDumpJSON);
    
    if (!memoryDump || !memoryDump.nodes || memoryDump.nodes.length === 0) {
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
      
      const mdPath = path.join(reportsDir, `drogonclaw_report_${Date.now()}.md`);
      fs.writeFileSync(mdPath, reportContent);

      const htmlContent = `
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  body { font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; color: #333; line-height: 1.6; max-width: 800px; margin: 0 auto; padding: 2em; }
  h1 { color: #d32f2f; border-bottom: 2px solid #d32f2f; padding-bottom: 0.2em; }
  h2 { color: #1976d2; margin-top: 1.5em; }
  code { background: #f4f4f4; padding: 0.2em 0.4em; border-radius: 3px; font-family: monospace; }
  pre { background: #282c34; color: #abb2bf; padding: 1em; border-radius: 5px; overflow-x: auto; }
  table { border-collapse: collapse; width: 100%; margin: 1em 0; }
  th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
  th { background-color: #f2f2f2; }
</style>
</head>
<body>
${await marked.parse(reportContent)}
</body>
</html>`;

      const htmlPath = mdPath.replace(".md", ".html");
      fs.writeFileSync(htmlPath, htmlContent);

      return { mdPath, htmlPath };
    } catch (e: any) {
      throw new Error(`Failed to generate report: ${e.message}`);
    }
  }

  // Instead of relying on a slow npm package for PDF, we generate a high-quality HTML report
  // and attempt to compile it to PDF using system tools if available, else fallback to HTML.
  public async generateReport(): Promise<{ textPath: string, docPath: string }> {
     const { mdPath, htmlPath } = await this.generateMarkdownReport();
     
     // Attempt to use 'wkhtmltopdf' if installed on the system (very common on Kali/Ubuntu)
     return new Promise((resolve) => {
        const pdfPath = mdPath.replace(".md", ".pdf");
        exec(`wkhtmltopdf ${htmlPath} ${pdfPath}`, (err) => {
           if (err) {
              // If wkhtmltopdf isn't available, return the styled HTML as the final document
              console.log(chalk.gray(`  [!] PDF renderer not found natively. Outputting styled HTML instead: ${htmlPath}`));
              resolve({ textPath: mdPath, docPath: htmlPath });
           } else {
              resolve({ textPath: mdPath, docPath: pdfPath });
           }
        });
     });
  }
}
