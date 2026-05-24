import Docker from "dockerode";
import fs from "fs";
import path from "path";
import { PassThrough } from "stream";

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
  }

  public async initialize(): Promise<void> {
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
}
