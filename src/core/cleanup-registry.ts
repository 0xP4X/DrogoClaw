import { execSync } from "child_process";
import fs from "fs";

export class CleanupRegistry {
  private static instance: CleanupRegistry;
  private commands: Array<{ command: string; description: string }> = [];

  private constructor() {}

  public static getInstance(): CleanupRegistry {
    if (!CleanupRegistry.instance) {
      CleanupRegistry.instance = new CleanupRegistry();
    }
    return CleanupRegistry.instance;
  }

  public register(command: string, description: string): void {
    // Avoid duplicate cleanup commands
    if (!this.commands.some((c) => c.command === command)) {
      this.commands.push({ command, description });
      console.log(`\x1b[90m  [OPSEC] Registered cleanup hook: ${description}\x1b[0m`);
    }
  }

  public async executeAll(): Promise<void> {
    if (this.commands.length === 0) return;

    console.log("\n\x1b[31m  [!] EXECUTING OPSEC CLEANUP ROUTINE\x1b[0m");

    // Execute in reverse order (LIFO)
    const reversed = [...this.commands].reverse();

    for (const { command, description } of reversed) {
      try {
        console.log(`\x1b[90m  ├── Cleaning: ${description}\x1b[0m`);
        // We use execSync with a hard timeout so a hanging cleanup command doesn't permanently brick the exit routine
        execSync(command, { timeout: 10000, stdio: "ignore" });
        console.log(`\x1b[32m  ├── [✓] Success\x1b[0m`);
      } catch (err: any) {
        console.log(`\x1b[33m  ├── [x] Failed: ${err.message || "Timeout or Exit Code 1"}\x1b[0m`);
      }
    }
    
    console.log(`\x1b[32m  └── [✓] OPSEC Cleanup Complete\x1b[0m\n`);
    this.commands = []; // Clear after execution
  }
}
