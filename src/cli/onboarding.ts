import { select, input, password, confirm } from "@inquirer/prompts";
import chalk from "chalk";
import boxen from "boxen";
import ora from "ora";
import os from "os";
import crypto from "crypto";
import { exec } from "child_process";
import { createClient } from "@supabase/supabase-js";
import { ConfigManager, DrogonConfig } from "../core/config-manager.js";

// ─────────────────────────────────────────────────────────────────────────────
// F1 fix: Supabase credentials are NEVER hardcoded.
// They must be set in environment variables or .drogonclaw/.env (local only).
// ─────────────────────────────────────────────────────────────────────────────
function getSupabaseCredentials(): { url: string; anonKey: string } {
  const url = process.env.SUPABASE_URL;
  const anonKey = process.env.SUPABASE_ANON_KEY;

  if (!url || !anonKey) {
    console.error(chalk.red(
      "\n  [x] Missing SUPABASE_URL or SUPABASE_ANON_KEY environment variables.\n" +
      "      These must be set before running DrogonClaw.\n" +
      "      See .env.example for configuration reference.\n"
    ));
    process.exit(1);
  }
  return { url, anonKey };
}

// ─────────────────────────────────────────────────────────────────────────────
// F13 fix: A stable, harder-to-spoof hardware ID derived from multiple
// system attributes hashed with SHA-256. Not cryptographically binding,
// but significantly harder to trivially spoof than hostname alone.
// ─────────────────────────────────────────────────────────────────────────────
export function computeHardwareId(): string {
  const cpus = os.cpus();
  const cpuModel = cpus.length > 0 ? cpus[0].model : "unknown-cpu";
  const raw = [
    os.hostname(),
    os.arch(),
    os.platform(),
    cpuModel,
    String(os.totalmem()),
  ].join("|");
  return crypto.createHash("sha256").update(raw).digest("hex").slice(0, 48);
}

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
    // Ignore — server might be down or not configured
  }
  return null;
}

