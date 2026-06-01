import Docker from "dockerode";
import fs from "fs";
import path from "path";
import { PassThrough } from "stream";
import { exec } from "child_process";
import { promisify } from "util";

const execAsync = promisify(exec);

/**
 * Autonomous Execution Layer: Docker Sandbox
 * 
 * Executes arbitrary shell commands and pentesting tools inside an isolated Docker container.
 * This prevents the AI agent from accidentally modifying the host system during autonomous operation.
 */
export class DockerSandbox {
  private docker: Docker;
  private containerImage: string = "kalilinux/kali-rolling";
  private containerName: string = "drogonclaw-sandbox";
  private activeContainer: Docker.Container | null = null;
  private currentCwd: string = "/workspace";

  constructor() {
    this.docker = new Docker({ socketPath: process.platform === "win32" ? "//./pipe/docker_engine" : "/var/run/docker.sock" });
    this.autoDetectEnvironment();
  }

  private autoDetectEnvironment() {
    if (process.env.USE_NATIVE_KALI !== undefined) return;
    
    try {
      if (fs.existsSync("/etc/os-release")) {
        const osRelease = fs.readFileSync("/etc/os-release", "utf8").toLowerCase();
        if (osRelease.includes("id=kali") || osRelease.includes("kali linux")) {
          process.env.USE_NATIVE_KALI = "true";
        }
      }
    } catch (e) {
      // Ignore read errors
    }
  }

  public async initialize(): Promise<void> {
    if (process.env.USE_NATIVE_KALI === "true") {
      console.log("\n  [!] NATIVE KALI MODE ACTIVE: Bypassing Docker Sandbox.");
      this.currentCwd = process.cwd(); // Use actual host CWD instead of /workspace
      return;
    }
    
    // Quick test to see if docker is actually available
    try {
      await this.docker.ping().catch((e) => { throw new Error('Docker ping failed'); });
    } catch (e) {
      console.log("\n  [!] DOCKER DAEMON UNAVAILABLE: Automatically falling back to Native Mode.");
      process.env.USE_NATIVE_KALI = "true";
      this.currentCwd = process.cwd();
      return;
    }

    return new Promise((resolve, reject) => {
      this.docker.pull(this.containerImage, async (err: any, stream: any) => {
        if (err) return reject(err);
        this.docker.modem.followProgress(stream, async (err: any) => {
          if (err) return reject(err);
          await this.setupPersistentContainer();
          resolve();
        });
      });
    });
  }

  private async setupPersistentContainer(): Promise<void> {
    const workDir = path.join(process.cwd(), "data", "sandbox");
    if (!fs.existsSync(workDir)) fs.mkdirSync(workDir, { recursive: true });

    try {
      const container = this.docker.getContainer(this.containerName);
      const info = await container.inspect();
      if (!info.State.Running) {
        await container.start();
      }
      this.activeContainer = container;
      return;
    } catch (e: any) {
      if (e.statusCode !== 404) throw e;
    }

    // Create long-running persistent container
    this.activeContainer = await this.docker.createContainer({
      Image: this.containerImage,
      name: this.containerName,
      Cmd: ["tail", "-f", "/dev/null"],
      Tty: false,
      HostConfig: {
        Binds: [`${workDir}:/workspace`],
        NetworkMode: "host",
        Memory: 1024 * 1024 * 1024, // 1GB
        CpuShares: 1024,
      }
    });

    await this.activeContainer.start();
  }

  public async executeCommand(command: string, timeoutMs: number = 60000): Promise<string> {
    if (!this.activeContainer) {
      await this.setupPersistentContainer();
    }

    // Handle 'cd' commands locally to maintain state across execs
    if (command.trim().startsWith("cd ")) {
      const newDir = command.trim().substring(3).trim();
      this.currentCwd = path.isAbsolute(newDir) ? newDir : path.join(this.currentCwd, newDir);
      return `[State] Changed directory to ${this.currentCwd}`;
    }

    if (process.env.USE_NATIVE_KALI === "true") {
      try {
        const { stdout, stderr } = await execAsync(command, { 
          cwd: this.currentCwd, 
          timeout: timeoutMs 
        });
        return stdout || stderr || "[Native Execution Success: No Output]";
      } catch (e: any) {
        return `[Native Execution Error] ${e.message}\n${e.stderr || ""}`;
      }
    }

    try {
      const exec = await this.activeContainer!.exec({
        Cmd: ["/bin/sh", "-c", command],
        AttachStdout: true,
        AttachStderr: true,
        WorkingDir: this.currentCwd
      });

      // @ts-ignore - The types for dockerode exec start are sometimes incomplete regarding streams
      const stream: NodeJS.ReadWriteStream = await exec.start({ Detach: false, Tty: false });

      return new Promise((resolve, reject) => {
        let output = "";
        
        const timer = setTimeout(() => {
          resolve(`[TIMEOUT] Execution killed after ${timeoutMs}ms.\nPartial Output:\n${output}`);
        }, timeoutMs);

        const stdoutStream = new PassThrough();
        const stderrStream = new PassThrough();
        
        stdoutStream.on("data", (chunk: Buffer) => { output += chunk.toString("utf8"); });
        stderrStream.on("data", (chunk: Buffer) => { output += chunk.toString("utf8"); });

        // Docker modem mux demultiplexing
        this.docker.modem.demuxStream(stream, stdoutStream, stderrStream);

        stream.on("end", () => {
          clearTimeout(timer);
          resolve(output);
        });
      });
    } catch (e: any) {
      return `[Docker Sandbox Error] ${e.message}`;
    }
  }

  /**
   * Securely writes a file directly into the sandbox /workspace directory.
   * This circumvents brittle Docker volume-mounting bugs on Windows/WSL by
   * sending the payload as Base64 encoded chunks via 'executeCommand'.
   *
   * @param filename The relative path/filename inside /workspace
   * @param content The raw string content of the file
   */
  public async writeWorkspaceFile(filename: string, content: string): Promise<string> {
    try {
      const b64 = Buffer.from(content).toString("base64");
      const targetPath = path.posix.join("/workspace", filename.replace(/\\\\/g, "/"));
      
      // Ensure the directory exists
      const dirName = path.posix.dirname(targetPath);
      await this.executeCommand(`mkdir -p "${dirName}"`, 10000);

      // We split the b64 string into chunks if it's extremely large to avoid ARG_MAX limits,
      // but typically it's fine for small/medium scripts.
      const cmd = `echo "${b64}" | base64 -d > "${targetPath}"`;
      
      const out = await this.executeCommand(cmd, 30000);
      if (out && out.toLowerCase().includes("error")) {
        return `[WriteWorkspace Error] Failed to write file: ${out}`;
      }
      return targetPath;
    } catch (e: any) {
      return `[WriteWorkspace Error] ${e.message}`;
    }
  }
}
