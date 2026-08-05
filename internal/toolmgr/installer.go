package toolmgr

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
)

func shellQuote(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

// InstallMethod represents the strategy used to install a tool.
type InstallMethod string

const (
	MethodApt    InstallMethod = "apt"
	MethodGo     InstallMethod = "go"
	MethodPip    InstallMethod = "pip"
	MethodGitHub InstallMethod = "github"
)

// knownTools maps tool names to their install commands per method.
var knownTools = map[string]struct {
	Method  InstallMethod
	Package string
}{
	"nuclei":       {MethodGo, "github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest"},
	"gobuster":     {MethodApt, "gobuster"},
	"ffuf":         {MethodGo, "github.com/ffuf/ffuf/v2@latest"},
	"amass":        {MethodGo, "github.com/owasp-amass/amass/v4/...@master"},
	"subfinder":    {MethodGo, "github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest"},
	"httpx":        {MethodGo, "github.com/projectdiscovery/httpx/cmd/httpx@latest"},
	"nmap":         {MethodApt, "nmap"},
	"masscan":      {MethodApt, "masscan"},
	"sqlmap":       {MethodApt, "sqlmap"},
	"searchsploit": {MethodApt, "exploitdb"},
	"msfconsole":   {MethodApt, "metasploit-framework"},
	"john":         {MethodApt, "john"},
	"hashcat":      {MethodApt, "hashcat"},
	"hydra":        {MethodApt, "hydra"},
	"nikto":        {MethodApt, "nikto"},
	"enum4linux":   {MethodApt, "enum4linux"},
	"smbclient":    {MethodApt, "smbclient"},
	"impacket-scripts": {MethodPip, "impacket"},
	"crackmapexec": {MethodPip, "crackmapexec"},
	"evil-winrm":   {MethodApt, "evil-winrm"},
}

// InstallTool attempts to install a tool using the best available strategy.
// Returns the command that was run and whether it succeeded.
func InstallTool(ctx context.Context, toolName string, sb *sandbox.Docker) (string, error) {
	toolName = strings.ToLower(strings.TrimSpace(toolName))

	if info, ok := knownTools[toolName]; ok {
		return install(ctx, sb, info.Method, info.Package, toolName)
	}

	// Unknown tool — try apt first, then pip
	out, err := install(ctx, sb, MethodApt, toolName, toolName)
	if err == nil && !strings.Contains(out, "Unable to locate package") {
		return out, nil
	}

	// Last resort: pip
	return install(ctx, sb, MethodPip, toolName, toolName)
}

func install(ctx context.Context, sb *sandbox.Docker, method InstallMethod, pkg, toolName string) (string, error) {
	var cmd string
	switch method {
	case MethodApt:
		cmd = fmt.Sprintf("apt-get update -qq && apt-get install -y %s 2>&1", shellQuote(pkg))
	case MethodGo:
		cmd = fmt.Sprintf("go install %s 2>&1 && echo '[OK] %s installed via go'", shellQuote(pkg), shellQuote(toolName))
	case MethodPip:
		cmd = fmt.Sprintf("pip3 install --quiet %s 2>&1 && echo '[OK] %s installed via pip'", shellQuote(pkg), shellQuote(toolName))
	default:
		return "", fmt.Errorf("unknown install method: %s", method)
	}

	out, err := sb.Execute(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("install command failed: %w\nOutput: %s", err, out)
	}

	checkOut, _ := sb.Execute(ctx, fmt.Sprintf("which %s 2>/dev/null || command -v %s 2>/dev/null", shellQuote(toolName), shellQuote(toolName)))
	if strings.TrimSpace(checkOut) == "" {
		return out, fmt.Errorf("installation ran but '%s' is still not found in PATH", toolName)
	}

	return fmt.Sprintf("[ToolMgr] Successfully installed '%s' via %s. Path: %s", toolName, method, strings.TrimSpace(checkOut)), nil
}

// IsInstalled checks whether a tool is available in the sandbox.
func IsInstalled(ctx context.Context, toolName string, sb *sandbox.Docker) bool {
	out, err := sb.Execute(ctx, fmt.Sprintf("which %s 2>/dev/null", shellQuote(toolName)))
	return err == nil && strings.TrimSpace(out) != "" && !strings.Contains(out, "not found")
}
