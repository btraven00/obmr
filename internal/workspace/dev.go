package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// WriteLocalYAML reads benchYAML, rewrites every modules[].repository.url
// to the corresponding module's local path and modules[].repository.commit
// to the module's current local branch, and writes the result to
// <bench>.local.yaml. Comments are preserved.
func WriteLocalYAML(benchYAML string, lock *Lock) (string, error) {
	root, err := readYAML(benchYAML)
	if err != nil {
		return "", err
	}
	benchDir := filepath.Dir(benchYAML)
	urlToPath := map[string]string{}
	urlToBranch := map[string]string{}
	for _, m := range lock.Modules {
		urlToPath[normRemote(m.Remote)] = m.Path
		dir := m.Path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(benchDir, m.Path)
		}
		if branch, err := Git(dir, "rev-parse", "--abbrev-ref", "HEAD"); err == nil && branch != "HEAD" {
			urlToBranch[normRemote(m.Remote)] = branch
		}
	}
	walkRepositories(root, func(repo *yaml.Node) {
		url := repoURL(repo)
		key := normRemote(url)
		setMapStringValue(repo, "url", func(_ string) (string, bool) {
			p, ok := urlToPath[key]
			return p, ok
		})
		setMapStringValue(repo, "commit", func(_ string) (string, bool) {
			b, ok := urlToBranch[key]
			return b, ok
		})
	})
	return writeYAML(root, localOutputPath(benchYAML))
}

func localOutputPath(benchYAML string) string {
	dir := filepath.Dir(benchYAML)
	base := filepath.Base(benchYAML)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, name+".local"+ext)
}

func readYAML(path string) (*yaml.Node, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(src, &root); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &root, nil
}

func writeYAML(root *yaml.Node, path string) (string, error) {
	out, err := yaml.Marshal(root)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return "", err
	}
	return path, nil
}

// walkRepositories invokes fn for every mapping node found under a
// `repository:` key.
func walkRepositories(n *yaml.Node, fn func(repo *yaml.Node)) {
	if n == nil {
		return
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i]
			v := n.Content[i+1]
			if k.Value == "repository" && v.Kind == yaml.MappingNode {
				fn(v)
			}
			walkRepositories(v, fn)
		}
		return
	}
	for _, c := range n.Content {
		walkRepositories(c, fn)
	}
}

// setMapStringValue updates the scalar value of `key` in mapping m. fn
// receives the existing value and returns (new, true) to write or (_, false)
// to skip.
func setMapStringValue(m *yaml.Node, key string, fn func(old string) (string, bool)) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		k := m.Content[i]
		v := m.Content[i+1]
		if k.Value == key && v.Kind == yaml.ScalarNode {
			if newV, ok := fn(v.Value); ok {
				v.Value = newV
				v.Style = 0
				v.Tag = "!!str"
			}
			return
		}
	}
}

// repoURL returns the `url` scalar value from a repository mapping, or "".
func repoURL(m *yaml.Node) string {
	for i := 0; i+1 < len(m.Content); i += 2 {
		k := m.Content[i]
		v := m.Content[i+1]
		if k.Value == "url" && v.Kind == yaml.ScalarNode {
			return v.Value
		}
	}
	return ""
}
