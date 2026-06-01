import { ChatOpenAI } from "@langchain/openai";
import { HumanMessage } from "@langchain/core/messages";

async function test() {
  console.log("Starting ping...");
  const llm = new ChatOpenAI({
    modelName: "llama3:latest",
    temperature: 0,
    configuration: {
      baseURL: "http://127.0.0.1:11434/v1",
    },
    openAIApiKey: "ollama",
    maxRetries: 0,
  });

  try {
    const start = Date.now();
    const res = await llm.invoke([new HumanMessage("ping")]);
    console.log("Response:", res.content);
    console.log("Time taken:", Date.now() - start, "ms");
  } catch (err: any) {
    console.error("Error Name:", err.name);
    console.error("Error Message:", err.message);
    if (err.cause) console.error("Error Cause:", err.cause);
  }
}

test();
