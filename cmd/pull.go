package cmd

import (
	"fmt"

	"github.com/btraven00/obmr/internal/workspace"
	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull [bench.yaml]",
		Short: "git pull --ff-only in every module on its current branch",
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
				return workspace.Git(dir, "pull", "--ff-only")
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
}
