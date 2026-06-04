import { execSync } from "child_process";
import chalk from "chalk";
import { confirm } from "@inquirer/prompts";
import ora from "ora";

interface ToolDef {
  name: string;
  category: string;
  pkg: string; // the apt package name (may differ from the binary name)
}

const REQUIRED_TOOLS: ToolDef[] = [
  // Reconnaissance
  { name: "nmap", category: "Reconnaissance", pkg: "nmap" },
  { name: "masscan", category: "Reconnaissance", pkg: "masscan" },
  { name: "amass", category: "Reconnaissance", pkg: "amass" },
  
  // Web Enumeration
  { name: "gobuster", category: "Web Enumeration", pkg: "gobuster" },
  { name: "ffuf", category: "Web Enumeration", pkg: "ffuf" },
  { name: "sqlmap", category: "Web Enumeration", pkg: "sqlmap" },
  
  // Vulnerability Scanning
  { name: "nuclei", category: "Vulnerability Scanning", pkg: "nuclei" },
  { name: "searchsploit", category: "Vulnerability Scanning", pkg: "exploitdb" },

  // Post-Exploitation
  { name: "go", category: "Post-Exploitation", pkg: "golang" },
  { name: "msfconsole", category: "Post-Exploitation", pkg: "metasploit-framework" }
];

export class HealthChecker {
  
  /**
   * Detect if we're in a Linux-like environment (native Linux, WSL, or Docker)
   * where apt-get is available.
   */
  private canUseApt(): boolean {
    // Native Linux or WSL — both report process.platform === "linux"
    if (process.platform === "linux" || process.platform === "darwin") {
      try {
        execSync("which apt-get", { stdio: "ignore" });
        return true;
      } catch {
        return false;
      }
    }
    return false;
  }

  private checkTool(toolName: string): boolean {
    try {
      const cmd = process.platform === "win32" ? `where ${toolName}` : `which ${toolName}`;
      execSync(cmd, { stdio: "ignore" });
      return true;
    } catch {
      return false;
    }
  }

  private needsSudo(): boolean {
    try {
      // If we're already root, no sudo needed
      const uid = execSync("id -u", { encoding: "utf8" }).trim();
      return uid !== "0";
    } catch {
      return true;
    }
  }

  public async runDiagnostics(): Promise<void> {
    console.log(chalk.bold.white("\n  SYSTEM DIAGNOSTICS\n"));

    const categories = Array.from(new Set(REQUIRED_TOOLS.map(t => t.category)));
    const missingTools: ToolDef[] = [];
    
    let totalInstalled = 0;

    categories.forEach(category => {
      console.log(chalk.cyan(`  ${category}`));
      const toolsInCategory = REQUIRED_TOOLS.filter(t => t.category === category);
      let installedInCategory = 0;

      toolsInCategory.forEach(tool => {
        const isInstalled = this.checkTool(tool.name);
        if (isInstalled) {
          installedInCategory++;
          totalInstalled++;
          console.log(chalk.green(`    [+] ${tool.name}`));
        } else {
          missingTools.push(tool);
          console.log(chalk.red(`    [-] ${tool.name}`) + chalk.gray(" — not installed"));
        }
      });

      const pct = Math.round((installedInCategory / toolsInCategory.length) * 100);
      const pctColor = pct === 100 ? chalk.green : pct > 0 ? chalk.yellow : chalk.red;
      console.log(chalk.gray(`    Readiness: `) + pctColor(`${pct}%`) + "\n");
    });

    const totalPct = Math.round((totalInstalled / REQUIRED_TOOLS.length) * 100);
    const totalColor = totalPct === 100 ? chalk.green : totalPct > 50 ? chalk.yellow : chalk.red;
    console.log(chalk.cyan("  ──────────────────────────────────────────────────────────"));
    console.log(chalk.white("  Overall Readiness: ") + totalColor.bold(`${totalPct}%`));
    console.log(chalk.cyan("  ──────────────────────────────────────────────────────────\n"));

    if (missingTools.length === 0) {
      console.log(chalk.green("  [+] ") + chalk.gray("All tools are installed and ready.\n"));
      return;
    }

    const aptAvailable = this.canUseApt();

    if (!aptAvailable) {
      console.log(chalk.yellow("  ⬡ ") + chalk.gray("Auto-install is not available on this platform."));
      console.log(chalk.gray("    Install tools manually, or run from WSL / a Debian-based system."));
      console.log(chalk.gray("    The agent will also attempt to install missing tools at runtime.\n"));
      return;
    }

    const shouldInstall = await confirm({ message: "Install missing tools now?", default: true });

    if (!shouldInstall) {
      console.log(chalk.gray("\n  Skipped. The agent will install tools as needed during missions.\n"));
      return;
    }

    console.log("");
    const sudo = this.needsSudo() ? "sudo " : "";

    // Run apt-get update once
    const updateSpinner = ora({
      text: chalk.gray("Updating package index..."),
      color: "cyan",
      spinner: "dots",
    }).start();

    try {
      execSync(`${sudo}apt-get update -qq`, { stdio: "ignore", timeout: 60000 });
      updateSpinner.succeed(chalk.gray("Package index updated"));
    } catch {
      updateSpinner.fail(chalk.gray("Failed to update package index — attempting installs anyway"));
    }

    // Install each missing tool
    for (const tool of missingTools) {
      const spinner = ora({
        text: chalk.gray(`Installing ${tool.name}...`),
        color: "cyan",
        spinner: "dots",
      }).start();

      try {
        execSync(`${sudo}apt-get install -y -qq ${tool.pkg}`, { stdio: "ignore", timeout: 120000 });
        // Verify it actually installed
        if (this.checkTool(tool.name)) {
          spinner.succeed(chalk.green(`${tool.name}`) + chalk.gray(" installed"));
        } else {
          spinner.warn(chalk.yellow(`${tool.name}`) + chalk.gray(" — package installed but binary not found in PATH"));
        }
      } catch {
        spinner.fail(chalk.red(`${tool.name}`) + chalk.gray(` — failed. Run manually: ${sudo}apt-get install -y ${tool.pkg}`));
      }
    }

    console.log(chalk.green("\n  [+] ") + chalk.gray("Installation complete.\n"));
  }
}
