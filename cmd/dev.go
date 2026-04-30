package cmd

import (
	"fmt"

	"github.com/btraven00/obflow/internal/workspace"
	"github.com/spf13/cobra"
)

func newDevCmd() *cobra.Command {
	var fork bool
	c := &cobra.Command{
		Use:   "dev [bench.yaml]",
		Short: "Switch to dev mode: write <bench>.local.yaml; with --fork also ensure a 'fork' remote per module",
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
			out, err := workspace.WriteLocalYAML(plan, lock)
			if err != nil {
				return err
			}
			fmt.Printf("wrote %s\n", out)

			// Switch every module to its origin default branch.
			swResults := workspace.Fanout(benchDir, lock, func(dir string, _ workspace.LockedModule) (string, error) {
				return workspace.EnsureOnDefault(dir)
			})
			for _, r := range swResults {
				if r.Err != nil {
					fmt.Printf("%-20s ERROR: %v\n", r.Module.ID, r.Err)
					continue
				}
				fmt.Printf("%-20s %s\n", r.Module.ID, r.Out)
			}

			if fork {
				results := workspace.Fanout(benchDir, lock, func(dir string, _ workspace.LockedModule) (string, error) {
					return workspace.EnsureForkRemote(dir)
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
					return fmt.Errorf("one or more modules failed to set up fork remote")
				}
			}
			return nil
		},
	}
	c.Flags().BoolVar(&fork, "fork", false, "ensure each module has a 'fork' remote (creates via `gh repo fork`)")
	return c
}
