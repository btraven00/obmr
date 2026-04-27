package cmd

import (
	"fmt"

	"github.com/btraven00/obmr/internal/benchmark"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [bench.yaml]",
		Short: "List modules declared in a benchmark YAML",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			plan, err := resolvePlan(firstArg(args))
			if err != nil {
				return err
			}
			f, err := benchmark.Load(plan)
			if err != nil {
				return err
			}
			for _, s := range f.Stages {
				for _, m := range s.Modules {
					fmt.Printf("%s\t%s\t%s\t%s\n", s.ID, m.ID, m.Repository.URL, m.Repository.Commit)
				}
			}
			return nil
		},
	}
}
