package cmd

import (
	"fmt"

	"github.com/btraven00/obflow/internal/workspace"
	"github.com/spf13/cobra"
)

func newTrimCmd() *cobra.Command {
	var branch string
	var force bool
	c := &cobra.Command{
		Use:   "trim [bench.yaml]",
		Short: "Delete merged local branches per module (skips dirty trees)",
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
			results := workspace.Fanout(benchDir, lock, func(dir string, _ workspace.LockedModule) (string, error) {
				return workspace.TrimModule(dir, branch, force)
			})
			anyErr := false
			for _, r := range results {
				if r.Err != nil {
					fmt.Printf("%-20s ERROR: %v\n", r.Module.ID, r.Err)
					anyErr = true
					continue
				}
				fmt.Printf("%-20s %s\n", r.Module.ID, r.Out)
			}
			if anyErr {
				return fmt.Errorf("one or more modules failed")
			}
			return nil
		},
	}
	c.Flags().StringVar(&branch, "branch", "", "delete only this branch (across modules)")
	c.Flags().BoolVar(&force, "force", false, "use `git branch -D` to delete unmerged branches too")
	return c
}
