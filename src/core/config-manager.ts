import fs from "fs";
import path from "path";

export interface DrogonConfig {
  AI_PROVIDER?: string;
  OPENAI_API_KEY?: string;
  OPENAI_MODEL_NAME?: string;
  ANTHROPIC_API_KEY?: string;
  ANTHROPIC_MODEL_NAME?: string;
  GOOGLE_API_KEY?: string;
  GEMINI_MODEL_NAME?: string;
  OPENROUTER_API_KEY?: string;
  OPENROUTER_MODEL_NAME?: string;
  OLLAMA_BASE_URL?: string;
  OLLAMA_MODEL_NAME?: string;
  TELEGRAM_TOKEN?: string;
  TELEGRAM_CHAT_ID?: string;
  NEO4J_URI?: string;
  NEO4J_USER?: string;
  NEO4J_PASSWORD?: string;
}

class ConfigManagerSingleton {
  private config: DrogonConfig = {};
  private currentProfilePath: string;

  constructor() {
    const profilesDir = path.join(process.cwd(), ".drogonclaw", "profiles");
    if (!fs.existsSync(profilesDir)) {
      fs.mkdirSync(profilesDir, { recursive: true });
    }
    this.currentProfilePath = path.join(profilesDir, "default.json");
    this.load();
  }

  public load(profileName: string = "default"): void {
    this.currentProfilePath = path.join(process.cwd(), ".drogonclaw", "profiles", `${profileName}.json`);
    if (fs.existsSync(this.currentProfilePath)) {
      try {
        const data = fs.readFileSync(this.currentProfilePath, "utf-8");
        this.config = JSON.parse(data);
        
        // Push config into process.env for legacy compatibility
        for (const key in this.config) {
          process.env[key] = (this.config as any)[key];
        }
      } catch (e) {
        console.error(`Failed to load profile ${profileName}:`, e);
      }
    } else {
      this.config = {};
    }
  }

  public save(newConfig: DrogonConfig): void {
    this.config = { ...this.config, ...newConfig };
    fs.writeFileSync(this.currentProfilePath, JSON.stringify(this.config, null, 2), "utf-8");
    this.load(path.basename(this.currentProfilePath, ".json"));
  }

  public get(key: keyof DrogonConfig): string | undefined {
    return this.config[key] || process.env[key];
  }
}

export const ConfigManager = new ConfigManagerSingleton();
