package cmd

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/btraven00/obmr/internal/workspace"
	"github.com/spf13/cobra"
)

func newForeachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "foreach -- <cmd> [args...]",
		Short: "Run a shell command in every module dir (uses default plan)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			plan, err := resolvePlan("")
			if err != nil {
				return err
			}
			lock, benchDir, err := loadLock(plan)
			if err != nil {
				return err
			}
			cmdArgs := args
			results := workspace.Fanout(benchDir, lock, func(dir string, _ workspace.LockedModule) (string, error) {
				c := exec.Command(cmdArgs[0], cmdArgs[1:]...)
				c.Dir = dir
				var out bytes.Buffer
				c.Stdout = &out
				c.Stderr = &out
				err := c.Run()
				return out.String(), err
			})
			anyErr := false
			for _, r := range results {
				prefix := fmt.Sprintf("[%s]", r.Module.ID)
				for _, line := range strings.Split(strings.TrimRight(r.Out, "\n"), "\n") {
					fmt.Printf("%s %s\n", prefix, line)
				}
				if r.Err != nil {
					fmt.Printf("%s exit: %v\n", prefix, r.Err)
					anyErr = true
				}
			}
			if anyErr {
				return fmt.Errorf("one or more modules failed")
			}
			return nil
		},
	}
}
