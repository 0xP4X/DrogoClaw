import Database from "better-sqlite3";
import fs from "fs";
import path from "path";

// Define schema version for future migrations
const SCHEMA_VERSION = 1;

export class LootDB {
  private static instance: LootDB;
  private db: Database.Database;

  private constructor() {
    const dbPath = path.join(process.cwd(), "drogonclaw_loot.db");
    
    // Ensure the database file isn't created in an invalid state
    this.db = new Database(dbPath);
    this.initSchema();
  }

  public static getInstance(): LootDB {
    if (!LootDB.instance) {
      LootDB.instance = new LootDB();
    }
    return LootDB.instance;
  }

  private initSchema() {
    // Initialize tables if they do not exist
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS meta (
        key TEXT PRIMARY KEY,
        value TEXT
      );

      CREATE TABLE IF NOT EXISTS ports (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        ip TEXT NOT NULL,
        port INTEGER NOT NULL,
        service TEXT,
        version TEXT,
        discovered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(ip, port)
      );

      CREATE TABLE IF NOT EXISTS credentials (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        target TEXT NOT NULL,
        username TEXT,
        password TEXT,
        hash TEXT,
        discovered_at DATETIME DEFAULT CURRENT_TIMESTAMP
      );

      CREATE TABLE IF NOT EXISTS vulnerabilities (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        target TEXT NOT NULL,
        cve TEXT,
        description TEXT,
        severity TEXT,
        discovered_at DATETIME DEFAULT CURRENT_TIMESTAMP
      );
    `);
    
    this.db.prepare("INSERT OR IGNORE INTO meta (key, value) VALUES ('version', ?)").run(SCHEMA_VERSION.toString());
  }

  // PORT METHODS
  public insertPort(ip: string, port: number, service: string = "", version: string = ""): void {
    const stmt = this.db.prepare(`
      INSERT OR IGNORE INTO ports (ip, port, service, version)
      VALUES (?, ?, ?, ?)
    `);
    stmt.run(ip, port, service, version);
  }

  public getPorts(ip?: string): any[] {
    if (ip) {
      return this.db.prepare("SELECT * FROM ports WHERE ip = ? ORDER BY port ASC").all(ip);
    }
    return this.db.prepare("SELECT * FROM ports ORDER BY ip ASC, port ASC").all();
  }

  // CREDENTIAL METHODS
  public insertCredential(target: string, username: string, password: string = "", hash: string = ""): void {
    const stmt = this.db.prepare(`
      INSERT INTO credentials (target, username, password, hash)
      VALUES (?, ?, ?, ?)
    `);
    stmt.run(target, username, password, hash);
  }

  public getCredentials(): any[] {
    return this.db.prepare("SELECT * FROM credentials").all();
  }

  // VULNERABILITY METHODS
  public insertVulnerability(target: string, description: string, severity: string = "UNKNOWN", cve: string = ""): void {
    const stmt = this.db.prepare(`
      INSERT INTO vulnerabilities (target, cve, description, severity)
      VALUES (?, ?, ?, ?)
    `);
    stmt.run(target, cve, description, severity);
  }

  public getVulnerabilities(): any[] {
    return this.db.prepare("SELECT * FROM vulnerabilities ORDER BY severity DESC").all();
  }
}
