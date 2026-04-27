package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DirName  = ".obmr"
	FileName = "config.yaml"
)

type Config struct {
	Default Default `yaml:"default"`
}

type Default struct {
	Plan string `yaml:"plan"`
}

// Find walks up from start looking for .obmr/config.yaml. Returns the path
// to the config file, or "" if none found.
func Find(start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for {
		p := filepath.Join(dir, DirName, FileName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// Load reads the config at path. Returns (nil, nil) if path is empty.
func Load(path string) (*Config, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &c, nil
}

// Save writes c to <baseDir>/.obmr/config.yaml, creating the dir as needed.
func Save(baseDir string, c *Config) (string, error) {
	dir := filepath.Join(baseDir, DirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	p := filepath.Join(dir, FileName)
	b, err := yaml.Marshal(c)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(p, b, 0644); err != nil {
		return "", err
	}
	return p, nil
}

// ResolvePlan returns the configured default plan from a config found by
// walking up from cwd. Returns "" if no config or no plan set.
func ResolvePlan(cwd string) (string, error) {
	cp := Find(cwd)
	if cp == "" {
		return "", nil
	}
	c, err := Load(cp)
	if err != nil {
		return "", err
	}
	if c == nil || c.Default.Plan == "" {
		return "", nil
	}
	plan := c.Default.Plan
	if !filepath.IsAbs(plan) {
		plan = filepath.Join(filepath.Dir(filepath.Dir(cp)), plan)
	}
	if _, err := os.Stat(plan); err != nil {
		return "", fmt.Errorf("config at %s points to missing plan %s", cp, plan)
	}
	return plan, nil
}

var ErrNoPlan = errors.New("no plan: pass <bench.yaml> or run `obmr use <plan>` first")
