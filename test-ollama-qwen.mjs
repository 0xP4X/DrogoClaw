import { ChatOpenAI } from "@langchain/openai";
import { z } from "zod";
import { tool } from "@langchain/core/tools";

console.log("[*] Initializing ChatOpenAI with Ollama backend...");
const llm = new ChatOpenAI({
  modelName: "qwen3:latest",
  temperature: 0,
  configuration: {
    baseURL: "http://localhost:11434/v1",
  },
  openAIApiKey: "ollama",
});

const dummyTool = tool(
  async ({ input }) => `Hello ${input}`,
  {
    name: "dummy_tool",
    description: "A dummy tool",
    schema: z.object({ input: z.string() }),
  }
);

console.log("[*] Binding tools...");
const llmWithTools = llm.bindTools([dummyTool]);

console.log("[*] Invoking model (this is where it hangs)...");
try {
  const res = await llmWithTools.invoke("hi");
  console.log("\n[+] Response received:");
  console.log(res.content);
  if (res.tool_calls && res.tool_calls.length > 0) {
    console.log("[+] Tool calls:");
    console.log(res.tool_calls);
  }
} catch (e) {
  console.error("\n[x] Error occurred:");
  console.error(e);
}
console.log("[*] Done.");
