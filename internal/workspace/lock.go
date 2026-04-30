package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const LockName = ".obflow.lock"

type Lock struct {
	BenchmarkFile string         `yaml:"benchmark_file"`
	ParentDir     string         `yaml:"parent_dir"`
	Modules       []LockedModule `yaml:"modules"`
}

type LockedModule struct {
	ID     string `yaml:"id"`
	Stage  string `yaml:"stage"`
	Remote string `yaml:"remote"`
	Commit string `yaml:"commit"`
	Path   string `yaml:"path"`
}

// LockPath returns the lock file path for a benchmark YAML.
func LockPath(benchYAML string) string {
	return filepath.Join(filepath.Dir(benchYAML), LockName)
}

func LoadLock(path string) (*Lock, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var l Lock
	if err := yaml.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &l, nil
}

func (l *Lock) Save(path string) error {
	b, err := yaml.Marshal(l)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
