import { ChatOpenAI } from "@langchain/openai";

async function testOpenRouter() {
  const model = new ChatOpenAI({
    modelName: "fake/model",
    temperature: 0,
    configuration: {
      baseURL: "https://openrouter.ai/api/v1",
    },
    openAIApiKey: "sk-or-v1-fake-key",
  });

  try {
    await model.invoke("Hello");
  } catch (e: any) {
    console.log("OpenRouter error:", e.message);
  }
}

testOpenRouter();

async function testAnthropic() {
  const model = new ChatAnthropic({
    modelName: "claude-sonnet-4-6-20260218",
    temperature: 0,
    anthropicApiKey: "sk-ant-fake-key",
  });

  try {
    await model.invoke("Hello");
  } catch (e: any) {
    console.log("Anthropic error:", e.message);
  }
}

testOpenRouter().then(testAnthropic);
