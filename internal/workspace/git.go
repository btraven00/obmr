package workspace

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Git runs `git <args...>` in dir. Returns trimmed stdout.
func Git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// IsGitRepo reports whether dir is a git working tree.
func IsGitRepo(dir string) bool {
	if _, err := os.Stat(dir); err != nil {
		return false
	}
	_, err := Git(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

// Clone clones remote into dir. dir must not exist.
func Clone(remote, dir string) error {
	cmd := exec.Command("git", "clone", remote, dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RemoteURL returns the configured `origin` URL of the repo at dir.
func RemoteURL(dir string) (string, error) {
	return Git(dir, "config", "--get", "remote.origin.url")
}

// DefaultBranch returns the branch that origin/HEAD points to, e.g. "main".
// Falls back to "main" if origin/main exists when origin/HEAD is unset.
func DefaultBranch(dir string) (string, error) {
	out, err := Git(dir, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		if _, e2 := Git(dir, "show-ref", "--verify", "--quiet", "refs/remotes/origin/main"); e2 == nil {
			return "main", nil
		}
		return "", err
	}
	parts := strings.Split(out, "/")
	return parts[len(parts)-1], nil
}

// EnsureOnDefault switches dir to its origin default branch. Skips if the
// working tree is dirty or the module is already on the default branch.
// Returns a short status message.
func EnsureOnDefault(dir string) (string, error) {
	def, err := DefaultBranch(dir)
	if err != nil {
		return "", err
	}
	current, err := Git(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if current == def {
		return "on " + def, nil
	}
	if dirty, _ := Git(dir, "status", "--porcelain"); strings.TrimSpace(dirty) != "" {
		return "skip dirty (on " + current + ", default " + def + ")", nil
	}
	if _, err := Git(dir, "checkout", def); err != nil {
		return "", err
	}
	return current + " -> " + def, nil
}
