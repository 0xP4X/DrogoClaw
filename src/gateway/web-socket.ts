import { WebSocketServer, WebSocket } from "ws";
import chalk from "chalk";
import { AgentOrchestrator } from "../agent/orchestrator";
import fs from "fs";
import path from "path";
import dotenv from "dotenv";

export class WebGUIServer {
  private wss: WebSocketServer;
  private orchestrator: AgentOrchestrator;

  constructor(orchestrator: AgentOrchestrator, port: number = 8080) {
    this.orchestrator = orchestrator;
    try {
      this.wss = new WebSocketServer({ port });
      console.log(chalk.cyan(`\n🌐 Web GUI API running on ws://localhost:${port}`));
    } catch (e: any) {
      if (e.code === 'EADDRINUSE') {
        console.log(chalk.red(`❌ Port ${port} is already in use.`));
        console.log(chalk.yellow(`💡 Try running 'npx kill-port ${port}' or closing other DrogonClaw instances.`));
      }
      throw e;
    }
  }

  public start(): void {
    this.wss.on("connection", (ws: WebSocket) => {
      console.log(chalk.green("[Web GUI] Client connected to WebSocket."));
      
      const setupGraphListeners = () => {
        const graph = this.orchestrator.getMemoryGraph();
        if (graph) {
          graph.on("nodeAdded", (node) => {
            ws.send(JSON.stringify({ type: "GRAPH_NODE_ADDED", data: node }));
          });
          graph.on("edgeAdded", (edge) => {
            ws.send(JSON.stringify({ type: "GRAPH_EDGE_ADDED", data: edge }));
          });
        }
      };

      if (!this.orchestrator.isReady()) {
        ws.send(JSON.stringify({ type: "NEEDS_SETUP" }));
      } else {
        const graph = this.orchestrator.getMemoryGraph();
        ws.send(JSON.stringify({
          type: "MEMORY_GRAPH_DUMP",
          data: graph ? JSON.parse(graph.getFullGraphJSON()) : { nodes: [], edges: [] }
        }));
        setupGraphListeners();
      }

      ws.on("message", async (message: string) => {
        try {
          const parsed = JSON.parse(message);
          
          if (parsed.type === "SAVE_CONFIG") {
            const config = parsed.data;
            let envContent = `AI_PROVIDER=${config.provider}\n`;
            
            if (config.provider === "openai") {
              envContent += `OPENAI_API_KEY=${config.apiKey}\nOPENAI_MODEL_NAME=${config.model}\n`;
            } else if (config.provider === "anthropic") {
              envContent += `ANTHROPIC_API_KEY=${config.apiKey}\nANTHROPIC_MODEL_NAME=${config.model}\n`;
            } else if (config.provider === "gemini") {
              envContent += `GOOGLE_API_KEY=${config.apiKey}\nGEMINI_MODEL_NAME=${config.model}\n`;
            } else if (config.provider === "openrouter") {
              envContent += `OPENROUTER_API_KEY=${config.apiKey}\nOPENROUTER_MODEL_NAME=${config.model}\n`;
            } else if (config.provider === "ollama") {
              envContent += `OLLAMA_BASE_URL=${config.baseUrl}\nOLLAMA_MODEL_NAME=${config.model}\n`;
            }

            fs.writeFileSync(path.join(process.cwd(), ".env"), envContent, "utf-8");
            dotenv.config({ override: true });

            this.orchestrator = new AgentOrchestrator();
            const success = await this.orchestrator.initialize();

            if (success && this.orchestrator.isReady()) {
              ws.send(JSON.stringify({ type: "SETUP_COMPLETE" }));
              const graph = this.orchestrator.getMemoryGraph();
              ws.send(JSON.stringify({
                type: "MEMORY_GRAPH_DUMP",
                data: graph ? JSON.parse(graph.getFullGraphJSON()) : { nodes: [], edges: [] }
              }));
              setupGraphListeners();
            } else {
              // Extract specific error if available
              const lastErr = this.orchestrator.getLastError();
              ws.send(JSON.stringify({ 
                type: "ERROR", 
                data: lastErr || "Initialization failed. Check API key validity and network connectivity." 
              }));
            }
          }
          
          if (parsed.type === "EXECUTE_MISSION" && this.orchestrator.isReady()) {
            const mission = parsed.data;
            ws.send(JSON.stringify({ type: "MISSION_STATUS", data: "Mission started..." }));
            
            // Execute the mission and stream tool calls
            const response = await this.orchestrator.execute(mission, (toolName, args) => {
              const argStr = args ? ` - ${JSON.stringify(args)}` : "";
              ws.send(JSON.stringify({ type: "TOOL_UPDATE", data: `${toolName}${argStr}` }));
            });
            
            // Send final response and updated graph
            ws.send(JSON.stringify({ type: "MISSION_COMPLETE", data: response }));
            
            const updatedGraph = this.orchestrator.getMemoryGraph();
            ws.send(JSON.stringify({
              type: "MEMORY_GRAPH_DUMP",
              data: updatedGraph ? JSON.parse(updatedGraph.getFullGraphJSON()) : {}
            }));
          }
        } catch (e: any) {
          ws.send(JSON.stringify({ type: "ERROR", data: e.message }));
        }
      });

      ws.on("close", () => {
        console.log(chalk.gray("[Web GUI] Client disconnected."));
      });
    });
  }
}

// Start server if run directly
const orchestrator = new AgentOrchestrator();

// Start the server IMMEDIATELY so the UI can connect 
// even while the core is trying to initialize or awaiting setup.
const server = new WebGUIServer(orchestrator);
server.start();

// Try initial background initialization
orchestrator.initialize().then(() => {
  if (!orchestrator.isReady()) {
    console.log(chalk.yellow("\n⚠️ Agent Core is offline. Awaiting Web GUI configuration..."));
  } else {
    console.log(chalk.green("\n✅ Agent Core initialized from existing environment."));
  }
});
