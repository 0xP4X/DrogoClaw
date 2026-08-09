package sandbox

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/0xP4X/drogonclaw-go/internal/core"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	containerImage = "kalilinux/kali-rolling"
	containerName  = "drogonclaw-sandbox"
	defaultWorkDir = "/workspace"
)

// commandTimeout caps how long a single command may run. Most pentest commands
// complete in seconds; reconnaissance sweeps may need a couple of minutes. The
// parent context (mission timeout) still applies as an upper bound.
const commandTimeout = 5 * time.Minute

// Docker wraps the Docker SDK for sandboxed command execution.
type Docker struct {
	cli         *client.Client
	containerID string
	currentCwd  string
	nativeMode  bool
}

// New creates a Docker sandbox client.
func New() (*Docker, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return &Docker{currentCwd: defaultWorkDir, nativeMode: true}, nil
	}
	return &Docker{cli: cli, currentCwd: defaultWorkDir}, nil
}

// IsNativeMode returns whether the sandbox is running in native mode (host OS) vs Docker.
func (d *Docker) IsNativeMode() bool {
	return d.nativeMode
}

// RuntimeLabel returns a human-readable label for the current execution environment.
func (d *Docker) RuntimeLabel() string {
	if d == nil {
		return "unavailable"
	}
	if d.nativeMode {
		return "native host"
	}
	return "Docker sandbox"
}

// IsReady reports whether Execute can run commands in the selected runtime.
func (d *Docker) IsReady() bool {
	if d == nil {
		return false
	}
	return d.nativeMode || d.containerID != ""
}

// Initialize ensures the Kali container is running.
func (d *Docker) Initialize(ctx context.Context, native bool) error {
	d.nativeMode = native
	if native {
		d.currentCwd, _ = os.Getwd()
		fmt.Println("  [!] NATIVE MODE: commands run directly on host OS")
		return nil
	}

	// Ensure Docker daemon is alive
	if d.cli == nil {
		return fmt.Errorf("docker client unavailable")
	}
	if _, err := d.cli.Ping(ctx); err != nil {
		return fmt.Errorf("docker daemon not reachable: %w", err)
	}

	// Ensure workspace dir exists for volume mount
	workDir := filepath.Join("data", "sandbox")
	_ = os.MkdirAll(workDir, 0755)

	// Try to find existing container
	inspect, err := d.cli.ContainerInspect(ctx, containerName)
	if err == nil {
		if !inspect.State.Running {
			if err := d.cli.ContainerStart(ctx, inspect.ID, container.StartOptions{}); err != nil {
				return fmt.Errorf("failed to start existing container: %w", err)
			}
		}
		d.containerID = inspect.ID
		core.RegisterDocker(d.containerID)
		return nil
	}

	// Pull image if needed
	fmt.Printf("  [*] Pulling %s (first run only)...\n", containerImage)
	reader, err := d.cli.ImagePull(ctx, containerImage, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("image pull failed: %w", err)
	}
	defer reader.Close()
	io.Copy(io.Discard, reader)

	// Create persistent container
	absWorkDir, _ := filepath.Abs(workDir)
	resp, err := d.cli.ContainerCreate(ctx, &container.Config{
		Image:      containerImage,
		Cmd:        []string{"tail", "-f", "/dev/null"},
		WorkingDir: defaultWorkDir,
		Env:        []string{"DEBIAN_FRONTEND=noninteractive", "TERM=xterm-256color"},
	}, &container.HostConfig{
		Binds:      []string{absWorkDir + ":/workspace:rw"},
		AutoRemove: false,
		// NET_RAW  — required for raw socket tools: scapy, ping -p, tcpdump, nmap -sS
		// NET_ADMIN — required for interface manipulation, iptables, iproute2 inside container
		CapAdd: []string{"NET_RAW", "NET_ADMIN"},
	}, nil, nil, containerName)
	if err != nil {
		return fmt.Errorf("container create failed: %w", err)
	}

	if err := d.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("container start failed: %w", err)
	}
	d.containerID = resp.ID
	core.RegisterDocker(d.containerID)

	// The bare Kali image has nothing installed. We must provision essential tools on boot.
	fmt.Printf("  [*] Provisioning advanced Kali toolset in sandbox (nmap, sqlmap, metasploit, etc)...\n")

	// Check if already provisioned
	out, _ := d.Execute(ctx, "which sqlmap")
	if !strings.Contains(out, "/usr/bin/sqlmap") {
		_, _ = d.Execute(ctx, "apt-get update && apt-get install -y iputils-ping nmap dnsutils curl wget sqlmap metasploit-framework gobuster ffuf nuclei crackmapexec impacket-scripts python3-impacket john hashcat seclists subfinder whatweb wpscan")
	}

	return nil
}

