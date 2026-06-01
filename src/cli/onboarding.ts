import { select, input, password, confirm } from "@inquirer/prompts";
import chalk from "chalk";
import { ConfigManager, DrogonConfig } from "../core/config-manager.js";

async function fetchOllamaModels(baseUrl: string): Promise<string[] | null> {
  try {
    const url = baseUrl.replace(/\/$/, "") + "/api/tags";
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 3000);
    const res = await fetch(url, { signal: controller.signal });
    clearTimeout(timeoutId);
    if (res.ok) {
      const json = await res.json() as any;
      if (json.models && json.models.length > 0) {
        return json.models.map((m: any) => m.name);
      }
    }
  } catch (e) {
    // Ignore error, server might be down or not configured
  }
  return null;
}

export async function runOnboarding(): Promise<void> {
  console.log(chalk.red("🐉 DrogonClaw — I speak fluent bash, mild sarcasm, and aggressive tab-completion energy.\n"));
  
  console.log(chalk.red("▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄"));
  console.log(chalk.red("██████╗ ██████╗  ██████╗  ██████╗  ██████╗ ███╗   ██╗ ██████╗██╗      █████╗ ██╗    ██╗"));
  console.log(chalk.red("██╔══██╗██╔══██╗██╔═══██╗██╔════╝ ██╔═══██╗████╗  ██║██╔════╝██║     ██╔══██╗██║    ██║"));
  console.log(chalk.red("██║  ██║██████╔╝██║   ██║██║  ███╗██║   ██║██╔██╗ ██║██║     ██║     ███████║██║ █╗ ██║"));
  console.log(chalk.red("██║  ██║██╔══██╗██║   ██║██║   ██║██║   ██║██║╚██╗██║██║     ██║     ██╔══██║██║███╗██║"));
  console.log(chalk.red("██████╔╝██║  ██║╚██████╔╝╚██████╔╝╚██████╔╝██║ ╚████║╚██████╗███████╗██║  ██║╚███╔███╔╝"));
  console.log(chalk.red("╚═════╝ ╚═╝  ╚═╝ ╚═════╝  ╚═════╝  ╚═════╝ ╚═╝  ╚═══╝ ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝ "));
  console.log(chalk.red("▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀"));
  console.log(chalk.red("                                  🐉 DROGONCLAW 🐉\n"));

  console.log(chalk.cyan("┌  DrogonClaw setup"));
  console.log(chalk.cyan("│"));
  console.log(chalk.cyan("◇  Security disclaimer ──────────────────────────────────────────────────────────────────────╮"));
  console.log(chalk.cyan("│                                                                                            │"));
  console.log(chalk.cyan("│  DrogonClaw is a hobby project and still in beta. Expect sharp edges.                      │"));
  console.log(chalk.cyan("│  By default, DrogonClaw is a personal agent: one trusted operator boundary.                │"));
  console.log(chalk.cyan("│  This bot can read files and run actions if tools are enabled.                             │"));
  console.log(chalk.cyan("│  A bad prompt can trick it into doing unsafe things.                                       │"));
  console.log(chalk.cyan("│                                                                                            │"));
  console.log(chalk.cyan("│  DrogonClaw is not a hostile multi-tenant boundary by default.                             │"));
  console.log(chalk.cyan("│  If you're not comfortable with security hardening, do not expose this to the internet.    │"));
  console.log(chalk.cyan("│                                                                                            │"));
  console.log(chalk.cyan("├────────────────────────────────────────────────────────────────────────────────────────────╯"));
  console.log(chalk.cyan("│"));

  const proceed = await confirm({
    message: "I understand this is personal-by-default and requires lock-down if exposed. Continue?",
    default: true
  });

  if (!proceed) {
    console.log(chalk.yellow("\n  [-] DrogonClaw initialization aborted for safety."));
    process.exit(1);
  }

  const envConfigMap = new Map<string, string>();

  // Helper Functions for Wizards
  async function configureAI() {
    while (true) {
      console.log(chalk.cyan("\n  [Step 1: AI Provider Configuration]"));
      console.log(chalk.gray("  For Novices: We recommend OpenAI (gpt-4o) or Anthropic (Claude 3.5 Sonnet)"));
      console.log(chalk.gray("  For Experts: Local models (Ollama) can be used, but require tool-support (e.g. llama3.1)"));
      
      const provider = await select({
        message: "Select your AI Backend:",
        choices: [
          { name: "OpenAI (Recommended for Novices)", value: "openai" },
          { name: "Anthropic (Claude) (Recommended for Advanced Coding)", value: "anthropic" },
          { name: "Google Gemini", value: "gemini" },
          { name: "OpenRouter (Model Aggregator)", value: "openrouter" },
          { name: "Ollama (Local / On-Premise) (Advanced)", value: "ollama" },
        ],
      });

      // Clear old keys
      const aiKeys = ["AI_PROVIDER", "OPENAI_API_KEY", "OPENAI_MODEL_NAME", "ANTHROPIC_API_KEY", "ANTHROPIC_MODEL_NAME", "GOOGLE_API_KEY", "GEMINI_MODEL_NAME", "OPENROUTER_API_KEY", "OPENROUTER_MODEL_NAME", "OLLAMA_BASE_URL", "OLLAMA_URL", "OLLAMA_MODEL_NAME", "OLLAMA_MODEL", "OLLAMA_PING_TIMEOUT_MS"];
      aiKeys.forEach(k => envConfigMap.delete(k));

      envConfigMap.set("AI_PROVIDER", provider);

      if (provider === "openai") {
        const apiKey = await password({ 
          message: "Enter your OpenAI API Key (sk-...):", 
          validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
        });
        const model = await select({
          message: "Select the OpenAI model:",
          choices: [
            { name: "gpt-4o (Best overall)", value: "gpt-4o" },
            { name: "gpt-4o-mini (Faster & cheaper)", value: "gpt-4o-mini" },
            { name: "o1-preview (Advanced reasoning)", value: "o1-preview" },
            { name: "Cancel & Re-select Provider", value: "back" }
          ]
        });
        if (model === "back") continue;

        envConfigMap.set("OPENAI_API_KEY", apiKey);
        envConfigMap.set("OPENAI_MODEL_NAME", model);

      } else if (provider === "anthropic") {
        const apiKey = await password({ 
          message: "Enter your Anthropic API Key (sk-ant-...):", 
          validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
        });
        const model = await select({
          message: "Select the Anthropic model:",
          choices: [
            { name: "Claude 3.5 Sonnet (Best overall)", value: "claude-3-5-sonnet-20240620" },
            { name: "Claude 3 Opus (Most powerful)", value: "claude-3-opus-20240229" },
            { name: "Cancel & Re-select Provider", value: "back" }
          ]
        });
        if (model === "back") continue;

        envConfigMap.set("ANTHROPIC_API_KEY", apiKey);
        envConfigMap.set("ANTHROPIC_MODEL_NAME", model);

      } else if (provider === "gemini") {
        const apiKey = await password({ 
          message: "Enter your Google Gemini API Key:", 
          validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
        });
        const model = await select({
          message: "Select the Gemini model:",
          choices: [
            { name: "Gemini 2.5 Pro (Best performance)", value: "gemini-2.5-pro" },
            { name: "Gemini 2.5 Flash (Fastest)", value: "gemini-2.5-flash" },
            { name: "Cancel & Re-select Provider", value: "back" }
          ]
        });
        if (model === "back") continue;

        envConfigMap.set("GOOGLE_API_KEY", apiKey);
        envConfigMap.set("GEMINI_MODEL_NAME", model);

      } else if (provider === "openrouter") {
        const apiKey = await password({ 
          message: "Enter your OpenRouter API Key:", 
          validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
        });
        const model = await input({ 
          message: "Enter the OpenRouter Model string (e.g. anthropic/claude-3.5-sonnet):",
          default: "anthropic/claude-3.5-sonnet",
        });

        envConfigMap.set("OPENROUTER_API_KEY", apiKey);
        envConfigMap.set("OPENROUTER_MODEL_NAME", model);

      } else if (provider === "ollama") {
        console.log(chalk.gray("\n  [Ollama Local Engine]"));
        const baseUrl = await input({ message: "Enter your Ollama Base URL:", default: "http://localhost:11434" });
        
        console.log(chalk.gray("  Querying local Ollama instance for available models..."));
        const availableModels = await fetchOllamaModels(baseUrl);
        
        let finalModel = "";

        if (availableModels && availableModels.length > 0) {
          console.log(chalk.green(`  [+] Found ${availableModels.length} local models.`));
          const choices = availableModels.map(m => ({ name: m, value: m }));
          choices.push({ name: "Manual Entry (Type a model name manually)", value: "manual" });
          choices.push({ name: "Cancel & Re-select Provider", value: "back" });

          const selectedModel = await select({
            message: "Select an active local model:",
            choices: choices
          });

          if (selectedModel === "back") continue;
          
          if (selectedModel === "manual") {
            finalModel = await input({ message: "Enter the exact model name (e.g., llama3.1:8b):" });
          } else {
            finalModel = selectedModel;
          }
        } else {
          console.log(chalk.yellow("  [!] Could not connect to Ollama or no models found. Ensure Ollama is running."));
          const manualAction = await select({
             message: "How would you like to proceed?",
             choices: [
               { name: "Enter Model Name Manually", value: "manual" },
               { name: "Cancel & Re-select Provider", value: "back" }
             ]
          });
          if (manualAction === "back") continue;
          finalModel = await input({ message: "Enter the exact model name (e.g., llama3.1:8b):" });
        }

        envConfigMap.set("OLLAMA_BASE_URL", baseUrl);
        envConfigMap.set("OLLAMA_URL", baseUrl);
        envConfigMap.set("OLLAMA_MODEL_NAME", finalModel);
        envConfigMap.set("OLLAMA_MODEL", finalModel);
        envConfigMap.set("OLLAMA_PING_TIMEOUT_MS", "10000");
      }
      
      break; // Successfully configured AI
    }
  }

  async function configureTelegram() {
    console.log(chalk.cyan("\n  [Step 2: C2 Interface Setup]"));
    const tgAction = await select({
       message: "Would you like to setup remote Command & Control via Telegram?",
       choices: [
         { name: "Yes, configure Telegram Bot", value: "yes" },
         { name: "No, skip for now (CLI only)", value: "skip" }
       ]
    });
    
    if (tgAction === "skip") {
       envConfigMap.delete("TELEGRAM_TOKEN");
       envConfigMap.delete("TELEGRAM_CHAT_ID");
       return;
    }

    const telegramToken = await password({ 
      message: "Enter your Telegram Bot Token:", 
      validate: (input) => input.trim().length > 0 ? true : "Telegram Token cannot be empty!"
    });
    console.log(chalk.gray("\nTo secure your bot, we need your Telegram Chat ID so no one else can control it."));
    console.log(chalk.gray("You can find this by messaging @userinfobot on Telegram."));
    const chatId = await input({ 
      message: "Enter your Telegram Chat ID:", 
      validate: (input) => input.trim().length > 0 ? true : "Chat ID cannot be empty! Security is mandatory."
    });
    
    envConfigMap.set("TELEGRAM_TOKEN", telegramToken);
    envConfigMap.set("TELEGRAM_CHAT_ID", chatId);
  }

  // --- WIZARD FLOW ---
  console.log(chalk.cyan("\n╭─ Initializing Setup Wizard ─────────────────────────────"));
  await configureAI();
  await configureTelegram();

  // --- REVIEW STATE ---
  while (true) {
    console.log(chalk.cyan("\n╭─ Configuration Summary ─────────────────────────────────"));
    
    const provider = envConfigMap.get("AI_PROVIDER") || "Unknown";
    let model = "";
    if (provider === "openai") model = envConfigMap.get("OPENAI_MODEL_NAME") || "";
    if (provider === "anthropic") model = envConfigMap.get("ANTHROPIC_MODEL_NAME") || "";
    if (provider === "gemini") model = envConfigMap.get("GEMINI_MODEL_NAME") || "";
    if (provider === "openrouter") model = envConfigMap.get("OPENROUTER_MODEL_NAME") || "";
    if (provider === "ollama") model = envConfigMap.get("OLLAMA_MODEL_NAME") || "";

    console.log(chalk.gray("│  ") + chalk.white("AI Engine:    ") + chalk.green(`${provider} (${model})`));
    
    const tgStatus = envConfigMap.has("TELEGRAM_TOKEN") ? chalk.green("Enabled") : chalk.gray("Disabled");
    console.log(chalk.gray("│  ") + chalk.white("C2 Interface: ") + tgStatus);
    console.log(chalk.cyan("╰─────────────────────────────────────────────────────────"));

    const finalAction = await select({
      message: "Review your setup:",
      choices: [
        { name: "[+] Finish & Boot DrogonClaw", value: "boot" },
        { name: "[✎] Edit AI Provider", value: "edit_ai" },
        { name: "[✎] Edit Telegram Setup", value: "edit_tg" },
        { name: "[x] Abort Setup", value: "abort" }
      ]
    });

    if (finalAction === "abort") {
      console.log(chalk.yellow("\n  [-] Configuration aborted."));
      process.exit(1);
    }

    if (finalAction === "edit_ai") {
      await configureAI();
      continue;
    }

    if (finalAction === "edit_tg") {
      await configureTelegram();
      continue;
    }

    if (finalAction === "boot") {
      break;
    }
  }

  // Save the configuration
  const newConfig: DrogonConfig = {};
  for (const [key, value] of envConfigMap.entries()) {
    (newConfig as any)[key] = value;
  }
  
  ConfigManager.save(newConfig);
  console.log(chalk.green("\n  [+] Profile saved successfully!"));
  console.log(chalk.cyan("\n  ╭─ System Ready ────────────────────────────────────────"));
  console.log(chalk.gray("  │  Neural pathways are configured."));
  console.log(chalk.gray("  │  DrogonClaw will now initialize and seize control."));
  console.log(chalk.cyan("  ╰───────────────────────────────────────────────────────\n"));
}

export function isEnvConfigured(): boolean {
  return (
    !!ConfigManager.get("OPENAI_API_KEY") ||
    !!ConfigManager.get("ANTHROPIC_API_KEY") ||
    !!ConfigManager.get("GOOGLE_API_KEY") ||
    !!ConfigManager.get("OPENROUTER_API_KEY") ||
    !!ConfigManager.get("OLLAMA_BASE_URL") || !!ConfigManager.get("OLLAMA_MODEL_NAME") || !!ConfigManager.get("OLLAMA_MODEL")
  );
}
