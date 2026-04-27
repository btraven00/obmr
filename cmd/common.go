package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/btraven00/obmr/internal/config"
	"github.com/btraven00/obmr/internal/workspace"
)

// resolvePlan returns the YAML path to use: the explicit arg, or the
// configured default plan if arg is "".
func resolvePlan(arg string) (string, error) {
	if arg != "" {
		return arg, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	plan, err := config.ResolvePlan(cwd)
	if err != nil {
		return "", err
	}
	if plan == "" {
		return "", config.ErrNoPlan
	}
	return plan, nil
}

// firstArg returns args[0] or "" if no args were given.
func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// loadLock resolves the lock file at <bench-dir>/.obmr.lock and returns it
// plus the absolute benchmark dir.
func loadLock(yamlPath string) (*workspace.Lock, string, error) {
	benchDir, err := filepath.Abs(filepath.Dir(yamlPath))
	if err != nil {
		return nil, "", err
	}
	lockPath := workspace.LockPath(yamlPath)
	lock, err := workspace.LoadLock(lockPath)
	if err != nil {
		return nil, "", fmt.Errorf("load %s: %w (run `obmr init` first?)", lockPath, err)
	}
	return lock, benchDir, nil
}
