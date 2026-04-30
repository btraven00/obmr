package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/btraven00/obflow/internal/benchmark"
)

// Resolved is the result of resolving a target module's inputs against
// the bench `out/` tree.
type Resolved struct {
	Dataset string            // value substituted for {dataset}
	Inputs  map[string]string // input id -> absolute path of latest matching file
}

// ResolveInputs walks outRoot looking for files produced by each
// upstream stage that feeds the target module, as declared by the
// benchmark plan. No file patterns are hardcoded — everything is
// derived from plan stage Inputs/Outputs and Output.Path globs.
//
// Strategy:
//   - For each input id, look up the producer stage and its Output.Path
//     glob, then collect all candidate files in outRoot whose path
//     contains the stage id + a producer module id and whose basename
//     matches the glob.
//   - Anchor on the deepest candidate (longest path) of any input —
//     that's the most-nested stage in ob's DAG layout. Constrain
//     remaining inputs to candidates whose directory is an ancestor of
//     the anchor's path. This guarantees a single coherent upstream
//     chain (same normalization, same filter, …).
//   - Among constrained candidates, pick latest mtime.
//
// dataset (the {dataset} value) is read from the matched filename.
func ResolveInputs(plan *benchmark.File, moduleID, outRoot string) (*Resolved, error) {
	stage, _, ok := plan.FindModule(moduleID)
	if !ok {
		return nil, fmt.Errorf("module %q not found in plan", moduleID)
	}
	if len(stage.Inputs) == 0 {
		return &Resolved{Inputs: map[string]string{}}, nil
	}

	type lookup struct {
		inputID string
		producer benchmark.Stage
		output   benchmark.Output
		cands    []candidate
	}
	lookups := make([]*lookup, 0, len(stage.Inputs))
	for _, inputID := range stage.Inputs {
		producer, output, ok := plan.ProducerOf(inputID)
		if !ok {
			return nil, fmt.Errorf("input %q for module %q has no producing stage in plan",
				inputID, moduleID)
		}
		moduleIDs := make([]string, 0, len(producer.Modules))
		for _, m := range producer.Modules {
			moduleIDs = append(moduleIDs, m.ID)
		}
		cands, err := collectProducerOutputs(outRoot, producer.ID, moduleIDs, output.Path)
		if err != nil {
			return nil, fmt.Errorf("resolve input %q (from stage %q): %w", inputID, producer.ID, err)
		}
		if len(cands) == 0 {
			return nil, fmt.Errorf("no files matching %q under %q for stage %q (no successful upstream run?)",
				output.Path, outRoot, producer.ID)
		}
		lookups = append(lookups, &lookup{inputID: inputID, producer: producer, output: output, cands: cands})
	}

	// Anchor: the deepest single candidate across all inputs, breaking
	// ties by mtime. "Deepest" = most path separators in dir.
	var anchorPath string
	var anchorDepth int
	var anchorMtime int64
	for _, l := range lookups {
		for _, c := range l.cands {
			d := strings.Count(filepath.Dir(c.path), string(filepath.Separator))
			if d > anchorDepth || (d == anchorDepth && c.mtime > anchorMtime) {
				anchorDepth = d
				anchorMtime = c.mtime
				anchorPath = c.path
			}
		}
	}
	anchorDir := filepath.Dir(anchorPath)

	res := &Resolved{Inputs: make(map[string]string, len(stage.Inputs))}
	for _, l := range lookups {
		// Constrain to candidates whose dir is an ancestor of (or equal
		// to) anchorDir. The anchor itself trivially satisfies this.
		var pick string
		var pickMtime int64
		for _, c := range l.cands {
			cd := filepath.Dir(c.path)
			if !isAncestorOrEqual(cd, anchorDir) {
				continue
			}
			if pick == "" || c.mtime > pickMtime ||
				(c.mtime == pickMtime && c.path > pick) {
				pick = c.path
				pickMtime = c.mtime
			}
		}
		if pick == "" {
			return nil, fmt.Errorf("input %q has no candidate consistent with upstream chain anchored at %s",
				l.inputID, anchorDir)
		}
		res.Inputs[l.inputID] = pick
		if res.Dataset == "" {
			res.Dataset = extractDataset(filepath.Base(pick), l.output.Path)
		}
	}
	return res, nil
}

// isAncestorOrEqual reports whether candidate is an ancestor of (or
// equal to) anchor, in path-component terms. Both must be absolute.
func isAncestorOrEqual(candidate, anchor string) bool {
	if candidate == anchor {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(anchor+sep, candidate+sep)
}

type candidate struct {
	path  string
	mtime int64
}

// collectProducerOutputs returns ALL files in outRoot whose basename
// matches pathGlob (with {dataset} -> "*") and whose path includes
// "/<stageID>/" and "/<some moduleID>/".
func collectProducerOutputs(outRoot, stageID string, moduleIDs []string, pathGlob string) ([]candidate, error) {
	if _, err := os.Stat(outRoot); err != nil {
		return nil, fmt.Errorf("out dir %s missing: %w (no successful runs yet?)", outRoot, err)
	}
	wildGlob, _ := globWithDatasetWildcard(pathGlob)
	stageMarker := string(filepath.Separator) + stageID + string(filepath.Separator)
	moduleMarkers := make([]string, len(moduleIDs))
	for i, m := range moduleIDs {
		moduleMarkers[i] = string(filepath.Separator) + m + string(filepath.Separator)
	}
	var out []candidate
	err := filepath.WalkDir(outRoot, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.Contains(p, stageMarker) {
			return nil
		}
		hasModule := false
		for _, mm := range moduleMarkers {
			if strings.Contains(p, mm) {
				hasModule = true
				break
			}
		}
		if !hasModule {
			return nil
		}
		ok, err := filepath.Match(wildGlob, filepath.Base(p))
		if err != nil || !ok {
			return nil
		}
		fi, err := d.Info()
		if err != nil || fi.Size() == 0 {
			return nil
		}
		out = append(out, candidate{path: p, mtime: fi.ModTime().UnixNano()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// globWithDatasetWildcard returns (glob with {dataset} replaced by "*",
// index of the {dataset} placeholder in the original path or -1).
func globWithDatasetWildcard(pathGlob string) (string, int) {
	const tok = "{dataset}"
	idx := strings.Index(pathGlob, tok)
	if idx < 0 {
		return pathGlob, -1
	}
	return strings.Replace(pathGlob, tok, "*", -1), idx
}

// extractDataset finds the substring of basename that matches the
// {dataset} placeholder in the original glob. e.g. glob
// "{dataset}_normalized.h5" + basename "datasets_normalized.h5"
// -> "datasets".
func extractDataset(basename, pathGlob string) string {
	const tok = "{dataset}"
	idx := strings.Index(pathGlob, tok)
	if idx < 0 {
		return ""
	}
	prefix := pathGlob[:idx]
	suffix := pathGlob[idx+len(tok):]
	if !strings.HasPrefix(basename, prefix) || !strings.HasSuffix(basename, suffix) {
		return ""
	}
	return basename[len(prefix) : len(basename)-len(suffix)]
}

// EnvVarName converts an input/output id like "normalized.h5" or
// "filtered.cellids" into a shell-safe upper-case env var name:
// "NORMALIZED_H5", "FILTERED_CELLIDS".
func EnvVarName(id string) string {
	r := strings.NewReplacer(".", "_", "-", "_")
	return strings.ToUpper(r.Replace(id))
}
