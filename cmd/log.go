package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newLogCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "log",
		Short: "Show the rule log from the last failed snakemake run",
		RunE: func(_ *cobra.Command, _ []string) error {
			plan, err := resolvePlan("")
			if err != nil {
				return err
			}
			benchDir, err := filepath.Abs(filepath.Dir(plan))
			if err != nil {
				return err
			}
			logsDir := filepath.Join(benchDir, "out", ".snakemake", "log")

			snakeLog, err := latestSnakemakeLog(logsDir)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "%s snakemake log: %s\n",
				paint("==>", ansiGreen+ansiBold), snakeLog)

			ruleName, logPath, err := failedRuleLog(snakeLog)
			if err != nil {
				return err
			}
			ruleLogAbs := filepath.Join(benchDir, "out", logPath)
			fmt.Fprintf(os.Stderr, "%s rule %s → %s\n",
				paint("==>", ansiGreen+ansiBold),
				paint(ruleName, ansiBold),
				ruleLogAbs)
			fmt.Fprintln(os.Stderr, strings.Repeat("─", 60))

			b, err := os.ReadFile(ruleLogAbs)
			if err != nil {
				return fmt.Errorf("read rule log: %w", err)
			}
			os.Stdout.Write(b)
			return nil
		},
	}
	return c
}

// latestSnakemakeLog returns the most recently modified snakemake_*.log in dir.
func latestSnakemakeLog(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read logs dir %s: %w", dir, err)
	}
	var best string
	var bestMod int64
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".snakemake.log") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if t := info.ModTime().UnixNano(); t > bestMod {
			best = filepath.Join(dir, name)
			bestMod = t
		}
	}
	if best == "" {
		return "", fmt.Errorf("no *.snakemake.log found in %s", dir)
	}
	return best, nil
}

// failedRuleLog scans a snakemake log and returns the last failed rule name
// and its log path (relative to out/).
func failedRuleLog(snakeLog string) (string, string, error) {
	f, err := os.Open(snakeLog)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	var ruleName, logPath string
	inError := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, "Error in rule "); ok {
			ruleName = strings.TrimSuffix(after, ":")
			logPath = ""
			inError = true
			continue
		}
		if inError {
			if after, ok := strings.CutPrefix(trimmed, "log: "); ok {
				// strip trailing " (check log file(s) for error details)"
				if i := strings.Index(after, " ("); i >= 0 {
					after = after[:i]
				}
				logPath = after
				inError = false
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("scan %s: %w", snakeLog, err)
	}
	if ruleName == "" {
		return "", "", fmt.Errorf("no failed rule found in %s", snakeLog)
	}
	if logPath == "" {
		return "", "", fmt.Errorf("failed rule %q has no log path in %s", ruleName, snakeLog)
	}
	return ruleName, logPath, nil
}
