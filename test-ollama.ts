import { ChatOpenAI } from "@langchain/openai";
import { z } from "zod";
import { tool } from "@langchain/core/tools";

async function testOllama() {
  console.log("Connecting to Ollama...");
  
  const llm = new ChatOpenAI({
    modelName: "qwen2.5:latest",
    temperature: 0,
    configuration: {
      baseURL: "http://localhost:11434/v1",
    },
    openAIApiKey: "ollama",
  });

  const weatherTool = tool(
    async ({ location }) => {
      return `The weather in ${location} is 72 degrees.`;
    },
    {
      name: "get_weather",
      description: "Get the current weather for a location",
      schema: z.object({
        location: z.string().describe("The city and state, e.g. San Francisco, CA"),
      }),
    }
  );

  const llmWithTools = llm.bindTools([weatherTool]);

  console.log("Sending request to qwen2.5:latest to use the weather tool...");
  
  try {
    const res = await llmWithTools.invoke("What is the weather in San Francisco?");
    console.log("Response received!");
    console.log(JSON.stringify(res.tool_calls, null, 2));
    if (res.tool_calls && res.tool_calls.length > 0) {
      console.log("✅ Tool calling is fully operational on this model!");
    } else {
      console.log("❌ The model responded, but did NOT trigger the tool call. It may not support tool binding properly.");
      console.log("Model output:", res.content);
    }
  } catch (error) {
    console.error("❌ Error during execution:");
    console.error(error);
  }
}

testOllama();
