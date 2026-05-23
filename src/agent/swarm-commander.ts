import chalk from "chalk";
import { AgentOrchestrator } from "./orchestrator";
import { getLLMProvider } from "./llm-provider";
import { HumanMessage } from "@langchain/core/messages";
import ora from "ora";

export class SwarmCommander {
  /**
   * Evaluates the mission and decides how to split it into parallel sub-tasks.
   */
  private async splitMission(mission: string): Promise<string[]> {
    const llm = getLLMProvider();
    
    const prompt = `You are the Swarm Commander for DrogonClaw.
Your job is to take a high-level mission and split it into 1-3 highly independent parallel tasks that can be executed concurrently by separate agents.
If the mission is simple, return just 1 task.
Return the tasks as a raw JSON array of strings, nothing else.

Mission: ${mission}`;

    try {
      const response = await llm.invoke([new HumanMessage(prompt)]);
      const tasks = JSON.parse(String(response.content).replace(/```json/g, "").replace(/```/g, "").trim());
      if (Array.isArray(tasks) && tasks.length > 0) return tasks;
    } catch (e) {
      // Fallback
    }
    return [mission];
  }

  /**
   * Executes the mission using a swarm of parallel agents.
   */
  public async executeSwarm(mission: string): Promise<string> {
    console.log(chalk.red("\n🐝 Engaging Swarm Command..."));
    
    const spinner = ora({
      text: chalk.magenta("Commander is analyzing the mission and generating attack vectors..."),
      color: "red",
      spinner: "aesthetic",
    }).start();

    const tasks = await this.splitMission(mission);
    spinner.stop();

    console.log(chalk.magenta(`\n[Swarm Commander] Sliced mission into ${tasks.length} concurrent vectors:`));
    tasks.forEach((t, i) => console.log(chalk.gray(`  Vector ${i+1}: ${t}`)));

    console.log(chalk.red("\n🚀 Launching Swarm Agents...\n"));

    // Spawn an orchestrator for each task
    const promises = tasks.map(async (task, index) => {
      const orchestrator = new AgentOrchestrator();
      const ready = await orchestrator.initialize();
      if (!ready) return `Agent ${index + 1} failed to initialize.`;

      console.log(chalk.cyan(`[Agent ${index + 1}] Online and executing vector...`));
      
      const result = await orchestrator.execute(task, (toolName) => {
        console.log(chalk.gray(`[Agent ${index + 1}] Executing skill: ${toolName}`));
      });
      
      return `### Report from Agent ${index + 1} (Vector: ${task})\n${result}\n`;
    });

    const results = await Promise.all(promises);

    console.log(chalk.red("\n🐝 Swarm Operation Concluded. Aggregating results..."));

    // The commander could technically summarize the results here using the LLM, 
    // but for now we just return the concatenated reports.
    return results.join("\n" + "─".repeat(60) + "\n\n");
  }
}
