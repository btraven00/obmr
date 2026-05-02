package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/btraven00/obflow/internal/config"
	"github.com/spf13/cobra"
)

func NewUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <bench.yaml>",
		Short: "Set the default benchmark plan in ./.obflow/config.yaml",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			abs, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			if _, err := os.Stat(abs); err != nil {
				return fmt.Errorf("plan not found: %s", abs)
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(cwd, abs)
			if err != nil {
				rel = abs
			}
			c := &config.Config{Default: config.Default{Plan: rel}}
			out, err := config.Save(cwd, c)
			if err != nil {
				return err
			}
			fmt.Printf("wrote %s (default plan = %s)\n", out, rel)
			return nil
		},
	}
}
