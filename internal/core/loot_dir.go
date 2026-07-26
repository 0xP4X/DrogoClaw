package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// LootDir returns the path to the operator's loot directory, creating it if needed.
func LootDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, "drogonclaw_loot")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create loot directory: %w", err)
	}
	return dir, nil
}
