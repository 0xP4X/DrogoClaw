import { ChatOpenAI } from "@langchain/openai";
import { ChatAnthropic } from "@langchain/anthropic";
import { ChatGoogleGenerativeAI } from "@langchain/google-genai";
import { BaseChatModel } from "@langchain/core/language_models/chat_models";
import chalk from "chalk";
import { ConfigManager } from "../core/config-manager.js";
import dotenv from "dotenv";

dotenv.config();

export interface LLMProviderOptions {
  maxRetries?: number;
}

export function getLLMProvider(options?: LLMProviderOptions): BaseChatModel {
  const provider = (ConfigManager.get("AI_PROVIDER") || process.env.AI_PROVIDER || "openai").toLowerCase();
  const maxRetries = options?.maxRetries;

  try {
    switch (provider) {
      case "openai": {
        const apiKey = ConfigManager.get("OPENAI_API_KEY") || process.env.OPENAI_API_KEY;
        if (!apiKey) throw new Error("OPENAI_API_KEY is not set in config.");
        return new ChatOpenAI({
          modelName: ConfigManager.get("OPENAI_MODEL_NAME") || process.env.OPENAI_MODEL_NAME || "gpt-4-turbo",
          temperature: 0,
          openAIApiKey: apiKey,
          maxTokens: 4000,
          maxRetries,
        });
      }

      case "anthropic":
      case "claude": {
        const apiKey = ConfigManager.get("ANTHROPIC_API_KEY") || process.env.ANTHROPIC_API_KEY;
        if (!apiKey) throw new Error("ANTHROPIC_API_KEY is not set in config.");
        return new ChatAnthropic({
          modelName: ConfigManager.get("ANTHROPIC_MODEL_NAME") || process.env.ANTHROPIC_MODEL_NAME || "claude-sonnet-4-6-20260218",
          temperature: 0,
          anthropicApiKey: apiKey,
          maxTokens: 4000,
          maxRetries,
        });
      }

      case "gemini":
      case "google": {
        const apiKey = ConfigManager.get("GOOGLE_API_KEY") || process.env.GOOGLE_API_KEY;
        if (!apiKey) throw new Error("GOOGLE_API_KEY is not set in config.");
        return new ChatGoogleGenerativeAI({
          modelName: ConfigManager.get("GEMINI_MODEL_NAME") || process.env.GEMINI_MODEL_NAME || "gemini-2.5-pro",
          temperature: 0,
          apiKey: apiKey,
          maxOutputTokens: 4000,
          maxRetries,
        });
      }

      case "ollama":
      case "local": {
        const baseUrl =
          ConfigManager.get("OLLAMA_BASE_URL") ||
          ConfigManager.get("OLLAMA_URL") ||
          process.env.OLLAMA_BASE_URL ||
          process.env.OLLAMA_URL ||
          "http://localhost:11434";
        const modelName =
          ConfigManager.get("OLLAMA_MODEL_NAME") ||
          ConfigManager.get("OLLAMA_MODEL") ||
          process.env.OLLAMA_MODEL_NAME ||
          process.env.OLLAMA_MODEL ||
          "llama3.1:latest";
        const normalizedBaseUrl = baseUrl.replace(/\/$/, "");
        return new ChatOpenAI({
          modelName,
          temperature: 0,
          configuration: {
            baseURL: `${normalizedBaseUrl}/v1`,
            defaultHeaders: {
              "User-Agent": "DrogonClaw/1.0",
            },
          },
          openAIApiKey: "ollama",
          maxRetries,
        });
      }
        
      case "openrouter": {
        const apiKey = ConfigManager.get("OPENROUTER_API_KEY") || process.env.OPENROUTER_API_KEY;
        if (!apiKey) throw new Error("OPENROUTER_API_KEY is not set in config.");
        return new ChatOpenAI({
          modelName: ConfigManager.get("OPENROUTER_MODEL_NAME") || process.env.OPENROUTER_MODEL_NAME || "anthropic/claude-3.5-sonnet",
          temperature: 0,
          configuration: {
            baseURL: "https://openrouter.ai/api/v1",
          },
          openAIApiKey: apiKey,
          maxTokens: 4000,
          maxRetries,
        });
      }
      
      default:
        throw new Error(
          `Unsupported AI_PROVIDER: "${provider}". Supported: openai, anthropic, gemini, ollama, openrouter`
        );
    }
  } catch (e: any) {
    console.error(chalk.red(`\n[LLM Error] ${e.message}`));
    throw e;
  }
}
