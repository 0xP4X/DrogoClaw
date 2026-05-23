/**
 * DrogonClaw LLM Provider Factory
 * 
 * Supports multiple AI providers — users can plug in any API key.
 * Providers: OpenAI, Anthropic (Claude), Google Gemini, Ollama (local models)
 */
import { ChatOpenAI } from "@langchain/openai";
import { ChatAnthropic } from "@langchain/anthropic";
import { ChatGoogleGenerativeAI } from "@langchain/google-genai";
import { BaseChatModel } from "@langchain/core/language_models/chat_models";
import dotenv from "dotenv";

dotenv.config();

export interface LLMProviderOptions {
  maxRetries?: number;
}

export function getLLMProvider(options?: LLMProviderOptions): BaseChatModel {
  const provider = (process.env.AI_PROVIDER || "openai").toLowerCase();
  const maxRetries = options?.maxRetries;

  switch (provider) {
    case "openai": {
      if (!process.env.OPENAI_API_KEY) {
        throw new Error("OPENAI_API_KEY is required when AI_PROVIDER=openai");
      }
      return new ChatOpenAI({
        modelName: process.env.OPENAI_MODEL_NAME || "gpt-4-turbo",
        temperature: 0,
        openAIApiKey: process.env.OPENAI_API_KEY,
        maxRetries,
      });
    }

    case "anthropic":
    case "claude": {
      if (!process.env.ANTHROPIC_API_KEY) {
        throw new Error("ANTHROPIC_API_KEY is required when AI_PROVIDER=anthropic");
      }
      return new ChatAnthropic({
        modelName: process.env.ANTHROPIC_MODEL_NAME || "claude-sonnet-4-20250514",
        temperature: 0,
        anthropicApiKey: process.env.ANTHROPIC_API_KEY,
        maxRetries,
      });
    }

    case "gemini":
    case "google": {
      if (!process.env.GOOGLE_API_KEY) {
        throw new Error("GOOGLE_API_KEY is required when AI_PROVIDER=gemini");
      }
      return new ChatGoogleGenerativeAI({
        modelName: process.env.GEMINI_MODEL_NAME || "gemini-2.5-pro",
        temperature: 0,
        apiKey: process.env.GOOGLE_API_KEY,
        maxRetries,
      });
    }

    case "ollama":
    case "local": {
      // Ollama uses the OpenAI-compatible API
      const baseUrl = process.env.OLLAMA_BASE_URL || "http://localhost:11434";
      return new ChatOpenAI({
        modelName: process.env.OLLAMA_MODEL_NAME || "llama3",
        temperature: 0,
        configuration: {
          baseURL: `${baseUrl}/v1`,
        },
        openAIApiKey: "ollama", // Ollama doesn't need a real key
        maxRetries,
      });
    }

    case "openrouter": {
      if (!process.env.OPENROUTER_API_KEY) {
        throw new Error("OPENROUTER_API_KEY is required when AI_PROVIDER=openrouter");
      }
      return new ChatOpenAI({
        modelName: process.env.OPENROUTER_MODEL_NAME || "anthropic/claude-sonnet-4.6",
        temperature: 0,
        maxTokens: 2000,
        configuration: {
          baseURL: "https://openrouter.ai/api/v1",
        },
        openAIApiKey: process.env.OPENROUTER_API_KEY,
        maxRetries,
      });
    }

    default:
      throw new Error(
        `Unsupported AI_PROVIDER: "${provider}". ` +
          `Supported: openai, anthropic, gemini, ollama, openrouter`
      );
  }
}
