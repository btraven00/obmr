package workspace

import (
	"fmt"
	"strings"
)

// TrimModule deletes local branches in dir that are merged into
// origin/HEAD's branch. If onlyBranch is non-empty, only that branch
// is considered. If force is true, uses `git branch -D` (deletes even
// unmerged branches).
//
// Skips modules with a dirty working tree (returns a status string).
// If the current branch is among those to be deleted, the module is
// switched to the upstream default branch first.
func TrimModule(dir, onlyBranch string, force bool) (string, error) {
	// Refuse on dirty trees.
	if dirty, _ := Git(dir, "status", "--porcelain"); strings.TrimSpace(dirty) != "" {
		return "skip (dirty)", nil
	}
	defaultBranch, err := DefaultBranch(dir)
	if err != nil {
		return "", err
	}
	current, err := Git(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}

	var candidates []string
	if onlyBranch != "" {
		// Verify it exists locally before queueing.
		if _, err := Git(dir, "show-ref", "--verify", "--quiet", "refs/heads/"+onlyBranch); err == nil {
			candidates = []string{onlyBranch}
		}
	} else {
		out, err := Git(dir, "branch", "--format=%(refname:short)", "--merged", "origin/HEAD")
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(out, "\n") {
			b := strings.TrimSpace(line)
			if b == "" || b == defaultBranch {
				continue
			}
			candidates = append(candidates, b)
		}
	}

	if len(candidates) == 0 {
		return "nothing to trim", nil
	}

	// If we'd delete the current branch, switch away first.
	for _, b := range candidates {
		if b == current {
			if _, err := Git(dir, "checkout", defaultBranch); err != nil {
				return "", fmt.Errorf("switch to %s: %w", defaultBranch, err)
			}
			break
		}
	}

	flag := "-d"
	if force {
		flag = "-D"
	}
	deleted := []string{}
	for _, b := range candidates {
		if _, err := Git(dir, "branch", flag, b); err != nil {
			// On non-force runs, unmerged branches will fail; report and continue.
			return "", fmt.Errorf("delete %s: %w", b, err)
		}
		deleted = append(deleted, b)
	}
	return "deleted: " + strings.Join(deleted, ", "), nil
}

