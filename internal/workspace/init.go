package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/btraven00/obmr/internal/benchmark"
)

// Init clones every module declared in benchYAML into parentDir (resolved
// relative to the YAML's directory). Existing clones are reused if their
// remote matches; otherwise the operation aborts.
//
// Returns the resulting Lock (not yet written).
func Init(benchYAML string, parentDir string) (*Lock, error) {
	f, err := benchmark.Load(benchYAML)
	if err != nil {
		return nil, err
	}
	benchDir, err := filepath.Abs(filepath.Dir(benchYAML))
	if err != nil {
		return nil, err
	}
	parentAbs := parentDir
	if !filepath.IsAbs(parentAbs) {
		parentAbs = filepath.Join(benchDir, parentDir)
	}
	if err := os.MkdirAll(parentAbs, 0755); err != nil {
		return nil, fmt.Errorf("create parent dir: %w", err)
	}

	lock := &Lock{
		BenchmarkFile: filepath.Base(benchYAML),
		ParentDir:     parentDir,
	}

	for _, s := range f.Stages {
		for _, m := range s.Modules {
			dirName := repoDirName(m.Repository.URL, m.ID)
			absPath := filepath.Join(parentAbs, dirName)
			relPath, _ := filepath.Rel(benchDir, absPath)

			if err := ensureClone(m.Repository.URL, absPath); err != nil {
				return nil, fmt.Errorf("module %s: %w", m.ID, err)
			}
			sha, err := Git(absPath, "rev-parse", "HEAD")
			if err != nil {
				return nil, fmt.Errorf("module %s: %w", m.ID, err)
			}
			lock.Modules = append(lock.Modules, LockedModule{
				ID:     m.ID,
				Stage:  s.ID,
				Remote: m.Repository.URL,
				Commit: sha,
				Path:   relPath,
			})
		}
	}
	return lock, nil
}

func ensureClone(remote, dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "cloning %s -> %s\n", remote, dir)
		return Clone(remote, dir)
	} else if err != nil {
		return err
	}
	if !IsGitRepo(dir) {
		return fmt.Errorf("%s exists but is not a git repo", dir)
	}
	got, err := RemoteURL(dir)
	if err != nil {
		return err
	}
	if !sameRemote(got, remote) {
		return fmt.Errorf("%s exists with remote %s, expected %s", dir, got, remote)
	}
	fmt.Fprintf(os.Stderr, "reusing %s (remote ok)\n", dir)
	return nil
}

// repoDirName picks a local directory name from the repo URL, falling back
// to the module ID. e.g. https://github.com/org/3-normalize -> 3-normalize.
func repoDirName(url, fallback string) string {
	u := strings.TrimSuffix(url, ".git")
	if i := strings.LastIndex(u, "/"); i >= 0 && i < len(u)-1 {
		return u[i+1:]
	}
	return fallback
}

func sameRemote(a, b string) bool {
	return normRemote(a) == normRemote(b)
}

func normRemote(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimSuffix(s, "/")
	return s
}
