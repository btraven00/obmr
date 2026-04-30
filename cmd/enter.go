package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/btraven00/obflow/internal/benchmark"
	"github.com/btraven00/obflow/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newEnterCmd() *cobra.Command {
	var fromOutput string
	var outDir string
	var pixiEnv string
	var printOnly bool
	c := &cobra.Command{
		Use:   "enter <module-id>",
		Short: "Drop into a pixi shell for a module with upstream inputs preloaded as env vars",
		Long: `Resolve a module's upstream inputs from the latest successful run in
out/, export them as environment variables (one per declared input id),
and exec ` + "`pixi shell`" + ` inside that module's pixi env.

Inputs are discovered from the benchmark plan: for each id in the
target stage's ` + "`inputs:`" + `, obflow finds the producing stage
(via its ` + "`outputs:`" + `) and walks ` + "`out/`" + ` for the
latest file matching the producer's declared ` + "`path:`" + ` glob.
No file patterns are hardcoded in obflow.

Inside the spawned shell:
  $NORMALIZED_H5, $SELECTED_GENES, …  one per declared input id
  $OBMR_NAME                          dataset wildcard value
  $OBMR_OUTPUT_DIR                    ephemeral output dir
  $OBMR_MODULE, $OBMR_MODULE_DIR, $OBMR_ENTRYPOINT
`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			moduleID := args[0]
			plan, err := resolvePlan("")
			if err != nil {
				return err
			}
			if detectMode(plan) != "dev" {
				return fmt.Errorf("`obflow enter` only operates in dev mode (no %s found); run `obflow dev` first",
					filepath.Base(localYAMLPathFromCanonical(plan)))
			}
			localPlan := localYAMLPathFromCanonical(plan)
			f, err := benchmark.Load(localPlan)
			if err != nil {
				return err
			}
			benchDir, err := filepath.Abs(filepath.Dir(localPlan))
			if err != nil {
				return err
			}

			stage, mod, ok := f.FindModule(moduleID)
			if !ok {
				return fmt.Errorf("module %q not found in plan (try `obflow list`)", moduleID)
			}

			modulePath := mod.Repository.URL
			if !filepath.IsAbs(modulePath) {
				modulePath = filepath.Join(benchDir, modulePath)
			}
			pixiManifest := filepath.Join(modulePath, "pixi.toml")
			if _, err := os.Stat(pixiManifest); err != nil {
				return fmt.Errorf("module %q has no pixi.toml at %s: %w", moduleID, pixiManifest, err)
			}

			outRoot := filepath.Join(benchDir, "out")
			var resolved *workspace.Resolved
			if fromOutput != "" {
				resolved, err = resolveFromExplicit(f, stage, fromOutput)
			} else {
				resolved, err = workspace.ResolveInputs(f, moduleID, outRoot)
			}
			if err != nil {
				return err
			}

			// Ephemeral output dir.
			if outDir == "" {
				outDir, err = os.MkdirTemp("", fmt.Sprintf("obflow-enter-%s-", moduleID))
				if err != nil {
					return err
				}
			} else {
				if err := os.MkdirAll(outDir, 0755); err != nil {
					return err
				}
			}

			env := os.Environ()
			env = setEnv(env, "OBMR_MODULE", moduleID)
			env = setEnv(env, "OBMR_MODULE_DIR", modulePath)
			if mod.Repository.Entrypoint != "" {
				env = setEnv(env, "OBMR_ENTRYPOINT", mod.Repository.Entrypoint)
			}
			env = setEnv(env, "OBMR_OUTPUT_DIR", outDir)
			if resolved.Dataset != "" {
				env = setEnv(env, "OBMR_NAME", resolved.Dataset)
			}
			// Stable iteration order for the banner and env.
			inputIDs := make([]string, 0, len(resolved.Inputs))
			for k := range resolved.Inputs {
				inputIDs = append(inputIDs, k)
			}
			sort.Strings(inputIDs)
			for _, id := range inputIDs {
				env = setEnv(env, workspace.EnvVarName(id), resolved.Inputs[id])
			}

			// Build a `runit` script that echoes-then-runs the module's
			// entrypoint with all required inputs already substituted;
			// extra args passed to runit are appended (e.g.
			// `runit --pca_type scanpy_arpack --n_components 50`).
			scriptsDir, runitPath, err := writeRunit(moduleID, modulePath, mod, inputIDs)
			if err != nil {
				return fmt.Errorf("write runit script: %w", err)
			}
			env = prependPath(env, scriptsDir)
			env = setEnv(env, "OBMR_RUNIT", runitPath)

			if printOnly {
				printEnterBanner(moduleID, mod, modulePath, resolved, inputIDs, outDir, runitPath)
				printExports(os.Stdout, env, inputIDs, resolved.Dataset != "")
				fmt.Fprintln(os.Stdout, "export PATH='"+scriptsDir+":'\"$PATH\"")
				return nil
			}

			printEnterBanner(moduleID, mod, modulePath, resolved, inputIDs, outDir, runitPath)

			pixiArgs := []string{"shell", "--manifest-path", pixiManifest}
			if pixiEnv != "" {
				pixiArgs = append(pixiArgs, "-e", pixiEnv)
			}
			ex := exec.Command("pixi", pixiArgs...)
			ex.Stdin = os.Stdin
			ex.Stdout = os.Stdout
			ex.Stderr = os.Stderr
			ex.Env = env
			ex.Dir = modulePath
			if err := ex.Run(); err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
						os.Exit(ws.ExitStatus())
					}
				}
				return err
			}
			return nil
		},
	}
	c.Flags().StringVar(&fromOutput, "from", "", "use an explicit upstream output directory instead of resolving the latest run")
	c.Flags().StringVar(&outDir, "out", "", "use this directory for OBMR_OUTPUT_DIR (default: a fresh /tmp dir)")
	c.Flags().StringVar(&pixiEnv, "env", "", "pixi environment to enter (forwarded as `pixi shell -e <env>`)")
	c.Flags().BoolVar(&printOnly, "print", false, "print resolved exports to stdout instead of exec'ing pixi shell (eval-friendly)")
	return c
}

