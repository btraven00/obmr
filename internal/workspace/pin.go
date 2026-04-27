package workspace

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PinResult records what happened for one module during Pin.
type PinResult struct {
	ModuleID string
	URL      string
	OldSHA   string
	NewSHA   string
	Warning  string // non-fatal issue (e.g., not an ancestor)
	Err      error
}

// Pin updates the canonical YAML's repository.commit for each module by
// reading `origin/<ref>` (after `git fetch origin`) in each module's
// clone. ref defaults to "HEAD" (i.e., origin/HEAD = upstream default
// branch) when empty.
//
// Mutates benchYAML in place. Returns per-module results so the caller
// can present diffs/warnings.
func Pin(benchYAML string, lock *Lock, ref string) ([]PinResult, error) {
	if ref == "" {
		ref = "HEAD"
	}
	benchDir, err := filepath.Abs(filepath.Dir(benchYAML))
	if err != nil {
		return nil, err
	}

	// Fetch + rev-parse for each module, in parallel.
	fetchResults := Fanout(benchDir, lock, func(dir string, _ LockedModule) (string, error) {
		if _, err := Git(dir, "fetch", "origin", "--quiet"); err != nil {
			return "", err
		}
		sha, err := Git(dir, "rev-parse", "origin/"+ref)
		if err != nil {
			return "", err
		}
		return sha, nil
	})

	urlToNew := map[string]string{}
	results := make([]PinResult, 0, len(lock.Modules))
	for _, fr := range fetchResults {
		pr := PinResult{ModuleID: fr.Module.ID, URL: fr.Module.Remote}
		if fr.Err != nil {
			pr.Err = fr.Err
			results = append(results, pr)
			continue
		}
		pr.NewSHA = fr.Out
		urlToNew[normRemote(fr.Module.Remote)] = fr.Out
		results = append(results, pr)
	}

	// Read YAML, capture old SHAs, rewrite, write back.
	root, err := readYAML(benchYAML)
	if err != nil {
		return results, err
	}
	urlToOld := map[string]string{}
	walkRepositories(root, func(repo *yaml.Node) {
		url := repoURL(repo)
		if url == "" {
			return
		}
		// capture old commit
		for i := 0; i+1 < len(repo.Content); i += 2 {
			k := repo.Content[i]
			v := repo.Content[i+1]
			if k.Value == "commit" && v.Kind == yaml.ScalarNode {
				urlToOld[normRemote(url)] = v.Value
			}
		}
		setMapStringValue(repo, "commit", func(old string) (string, bool) {
			n, ok := urlToNew[normRemote(url)]
			return n, ok
		})
	})

	for i := range results {
		key := normRemote(results[i].URL)
		results[i].OldSHA = urlToOld[key]
	}

	if _, err := writeYAML(root, benchYAML); err != nil {
		return results, err
	}

	// Ancestry checks (post-write; informational).
	for i := range results {
		if results[i].Err != nil || results[i].OldSHA == "" || results[i].NewSHA == "" {
			continue
		}
		mod := findModule(lock, results[i].ModuleID)
		if mod == nil {
			continue
		}
		dir := filepath.Join(benchDir, mod.Path)
		// Is OldSHA an ancestor of NewSHA? Non-fast-forward warning if not.
		_, err := Git(dir, "merge-base", "--is-ancestor", results[i].OldSHA, results[i].NewSHA)
		if err != nil && !strings.Contains(err.Error(), "exit status 1") {
			// Some other error; ignore for now.
			continue
		}
		if err != nil {
			results[i].Warning = fmt.Sprintf("not a fast-forward (old %s not ancestor of %s)", short(results[i].OldSHA), short(results[i].NewSHA))
		}
	}

	return results, nil
}

func findModule(lock *Lock, id string) *LockedModule {
	for i := range lock.Modules {
		if lock.Modules[i].ID == id {
			return &lock.Modules[i]
		}
	}
	return nil
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
