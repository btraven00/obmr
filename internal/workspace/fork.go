package workspace

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// EnsureForkRemote checks whether a `fork` remote exists in dir; if not,
// runs `gh repo fork --remote=fork --clone=false` to create one. Idempotent.
// Returns a short status message describing what happened.
func EnsureForkRemote(dir string) (string, error) {
	if has, err := HasRemote(dir, "fork"); err != nil {
		return "", err
	} else if has {
		url, _ := Git(dir, "config", "--get", "remote.fork.url")
		return "fork already set: " + url, nil
	}
	// gh repo fork uses cwd's git remote 'origin' to know what to fork.
	cmd := exec.Command("gh", "repo", "fork", "--remote=true", "--remote-name=fork", "--clone=false")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh repo fork: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	url, _ := Git(dir, "config", "--get", "remote.fork.url")
	return "fork created: " + url, nil
}

// HasRemote reports whether the named git remote exists in dir.
func HasRemote(dir, name string) (bool, error) {
	out, err := Git(dir, "remote")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == name {
			return true, nil
		}
	}
	return false, nil
}
