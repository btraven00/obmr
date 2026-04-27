package cmd

import (
	"fmt"

	"github.com/btraven00/obmr/internal/workspace"
	"github.com/spf13/cobra"
)

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push [bench.yaml]",
		Short: "Push current branch to 'fork' if present else 'origin' per module; skip clean modules",
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
				return workspace.PushModule(dir)
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
