import Docker from "dockerode";
import fs from "fs";
import path from "path";

/**
 * Autonomous Execution Layer: Docker Sandbox
 * 
 * Executes arbitrary shell commands and pentesting tools inside an isolated Docker container.
 * This prevents the AI agent from accidentally modifying the host system during autonomous operation.
 */
export class DockerSandbox {
  private docker: Docker;
  private containerImage: string = "kalilinux/kali-rolling"; // Default to Kali for pentest tools

  constructor() {
    this.docker = new Docker({ socketPath: process.platform === "win32" ? "//./pipe/docker_engine" : "/var/run/docker.sock" });
  }

  /**
   * Ensure the pentesting image is pulled.
   */
  public async initialize(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.docker.pull(this.containerImage, (err: any, stream: any) => {
        if (err) return reject(err);
        this.docker.modem.followProgress(stream, onFinished);
        function onFinished(err: any, output: any) {
          if (err) return reject(err);
          resolve();
        }
      });
    });
  }

  /**
   * Execute a command inside the isolated container and return stdout/stderr.
   */
  public async executeCommand(command: string, timeoutMs: number = 60000): Promise<string> {
    const workDir = path.join(process.cwd(), "data", "sandbox");
    if (!fs.existsSync(workDir)) fs.mkdirSync(workDir, { recursive: true });

    try {
      const container = await this.docker.createContainer({
        Image: this.containerImage,
        Cmd: ["/bin/sh", "-c", command],
        Tty: false,
        HostConfig: {
          AutoRemove: true, // Clean up immediately after execution
          Binds: [`${workDir}:/workspace`], // Mount a shared workspace for file operations
          NetworkMode: "host", // Required for local pentesting/CTFs
          Memory: 512 * 1024 * 1024, // 512MB RAM limit
          CpuShares: 512,
        }
      });

      await container.start();

      return new Promise((resolve, reject) => {
        let output = "";
        
        // Timeout safety monitor
        const timer = setTimeout(async () => {
          try {
            await container.stop();
            resolve(`[TIMEOUT] Execution killed after ${timeoutMs}ms.\nPartial Output:\n${output}`);
          } catch (e) {
            reject(new Error("Failed to stop timed out container."));
          }
        }, timeoutMs);

        container.logs({
          follow: true,
          stdout: true,
          stderr: true
        }, (err, stream) => {
          if (err) return reject(err);
          if (stream) {
            stream.on("data", (chunk) => {
              output += chunk.toString("utf8");
            });
            stream.on("end", () => {
              clearTimeout(timer);
              resolve(output);
            });
          }
        });
      });
    } catch (e: any) {
      return `[Docker Sandbox Error] ${e.message}`;
    }
  }
}
