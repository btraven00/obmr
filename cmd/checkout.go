package cmd

import (
	"fmt"

	"github.com/btraven00/obmr/internal/workspace"
	"github.com/spf13/cobra"
)

func newCheckoutCmd() *cobra.Command {
	var create bool
	c := &cobra.Command{
		Use:   "checkout <branch> [bench.yaml]",
		Short: "Checkout (or create) the same branch in every module",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			branch := args[0]
			var arg string
			if len(args) == 2 {
				arg = args[1]
			}
			plan, err := resolvePlan(arg)
			if err != nil {
				return err
			}
			lock, benchDir, err := loadLock(plan)
			if err != nil {
				return err
			}
			results := workspace.Fanout(benchDir, lock, func(dir string, _ workspace.LockedModule) (string, error) {
				gitArgs := []string{"checkout"}
				if create {
					gitArgs = append(gitArgs, "-B")
				}
				gitArgs = append(gitArgs, branch)
				return workspace.Git(dir, gitArgs...)
			})
			anyErr := false
			for _, r := range results {
				if r.Err != nil {
					fmt.Printf("%-20s ERROR: %v\n", r.Module.ID, r.Err)
					anyErr = true
					continue
				}
				fmt.Printf("%-20s ok (%s)\n", r.Module.ID, branch)
			}
			if anyErr {
				return fmt.Errorf("one or more modules failed")
			}
			return nil
		},
	}
	c.Flags().BoolVarP(&create, "create", "b", false, "create the branch if it doesn't exist")
	return c
}
