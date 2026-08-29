package memory

import (
	"os"
	"strings"
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

	// Round-trip: Findings() must decrypt the credential and count items.
	if err := db.InsertPort("10.0.0.7", 443, "https", ""); err != nil {
		t.Fatalf("InsertPort failed: %v", err)
	}
	if err := db.InsertVulnerability("10.0.0.7", "CVE-2026-0001", "RCE", "high"); err != nil {
		t.Fatalf("InsertVulnerability failed: %v", err)
	}

	rep, err := db.Findings(10)
	if err != nil {
		t.Fatalf("Findings failed: %v", err)
	}
	if rep.Ports != 1 || rep.Vulnerabilities != 1 || rep.Credentials != 1 {
		t.Errorf("Findings totals wrong: %+v", rep)
	}
	var sawPass, sawVuln bool
	for _, it := range rep.Items {
		switch it.Category {
		case "cred":
			if !strings.Contains(it.Detail, "admin") || !strings.Contains(it.Detail, "secret123") {
				t.Errorf("credential not decrypted into detail: %q", it.Detail)
			}
			sawPass = true
		case "vuln":
			if !strings.Contains(it.Detail, "CVE-2026-0001") {
				t.Errorf("vuln detail wrong: %q", it.Detail)
			}
			sawVuln = true
		}
	}
	if !sawPass || !sawVuln {
		t.Errorf("Findings items missing decrypted credential or vulnerability: %+v", rep.Items)
	}
}
