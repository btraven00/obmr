package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/btraven00/obmr/internal/benchmark"
	"github.com/btraven00/obmr/internal/config"
	"github.com/btraven00/obmr/internal/runner"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var prod bool
	c := &cobra.Command{
		Use:   "run [-- snakemake-args...]",
		Short: "Invoke `ob run` (uv by default; pixi when software_backend is conda)",
		Long: `Runs the configured omnibenchmark.

By default (dev mode), passes --dirty to ob so it uses your local clones.
Use --prod to run without --dirty (upstream-pinned).

If the benchmark's top-level software_backend is "conda", obmr generates
a pixi manifest at .obmr/pixi.toml (with python + conda + omnibenchmark)
and runs via ` + "`pixi run`" + `. Otherwise it runs via ` + "`uv tool run`" + `.

The omnibenchmark version is resolved from config (priority: pr > branch
> version > latest pypi). See ` + "`obmr config`" + `.

Extra arguments after -- are passed through to snakemake (via ob run).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := resolvePlan("")
			if err != nil {
				return err
			}
			yamlPath := plan
			if !prod {
				local := localYAMLPathFromCanonical(plan)
				if _, err := os.Stat(local); err != nil {
					return fmt.Errorf("%s not found (run `obmr dev` first, or pass --prod)", local)
				}
				yamlPath = local
			}
			var passThrough []string
			passIdx := cmd.Flags().ArgsLenAtDash()
			if passIdx >= 0 {
				passThrough = args[passIdx:]
			} else {
				passThrough = args
			}
			subArgs := []string{"run", yamlPath}
			if !prod {
				subArgs = append(subArgs, "--dirty")
			}
			if len(passThrough) > 0 {
				subArgs = append(subArgs, "--")
				subArgs = append(subArgs, passThrough...)
			}
			return dispatchOb(plan, subArgs)
		},
	}
	c.Flags().BoolVar(&prod, "prod", false, "run without --dirty (upstream-pinned mode)")
	return c
}

// dispatchOb runs `ob <subArgs...>` via uv (default) or pixi (when the
// plan's software_backend is "conda"), using the configured omnibenchmark
// spec from .obmr/config.yaml.
func dispatchOb(plan string, subArgs []string) error {
	cwd, _ := os.Getwd()
	cp := config.Find(cwd)
	var cfg *config.Config
	workspaceDir := cwd
	if cp != "" {
		cfg, _ = config.Load(cp)
		workspaceDir = filepath.Dir(filepath.Dir(cp))
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	useConda := false
	if f, err := benchmark.Load(plan); err == nil && strings.EqualFold(f.SoftwareBackend, "conda") {
		useConda = true
	}
	printOmniBanner(cfg.Omnibenchmark, useConda)
	if useConda {
		return runPixi(workspaceDir, cfg.Omnibenchmark, subArgs)
	}
	return runUv(cfg.Omnibenchmark, subArgs)
}

func runUv(omni config.Omnibenchmark, subArgs []string) error {
	if err := requireTool("uv", "https://docs.astral.sh/uv/", "curl -LsSf https://astral.sh/uv/install.sh | sh"); err != nil {
		return err
	}
	uvArgs := append([]string{"tool", "run", "--from", omniSpec(omni), "ob"}, subArgs...)
	fmt.Fprintf(os.Stderr, "+ uv %s\n", strings.Join(uvArgs, " "))
	ex := exec.Command("uv", uvArgs...)
	ex.Stdin = os.Stdin
	ex.Stdout = os.Stdout
	ex.Stderr = os.Stderr
	return ex.Run()
}

func runPixi(workspaceDir string, omni config.Omnibenchmark, subArgs []string) error {
	if err := requireTool("pixi", "https://pixi.sh", "curl -fsSL https://pixi.sh/install.sh | sh"); err != nil {
		return err
	}
	manifest, changed, err := runner.EnsurePixiManifest(workspaceDir, omni)
	if err != nil {
		return fmt.Errorf("write pixi manifest: %w", err)
	}
	switch {
	case omni.PR != 0 || omni.Branch != "":
		// Mutable git ref: force a re-resolve so new commits land.
		fmt.Fprintf(os.Stderr, "+ pixi update --manifest-path %s omnibenchmark\n", manifest)
		up := exec.Command("pixi", "update", "--manifest-path", manifest, "omnibenchmark")
		up.Stdout = os.Stdout
		up.Stderr = os.Stderr
		if err := up.Run(); err != nil {
			return fmt.Errorf("pixi update: %w", err)
		}
	case changed:
		fmt.Fprintf(os.Stderr, "+ pixi install --manifest-path %s\n", manifest)
		install := exec.Command("pixi", "install", "--manifest-path", manifest)
		install.Stdout = os.Stdout
		install.Stderr = os.Stderr
		if err := install.Run(); err != nil {
			return fmt.Errorf("pixi install: %w", err)
		}
	}
	pxArgs := append([]string{"run", "--manifest-path", manifest, "ob"}, subArgs...)
	fmt.Fprintf(os.Stderr, "+ pixi %s\n", strings.Join(pxArgs, " "))
	ex := exec.Command("pixi", pxArgs...)
	ex.Stdin = os.Stdin
	ex.Stdout = os.Stdout
	ex.Stderr = os.Stderr
	return ex.Run()
}

// printOmniBanner prints a one-line banner identifying which omnibenchmark
// build is about to run, plus a hint on how to revert any non-default
// override.
func printOmniBanner(o config.Omnibenchmark, conda bool) {
	runner := "uv"
	if conda {
		runner = "pixi"
	}
	var src, unsetKey string
	switch {
	case o.PR != 0:
		src = paint(fmt.Sprintf("PR #%d", o.PR), ansiYellow+ansiBold)
		unsetKey = "omnibenchmark.pr"
	case o.Branch != "":
		src = paint("branch "+o.Branch, ansiYellow+ansiBold)
		unsetKey = "omnibenchmark.branch"
	case o.Version != "":
		src = paint("v"+o.Version, ansiCyan)
		unsetKey = "omnibenchmark.version"
	default:
		src = paint("pypi (latest)", ansiDim)
	}
	fmt.Fprintf(os.Stderr, "%s omnibenchmark %s via %s\n",
		paint("==>", ansiGreen+ansiBold), src, paint(runner, ansiBlue+ansiBold))
	if unsetKey != "" {
		fmt.Fprintf(os.Stderr, "    %s revert with `%s`\n",
			paint("hint:", ansiDim),
			paint("obmr config --unset "+unsetKey, ansiBold))
	}
}

// requireTool returns a friendly error if name is not on PATH, including a
// copy-pastable install command.
func requireTool(name, homepage, installCmd string) error {
	if _, err := exec.LookPath(name); err == nil {
		return nil
	}
	return fmt.Errorf("`%s` not found on PATH (see %s)\nInstall: %s\nThen re-run.", name, homepage, installCmd)
}

// omniSpec returns the `--from` argument for `uv tool run`.
// Priority: pr > branch > version > "omnibenchmark" (latest pypi).
func omniSpec(o config.Omnibenchmark) string {
	if o.PR != 0 {
		return fmt.Sprintf("git+%s@refs/pull/%d/head", config.UpstreamRepo, o.PR)
	}
	if o.Branch != "" {
		return fmt.Sprintf("git+%s@%s", config.UpstreamRepo, o.Branch)
	}
	if o.Version != "" {
		return "omnibenchmark==" + o.Version
	}
	return "omnibenchmark"
}
