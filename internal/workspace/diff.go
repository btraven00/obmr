package workspace

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LocalDiff summarizes how the on-disk <bench>.local.yaml diverges from
// what `obflow dev` would have produced from the current canonical + lock.
// Anything beyond the rewritten repository.url / repository.commit fields
// counts as user edits.
type LocalDiff struct {
	Missing         bool     // .local.yaml does not exist
	Diverged        bool     // any difference detected
	AddedStages     []string // stage IDs only in actual
	RemovedStages   []string // stage IDs only in expected
	ModifiedStages  []string // stage IDs whose stage-level fields differ (excluding modules)
	AddedModules    []string // "stage/module"
	RemovedModules  []string // "stage/module"
	ModifiedModules []string // "stage/module" with content differences
}

// DiffLocal compares <bench>.local.yaml on disk against what would be
// regenerated from the canonical YAML + lock right now. Returns a
// summary; LocalDiff.Missing=true if the file does not exist.
func DiffLocal(canonicalYAML string, lock *Lock) (*LocalDiff, error) {
	localPath := localOutputPath(canonicalYAML)
	actualBytes, err := os.ReadFile(localPath)
	if os.IsNotExist(err) {
		return &LocalDiff{Missing: true}, nil
	}
	if err != nil {
		return nil, err
	}

	expectedBytes, err := renderExpectedLocal(canonicalYAML, lock)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(actualBytes, expectedBytes) {
		return &LocalDiff{}, nil
	}

	d := &LocalDiff{Diverged: true}
	var expectedRoot, actualRoot yaml.Node
	if err := yaml.Unmarshal(expectedBytes, &expectedRoot); err != nil {
		return d, err
	}
	if err := yaml.Unmarshal(actualBytes, &actualRoot); err != nil {
		return d, err
	}
	diffStages(&expectedRoot, &actualRoot, d)
	return d, nil
}

// renderExpectedLocal returns the byte content `obflow dev` would write
// right now (without touching disk). Mirrors WriteLocalYAML: rewrites
// both repository.url and repository.commit per module.
func renderExpectedLocal(canonicalYAML string, lock *Lock) ([]byte, error) {
	root, err := readYAML(canonicalYAML)
	if err != nil {
		return nil, err
	}
	benchDir := filepath.Dir(canonicalYAML)
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
	return yaml.Marshal(root)
}

// findStagesNode descends from a document/mapping node and returns the
// sequence node holding the stages list.
func findStagesNode(root *yaml.Node) *yaml.Node {
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		return findStagesNode(root.Content[0])
	}
	if root.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "stages" && root.Content[i+1].Kind == yaml.SequenceNode {
			return root.Content[i+1]
		}
	}
	return nil
}

// stageMap returns stage-id -> stage mapping node.
func stageMap(stages *yaml.Node) map[string]*yaml.Node {
	out := map[string]*yaml.Node{}
	if stages == nil {
		return out
	}
	for _, s := range stages.Content {
		id := mapStringValue(s, "id")
		if id != "" {
			out[id] = s
		}
	}
	return out
}

func moduleMap(stage *yaml.Node) map[string]*yaml.Node {
	out := map[string]*yaml.Node{}
	for i := 0; i+1 < len(stage.Content); i += 2 {
		k := stage.Content[i]
		v := stage.Content[i+1]
		if k.Value == "modules" && v.Kind == yaml.SequenceNode {
			for _, m := range v.Content {
				id := mapStringValue(m, "id")
				if id != "" {
					out[id] = m
				}
			}
		}
	}
	return out
}

func mapStringValue(m *yaml.Node, key string) string {
	if m == nil || m.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key && m.Content[i+1].Kind == yaml.ScalarNode {
			return m.Content[i+1].Value
		}
	}
	return ""
}

