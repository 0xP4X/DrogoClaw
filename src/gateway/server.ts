import { Telegraf } from "telegraf";
import chalk from "chalk";
import { AgentOrchestrator } from "../agent/orchestrator";

/**
 * DrogonClaw Telegram Gateway
 * 
 * Allows remote Command & Control of the AI Agent via a Telegram Bot.
 * Supports natural language interaction and streams execution updates securely.
 */
export class GatewayServer {
  private bot: Telegraf;
  private orchestrator: AgentOrchestrator;
  private allowedChatId: string;

  constructor() {
    const token = process.env.TELEGRAM_TOKEN;
    this.allowedChatId = process.env.TELEGRAM_CHAT_ID || "";

    if (!token) {
      console.log(chalk.red("❌ TELEGRAM_TOKEN is not set in .env"));
      process.exit(1);
    }

    this.bot = new Telegraf(token);
    this.orchestrator = new AgentOrchestrator();
  }

  public async start(): Promise<void> {
    await this.orchestrator.initialize();
    
    if (!this.orchestrator.isReady()) {
      console.log(chalk.red("❌ Agent Core failed to initialize. Check your AI Provider keys."));
      process.exit(1);
    }

    // Security Whitelist Middleware
    this.bot.use(async (ctx, next) => {
      const chatId = String(ctx.chat?.id);
      
      if (!this.allowedChatId) {
        console.log(chalk.red(`[Security Block] No TELEGRAM_CHAT_ID set in .env. Dropping message from Chat ID: ${chatId}. Security is mandatory.`));
        await ctx.reply("❌ Administrator has not configured a secure Chat ID whitelist. Access denied.");
        return;
      }

      if (chatId !== this.allowedChatId) {
        console.log(chalk.yellow(`[Security Warning] Unauthorized access attempt from Chat ID: ${chatId}`));
        await ctx.reply("❌ Unauthorized. You are not the administrator of this DrogonClaw instance.");
        return;
      }

      return next();
    });

    // Start command (optional, but standard for Telegram bots)
    this.bot.start((ctx) => {
      ctx.reply(
        "🐉🔥 *DrogonClaw Online*\n\n" +
        "I am your autonomous offensive security agent. Send me a natural language instruction (e.g., 'Scan target.com for open ports') and I will execute the mission.",
        { parse_mode: "Markdown" }
      );
    });

    // Handle all natural language text messages
    this.bot.on("text", async (ctx) => {
      const instruction = ctx.message.text;

      // Ignore standard commands if user types them by habit
      if (instruction.startsWith("/")) {
        if (instruction === "/new") {
          this.orchestrator.newSession();
          await ctx.reply("🔄 Agent memory wiped. Ready for a new mission.");
          return;
        }
        if (instruction !== "/start") {
          await ctx.reply("I operate on natural language. Just tell me what to do directly. (Type /new to wipe memory)");
          return;
        }
      }

      console.log(chalk.cyan(`\n[Telegram Gateway] Received mission: "${instruction}"`));
      
      // Send initial acknowledgment
      const statusMessage = await ctx.reply("🔥 Mission acknowledged. Engaging Orchestration Core...");

      try {
        // Execute the agent, passing a callback to stream tool updates back to Telegram
        const finalResponse = await this.orchestrator.execute(instruction, async (toolName, args) => {
          // Send intermediate status updates as the agent works
          try {
            await ctx.telegram.editMessageText(
              ctx.chat.id,
              statusMessage.message_id,
              undefined,
              `⚙️ *Executing Tool:* \`${toolName}\`\n_Analyzing findings..._`,
              { parse_mode: "Markdown" }
            );
          } catch (e) {
            // Ignore Telegram rate limit / same-text edit errors
          }
        });

        // Delete the "Executing Tool" placeholder
        try {
          await ctx.telegram.deleteMessage(ctx.chat.id, statusMessage.message_id);
        } catch (e) { }

        // Send the final comprehensive report
        // Telegram limits messages to 4096 chars, so we chunk if necessary
        const chunks = this.chunkString(finalResponse, 4000);
        for (const chunk of chunks) {
          await ctx.reply(`📋 *Agent Report:*\n\n${chunk}`, { parse_mode: "Markdown" });
        }

        console.log(chalk.green(`[Telegram Gateway] Mission complete. Report delivered.`));

      } catch (error: any) {
        console.error(chalk.red(`[Telegram Gateway Error] ${error.message}`));
        await ctx.reply(`❌ *Mission Failed:*\n\n${error.message}`, { parse_mode: "Markdown" });
      }
    });

    // Launch the bot polling loop
    this.bot.launch();
    console.log(chalk.green("✅ DrogonClaw Telegram Gateway is listening for instructions..."));

    // Enable graceful stop
    process.once("SIGINT", () => this.bot.stop("SIGINT"));
    process.once("SIGTERM", () => this.bot.stop("SIGTERM"));
  }

  /**
   * Helper to chunk large agent responses so they fit within Telegram's message limits
   */
  private chunkString(str: string, size: number): string[] {
    const numChunks = Math.ceil(str.length / size);
    const chunks = new Array(numChunks);
    for (let i = 0, o = 0; i < numChunks; ++i, o += size) {
      chunks[i] = str.substr(o, size);
    }
    return chunks;
  }
}
