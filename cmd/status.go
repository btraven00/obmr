package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/btraven00/obflow/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)


func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [bench.yaml]",
		Short: "Per-module branch and dirty state",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			plan, err := resolvePlan(firstArg(args))
			if err != nil {
				return err
			}
			lock, benchDir, err := loadLock(plan)
			if err != nil {
				return err
			}
			mode := detectMode(plan)
			modeColor := ansiYellow
			if mode == "dev" {
				modeColor = ansiGreen
			}
			displayPlan := plan
			if cwd, err := os.Getwd(); err == nil {
				if rel, err := filepath.Rel(cwd, plan); err == nil && !strings.HasPrefix(rel, "../../..") {
					displayPlan = rel
				}
			}
			planStatus := paint("ok", ansiGreen)
			if msg := yamlParseError(plan); msg != "" {
				planStatus = paint("parse error: "+msg, ansiRed+ansiBold)
			}
			fmt.Printf("plan:  %s  (%s)\n", displayPlan, planStatus)
			fmt.Printf("mode:  %s\n", paint(mode, modeColor+ansiBold))
			if mode == "dev" {
				d, err := workspace.DiffLocal(plan, lock)
				switch {
				case err != nil:
					fmt.Printf("local: %s %v\n", paint("parse error:", ansiRed+ansiBold), err)
				case d == nil || (!d.Diverged && !d.Missing):
					fmt.Printf("local: %s\n", paint("clean", ansiGreen))
				case d.Diverged:
					fmt.Printf("local: %s  %s\n", paint("edited", ansiMagenta+ansiBold), paint(d.Summary(), ansiDim))
					printDiffDetails(d)
					fmt.Printf("       %s use `%s` to apply edits to the canonical plan\n",
						paint("hint:", ansiDim), paint("obflow plan promote", ansiBold))
				}
			}
			results := workspace.Fanout(benchDir, lock, func(dir string, _ workspace.LockedModule) (string, error) {
				branch, err := workspace.Git(dir, "rev-parse", "--abbrev-ref", "HEAD")
				if err != nil {
					return "", err
				}
				dirty, err := workspace.Git(dir, "status", "--porcelain")
				if err != nil {
					return "", err
				}
				if strings.TrimSpace(dirty) != "" {
					return fmt.Sprintf("%s\t%s", paint(branch, ansiCyan), paint("DIRTY", ansiRed+ansiBold)), nil
				}
				return fmt.Sprintf("%s\t%s", paint(branch, ansiCyan), paint("clean", ansiGreen)), nil
			})
			for _, r := range results {
				if r.Err != nil {
					fmt.Printf("%-20s %s %v\n", r.Module.ID, paint("ERROR:", ansiRed+ansiBold), r.Err)
					continue
				}
				fmt.Printf("%-20s %s\n", r.Module.ID, r.Out)
			}
			if mode == "dev" {
				extras, err := workspace.LocalOnlyModules(plan, lock)
				if err != nil {
					fmt.Printf("%s %v\n", paint("local-only: error:", ansiRed+ansiBold), err)
				} else if len(extras) > 0 {
					fmt.Printf("%s\n", paint("local-only:", ansiDim))
					for _, e := range extras {
						id := e.Stage + "/" + e.ID
						branch, gerr := workspace.Git(e.AbsDir, "rev-parse", "--abbrev-ref", "HEAD")
						if gerr != nil {
							fmt.Printf("%-20s %s %v\n", id, paint("ERROR:", ansiRed+ansiBold), gerr)
							continue
						}
						dirty, derr := workspace.Git(e.AbsDir, "status", "--porcelain")
						if derr != nil {
							fmt.Printf("%-20s %s %v\n", id, paint("ERROR:", ansiRed+ansiBold), derr)
							continue
						}
						state := paint("clean", ansiGreen)
						if strings.TrimSpace(dirty) != "" {
							state = paint("DIRTY", ansiRed+ansiBold)
						}
						fmt.Printf("%-20s %s\t%s\n", id, paint(branch, ansiCyan), state)
					}
				}
			}
			return nil
		},
	}
}

// yamlParseError returns "" if path parses as YAML, else the error message.
func yamlParseError(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return err.Error()
	}
	var n yaml.Node
	if err := yaml.Unmarshal(b, &n); err != nil {
		return err.Error()
	}
	return ""
}

func printDiffDetails(d *workspace.LocalDiff) {
	prefix := "       "
	if len(d.AddedStages) > 0 {
		fmt.Printf("%s%s %s\n", prefix, paint("+stages", ansiGreen), strings.Join(d.AddedStages, ", "))
	}
	if len(d.RemovedStages) > 0 {
		fmt.Printf("%s%s %s\n", prefix, paint("-stages", ansiRed), strings.Join(d.RemovedStages, ", "))
	}
	if len(d.ModifiedStages) > 0 {
		fmt.Printf("%s%s %s\n", prefix, paint("~stages", ansiYellow), strings.Join(d.ModifiedStages, ", "))
	}
	if len(d.AddedModules) > 0 {
		fmt.Printf("%s%s %s\n", prefix, paint("+modules", ansiGreen), strings.Join(d.AddedModules, ", "))
	}
	if len(d.RemovedModules) > 0 {
		fmt.Printf("%s%s %s\n", prefix, paint("-modules", ansiRed), strings.Join(d.RemovedModules, ", "))
	}
	if len(d.ModifiedModules) > 0 {
		fmt.Printf("%s%s %s\n", prefix, paint("~modules", ansiYellow), strings.Join(d.ModifiedModules, ", "))
	}
}

// detectMode reports "local" if a sibling <bench>.local.yaml exists next
// to the canonical, otherwise "canonical".
func detectMode(plan string) string {
	dir := filepath.Dir(plan)
	base := filepath.Base(plan)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	local := filepath.Join(dir, name+".local"+ext)
	if _, err := os.Stat(local); err == nil {
		return "dev"
	}
	return "prod"
}