// Execute runs a shell command in the sandbox and returns combined output.
func (d *Docker) Execute(ctx context.Context, command string) (string, error) {
	if isDangerous(command) {
		return core.GlobalHitL.RequestApproval() + fmt.Sprintf(" | Command: %s", command), nil
	}
	if d.nativeMode {
		return d.executeNative(ctx, command)
	}
	return d.executeDocker(ctx, command)
}

func (d *Docker) executeDocker(ctx context.Context, command string) (string, error) {
	if d.containerID == "" {
		return "", fmt.Errorf("sandbox not initialized — call Initialize() first")
	}

	// Handle `cd` to update tracked CWD
	if strings.HasPrefix(strings.TrimSpace(command), "cd ") {
		dir := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(command), "cd "))
		if strings.HasPrefix(dir, "/") {
			d.currentCwd = dir
		} else {
			d.currentCwd = defaultWorkDir + "/" + dir
		}
		return fmt.Sprintf("Changed directory to %s", d.currentCwd), nil
	}

	execCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	execResp, err := d.cli.ContainerExecCreate(execCtx, d.containerID, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   d.currentCwd,
		Cmd:          []string{"/bin/bash", "-c", command},
		Env:          []string{"DEBIAN_FRONTEND=noninteractive", "TERM=xterm-256color", "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
	})
	if err != nil {
		return "", fmt.Errorf("exec create failed: %w", err)
	}

	attach, err := d.cli.ContainerExecAttach(execCtx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("exec attach failed: %w", err)
	}
	defer attach.Close()

	// Ensure we unblock StdCopy if context expires
	doneCh := make(chan struct{})
	go func() {
		select {
		case <-execCtx.Done():
			attach.Close()
		case <-doneCh:
		}
	}()

	var stdout, stderr bytes.Buffer
	// stdcopy.StdCopy demultiplexes Docker's multiplexed stream (Tty:false)
	if _, err := stdcopy.StdCopy(&stdout, &stderr, attach.Reader); err != nil && err != io.EOF {
		close(doneCh)
		return "", fmt.Errorf("output read failed (context expired?): %w", err)
	}
	close(doneCh)

	out := stdout.String()
	if stderr.Len() > 0 {
		out += "\n[stderr]\n" + stderr.String()
	}
	if out == "" {
		out = "(command produced no output)"
	}
	out = strings.TrimRight(out, "\n")
	inspect, err := d.cli.ContainerExecInspect(execCtx, execResp.ID)
	if err != nil {
		return out, fmt.Errorf("inspect command exit status: %w", err)
	}
	if inspect.ExitCode != 0 {
		return out, fmt.Errorf("command exited with status %d", inspect.ExitCode)
	}
	return out, nil
}

func (d *Docker) executeNative(ctx context.Context, command string) (string, error) {
	execCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "/bin/bash", "-c", command)
	cmd.Dir = d.currentCwd
	out, err := cmd.CombinedOutput()
	result := strings.TrimRight(string(out), "\n")
	if err != nil {
		return result, fmt.Errorf("command failed: %w", err)
	}
	return result, nil
}

// GetContainerIP returns the Docker bridge network IP of the sandbox container.
// This is the address targets must connect back to for reverse shells and tunnels.
// Falls back to the host's outbound IP if running in native mode or if Docker inspection fails.
func (d *Docker) GetContainerIP(ctx context.Context) string {
	if d.nativeMode || d.containerID == "" {
		// Native mode: use the host's primary outbound IP
		if ip := getHostOutboundIP(); ip != "" {
			return ip
		}
		return "127.0.0.1"
	}

	inspect, err := d.cli.ContainerInspect(ctx, d.containerID)
	if err != nil {
		return getHostOutboundIP()
	}

	// Prefer the bridge network IP
	for _, net := range inspect.NetworkSettings.Networks {
		if net.IPAddress != "" {
			return net.IPAddress
		}
	}

	return getHostOutboundIP()
}

