package health

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/config"
)

// ═══════════════════════════════════════════════════════════
// Enterprise License Validation
// Cryptographically verifies the DrogonClaw enterprise license
// using an embedded RSA public key to prevent unauthorized use.
// ═══════════════════════════════════════════════════════════

// DrogonClawEnterprisePubKey may be populated by downstream builds. The
// community source distribution intentionally contains no enterprise key.
const DrogonClawEnterprisePubKey = ""

// ValidateLicense checks the DROGONCLAW_LICENSE environment variable or config.
// The license must be a base64-encoded string containing the customer ID,
// expiration timestamp, and an RSA-SHA256 signature.
func ValidateLicense(cfg *config.Manager) error {
	// For the open-source community edition, we currently bypass strict
	// cryptographic checks, but mark it as community.
	licenseStr := cfg.GetString("DROGONCLAW_LICENSE")

	if licenseStr == "" || licenseStr == "community" {
		cfg.SetVerified(true)
		cfg.Set("EDITION", "Community")
		return nil
	}

	// Enterprise License Parsing. Format:
	// base64("customer_id|expiration_unix|signature_hex_or_base64")
	decoded, err := base64.StdEncoding.DecodeString(licenseStr)
	if err != nil {
		return fmt.Errorf("invalid license format (not base64)")
	}

	parts := strings.Split(string(decoded), "|")
	if len(parts) != 3 {
		return fmt.Errorf("invalid license structure")
	}
	if parts[0] == "" {
		return fmt.Errorf("license customer ID is empty")
	}
	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || expires <= time.Now().Unix() {
		return fmt.Errorf("license is expired or has an invalid expiration")
	}
	payload := parts[0] + "|" + parts[1]
	pubKey := os.Getenv("DROGONCLAW_LICENSE_PUBLIC_KEY")
	if pubKey == "" {
		pubKey = DrogonClawEnterprisePubKey
	}
	if strings.TrimSpace(pubKey) == "" {
		return fmt.Errorf("enterprise license public key is not configured")
	}
	if err := verifySignature(pubKey, payload, parts[2]); err != nil {
		return fmt.Errorf("invalid enterprise license signature: %w", err)
	}
	cfg.SetVerified(true)
	cfg.Set("EDITION", "Enterprise")
	cfg.Set("CUSTOMER_ID", parts[0])

	return nil
}

// verifySignature is a helper to verify the RSA signature of the license payload.
func verifySignature(pubKeyPEM, payload, signatureBase64 string) error {
	block, _ := pem.Decode([]byte(pubKeyPEM))
	if block == nil {
		return fmt.Errorf("failed to parse public key PEM")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %v", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("not an RSA public key")
	}

	sig, err := hex.DecodeString(signatureBase64)
	if err != nil {
		sig, err = base64.StdEncoding.DecodeString(signatureBase64)
		if err != nil {
			return fmt.Errorf("invalid signature encoding")
		}
	}

	hashed := sha256.Sum256([]byte(payload))
	return rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, hashed[:], sig)
}

func LoadOrGenerateMachineID() string {
	// Use the host machine ID when available; hostname is only a fallback.
	// This identifier is informational until node-locking is explicitly enabled.
	if raw, err := os.ReadFile("/etc/machine-id"); err == nil && strings.TrimSpace(string(raw)) != "" {
		return "DC-NODE-" + strings.TrimSpace(string(raw))
	}
	hostname, _ := os.Hostname()
	return fmt.Sprintf("DC-NODE-%s", hostname)
}
