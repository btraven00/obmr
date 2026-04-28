package benchmark

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type File struct {
	Name            string  `yaml:"name"`
	ID              string  `yaml:"id"`
	Description     string  `yaml:"description"`
	Version         string  `yaml:"version"`
	SoftwareBackend string  `yaml:"software_backend"`
	Stages          []Stage `yaml:"stages"`
}

type Stage struct {
	ID      string   `yaml:"id"`
	Inputs  []string `yaml:"inputs,omitempty"`
	Outputs []Output `yaml:"outputs,omitempty"`
	Modules []Module `yaml:"modules"`
}

type Output struct {
	ID   string `yaml:"id"`
	Path string `yaml:"path"`
}

type Module struct {
	ID         string                   `yaml:"id"`
	Name       string                   `yaml:"name"`
	Repository Repository               `yaml:"repository"`
	Parameters []map[string]interface{} `yaml:"parameters,omitempty"`
}

type Repository struct {
	URL        string `yaml:"url"`
	Commit     string `yaml:"commit"`
	Entrypoint string `yaml:"entrypoint,omitempty"`
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

// ProducerOf returns the stage and output declaration that produces
// the given output id. ok is false if no stage declares it.
func (f *File) ProducerOf(outputID string) (Stage, Output, bool) {
	for _, s := range f.Stages {
		for _, o := range s.Outputs {
			if o.ID == outputID {
				return s, o, true
			}
		}
	}
	return Stage{}, Output{}, false
}

// FindModule returns the stage and module with the given module id.
func (f *File) FindModule(moduleID string) (Stage, Module, bool) {
	for _, s := range f.Stages {
		for _, m := range s.Modules {
			if m.ID == moduleID {
				return s, m, true
			}
		}
	}
	return Stage{}, Module{}, false
}