// printExports writes `export KEY=VALUE` lines for the OBMR_* and
// input env vars. Quotes values with single quotes, escaping inner '.
func printExports(w *os.File, env, inputIDs []string, hasName bool) {
	wanted := map[string]bool{
		"OBMR_MODULE":     true,
		"OBMR_MODULE_DIR": true,
		"OBMR_ENTRYPOINT": true,
		"OBMR_OUTPUT_DIR": true,
	}
	if hasName {
		wanted["OBMR_NAME"] = true
	}
	for _, id := range inputIDs {
		wanted[workspace.EnvVarName(id)] = true
	}
	for _, e := range env {
		eq := strings.IndexByte(e, '=')
		if eq < 0 {
			continue
		}
		k := e[:eq]
		if !wanted[k] {
			continue
		}
		v := strings.ReplaceAll(e[eq+1:], `'`, `'\''`)
		fmt.Fprintf(w, "export %s='%s'\n", k, v)
	}
}

// resolveFromExplicit treats the user-supplied directory as the
// canonical producer output dir and pulls one file per declared input
// of the target stage from it (matching the producer Output.Path glob,
// with {dataset} -> "*"). Useful when auto-resolution is ambiguous.
func resolveFromExplicit(f *benchmark.File, target benchmark.Stage, fromOutput string) (*workspace.Resolved, error) {
	abs, err := filepath.Abs(fromOutput)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, fmt.Errorf("--from %s: %w", abs, err)
	}
	res := &workspace.Resolved{Inputs: map[string]string{}}
	for _, inputID := range target.Inputs {
		_, output, ok := f.ProducerOf(inputID)
		if !ok {
			return nil, fmt.Errorf("input %q has no producing stage in plan", inputID)
		}
		const tok = "{dataset}"
		glob := strings.Replace(output.Path, tok, "*", -1)
		matches, err := filepath.Glob(filepath.Join(abs, glob))
		if err != nil || len(matches) == 0 {
			return nil, fmt.Errorf("no file matching %q in %s", glob, abs)
		}
		res.Inputs[inputID] = matches[0]
		if res.Dataset == "" && strings.Contains(output.Path, tok) {
			res.Dataset = strings.TrimSuffix(strings.TrimPrefix(filepath.Base(matches[0]),
				output.Path[:strings.Index(output.Path, tok)]),
				output.Path[strings.Index(output.Path, tok)+len(tok):])
		}
	}
	return res, nil
}

func setEnv(env []string, key, val string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + val
			return env
		}
	}
	return append(env, prefix+val)
}

func printEnterBanner(moduleID string, mod benchmark.Module, modulePath string,
	resolved *workspace.Resolved, inputIDs []string, outDir, runitPath string) {
	fmt.Fprintf(os.Stderr, "%s obflow enter %s\n",
		paint("==>", ansiGreen+ansiBold), paint(moduleID, ansiBold))
	if mod.Repository.Entrypoint != "" {
		fmt.Fprintf(os.Stderr, "    %s   %s/%s @ %s\n",
			paint("module:", ansiDim), filepath.Base(modulePath),
			mod.Repository.Entrypoint, shortSHA(mod.Repository.Commit))
	} else {
		fmt.Fprintf(os.Stderr, "    %s   %s @ %s\n",
			paint("module:", ansiDim), filepath.Base(modulePath), shortSHA(mod.Repository.Commit))
	}
	if len(inputIDs) > 0 {
		first := true
		for _, id := range inputIDs {
			label := paint("inputs:", ansiDim)
			if !first {
				label = "        "
			}
			fmt.Fprintf(os.Stderr, "    %s   %s=%s\n", label,
				paint(workspace.EnvVarName(id), ansiCyan), resolved.Inputs[id])
			first = false
		}
	}
	fmt.Fprintf(os.Stderr, "    %s   %s=%s (ephemeral)\n",
		paint("output:", ansiDim), paint("OBMR_OUTPUT_DIR", ansiCyan), outDir)
	if resolved.Dataset != "" {
		fmt.Fprintf(os.Stderr, "    %s     %s=%s\n",
			paint("name:", ansiDim), paint("OBMR_NAME", ansiCyan), resolved.Dataset)
	}
	if runitPath != "" {
		fmt.Fprintf(os.Stderr, "    %s    %s in PATH (echoes & runs the module; pass extra flags after it)\n",
			paint("runit:", ansiDim), paint("runit", ansiCyan+ansiBold))
	}
	fmt.Fprintln(os.Stderr)
}

