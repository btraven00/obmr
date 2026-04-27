package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/btraven00/obmr/internal/config"
	"github.com/spf13/cobra"
)

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <bench.yaml>",
		Short: "Set the default benchmark plan in ./.obmr/config.yaml",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			plan, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			if _, err := os.Stat(plan); err != nil {
				return fmt.Errorf("plan not found: %s", plan)
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			c := &config.Config{Default: config.Default{Plan: plan}}
			out, err := config.Save(cwd, c)
			if err != nil {
				return err
			}
			fmt.Printf("wrote %s (default plan = %s)\n", out, plan)
			return nil
		},
	}
}
