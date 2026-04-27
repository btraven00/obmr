package workspace

import (
	"fmt"
	"strings"
)

// PushModule pushes the current branch of dir to `fork` if that remote
// exists, otherwise to `origin`. Skips if there are no commits ahead of
// the upstream tracking branch (or if no upstream is configured and the
// branch matches the chosen remote's default branch state -- in which
// case we still push -u so future runs have an upstream).
func PushModule(dir string) (string, error) {
	branch, err := Git(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if branch == "HEAD" {
		return "", fmt.Errorf("detached HEAD")
	}
	remote := "origin"
	if has, _ := HasRemote(dir, "fork"); has {
		remote = "fork"
	}
	// Determine whether there's anything to push. If we have an upstream,
	// compare; otherwise fall through to push -u.
	if upstream, err := Git(dir, "rev-parse", "--abbrev-ref", "@{u}"); err == nil && upstream != "" {
		ahead, err := Git(dir, "rev-list", "--count", upstream+"..HEAD")
		if err == nil && strings.TrimSpace(ahead) == "0" {
			return fmt.Sprintf("skip (clean: %s tracks %s)", branch, upstream), nil
		}
	}
	out, err := Git(dir, "push", "-u", remote, branch)
	if err != nil {
		return "", err
	}
	if out == "" {
		out = fmt.Sprintf("pushed %s -> %s", branch, remote)
	}
	return out, nil
}
