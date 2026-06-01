import { Telegraf } from "telegraf";
import chalk from "chalk";
import { AgentOrchestrator } from "../agent/orchestrator";
import { HitL } from "../core/hitl";
import { ConfigManager } from "../core/config-manager";

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
  private noisyMode: boolean = false;

  constructor() {
    const token = ConfigManager.get("TELEGRAM_TOKEN");
    this.allowedChatId = ConfigManager.get("TELEGRAM_CHAT_ID") || "";

    if (!token) {
      console.log(chalk.red("❌ TELEGRAM_TOKEN is not set in .env"));
      process.exit(1);
    }

    this.bot = new Telegraf(token);
    this.orchestrator = new AgentOrchestrator();

    // Global Error Handler for the Bot
    this.bot.catch((err: any) => {
      if (String(err.message).includes("409")) {
        console.error(chalk.red("\n[FATAL] Telegram Bot Conflict detected."));
        console.error(chalk.yellow("You have another instance of this bot running. Terminal C2 and Telegram can coexist, but you cannot run TWO Telegram gateways at once."));
        process.exit(1);
      }
    });
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

    // Human-in-the-Loop Event Listener
    HitL.on("approval_requested", async ({ question, requestId }) => {
      try {
        await this.bot.telegram.sendMessage(this.allowedChatId, `⚠️ *AGENT REQUIRES APPROVAL*\n\n_${question}_\n\nPlease reply directly to this message with your decision.`, { parse_mode: "Markdown" });
      } catch (e) {
        console.error(chalk.red(`[Gateway Error] Failed to send HitL request: ${e}`));
      }
    });

    // Handle all natural language text messages
    this.bot.on("text", async (ctx) => {
      let instruction = ctx.message.text;

      // Ignore standard commands if user types them by habit
      if (instruction.startsWith("/")) {
        if (instruction === "/new") {
          this.orchestrator.newSession();
          await ctx.reply("🔄 Agent memory wiped. Ready for a new mission.");
          return;
        }
        if (instruction === "/report") {
          const statusMessage = await ctx.reply("📝 Compiling raw intelligence into compliance report. Please wait...");
          try {
            const { ReportGenerator } = await import("../core/report-generator");
            const generator = new ReportGenerator(this.orchestrator.getMemoryGraph());
            const { docPath } = await generator.generateReport();
            
            await ctx.telegram.deleteMessage(ctx.chat.id, statusMessage.message_id);
            await ctx.replyWithDocument({ source: docPath }, { caption: "📋 Mission Intelligence Report Generated." });
          } catch (e: any) {
            await ctx.telegram.editMessageText(ctx.chat.id, statusMessage.message_id, undefined, `❌ Report generation failed: ${e.message}`);
          }
          return;
        }
        if (instruction === "/autopilot on") {
          this.orchestrator.autopilotEnabled = true;
          await ctx.reply("🔥 *AUTOPILOT MODE ACTIVATED*\nI will now run autonomously, compile my own tools, and only stop when the objective is completely destroyed. Send your mission.", { parse_mode: "Markdown" });
          return;
        }
        if (instruction === "/autopilot off") {
          this.orchestrator.autopilotEnabled = false;
          await ctx.reply("🛑 Autopilot deactivated. I will return to standard step-by-step execution.");
          return;
        }
        if (instruction === "/noisy") {
          this.noisyMode = !this.noisyMode;
          const status = this.noisyMode ? "ENABLED" : "DISABLED";
          await ctx.reply(`⚡ *Noisy Mode ${status}*\nLive neural telemetry stream is now active.`, { parse_mode: "Markdown" });
          return;
        }

        // Handle mission shortcuts from documentation
        const cmdParts = instruction.split(" ");
        const baseCmd = cmdParts[0].toLowerCase();
        if (["/scan", "/enum", "/recon", "/exploit", "/attack", "/whois", "/nmap"].includes(baseCmd)) {
             // Let these through to the natural language processor by stripping the slash
             instruction = instruction.substring(1); 
        } else if (instruction !== "/start") {
          await ctx.reply("I operate on natural language. Just tell me what to do directly. (Type /new to wipe memory, /autopilot on to enable infinite autonomy, or /report to get a final document)");
          return;
        }
      }

      // If HitL is waiting for a reply, resolve it instead of launching a new execution!
      if (HitL.hasPendingRequest()) {
        HitL.resolveRequest(instruction);
        await ctx.reply("✅ Answer received. Agent execution resuming...");
        return;
      }

      console.log(chalk.cyan(`\n[Telegram Gateway] Received mission: "${instruction}"`));
      
      // Send initial acknowledgment
      const statusMessage = await ctx.reply("🔥 *Mission Acknowledged.*\nInitializing Orchestration Core...", { parse_mode: "Markdown" });

      try {
        // Execute the agent, passing a callback to stream tool updates back to Telegram
        const finalResponse = await this.orchestrator.execute(instruction, async (toolName, args) => {
          // Send intermediate status updates as the agent works
          try {
            if (toolName === "thought") {
              if (this.noisyMode) {
                await ctx.reply(`🧠 *Neural Thought:*\n_${args.thought.substring(0, 1000)}_`, { parse_mode: "Markdown" });
              }
            } else if (toolName === "status") {
              await ctx.telegram.editMessageText(
                ctx.chat.id,
                statusMessage.message_id,
                undefined,
                `📡 *Tactical Status:* ${args.message}`,
                { parse_mode: "Markdown" }
              );
            } else {
              await ctx.telegram.editMessageText(
                ctx.chat.id,
                statusMessage.message_id,
                undefined,
                `⚙️ *Executing Tool:* \`${toolName}\`\n\`\`\`json\n${JSON.stringify(args, null, 2).substring(0, 500)}\`\`\``,
                { parse_mode: "Markdown" }
              );
            }
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
        if (error.name === "HitLPauseError") {
          console.log(chalk.yellow(`\n[HitL] Agent execution suspended. Awaiting human input...`));
          // Do not send a failure message, the user is already pinged.
          return;
        }

        console.error(chalk.red(`[Telegram Gateway Error] ${error.message}`));
        await ctx.reply(`❌ *Mission Failed:*\n\n${error.message}`, { parse_mode: "Markdown" });
      }
    });

    // Launch the bot polling loop
    try {
      this.bot.launch();
      console.log(chalk.green("✅ DrogonClaw Telegram Gateway is listening for instructions..."));
    } catch (e: any) {
      if (e.message.includes("409")) {
        console.error(chalk.red("\n[FATAL ERROR] Telegram Bot Conflict (409)"));
        console.error(chalk.yellow("Another instance of this bot is already running elsewhere."));
        console.error(chalk.gray("This usually happens if you have another terminal open or didn't kill the previous process correctly.\n"));
        console.error(chalk.white("Try running: ") + chalk.cyan("killall node") + chalk.white(" or find the process in Task Manager.\n"));
      } else {
        console.error(chalk.red(`[Fatal Gateway Error] ${e.message}`));
      }
      process.exit(1);
    }

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

// Start the gateway if called directly
const gateway = new GatewayServer();
gateway.start().catch((err) => {
  console.error(chalk.red("❌ Failed to start Telegram Gateway:"), err);
});
