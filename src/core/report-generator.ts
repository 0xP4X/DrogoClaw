import { MemoryGraph } from "./memory-graph.js";
import { getLLMProvider } from "../agent/llm-provider.js";
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
     
     try {
       // Utilize Playwright for native cross-platform PDF generation
       const { chromium } = await import("playwright");
       const browser = await chromium.launch({ headless: true });
       const page = await browser.newPage();
       
       // Load the local HTML file
       await page.goto(`file://${htmlPath}`, { waitUntil: 'networkidle' });
       
       const pdfPath = mdPath.replace(".md", ".pdf");
       
       // Generate PDF with professional styling
       await page.pdf({
         path: pdfPath,
         format: 'A4',
         printBackground: true,
         margin: { top: '20mm', bottom: '20mm', left: '20mm', right: '20mm' }
       });
       
       await browser.close();
       console.log(chalk.green(`  [+] PDF generated successfully: ${pdfPath}`));
       return { textPath: mdPath, docPath: pdfPath };
     } catch (e: any) {
       console.log(chalk.yellow(`  [!] PDF rendering failed (${e.message}). Falling back to styled HTML.`));
       return { textPath: mdPath, docPath: htmlPath };
     }
  }
}
