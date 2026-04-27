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

			// Compute column widths from data.
			stageW, modW := 5, 6
			for _, s := range f.Stages {
				if len(s.ID) > stageW {
					stageW = len(s.ID)
				}
				for _, m := range s.Modules {
					if len(m.ID) > modW {
						modW = len(m.ID)
					}
				}
			}

			for _, s := range f.Stages {
				for _, m := range s.Modules {
					stage := paint(padRight(s.ID, stageW), ansiBold)
					mod := padRight(m.ID, modW)
					url := paint(m.Repository.URL, ansiBlue)
					commit := paint(shortHash(m.Repository.Commit), ansiYellow)
					fmt.Printf("%s  %s  %s  %s\n", stage, mod, commit, url)
				}
			}
			return nil
		},
	}
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + spaces(n-len(s))
}

func spaces(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}

// shortHash returns a 12-char prefix if the commit looks SHA-like, else the
// full string. Branch names and tags are returned unchanged.
func shortHash(c string) string {
	if isHexSHA(c) && len(c) > 12 {
		return c[:12]
	}
	return c
}

func isHexSHA(s string) bool {
	if len(s) < 7 {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
