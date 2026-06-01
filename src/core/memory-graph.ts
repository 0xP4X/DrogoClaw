import fs from "fs";
import path from "path";
import { EventEmitter } from "events";
import neo4j, { Driver } from "neo4j-driver";
import { ConfigManager } from "./config-manager.js";
import "dotenv/config";

export interface GraphNode {
  id: string;
  label: "Target" | "Asset" | "Port" | "Service" | "Vulnerability" | "Credential" | "Flag";
  properties: Record<string, any>;
}

export interface GraphEdge {
  sourceId: string;
  targetId: string;
  relationship: "HAS_ASSET" | "HAS_PORT" | "RUNS_SERVICE" | "HAS_VULNERABILITY" | "EXPLOITED_BY" | "CONTAINS_FLAG";
  properties?: Record<string, any>;
}

/**
 * DrogonClaw Intelligence Graph (Local JSON implementation)
 * Maps the relationships between discovered assets, vulnerabilities, and evidence.
 */
export interface OperatorProfile {
  name: string;
  skillLevel?: string;
  preferences?: string;
}

export class MemoryGraph extends EventEmitter {
  private nodes: Map<string, GraphNode> = new Map();
  private edges: GraphEdge[] = [];
  private dbPath: string;
  private neo4jDriver: Driver | null = null;
  private operatorProfile: OperatorProfile | null = null;

  constructor(sessionId: string = "default") {
    super();
    this.dbPath = path.join(process.cwd(), "data", `graph_${sessionId}.json`);
    this.load();
    
    this.connect();
  }

  public async connect(): Promise<boolean> {
    const uri = ConfigManager.get("NEO4J_URI");
    const user = ConfigManager.get("NEO4J_USER");
    const password = ConfigManager.get("NEO4J_PASSWORD");
    
    if (uri && user && password) {
      try {
        this.neo4jDriver = neo4j.driver(uri, neo4j.auth.basic(user, password));
        return true;
      } catch (e) {
        console.warn("[MemoryGraph] Neo4j connection failed. Falling back to local JSON.");
        return false;
      }
    }
    return false;
  }

  public async addNode(node: GraphNode): Promise<void> {
    this.nodes.set(node.id, node);
    this.save();
    this.emit("nodeAdded", node);
    
    if (this.neo4jDriver) {
      const session = this.neo4jDriver.session();
      try {
        await session.run(
          `MERGE (n:${node.label} {id: $id}) SET n += $props`,
          { id: node.id, props: node.properties }
        );
      } catch (e) {
        // Silent catch for background DB pushes
      } finally {
        await session.close();
      }
    }
  }

  public async addEdge(edge: GraphEdge): Promise<void> {
    // Avoid duplicates
    const exists = this.edges.find(e => e.sourceId === edge.sourceId && e.targetId === edge.targetId && e.relationship === edge.relationship);
    if (!exists) {
      this.edges.push(edge);
      this.save();
      this.emit("edgeAdded", edge);
    }
    
    if (this.neo4jDriver) {
      const session = this.neo4jDriver.session();
      try {
        await session.run(
          `MATCH (a {id: $sourceId}), (b {id: $targetId})
           MERGE (a)-[r:${edge.relationship}]->(b)
           SET r += $props`,
          { sourceId: edge.sourceId, targetId: edge.targetId, props: edge.properties || {} }
        );
      } catch (e) {
        // Silent catch for background DB pushes
      } finally {
        await session.close();
      }
    }
  }

  public getContextForAsset(assetId: string): any {
    const asset = this.nodes.get(assetId);
    if (!asset) return null;

    const relatedEdges = this.edges.filter(e => e.sourceId === assetId || e.targetId === assetId);
    const relatedNodes = relatedEdges.map(e => {
      const otherId = e.sourceId === assetId ? e.targetId : e.sourceId;
      return {
        relationship: e.relationship,
        node: this.nodes.get(otherId)
      };
    });

    return {
      asset,
      relationships: relatedNodes
    };
  }

  /**
   * Resolves token explosion by returning only a 1st-degree relationship subgraph.
   */
  public getRelevantContext(targetAssetId?: string): string {
    if (!targetAssetId || this.nodes.size < 50) {
      return this.getFullGraphJSON();
    }
    
    const subgraphNodes = new Map<string, GraphNode>();
    const subgraphEdges = this.edges.filter(e => e.sourceId === targetAssetId || e.targetId === targetAssetId);
    
    subgraphEdges.forEach(e => {
      const otherId = e.sourceId === targetAssetId ? e.targetId : e.sourceId;
      const node = this.nodes.get(otherId);
      if (node) subgraphNodes.set(node.id, node);
    });
    
    const targetNode = this.nodes.get(targetAssetId);
    if (targetNode) subgraphNodes.set(targetNode.id, targetNode);
    
    return JSON.stringify({
      nodes: Array.from(subgraphNodes.values()),
      edges: subgraphEdges
    }, null, 2);
  }

  public queryVulnerabilities(): GraphNode[] {
    return Array.from(this.nodes.values()).filter(n => n.label === "Vulnerability");
  }

  public async getRecentContext(limit: number = 5): Promise<string> {
    const nodes = Array.from(this.nodes.values())
      .sort((a: any, b: any) => (b.timestamp || 0) - (a.timestamp || 0))
      .slice(0, limit);

    if (nodes.length === 0) return "Memory is empty.";

    return nodes
      .map((n) => `[${n.label.toUpperCase()}] ${JSON.stringify(n.properties)}`)
      .join("\n");
  }

  public getOperatorProfile(): OperatorProfile | null {
    return this.operatorProfile;
  }

  public async updateOperatorProfile(profile: Partial<OperatorProfile>): Promise<void> {
    if (!this.operatorProfile) {
      this.operatorProfile = { name: profile.name || "Unknown" };
    }
    this.operatorProfile = { ...this.operatorProfile, ...profile };
    this.save();
    console.log(`\n  [Memory] Neural pathway updated: Operator Identity -> ${this.operatorProfile.name}`);
  }

  public getNodesCount(): number {
    return this.nodes.size;
  }

  public getFullGraphJSON(): string {
    return JSON.stringify({
      nodes: Array.from(this.nodes.values()),
      edges: this.edges,
      operatorProfile: this.operatorProfile
    }, null, 2);
  }

  private save(): void {
    const data = {
      nodes: Array.from(this.nodes.values()),
      edges: this.edges,
      operatorProfile: this.operatorProfile
    };
    const dir = path.dirname(this.dbPath);
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
    
    fs.writeFileSync(this.dbPath, JSON.stringify(data, null, 2));
  }

  private load(): void {
    if (fs.existsSync(this.dbPath)) {
      try {
        const data = JSON.parse(fs.readFileSync(this.dbPath, "utf-8"));
        if (data.nodes) {
          for (const node of data.nodes) {
            this.nodes.set(node.id, node);
          }
        }
        if (data.edges) {
          this.edges = data.edges;
        }
        if (data.operatorProfile) {
          this.operatorProfile = data.operatorProfile;
        }
      } catch (e) {
        console.warn("[MemoryGraph] Failed to load local JSON graph.");
      }
    }
  }
}
