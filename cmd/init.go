package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/btraven00/obflow/internal/workspace"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var parent string
	c := &cobra.Command{
		Use:   "init [bench.yaml]",
		Short: "Clone all modules as siblings and write .obflow.lock",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			yamlPath, err := resolvePlan(firstArg(args))
			if err != nil {
				return err
			}
			p := parent
			if p == "" {
				p = defaultParent(yamlPath)
			}
			lock, err := workspace.Init(yamlPath, p)
			if err != nil {
				return err
			}
			lockPath := workspace.LockPath(yamlPath)
			if err := lock.Save(lockPath); err != nil {
				return err
			}
			fmt.Printf("wrote %s (%d modules, parent=%s)\n", lockPath, len(lock.Modules), p)
			return nil
		},
	}
	c.Flags().StringVar(&parent, "parent", "", "parent dir for sibling clones (default: ../<bench>-modules)")
	return c
}

// defaultParent derives "../<benchdir>-modules" from the YAML's directory.
func defaultParent(yamlPath string) string {
	abs, err := filepath.Abs(yamlPath)
	if err != nil {
		return "../bench-modules"
	}
	name := filepath.Base(filepath.Dir(abs))
	return "../" + name + "-modules"
}
