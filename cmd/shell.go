package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const shellBlockBegin = "# >>> obmr shell integration >>>"
const shellBlockEnd = "# <<< obmr shell integration <<<"

func obmrShellSnippet(shell string) string {
	// Same body for bash and zsh — POSIX-compatible.
	body := `ocd() {
  local d
  d=$(obmr cd "$@") || return $?
  [ -n "$d" ] && cd "$d"
}
obrowse() {
  local d
  d=$(obmr browse "$@") || return $?
  [ -n "$d" ] && cd "$d"
}`
	completion := ""
	switch shell {
	case "zsh":
		completion = `if command -v obmr >/dev/null 2>&1; then
  source <(obmr completion zsh)
  compdef _obmr obmr
fi`
	case "bash":
		completion = `if command -v obmr >/dev/null 2>&1; then
  source <(obmr completion bash)
fi`
	}
	return shellBlockBegin + "\n" + body + "\n" + completion + "\n" + shellBlockEnd + "\n"
}

func detectShell() string {
	s := os.Getenv("SHELL")
	switch filepath.Base(s) {
	case "zsh":
		return "zsh"
	case "bash":
		return "bash"
	}
	return "bash"
}

func rcPathFor(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	}
	return "", fmt.Errorf("unsupported shell %q (use bash or zsh)", shell)
}

func newShellInitCmd() *cobra.Command {
	var shell string
	c := &cobra.Command{
		Use:   "shell-init",
		Short: "Print shell integration (ocd wrapper + completion) to stdout",
		Long: `Print the shell snippet that defines the ` + "`ocd`" + ` wrapper for ` + "`obmr cd`" + `
and enables tab completion. Source it manually, or run ` + "`obmr shell-install`" + `
to write it to your rc file.

  # one-shot eval (current session)
  eval "$(obmr shell-init)"
`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			s := shell
			if s == "" {
				s = detectShell()
			}
			fmt.Print(obmrShellSnippet(s))
			return nil
		},
	}
	c.Flags().StringVar(&shell, "shell", "", "shell to target: bash or zsh (default: $SHELL)")
	return c
}

func newShellInstallCmd() *cobra.Command {
	var shell string
	var rcPath string
	var force bool
	c := &cobra.Command{
		Use:   "shell-install",
		Short: "Append the shell integration block to your rc file (idempotent)",
		Long: `Write the obmr shell integration block to your rc file. The block is
delimited by markers, so re-running this command replaces the existing
block in place rather than appending duplicates.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			s := shell
			if s == "" {
				s = detectShell()
			}
			path := rcPath
			if path == "" {
				p, err := rcPathFor(s)
				if err != nil {
					return err
				}
				path = p
			}

			snippet := obmrShellSnippet(s)

			var existing []byte
			if b, err := os.ReadFile(path); err == nil {
				existing = b
			} else if !os.IsNotExist(err) {
				return err
			}

			updated, replaced := replaceBlock(existing, snippet)
			if !replaced {
				if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
					existing = append(existing, '\n')
				}
				updated = append(existing, []byte("\n"+snippet)...)
			}

			if !force && bytes.Equal(existing, updated) {
				fmt.Printf("%s already up to date\n", path)
				return nil
			}

			if err := os.WriteFile(path, updated, 0644); err != nil {
				return err
			}
			action := "updated"
			if !replaced {
				action = "appended to"
			}
			fmt.Printf("%s %s — restart your shell or run: source %s\n", action, path, path)
			return nil
		},
	}
	c.Flags().StringVar(&shell, "shell", "", "shell to target: bash or zsh (default: $SHELL)")
	c.Flags().StringVar(&rcPath, "rc", "", "rc file path (default: ~/.zshrc or ~/.bashrc)")
	c.Flags().BoolVar(&force, "force", false, "rewrite even if no changes detected")
	return c
}

// replaceBlock replaces the existing obmr block (if any) in src with newBlock.
// Returns the new buffer and whether a replacement happened.
func replaceBlock(src []byte, newBlock string) ([]byte, bool) {
	s := string(src)
	beginIdx := strings.Index(s, shellBlockBegin)
	if beginIdx < 0 {
		return src, false
	}
	endIdx := strings.Index(s[beginIdx:], shellBlockEnd)
	if endIdx < 0 {
		return src, false
	}
	endIdx += beginIdx + len(shellBlockEnd)
	// Consume trailing newline if present.
	if endIdx < len(s) && s[endIdx] == '\n' {
		endIdx++
	}
	out := s[:beginIdx] + newBlock + s[endIdx:]
	return []byte(out), true
}