// firstParamSet returns flag/value pairs from the FIRST entry of the
// module's plan-declared `parameters:` list. For multi-value entries
// (cartesian expansions like `selection_type: [a, b]`), it picks the
// first value. Returns ordered (key, value) pairs as a flat alternating
// slice for stable arg ordering. Empty if no parameters.
func firstParamSet(params []map[string]interface{}) []string {
	if len(params) == 0 {
		return nil
	}
	first := params[0]
	keys := make([]string, 0, len(first))
	for k := range first {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, 2*len(keys))
	for _, k := range keys {
		v := first[k]
		if list, ok := v.([]interface{}); ok {
			if len(list) == 0 {
				continue
			}
			v = list[0]
		}
		out = append(out, "--"+k, fmt.Sprintf("%v", v))
	}
	return out
}

// writeRunit creates a temp dir containing an executable `runit` shell
// script. The script echoes the resolved invocation to stderr (with a
// `+` prefix, like `set -x`) and then execs it, forwarding any extra
// args the user passes. The dir is meant to be prepended to PATH.
func writeRunit(moduleID, modulePath string, mod benchmark.Module, inputIDs []string) (string, string, error) {
	scriptsDir, err := os.MkdirTemp("", fmt.Sprintf("obflow-bin-%s-", moduleID))
	if err != nil {
		return "", "", err
	}
	entrypointFile, err := resolveEntrypointFile(modulePath, mod.Repository.Entrypoint)
	if err != nil {
		return "", "", err
	}
	interp := interpreterFor(entrypointFile)

	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("# generated by `obflow enter`; echoes then runs the module's entrypoint.\n")
	b.WriteString("set -e\n")
	b.WriteString("cd \"$OBMR_MODULE_DIR\"\n")
	b.WriteString("cmd=(")
	if interp != "" {
		b.WriteString(shellQuote(interp))
		b.WriteString(" ")
	}
	b.WriteString(shellQuote(entrypointFile))
	b.WriteString(" --output_dir \"$OBMR_OUTPUT_DIR\" --name \"$OBMR_NAME\"")
	for _, id := range inputIDs {
		flag := "--" + id
		envName := workspace.EnvVarName(id)
		b.WriteString(fmt.Sprintf(" %s \"$%s\"", flag, envName))
	}
	// Plan-declared parameters from the first param set act as defaults.
	// argparse takes the LAST occurrence, so user-supplied "$@" overrides.
	for _, tok := range firstParamSet(mod.Parameters) {
		b.WriteString(" ")
		b.WriteString(shellQuote(tok))
	}
	b.WriteString(" \"$@\")\n")
	b.WriteString(`printf '+ '; printf '%q ' "${cmd[@]}"; printf '\n' >&2`)
	b.WriteString("\n")
	b.WriteString(`exec "${cmd[@]}"`)
	b.WriteString("\n")

	runitPath := filepath.Join(scriptsDir, "runit")
	if err := os.WriteFile(runitPath, []byte(b.String()), 0755); err != nil {
		return "", "", err
	}
	return scriptsDir, runitPath, nil
}

// resolveEntrypointFile reads <modulePath>/omnibenchmark.yaml and looks
// up the entrypoint key (or "default") in its `entrypoints:` map.
// Returns the basename of the script (e.g. "pca.py").
func resolveEntrypointFile(modulePath, key string) (string, error) {
	if key == "" {
		key = "default"
	}
	manifestPath := filepath.Join(modulePath, "omnibenchmark.yaml")
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", manifestPath, err)
	}
	var doc struct {
		Entrypoints map[string]string `yaml:"entrypoints"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return "", fmt.Errorf("parse %s: %w", manifestPath, err)
	}
	file, ok := doc.Entrypoints[key]
	if !ok {
		return "", fmt.Errorf("entrypoint %q not declared in %s (have: %v)",
			key, manifestPath, mapKeys(doc.Entrypoints))
	}
	return file, nil
}

func mapKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// interpreterFor returns the interpreter to prefix in front of the
// entrypoint, picked by extension. Empty means execute directly.
func interpreterFor(file string) string {
	switch strings.ToLower(filepath.Ext(file)) {
	case ".py":
		return "python"
	case ".r":
		return "Rscript"
	default:
		return ""
	}
}

func prependPath(env []string, dir string) []string {
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			env[i] = "PATH=" + dir + ":" + e[len("PATH="):]
			return env
		}
	}
	return append(env, "PATH="+dir)
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func shortSHA(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