func diffStages(expected, actual *yaml.Node, d *LocalDiff) {
	es := stageMap(findStagesNode(expected))
	as := stageMap(findStagesNode(actual))

	for id := range as {
		if _, ok := es[id]; !ok {
			d.AddedStages = append(d.AddedStages, id)
		}
	}
	for id := range es {
		if _, ok := as[id]; !ok {
			d.RemovedStages = append(d.RemovedStages, id)
		}
	}
	for id, e := range es {
		a, ok := as[id]
		if !ok {
			continue
		}
		// Compare modules within the stage.
		em := moduleMap(e)
		am := moduleMap(a)
		for mid := range am {
			if _, ok := em[mid]; !ok {
				d.AddedModules = append(d.AddedModules, id+"/"+mid)
			}
		}
		for mid := range em {
			if _, ok := am[mid]; !ok {
				d.RemovedModules = append(d.RemovedModules, id+"/"+mid)
			}
		}
		for mid, en := range em {
			an, ok := am[mid]
			if !ok {
				continue
			}
			if !nodesEqualYAML(en, an) {
				d.ModifiedModules = append(d.ModifiedModules, id+"/"+mid)
			}
		}
		// Stage-level field diffs (anything other than `id` and `modules`).
		if !stageNonModuleFieldsEqual(e, a) {
			d.ModifiedStages = append(d.ModifiedStages, id)
		}
	}
}

func stageNonModuleFieldsEqual(e, a *yaml.Node) bool {
	mk := func(n *yaml.Node) map[string]*yaml.Node {
		out := map[string]*yaml.Node{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i].Value
			if k == "modules" {
				continue
			}
			out[k] = n.Content[i+1]
		}
		return out
	}
	em, am := mk(e), mk(a)
	if len(em) != len(am) {
		return false
	}
	for k, ev := range em {
		av, ok := am[k]
		if !ok {
			return false
		}
		if !nodesEqualYAML(ev, av) {
			return false
		}
	}
	return true
}

// nodesEqualYAML compares two YAML nodes by their re-serialized content.
// Cheaper than a structural walk and good enough for this use.
func nodesEqualYAML(a, b *yaml.Node) bool {
	ab, err := yaml.Marshal(a)
	if err != nil {
		return false
	}
	bb, err := yaml.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(ab, bb)
}

// LocalOnlyModule describes a module present in <bench>.local.yaml but not
// in the lock — typically one that the user added in dev mode and has not
// promoted/pinned yet.
type LocalOnlyModule struct {
	Stage  string
	ID     string
	URL    string // raw repository.url from .local.yaml
	AbsDir string // URL resolved relative to the benchmark dir
}

// LocalOnlyModules returns modules declared in <bench>.local.yaml whose
// (stage, id) pair is not in lock. Returns nil if .local.yaml is missing.
func LocalOnlyModules(canonicalYAML string, lock *Lock) ([]LocalOnlyModule, error) {
	localPath := localOutputPath(canonicalYAML)
	b, err := os.ReadFile(localPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	have := map[string]bool{}
	for _, m := range lock.Modules {
		have[m.Stage+"/"+m.ID] = true
	}
	benchDir := filepath.Dir(canonicalYAML)
	stages := findStagesNode(&root)
	if stages == nil {
		return nil, nil
	}
	var out []LocalOnlyModule
	for _, s := range stages.Content {
		sid := mapStringValue(s, "id")
		if sid == "" {
			continue
		}
		for mid, m := range moduleMap(s) {
			if have[sid+"/"+mid] {
				continue
			}
			url := ""
			for i := 0; i+1 < len(m.Content); i += 2 {
				if m.Content[i].Value == "repository" && m.Content[i+1].Kind == yaml.MappingNode {
					url = repoURL(m.Content[i+1])
					break
				}
			}
			abs := url
			if abs != "" && !filepath.IsAbs(abs) {
				abs = filepath.Join(benchDir, abs)
			}
			out = append(out, LocalOnlyModule{Stage: sid, ID: mid, URL: url, AbsDir: abs})
		}
	}
	return out, nil
}

// Summary returns a one-line summary like "+1 stages, -0 stages, ~2 modules"
// or "" if no divergence.
func (d *LocalDiff) Summary() string {
	if d == nil || d.Missing || !d.Diverged {
		return ""
	}
	return fmt.Sprintf("stages +%d -%d ~%d, modules +%d -%d ~%d",
		len(d.AddedStages), len(d.RemovedStages), len(d.ModifiedStages),
		len(d.AddedModules), len(d.RemovedModules), len(d.ModifiedModules))
}
