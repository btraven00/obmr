package benchmark

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type File struct {
	Name        string  `yaml:"name"`
	ID          string  `yaml:"id"`
	Description string  `yaml:"description"`
	Version     string  `yaml:"version"`
	Stages      []Stage `yaml:"stages"`
}

type Stage struct {
	ID      string   `yaml:"id"`
	Modules []Module `yaml:"modules"`
}

type Module struct {
	ID         string     `yaml:"id"`
	Name       string     `yaml:"name"`
	Repository Repository `yaml:"repository"`
}

type Repository struct {
	URL    string `yaml:"url"`
	Commit string `yaml:"commit"`
}

func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var f File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &f, nil
}

// Modules flattens all modules across stages, in declaration order.
func (f *File) Modules() []Module {
	var out []Module
	for _, s := range f.Stages {
		out = append(out, s.Modules...)
	}
	return out
}
