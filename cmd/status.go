package cmd

import (
	"fmt"
	"strings"

	"github.com/btraven00/obmr/internal/workspace"
	"github.com/spf13/cobra"
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
			results := workspace.Fanout(benchDir, lock, func(dir string, _ workspace.LockedModule) (string, error) {
				branch, err := workspace.Git(dir, "rev-parse", "--abbrev-ref", "HEAD")
				if err != nil {
					return "", err
				}
				dirty, err := workspace.Git(dir, "status", "--porcelain")
				if err != nil {
					return "", err
				}
				flag := "clean"
				if strings.TrimSpace(dirty) != "" {
					flag = "DIRTY"
				}
				return fmt.Sprintf("%s\t%s", branch, flag), nil
			})
			for _, r := range results {
				if r.Err != nil {
					fmt.Printf("%-20s ERROR: %v\n", r.Module.ID, r.Err)
					continue
				}
				fmt.Printf("%-20s %s\n", r.Module.ID, r.Out)
			}
			return nil
		},
	}
}
