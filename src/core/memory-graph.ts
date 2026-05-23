import fs from "fs";
import path from "path";

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
export class MemoryGraph {
  private nodes: Map<string, GraphNode> = new Map();
  private edges: GraphEdge[] = [];
  private dbPath: string;

  constructor(sessionId: string = "default") {
    this.dbPath = path.join(process.cwd(), "data", `graph_${sessionId}.json`);
    this.load();
  }

  public addNode(node: GraphNode): void {
    this.nodes.set(node.id, node);
    this.save();
  }

  public addEdge(edge: GraphEdge): void {
    // Avoid duplicates
    const exists = this.edges.find(e => e.sourceId === edge.sourceId && e.targetId === edge.targetId && e.relationship === edge.relationship);
    if (!exists) {
      this.edges.push(edge);
      this.save();
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

  public queryVulnerabilities(): GraphNode[] {
    return Array.from(this.nodes.values()).filter(n => n.label === "Vulnerability");
  }

  public getFullGraphJSON(): string {
    return JSON.stringify({
      nodes: Array.from(this.nodes.values()),
      edges: this.edges
    }, null, 2);
  }

  private save(): void {
    const dir = path.dirname(this.dbPath);
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
    
    fs.writeFileSync(this.dbPath, this.getFullGraphJSON(), "utf-8");
  }

  private load(): void {
    if (fs.existsSync(this.dbPath)) {
      try {
        const data = JSON.parse(fs.readFileSync(this.dbPath, "utf-8"));
        data.nodes.forEach((n: GraphNode) => this.nodes.set(n.id, n));
        this.edges = data.edges;
      } catch (e) {
        console.error("Failed to load intelligence graph from disk.", e);
      }
    }
  }
}
