import { select, input, password, confirm } from "@inquirer/prompts";
import chalk from "chalk";
import fs from "fs";
import path from "path";

const ENV_PATH = path.join(process.cwd(), ".env");

export async function runOnboarding(): Promise<void> {
  console.log(chalk.red("\n🔥 Welcome to DrogonClaw Initialization 🔥"));
  console.log(chalk.gray("Let's get your autonomous agent configured.\n"));

  const setupMethod = await select({
    message: "Choose your setup method:",
    choices: [
      { name: "⚡ Quick Setup (Default: OpenAI + gpt-4o)", value: "quick" },
      { name: "🛠️  Custom Setup (Choose Provider, Model, Telegram)", value: "custom" },
    ],
  });

  let envConfig = "";

  if (setupMethod === "quick") {
    console.log(chalk.cyan("\nQuick Setup Selected: Using OpenAI (gpt-4o)"));
    const apiKey = await input({ 
      message: "Enter your OpenAI API Key (sk-...):", 
      validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
    });
    envConfig = `AI_PROVIDER=openai\nOPENAI_API_KEY=${apiKey}\nOPENAI_MODEL_NAME=gpt-4o\n`;
  } else {
    const provider = await select({
      message: "Which AI Provider would you like to use for the agent?",
      choices: [
        { name: "OpenAI", value: "openai" },
        { name: "Anthropic (Claude)", value: "anthropic" },
        { name: "Google Gemini", value: "gemini" },
        { name: "OpenRouter (Aggregator)", value: "openrouter" },
        { name: "Ollama (Local)", value: "ollama" }
      ],
    });

    envConfig = `AI_PROVIDER=${provider}\n\n`;

    if (provider === "openai") {
      const apiKey = await input({ 
        message: "Enter your OpenAI API Key (sk-...) [Get one at https://platform.openai.com/api-keys]:", 
        validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
      });
      const model = await select({
        message: "Select the OpenAI model to use:",
        choices: [
          { name: "gpt-4o (Recommended - Best performance & speed)", value: "gpt-4o" },
          { name: "gpt-4o-mini (Faster & cheaper)", value: "gpt-4o-mini" },
          { name: "gpt-4-turbo", value: "gpt-4-turbo" },
          { name: "o1-preview (Advanced reasoning)", value: "o1-preview" }
        ]
      });
      envConfig += `OPENAI_API_KEY=${apiKey}\nOPENAI_MODEL_NAME=${model}\n`;
    } else if (provider === "anthropic") {
      const apiKey = await input({ 
        message: "Enter your Anthropic API Key (sk-ant-...) [Get one at https://console.anthropic.com/settings/keys]:", 
        validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
      });
      const model = await select({
        message: "Select the Anthropic model to use:",
        choices: [
          { name: "Claude 3.5 Sonnet (Recommended - Best overall)", value: "claude-3-5-sonnet-20240620" },
          { name: "Claude 3 Opus (Most powerful)", value: "claude-3-opus-20240229" },
          { name: "Claude 3 Haiku (Fastest)", value: "claude-3-haiku-20240307" }
        ]
      });
      envConfig += `ANTHROPIC_API_KEY=${apiKey}\nANTHROPIC_MODEL_NAME=${model}\n`;
    } else if (provider === "gemini") {
      const apiKey = await input({ 
        message: "Enter your Google Gemini API Key [Get one at https://aistudio.google.com/app/apikey]:", 
        validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
      });
      const model = await select({
        message: "Select the Google Gemini model to use:",
        choices: [
          { name: "Gemini 2.5 Pro (Recommended - Best performance)", value: "gemini-2.5-pro" },
          { name: "Gemini 2.5 Flash (Fastest)", value: "gemini-2.5-flash" },
          { name: "Gemini 3 Pro Preview (Experimental)", value: "gemini-3-pro-preview" },
          { name: "Gemini 2.0 Flash (Legacy free tier)", value: "gemini-2.0-flash" }
        ]
      });
      envConfig += `GOOGLE_API_KEY=${apiKey}\nGEMINI_MODEL_NAME=${model}\n`;
    } else if (provider === "openrouter") {
      const apiKey = await input({ 
        message: "Enter your OpenRouter API Key [Get one at https://openrouter.ai/keys]:", 
        validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
      });
      const model = await select({
        message: "Select the OpenRouter model to use:",
        choices: [
          { name: "Claude Sonnet 4.6 (anthropic/claude-sonnet-4.6)", value: "anthropic/claude-sonnet-4.6" },
          { name: "Llama 3.3 70B (meta-llama/llama-3.3-70b-instruct)", value: "meta-llama/llama-3.3-70b-instruct" },
          { name: "DeepSeek V3 (deepseek/deepseek-chat)", value: "deepseek/deepseek-chat" },
          { name: "Google Gemini 2.5 Pro (google/gemini-2.5-pro)", value: "google/gemini-2.5-pro" }
        ]
      });
      envConfig += `OPENROUTER_API_KEY=${apiKey}\nOPENROUTER_MODEL_NAME=${model}\n`;
    } else if (provider === "ollama") {
      const baseUrl = await input({ message: "Enter your Ollama base URL:", default: "http://localhost:11434" });
      const model = await select({
        message: "Select the Ollama model to use:",
        choices: [
          { name: "llama3 (Recommended)", value: "llama3" },
          { name: "mistral", value: "mistral" },
          { name: "gemma", value: "gemma" },
          { name: "qwen", value: "qwen" }
        ]
      });
      envConfig += `OLLAMA_BASE_URL=${baseUrl}\nOLLAMA_MODEL_NAME=${model}\n`;
    }

    console.log("");
    const setupTelegram = await confirm({ message: "Would you like to configure the Telegram Bot interface?", default: false });
    
    if (setupTelegram) {
      const telegramToken = await input({ 
        message: "Enter your Telegram Bot Token:", 
        validate: (input) => input.trim().length > 0 ? true : "Telegram Token cannot be empty!"
      });
      console.log(chalk.gray("\nTo secure your bot, we need your Telegram Chat ID so no one else can control it."));
      console.log(chalk.gray("You can find this by messaging @userinfobot on Telegram."));
      const chatId = await input({ 
        message: "Enter your Telegram Chat ID:", 
        validate: (input) => input.trim().length > 0 ? true : "Chat ID cannot be empty! Security is mandatory."
      });
      envConfig += `\nTELEGRAM_TOKEN=${telegramToken}\nTELEGRAM_CHAT_ID=${chatId}\n`;
    }
  }

  console.log("");
  const save = await confirm({ message: "Save configuration?", default: true });

  if (save) {
    fs.writeFileSync(ENV_PATH, envConfig, "utf-8");
    console.log(chalk.green("\n✅ Configuration saved!"));
  } else {
    console.log(chalk.yellow("\n⚠️ Configuration aborted. You will need to manually configure the agent to run DrogonClaw."));
    process.exit(1);
  }
}

export function isEnvConfigured(): boolean {
  if (!fs.existsSync(ENV_PATH)) return false;
  
  const content = fs.readFileSync(ENV_PATH, "utf-8");
  
  // Basic check: Ensure AI_PROVIDER exists
  if (!content.includes("AI_PROVIDER=")) return false;

  // Function to extract a specific key's value and check if it's non-empty
  const hasValidValue = (key: string): boolean => {
    const regex = new RegExp(`^${key}=(.*)$`, "m");
    const match = content.match(regex);
    return match ? match[1].trim().length > 0 : false;
  };

  // Ensure at least one required provider key has a non-empty value
  return (
    hasValidValue("OPENAI_API_KEY") ||
    hasValidValue("ANTHROPIC_API_KEY") ||
    hasValidValue("GOOGLE_API_KEY") ||
    hasValidValue("OPENROUTER_API_KEY") ||
    hasValidValue("OLLAMA_BASE_URL")
  );
}
