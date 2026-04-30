package workspace

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Promote rewrites canonicalYAML in place from <bench>.local.yaml,
// substituting each module's local path back to its canonical url and
// commit. Modules that are new in local (no lock entry) are passed
// through unchanged. Comments and structure of the local YAML are
// preserved.
func Promote(canonicalYAML string, lock *Lock) error {
	localPath := localOutputPath(canonicalYAML)
	if _, err := os.Stat(localPath); err != nil {
		return fmt.Errorf("no local YAML at %s (run `obflow dev` first)", localPath)
	}

	// Capture canonical url -> commit so we can restore SHAs.
	canonRoot, err := readYAML(canonicalYAML)
	if err != nil {
		return err
	}
	urlToCommit := map[string]string{}
	walkRepositories(canonRoot, func(repo *yaml.Node) {
		u := repoURL(repo)
		if u == "" {
			return
		}
		for i := 0; i+1 < len(repo.Content); i += 2 {
			k := repo.Content[i]
			v := repo.Content[i+1]
			if k.Value == "commit" && v.Kind == yaml.ScalarNode {
				urlToCommit[normRemote(u)] = v.Value
			}
		}
	})

	pathToRemote := map[string]string{}
	for _, m := range lock.Modules {
		pathToRemote[m.Path] = m.Remote
	}

	localRoot, err := readYAML(localPath)
	if err != nil {
		return err
	}
	var bad []string
	walkRepositories(localRoot, func(repo *yaml.Node) {
		current := repoURL(repo)
		if remote, ok := pathToRemote[current]; ok {
			setMapStringValue(repo, "url", func(_ string) (string, bool) { return remote, true })
			if commit, ok := urlToCommit[normRemote(remote)]; ok {
				setMapStringValue(repo, "commit", func(_ string) (string, bool) { return commit, true })
			}
			return
		}
		// Not in lock: must already be a remote URL, otherwise canonical
		// would end up with a local path in it.
		if !looksLikeRemoteURL(current) {
			bad = append(bad, current)
		}
	})
	if len(bad) > 0 {
		return fmt.Errorf("cannot promote: %d module(s) have no remote url; add an origin first:\n  - %s",
			len(bad), strings.Join(bad, "\n  - "))
	}

	_, err = writeYAML(localRoot, canonicalYAML)
	return err
}

func looksLikeRemoteURL(s string) bool {
	switch {
	case strings.HasPrefix(s, "http://"),
		strings.HasPrefix(s, "https://"),
		strings.HasPrefix(s, "ssh://"),
		strings.HasPrefix(s, "git@"),
		strings.HasPrefix(s, "git+"):
		return true
	}
	return false
}
