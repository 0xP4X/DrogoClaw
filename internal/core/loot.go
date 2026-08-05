package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const SchemaVersion = 1

var globalLootKey []byte

func initLootKey(dataDir string) error {
	keyPath := filepath.Join(dataDir, ".loot_key")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return err
		}
		if err := os.WriteFile(keyPath, key, 0600); err != nil {
			return err
		}
		globalLootKey = key
		return nil
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	globalLootKey = key
	return nil
}

func encryptLoot(plaintext string) string {
	if plaintext == "" {
		return ""
	}
	if globalLootKey == nil {
		return plaintext
	}
	block, err := aes.NewCipher(globalLootKey)
	if err != nil {
		return plaintext
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return plaintext
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return plaintext
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

// LootDB manages discovered assets, credentials, and vulnerabilities.
type LootDB struct {
	db *sql.DB
}

var globalLootDB *LootDB

// GetLootDB returns the singleton LootDB instance.
func GetLootDB() *LootDB {
	if globalLootDB == nil {
		dataDir := filepath.Join("data")
		_ = os.MkdirAll(dataDir, 0755)

		if err := initLootKey(dataDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to init loot key: %v\n", err)
			return nil
		}

		dbPath := filepath.Join(dataDir, "drogonclaw_loot.db")
		db, err := sql.Open("sqlite3", dbPath)
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
        password_enc TEXT,
        hash_enc TEXT,
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
	_, _ = l.db.Exec(`INSERT INTO credentials (target, username, password_enc, hash_enc, discovered_at) VALUES (?, ?, ?, ?, ?)`, target, username, encryptLoot(password), encryptLoot(hash), time.Now())
}

func (l *LootDB) InsertVulnerability(target, description, severity, cve string) {
	if severity == "" {
		severity = "UNKNOWN"
	}
	_, _ = l.db.Exec(`INSERT INTO vulnerabilities (target, cve, description, severity) VALUES (?, ?, ?, ?)`, target, cve, description, severity)
}
