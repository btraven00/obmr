package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/btraven00/obmr/internal/benchmark"
	"github.com/btraven00/obmr/internal/workspace"
	"github.com/spf13/cobra"
)

func newNewModuleCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "newmodule <name> [-- ob-args...]",
		Short: "Create a new module under the modules parent dir via `ob create module`",
		Long: `Wraps ` + "`ob create module <path>`" + `, placing the new module under the
parent directory recorded in .obmr.lock (typically ../<bench>-modules/).

Refuses if a module with the same id already exists in the plan YAML.

Extra arguments after -- are passed through to ob create module.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			plan, err := resolvePlan("")
			if err != nil {
				return err
			}
			lock, benchDir, err := loadLock(plan)
			if err != nil {
				return err
			}

			// Refuse if the name already exists as a module id in the plan.
			if f, err := benchmark.Load(plan); err == nil {
				for _, m := range f.Modules() {
					if m.ID == name {
						return fmt.Errorf("module %q already exists in %s (stage: search the YAML)", name, plan)
					}
				}
			}

			parent := lock.ParentDir
			if !filepath.IsAbs(parent) {
				parent = filepath.Join(benchDir, parent)
			}
			modulePath := filepath.Join(parent, name)
			if _, err := os.Stat(modulePath); err == nil {
				return fmt.Errorf("path already exists: %s", modulePath)
			}

			// Pass-through args after `--`.
			var passThrough []string
			passIdx := cmd.Flags().ArgsLenAtDash()
			if passIdx >= 0 {
				passThrough = args[passIdx:]
			} else if len(args) > 1 {
				passThrough = args[1:]
			}
			subArgs := append([]string{"create", "module", "--dirty"}, passThrough...)
			subArgs = append(subArgs, modulePath)

			return dispatchOb(plan, subArgs)
		},
	}
	return c
}

// silence unused import when workspace types are not referenced directly
var _ = workspace.LockedModule{}