// getHostOutboundIP discovers the host's primary outbound IP by opening a UDP "connection"
// (no packets are sent) to a public address and reading the local IP assigned to that route.
func getHostOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// Stop gracefully shuts down the container.
func (d *Docker) Stop(ctx context.Context) {
	if d.containerID != "" && !d.nativeMode {
		timeout := 5
		_ = d.cli.ContainerStop(ctx, d.containerID, container.StopOptions{Timeout: &timeout})
	}
}

// CopyFile copies a file from the sandbox container to a local destination path.
// In native mode, it copies from the local filesystem directly.
func (d *Docker) CopyFile(ctx context.Context, remotePath, localDest string) (int64, error) {
	if d.nativeMode {
		// In native mode remotePath is already on the host filesystem
		src := remotePath
		if !filepath.IsAbs(src) {
			src = filepath.Join(d.currentCwd, src)
		}
		in, err := os.Open(src)
		if err != nil {
			return 0, fmt.Errorf("open source: %w", err)
		}
		defer in.Close()
		out, err := os.Create(localDest)
		if err != nil {
			return 0, fmt.Errorf("create dest: %w", err)
		}
		defer out.Close()
		return io.Copy(out, in)
	}

	if d.containerID == "" {
		return 0, fmt.Errorf("sandbox not initialized")
	}

	// Use docker cp via exec since the SDK's CopyFromContainer needs tar extraction
	cmd := exec.CommandContext(ctx, "docker", "cp",
		fmt.Sprintf("%s:%s", d.containerID, remotePath),
		localDest,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("docker cp failed: %w — %s", err, string(out))
	}

	fi, err := os.Stat(localDest)
	if err != nil {
		return 0, nil
	}
	return fi.Size(), nil
}

// dangerousPatterns is the blocklist checked before every Execute call.
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)rm\s+-[a-z]*r[a-z]*f`),          // rm -rf
	regexp.MustCompile(`(?i)rm\s+-[a-z]*f[a-z]*r`),          // rm -fr
	regexp.MustCompile(`(?i):\(\)\s*\{`),                    // fork bomb
	regexp.MustCompile(`(?i)mkfs`),                          // format filesystem
	regexp.MustCompile(`(?i)dd\s+if=`),                      // dd overwrite
	regexp.MustCompile(`(?i)DROP\s+(TABLE|DATABASE)`),       // SQL destructive
	regexp.MustCompile(`(?i)shutdown`),                      // shutdown
	regexp.MustCompile(`(?i)reboot`),                        // reboot
	regexp.MustCompile(`(?i)halt`),                          // halt
	regexp.MustCompile(`(?i)masscan.*--rate\s*[5-9]\d{4,}`), // masscan >50k/s
}

// isDangerous returns true if the command matches a destructive pattern.
func isDangerous(cmd string) bool {
	for _, p := range dangerousPatterns {
		if p.MatchString(cmd) {
			return true
		}
	}
	return false
}

func shellQuote(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

// WriteFile writes string content directly to a file inside the sandbox container
// by echoing a base64 encoded string and decoding it inside the container.
func (d *Docker) WriteFile(ctx context.Context, remoteDest, content string) error {
	if d.nativeMode {
		dest := remoteDest
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(d.currentCwd, dest)
		}
		return os.WriteFile(dest, []byte(content), 0644)
	}

	if d.containerID == "" {
		return fmt.Errorf("sandbox not initialized")
	}

	remoteDest = filepath.Clean(remoteDest)
	if strings.Contains(remoteDest, "..") {
		return fmt.Errorf("invalid destination path: path traversal detected")
	}

	b64 := base64.StdEncoding.EncodeToString([]byte(content))

	cmd := fmt.Sprintf("mkdir -p \"$(dirname '%s')\" && echo '%s' | base64 -d > '%s'", shellQuote(remoteDest), shellQuote(b64), shellQuote(remoteDest))

	out, err := d.executeDocker(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to write file to sandbox: %w. Output: %s", err, out)
	}
	return nil
}
