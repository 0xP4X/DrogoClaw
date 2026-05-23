import { z } from "zod";

export interface SkillExecutionContext {
  target: string;
  args?: Record<string, any>;
}

export interface SkillPlugin {
  id: string;
  name: string;
  category: "recon" | "osint" | "browser" | "exploitation" | "post_exploitation" | "core";
  description: string;
  schema: z.ZodObject<any>;
  execute: (context: SkillExecutionContext) => Promise<string>;
}

/**
 * Skill Ecosystem: Plugin Registry
 * 
 * Manages modular offensive security plugins. Allows dynamic loading of tools
 * without modifying the orchestration core.
 */
export class PluginRegistry {
  private plugins: Map<string, SkillPlugin> = new Map();

  /**
   * Register a new offensive security skill plugin.
   */
  public register(plugin: SkillPlugin): void {
    if (this.plugins.has(plugin.id)) {
      console.warn(`[PluginRegistry] Plugin ${plugin.id} is already registered. Overwriting.`);
    }
    this.plugins.set(plugin.id, plugin);
    console.log(`[PluginRegistry] Loaded plugin: ${plugin.name} [${plugin.category}]`);
  }

  /**
   * Retrieve a specific plugin by ID.
   */
  public getPlugin(id: string): SkillPlugin | undefined {
    return this.plugins.get(id);
  }

  /**
   * Get all registered plugins, optionally filtered by category.
   */
  public getAllPlugins(category?: SkillPlugin["category"]): SkillPlugin[] {
    const all = Array.from(this.plugins.values());
    if (category) {
      return all.filter(p => p.category === category);
    }
    return all;
  }

  /**
   * Generates a LangChain compatible tool array from registered plugins.
   * (This acts as the bridge between the Plugin Ecosystem and the LangGraph orchestrator)
   */
  public getLangChainTools(): any[] {
    // In a full implementation, this dynamically wraps SkillPlugins into DynamicStructuredTools
    return [];
  }
}

// Global registry singleton
export const globalPluginRegistry = new PluginRegistry();
