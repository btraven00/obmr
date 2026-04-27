package cmd

import (
	"fmt"

	"github.com/btraven00/obmr/internal/workspace"
	"github.com/spf13/cobra"
)

func newPromoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "promote [bench.yaml]",
		Short: "Copy local YAML edits back into canonical (urls/commits restored)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			plan, err := resolvePlan(firstArg(args))
			if err != nil {
				return err
			}
			lock, _, err := loadLock(plan)
			if err != nil {
				return err
			}
			if err := workspace.Promote(plan, lock); err != nil {
				return err
			}
			fmt.Printf("promoted local edits into %s (review with `git diff`)\n", plan)
			return nil
		},
	}
}
