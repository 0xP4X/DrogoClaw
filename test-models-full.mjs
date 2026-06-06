import { ChatOllama } from "@langchain/ollama";
import { z } from "zod";
import { tool } from "@langchain/core/tools";

const TIMEOUT_MS = 45000;
const MODELS = ["qwen2.5:latest", "qwen2.5:7b", "llama3:latest"];

const dummyTool = tool(
  async ({ input }) => `Result: ${input}`,
  {
    name: "shell_execute",
    description: "Execute a shell command on the target system",
    schema: z.object({ input: z.string().describe("The shell command to run") }),
  }
);

async function testModel(modelName) {
  console.log(`\n${"=".repeat(50)}`);
  console.log(`[*] Testing model: ${modelName}`);
  console.log(`${"=".repeat(50)}`);

  const llm = new ChatOllama({
    baseUrl: "http://localhost:11434",
    model: modelName,
    temperature: 0,
  });

  // Step 1: bare chat (no tools)
  console.log(`[1] Bare chat test (no tools)...`);
  try {
    const start = Date.now();
    const result = await Promise.race([
      llm.invoke("Reply with just the word: READY"),
      new Promise((_, reject) => setTimeout(() => reject(new Error("TIMEOUT")), TIMEOUT_MS)),
    ]);
    console.log(`    [+] OK in ${Date.now() - start}ms — "${String(result.content).trim().slice(0, 60)}"`);
  } catch (e) {
    console.log(`    [-] FAILED: ${e.message}`);
    console.log(`[VERDICT] ${modelName}: BARE CHAT FAILED — skip tool test.\n`);
    return;
  }

  // Step 2: tool binding
  console.log(`[2] Tool binding + invocation...`);
  try {
    const llmWithTools = llm.bindTools([dummyTool]);
    const start = Date.now();
    const result = await Promise.race([
      llmWithTools.invoke("Use the shell_execute tool to run: echo hello"),
      new Promise((_, reject) => setTimeout(() => reject(new Error("TIMEOUT")), TIMEOUT_MS)),
    ]);
    const elapsed = Date.now() - start;
    if (result.tool_calls && result.tool_calls.length > 0) {
      console.log(`    [+] Tool call returned in ${elapsed}ms!`);
      console.log(`    [+] Tool name: ${result.tool_calls[0].name}`);
      console.log(`    [+] Args: ${JSON.stringify(result.tool_calls[0].args)}`);
      console.log(`[VERDICT] ${modelName}: FULLY COMPATIBLE ✓\n`);
    } else {
      console.log(`    [~] Response in ${elapsed}ms but no tool_calls. Content: "${String(result.content).slice(0, 100)}"`);
      console.log(`[VERDICT] ${modelName}: TOOL CALLING NOT WORKING — model responded but ignored the tool schema.\n`);
    }
  } catch (e) {
    console.log(`    [-] FAILED: ${e.message}`);
    console.log(`[VERDICT] ${modelName}: TOOL CALLING FAILED.\n`);
  }
}

(async () => {
  console.log("\n╔══════════════════════════════════════╗");
  console.log("║  DrogonClaw Local LLM Compatibility  ║");
  console.log("╚══════════════════════════════════════╝\n");
  for (const model of MODELS) {
    await testModel(model);
  }
  console.log("\n[*] Investigation complete.");
})();
