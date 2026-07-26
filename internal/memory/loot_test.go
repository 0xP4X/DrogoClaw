package memory

import (
	"os"
	"testing"
)

func TestLootDB_Encryption(t *testing.T) {
	// Create temp dir for the DB
	tempDir, err := os.MkdirTemp("", "lootdb_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Override data dir
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)
	
	os.MkdirAll("data", 0755)

	db, err := NewLootDB()
	if err != nil {
		t.Fatalf("failed to init lootdb: %v", err)
	}
	defer db.Close()

	// Test Insert Credential
	err = db.InsertCredential("test-target", "admin", "secret123", "hash123")
	if err != nil {
		t.Fatalf("InsertCredential failed: %v", err)
	}

	// Verify encryption directly from the DB
	rows, err := db.db.Query(`SELECT password_enc, hash_enc FROM credentials WHERE username = 'admin'`)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatalf("No rows found")
	}

	var passEnc, hashEnc string
	if err := rows.Scan(&passEnc, &hashEnc); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if passEnc == "secret123" {
		t.Errorf("Password was stored in plain text!")
	}
	if hashEnc == "hash123" {
		t.Errorf("Hash was stored in plain text!")
	}
}
