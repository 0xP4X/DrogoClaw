package core

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const SchemaVersion = 1

// LootDB manages discovered assets, credentials, and vulnerabilities.
type LootDB struct {
	db *sql.DB
}

var globalLootDB *LootDB

// GetLootDB returns the singleton LootDB instance.
func GetLootDB() *LootDB {
	if globalLootDB == nil {
		dbPath := filepath.Join("drogonclaw_loot.db")
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open Loot DB: %v\n", err)
			return nil
		}

		ldb := &LootDB{db: db}
		ldb.initSchema()
		globalLootDB = ldb
	}
	return globalLootDB
}

func (l *LootDB) initSchema() {
	schema := `
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
    `
	_, _ = l.db.Exec(schema)
	_, _ = l.db.Exec("INSERT OR IGNORE INTO meta (key, value) VALUES ('version', ?)", fmt.Sprintf("%d", SchemaVersion))
}

func (l *LootDB) InsertPort(ip string, port int, service, version string) {
	_, _ = l.db.Exec(`INSERT OR IGNORE INTO ports (ip, port, service, version) VALUES (?, ?, ?, ?)`, ip, port, service, version)
}

func (l *LootDB) InsertCredential(target, username, password, hash string) {
	_, _ = l.db.Exec(`INSERT INTO credentials (target, username, password, hash) VALUES (?, ?, ?, ?)`, target, username, password, hash)
}

func (l *LootDB) InsertVulnerability(target, description, severity, cve string) {
	if severity == "" {
		severity = "UNKNOWN"
	}
	_, _ = l.db.Exec(`INSERT INTO vulnerabilities (target, cve, description, severity) VALUES (?, ?, ?, ?)`, target, cve, description, severity)
}