export async function runOnboarding(): Promise<void> {
  console.log(chalk.red("[*] DrogonClaw — I speak fluent bash, mild sarcasm, and aggressive tab-completion energy.\n"));
  
  console.log(chalk.red("▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄"));
  console.log(chalk.red("██████╗ ██████╗  ██████╗  ██████╗  ██████╗ ███╗   ██╗ ██████╗██╗      █████╗ ██╗    ██╗"));
  console.log(chalk.red("██╔══██╗██╔══██╗██╔═══██╗██╔════╝ ██╔═══██╗████╗  ██║██╔════╝██║     ██╔══██╗██║    ██║"));
  console.log(chalk.red("██║  ██║██████╔╝██║   ██║██║  ███╗██║   ██║██╔██╗ ██║██║     ██║     ███████║██║ █╗ ██║"));
  console.log(chalk.red("██║  ██║██╔══██╗██║   ██║██║   ██║██║   ██║██║╚██╗██║██║     ██║     ██╔══██║██║███╗██║"));
  console.log(chalk.red("██████╔╝██║  ██║╚██████╔╝╚██████╔╝╚██████╔╝██║ ╚████║╚██████╗███████╗██║  ██║╚███╔███╔╝"));
  console.log(chalk.red("╚═════╝ ╚═╝  ╚═╝ ╚═════╝  ╚═════╝  ╚═════╝ ╚═╝  ╚═══╝ ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝ "));
  console.log(chalk.red("▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀"));
  console.log(chalk.red("                                  [*] DROGONCLAW [*]\n"));

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

  console.clear();

  const envConfigMap = new Map<string, string>();

  // ─────────────────────────────────────────────────────────────────────────────
  // License Configuration
  // F1 fix: no hardcoded Supabase credentials — read from environment
  // F2 fix: secret_token is generated locally and included in the INSERT
  // F13 fix: hardware ID is a SHA-256 hash of multiple stable system attributes
  // F16 fix: Realtime channel is scoped; secret_token stored locally
  // ─────────────────────────────────────────────────────────────────────────────
  async function configureLicense() {
    console.log(chalk.cyan("\n  [Step 0: License Verification]"));

    const { url: supabaseUrl, anonKey: supabaseKey } = getSupabaseCredentials();
    const supabase = createClient(supabaseUrl, supabaseKey);

    // F13 fix: stable SHA-256 hardware ID
    const hardwareId = computeHardwareId();

    // Generate device_code and secret_token locally
    const deviceCode = "dc_req_" + crypto.randomBytes(8).toString("hex");
    // F2/F16 fix: secret_token is a random UUID — stored locally and sent with INSERT
    // The RLS SELECT policy scopes Realtime delivery to only clients that know this token
    const secretToken = crypto.randomUUID();

    try {
      const expiresAt = new Date();
      expiresAt.setMinutes(expiresAt.getMinutes() + 15);

      const { error: insertError } = await supabase
        .from("device_requests")
        .insert({
          device_code: deviceCode,
          secret_token: secretToken,  // F2 fix: required field, non-empty
          hardware_id: hardwareId,
          expires_at: expiresAt.toISOString(),
        });

      if (insertError) {
        throw new Error(`Failed to initialize device flow: ${insertError.message}`);
      }

      const verificationUrl = `https://drogonclaw.xyz/checkout?device_code=${deviceCode}`;

      const boxContent = `
${chalk.white.bold("Authentication Required")}

To activate this device, please open the following link:
${chalk.blueBright.underline(verificationUrl)}

Device Code: ${chalk.yellow.bold(deviceCode)}
`;
      console.log(
        boxen(boxContent, {
          padding: 1,
          margin: 1,
          borderStyle: "round",
          borderColor: "cyan",
        })
      );

      await input({ message: chalk.gray("Press Enter to open your browser..."), default: "" });

      const platform = os.platform();
      const startCmd = platform === "darwin" ? "open" : platform === "win32" ? "start" : "xdg-open";
      exec(`${startCmd} "${verificationUrl}"`);

      const spinner = ora({
        text: chalk.cyan("Listening for payment confirmation via Realtime WebSockets..."),
        color: "cyan",
        spinner: "dots"
      }).start();

      // F16 fix: Subscribe to the Realtime channel; the RLS SELECT policy ensures
      // only clients whose device_code matches can receive the update.
      await new Promise<void>((resolve, reject) => {
        const timeout = setTimeout(() => {
          spinner.fail(chalk.red("Request timed out after 15 minutes."));
          reject(new Error("Request timed out."));
        }, 15 * 60 * 1000);

        supabase
          .channel(`device_request_${deviceCode}`)
          .on(
            "postgres_changes",
            {
              event: "UPDATE",
              schema: "public",
              table: "device_requests",
              filter: `device_code=eq.${deviceCode}`,
            },
            (payload: any) => {
              if (payload.new.status === "fulfilled" && payload.new.license_key) {
                clearTimeout(timeout);
                spinner.succeed(chalk.green("Payment Confirmed!"));

                const successBox = `
${chalk.green.bold("Device Activated")}
License Key successfully provisioned and bound to this hardware.
`;
                console.log(boxen(successBox, { padding: 1, margin: 1, borderColor: "green" }));

                envConfigMap.set("DROGONCLAW_LICENSE_KEY", payload.new.license_key);
                // Store computed hardware ID so validate-license can bind it
                envConfigMap.set("DROGONCLAW_HARDWARE_ID", hardwareId);
                supabase.removeChannel(supabase.getChannels()[0]);
                resolve();
              }
            }
          )
          .subscribe((status, err) => {
            if (err) {
              spinner.fail(chalk.red("WebSocket connection failed."));
              reject(err);
            }
          });
      });

    } catch (error: any) {
      console.log(chalk.red(`\n  [x] Error: ${error.message}`));
      process.exit(1);
    }
  }

  async function configureAI() {
    while (true) {
      console.log(chalk.cyan("\n  [Step 1: AI Provider Configuration]"));
      console.log(chalk.gray("  For Novices: We recommend OpenAI (gpt-4o) or Anthropic (Claude 3.5 Sonnet)"));
      console.log(chalk.gray("  For Experts: Local models (Ollama) can be used, but MUST support tool-calling (e.g. qwen2.5, mistral-nemo, phi3)"));

      const provider = await select({
        loop: false,
        message: "Select your AI Backend:",
        choices: [
          { name: "[*] OpenAI (Recommended for Novices)", value: "openai" },
          { name: "[*] Anthropic (Claude) (Recommended for Advanced Coding)", value: "anthropic" },
          { name: "[*] Google Gemini", value: "gemini" },
          { name: "🌐 OpenRouter (Model Aggregator)", value: "openrouter" },
          { name: "[*] Ollama (Local / On-Premise) (Advanced)", value: "ollama" },
        ],
      });

      const aiKeys = ["AI_PROVIDER", "OPENAI_API_KEY", "OPENAI_MODEL_NAME", "ANTHROPIC_API_KEY", "ANTHROPIC_MODEL_NAME", "GOOGLE_API_KEY", "GEMINI_MODEL_NAME", "OPENROUTER_API_KEY", "OPENROUTER_MODEL_NAME", "OLLAMA_BASE_URL", "OLLAMA_URL", "OLLAMA_MODEL_NAME", "OLLAMA_MODEL", "OLLAMA_PING_TIMEOUT_MS"];
      aiKeys.forEach(k => envConfigMap.delete(k));
      envConfigMap.set("AI_PROVIDER", provider);

      if (provider === "openai") {
        const apiKey = await password({
          message: "Enter your OpenAI API Key (sk-...):",
          mask: "*",
          validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
        });
        const model = await select({
          loop: false,
          message: "Select the OpenAI model:",
          choices: [
            { name: "[*] gpt-4o (Best overall)", value: "gpt-4o" },
            { name: "[+] gpt-4o-mini (Faster & cheaper)", value: "gpt-4o-mini" },
            { name: "[*] o1-pro (Advanced reasoning)", value: "o1-pro" },
            { name: "[x] Cancel & Re-select Provider", value: "back" }
          ]
        });
        if (model === "back") continue;
        envConfigMap.set("OPENAI_API_KEY", apiKey);
        envConfigMap.set("OPENAI_MODEL_NAME", model);

      } else if (provider === "anthropic") {
        const apiKey = await password({
          message: "Enter your Anthropic API Key (sk-ant-...):",
          mask: "*",
          validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
        });
        const model = await select({
          loop: false,
          message: "Select the Anthropic model:",
          choices: [
            { name: "[+] Claude 4.6 Sonnet (Best overall)", value: "claude-sonnet-4-6-20260218" },
            { name: "[*] Claude 4.8 Opus (Most powerful)", value: "claude-opus-4-8-20260515" },
            { name: "[x] Cancel & Re-select Provider", value: "back" }
          ]
        });
        if (model === "back") continue;
        envConfigMap.set("ANTHROPIC_API_KEY", apiKey);
        envConfigMap.set("ANTHROPIC_MODEL_NAME", model);

      } else if (provider === "gemini") {
        const apiKey = await password({
          message: "Enter your Google Gemini API Key:",
          mask: "*",
          validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
        });
        const model = await select({
          loop: false,
          message: "Select the Gemini model:",
          choices: [
            { name: "[*] Gemini 2.5 Pro (Best performance)", value: "gemini-2.5-pro" },
            { name: "[+] Gemini 2.5 Flash (Fastest)", value: "gemini-2.5-flash" },
            { name: "[x] Cancel & Re-select Provider", value: "back" }
          ]
        });
        if (model === "back") continue;
        envConfigMap.set("GOOGLE_API_KEY", apiKey);
        envConfigMap.set("GEMINI_MODEL_NAME", model);

      } else if (provider === "openrouter") {
        const apiKey = await password({
          message: "Enter your OpenRouter API Key:",
          mask: "*",
          validate: (input) => input.trim().length > 0 ? true : "API Key cannot be empty!"
        });
        const model = await select({
          loop: false,
          message: "Select the OpenRouter model:",
          choices: [
            { name: "[*] Anthropic: Claude 4.6 Sonnet", value: "anthropic/claude-sonnet-4.6" },
            { name: "[*] Anthropic: Claude 4.8 Opus", value: "anthropic/claude-opus-4.8" },
            { name: "[*] OpenAI: GPT-4o", value: "openai/gpt-4o" },
            { name: "[*] OpenAI: o1-pro", value: "openai/o1-pro" },
            { name: "[*] Google: Gemini 2.5 Pro", value: "google/gemini-2.5-pro" },
            { name: "[+] Google: Gemini 2.5 Flash", value: "google/gemini-2.5-flash" },
            { name: "[*] Meta: Hermes 3 Llama 405B", value: "nousresearch/hermes-3-llama-3.1-405b" },
            { name: "[*] Meta: Llama 3.1 70B Instruct", value: "meta-llama/llama-3.1-70b-instruct" },
            { name: "[*] Mistral: Mixtral 8x22B Instruct", value: "mistralai/mixtral-8x22b-instruct" },
            { name: "[*] Mistral: Mistral Large", value: "mistralai/mistral-large" },
            { name: "[x] Cancel & Re-select Provider", value: "back" }
          ]
        });
        if (model === "back") continue;
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
          choices.push({ name: "Cancel & Re-select Provider", value: "back" });

          const selectedModel = await select({
            loop: false,
            message: "Select an active local model:",
            choices,
          });
          if (selectedModel === "back") continue;
          finalModel = selectedModel;
        } else {
          console.log(chalk.yellow("  [!] Could not connect to Ollama or no models found. Ensure Ollama is running."));
          const fallbackModel = await select({
            loop: false,
            message: "Select a standard Ollama model (ensure you pull it later):",
            choices: [
              { name: "[✓] qwen2.5        — Confirmed tool-call support (Recommended)", value: "qwen2.5" },
              { name: "[✓] qwen2.5:7b     — Confirmed tool-call support (Fastest)", value: "qwen2.5:7b" },
              { name: "[✓] mistral-nemo   — Good tool-call support", value: "mistral-nemo" },
              { name: "[✓] phi3           — Lightweight tool-call support", value: "phi3" },
              { name: "[✓] gemma2         — Good reasoning model", value: "gemma2" },
              { name: "[✗] llama3         — WARNING: No tool-call support, will crash", value: "llama3" },
              { name: "[✗] llama3.1       — WARNING: Limited tool-call support", value: "llama3.1" },
              { name: "[x] Cancel & Re-select Provider", value: "back" }
            ]
          });
          if (fallbackModel === "back") continue;
          finalModel = fallbackModel;
        }

        envConfigMap.set("OLLAMA_BASE_URL", baseUrl);
        envConfigMap.set("OLLAMA_URL", baseUrl);
        envConfigMap.set("OLLAMA_MODEL_NAME", finalModel);
        envConfigMap.set("OLLAMA_MODEL", finalModel);
        envConfigMap.set("OLLAMA_PING_TIMEOUT_MS", "10000");
      }

      break;
    }
  }

  async function configureTelegram() {
    console.log(chalk.cyan("\n  [Step 2: C2 Interface Setup]"));
    const tgAction = await select({
      loop: false,
      message: "Would you like to setup remote Command & Control via Telegram?",
      choices: [
        { name: "[+] Yes, configure Telegram Bot", value: "yes" },
        { name: "[-] No, skip for now (CLI only)", value: "skip" }
      ]
    });

    if (tgAction === "skip") {
      envConfigMap.delete("TELEGRAM_TOKEN");
      envConfigMap.delete("TELEGRAM_CHAT_ID");
      return;
    }

    const telegramToken = await password({
      message: "Enter your Telegram Bot Token:",
      mask: "*",
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
  await configureLicense();
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
      loop: false,
      message: "Review your setup:",
      choices: [
        { name: "[+] Finish & Save Configuration", value: "boot" },
        { name: "[*] Edit License Key", value: "edit_license" },
        { name: "[*] Edit AI Provider", value: "edit_ai" },
        { name: "[*] Edit Telegram Setup", value: "edit_tg" },
        { name: "[x] Abort Setup", value: "abort" }
      ]
    });

    if (finalAction === "abort") {
      console.log(chalk.yellow("\n  [-] Configuration aborted."));
      process.exit(1);
    }
    if (finalAction === "edit_license") { await configureLicense(); continue; }
    if (finalAction === "edit_ai") { await configureAI(); continue; }
    if (finalAction === "edit_tg") { await configureTelegram(); continue; }
    if (finalAction === "boot") break;
  }

  // Save the configuration
  const newConfig: DrogonConfig = {};
  for (const [key, value] of envConfigMap.entries()) {
    (newConfig as any)[key] = value;
  }

  ConfigManager.save(newConfig);
  console.log(chalk.green("\n  [+] Profile saved successfully!"));
  console.log(chalk.cyan("\n  ╭─ Setup Complete ───────────────────────────────────────"));
  console.log(chalk.gray("  │  Configuration saved. Run 'drogonclaw start' to launch."));
  console.log(chalk.cyan("  ╰───────────────────────────────────────────────────────\n"));
}

export function isEnvConfigured(): boolean {
  return (
    !!ConfigManager.get("DROGONCLAW_LICENSE_KEY") &&
    (
      !!ConfigManager.get("OPENAI_API_KEY") ||
      !!ConfigManager.get("ANTHROPIC_API_KEY") ||
      !!ConfigManager.get("GOOGLE_API_KEY") ||
      !!ConfigManager.get("OPENROUTER_API_KEY") ||
      !!ConfigManager.get("OLLAMA_BASE_URL") || !!ConfigManager.get("OLLAMA_MODEL_NAME") || !!ConfigManager.get("OLLAMA_MODEL")
    )
  );
}
