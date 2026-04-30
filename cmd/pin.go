package cmd

import (
	"fmt"

	"github.com/btraven00/obflow/internal/workspace"
	"github.com/spf13/cobra"
)

func newPinCmd() *cobra.Command {
	var ref string
	c := &cobra.Command{
		Use:   "pin [bench.yaml]",
		Short: "Rewrite canonical YAML commit SHAs from origin/<ref> per module (in place)",
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
			results, err := workspace.Pin(plan, lock, ref)
			if err != nil {
				return err
			}
			for _, r := range results {
				if r.Err != nil {
					fmt.Printf("%-20s ERROR: %v\n", r.ModuleID, r.Err)
					continue
				}
				old := short(r.OldSHA)
				newS := short(r.NewSHA)
				switch {
				case r.OldSHA == r.NewSHA:
					fmt.Printf("%-20s unchanged (%s)\n", r.ModuleID, newS)
				case r.OldSHA == "":
					fmt.Printf("%-20s pinned -> %s\n", r.ModuleID, newS)
				default:
					line := fmt.Sprintf("%-20s %s -> %s", r.ModuleID, old, newS)
					if r.Warning != "" {
						line += "  WARN: " + r.Warning
					}
					fmt.Println(line)
				}
			}
			return nil
		},
	}
	c.Flags().StringVar(&ref, "ref", "", "upstream ref to pin from (default: HEAD i.e. origin's default branch)")
	return c
}

func short(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}
