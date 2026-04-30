package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/btraven00/obflow/internal/config"
	"github.com/btraven00/obflow/internal/workspace"
)

const (
	ansiReset   = "\x1b[0m"
	ansiBold    = "\x1b[1m"
	ansiDim     = "\x1b[2m"
	ansiRed     = "\x1b[31m"
	ansiGreen   = "\x1b[32m"
	ansiYellow  = "\x1b[33m"
	ansiBlue    = "\x1b[34m"
	ansiMagenta = "\x1b[35m"
	ansiCyan    = "\x1b[36m"
)

func isTTY() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	st, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) != 0
}

func paint(s, code string) string {
	if !isTTY() {
		return s
	}
	return code + s + ansiReset
}

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

// localYAMLPathFromCanonical mirrors workspace.localOutputPath: turns
// /a/b/foo.yaml into /a/b/foo.local.yaml.
func localYAMLPathFromCanonical(canonical string) string {
	dir := filepath.Dir(canonical)
	base := filepath.Base(canonical)
	name := base
	if i := len(base) - len(filepath.Ext(base)); i > 0 {
		name = base[:i]
	}
	return filepath.Join(dir, name+".local"+filepath.Ext(base))
}

// loadLock resolves the lock file at <bench-dir>/.obflow.lock and returns it
// plus the absolute benchmark dir.
func loadLock(yamlPath string) (*workspace.Lock, string, error) {
	benchDir, err := filepath.Abs(filepath.Dir(yamlPath))
	if err != nil {
		return nil, "", err
	}
	lockPath := workspace.LockPath(yamlPath)
	lock, err := workspace.LoadLock(lockPath)
	if err != nil {
		return nil, "", fmt.Errorf("load %s: %w (run `obflow init` first?)", lockPath, err)
	}
	return lock, benchDir, nil
}
