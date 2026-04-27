package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newFmtCmd() *cobra.Command {
	var local bool
	c := &cobra.Command{
		Use:   "fmt [path]",
		Short: "Reformat a YAML file in place (normalizes indentation, preserves comments)",
		Long: `Re-marshal a YAML file through yaml.v3 to fix indentation and quoting
inconsistencies. Comments are preserved.

With no path, uses the configured plan. With --local, uses <plan>.local.yaml.

Note: the file must parse as YAML first. If it's malformed (e.g. mixed
indentation that confuses the parser), fix the bad lines by hand before
running fmt.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := firstArg(args)
			if path == "" {
				plan, err := resolvePlan("")
				if err != nil {
					return err
				}
				localPath := localYAMLPathFromCanonical(plan)
				// In dev mode (local exists) default to local; with --local always.
				if local {
					path = localPath
				} else if _, err := os.Stat(localPath); err == nil {
					path = localPath
				} else {
					path = plan
				}
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var root yaml.Node
			if err := yaml.Unmarshal(src, &root); err != nil {
				return parseErrorWithContext(path, src, err)
			}
			out, err := yaml.Marshal(&root)
			if err != nil {
				return err
			}
			if err := os.WriteFile(path, out, 0644); err != nil {
				return err
			}
			fmt.Printf("formatted %s\n", path)
			return nil
		},
	}
	c.Flags().BoolVar(&local, "local", false, "format <plan>.local.yaml instead of the canonical plan")
	return c
}

var lineRE = regexp.MustCompile(`line (\d+)`)

// parseErrorWithContext wraps a yaml parse error with cargo-style source
// context: file:line pointer, surrounding lines, caret on the offending
// line, and a hint.
func parseErrorWithContext(path string, src []byte, err error) error {
	m := lineRE.FindStringSubmatch(err.Error())
	if m == nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	n, _ := strconv.Atoi(m[1])
	lines := strings.Split(string(src), "\n")
	start, end := n-3, n+2
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	gutterW := len(strconv.Itoa(end))
	pad := strings.Repeat(" ", gutterW)
	bar := paint("|", ansiBlue+ansiBold)

	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s\n", paint("error", ansiRed+ansiBold), err)
	fmt.Fprintf(&b, "%s%s %s:%d\n", pad, paint("-->", ansiBlue+ansiBold), path, n)
	fmt.Fprintf(&b, "%s %s\n", pad, bar)
	for i := start - 1; i < end; i++ {
		num := fmt.Sprintf("%*d", gutterW, i+1)
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		fmt.Fprintf(&b, "%s %s %s\n", paint(num, ansiBlue+ansiBold), bar, line)
		if i+1 == n {
			caret := strings.Repeat("^", maxInt(1, len(strings.TrimRight(line, " "))))
			fmt.Fprintf(&b, "%s %s %s\n", pad, bar, paint(caret, ansiRed+ansiBold))
		}
	}
	fmt.Fprintf(&b, "%s %s\n", pad, bar)
	fmt.Fprintf(&b, "%s %s mixed indentation likely; align list items to the rest of the file\n",
		pad, paint("=", ansiBlue+ansiBold))
	return fmt.Errorf("%s", b.String())
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
