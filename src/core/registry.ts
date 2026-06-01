import { MemoryGraph } from "./memory-graph.js";

export class CoreRegistry {
  private static graph: MemoryGraph | null = null;

  public static setGraph(graph: MemoryGraph): void {
    this.graph = graph;
  }

  public static getGraph(): MemoryGraph {
    if (!this.graph) {
      this.graph = new MemoryGraph();
    }
    return this.graph;
  }
}
