package core

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

var (
	cleanupMu     sync.Mutex
	activePIDs    = make(map[int]bool)
	activeDockers = make(map[string]bool)
)

// RegisterPID adds a raw OS process ID to the cleanup registry.
func RegisterPID(pid int) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	activePIDs[pid] = true
}

// UnregisterPID removes a PID from the registry.
func UnregisterPID(pid int) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	delete(activePIDs, pid)
}

// RegisterDocker adds a Docker container ID to the cleanup registry.
func RegisterDocker(containerID string) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	activeDockers[containerID] = true
}

// UnregisterDocker removes a Docker container ID from the registry.
func UnregisterDocker(containerID string) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	delete(activeDockers, containerID)
}

// PerformCleanup kills all registered processes and destroys registered containers.
func PerformCleanup() {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()

	for pid := range activePIDs {
		fmt.Printf("  [~] Cleanup: Killing orphaned PID %d\n", pid)
		process, err := os.FindProcess(pid)
		if err == nil {
			_ = process.Kill()
		}
	}

	for cid := range activeDockers {
		fmt.Printf("  [~] Cleanup: Terminating Docker sandbox %s\n", cid[:8])
		_ = exec.Command("docker", "rm", "-f", cid).Run()
	}
}

// InitCleanupHandler sets up the signal trap for graceful shutdown.
func InitCleanupHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\n\n  [!] Operator abort signal received (Ctrl+C). Initiating emergency teardown...")
		PerformCleanup()
		os.Exit(1)
	}()
}
