package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func newBrowseCmd() *cobra.Command {
	var rootArg string
	c := &cobra.Command{
		Use:   "browse [subdir]",
		Short: "Hierarchical directory browser (default root: <bench>/out); prints chosen path",
		Long: `Walk a directory tree under the current benchmark with arrow keys and pick a
folder to print on stdout. Defaults to <bench-root>/out; pass a subdir name to
start there instead (e.g. ` + "`obmr browse out/one-data`" + `, or any path under
the bench root).

Keys:
  ↑/↓ or j/k     move
  → or enter     descend into highlighted folder
  ← or backspace ascend (stops at the start root)
  s or .         select the current folder
  q or esc       cancel

The chosen absolute path is printed on stdout; the TUI runs on stderr, so the
` + "`ocd`" + ` shell wrapper works for this too:

  ocd "$(obmr browse)"
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			plan, err := resolvePlan("")
			if err != nil {
				return err
			}
			benchDir, err := filepath.Abs(filepath.Dir(plan))
			if err != nil {
				return err
			}

			start := filepath.Join(benchDir, "out")
			if rootArg != "" {
				start = rootArg
			}
			if len(args) == 1 {
				start = args[0]
			}
			if !filepath.IsAbs(start) {
				start = filepath.Join(benchDir, start)
			}
			start, err = filepath.Abs(start)
			if err != nil {
				return err
			}
			st, err := os.Stat(start)
			if err != nil {
				return fmt.Errorf("cannot browse %s: %w", start, err)
			}
			if !st.IsDir() {
				return fmt.Errorf("%s is not a directory", start)
			}

			m := browseModel{root: start, cwd: start}
			m.refresh()

			p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithAltScreen())
			final, err := p.Run()
			if err != nil {
				return err
			}
			fm := final.(browseModel)
			if fm.chosen == "" {
				return nil
			}
			fmt.Println(fm.chosen)
			return nil
		},
	}
	c.Flags().StringVar(&rootArg, "root", "", "explicit start directory (overrides default <bench>/out)")
	c.ValidArgsFunction = func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		plan, err := resolvePlan("")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		benchDir, err := filepath.Abs(filepath.Dir(plan))
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeBenchPath(benchDir, toComplete)
	}
	return c
}

// completeBenchPath returns directory completions under benchDir for the given
// partial input. When a candidate has exactly one subdirectory, the chain is
// extended automatically until ambiguity (or a leaf) is reached.
func completeBenchPath(benchDir, toComplete string) ([]string, cobra.ShellCompDirective) {
	dirPart, prefix := filepath.Split(toComplete)
	searchDir := filepath.Join(benchDir, dirPart)
	if dirPart == "" {
		searchDir = filepath.Join(benchDir, "out")
	}
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		base := dirPart
		if dirPart == "" {
			base = "out/"
		}
		candidate := base + name
		// Collapse single-child chains.
		full := filepath.Join(benchDir, candidate)
		for {
			subs, err := os.ReadDir(full)
			if err != nil {
				break
			}
			var sole string
			soleCount := 0
			for _, s := range subs {
				if s.IsDir() {
					sole = s.Name()
					soleCount++
				}
			}
			if soleCount != 1 {
				break
			}
			candidate = candidate + "/" + sole
			full = filepath.Join(full, sole)
		}
		out = append(out, candidate)
	}
	return out, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

type browseModel struct {
	root    string
	cwd     string
	entries []os.DirEntry
	cursor  int
	chosen  string
	errMsg  string
}

func (m *browseModel) refresh() {
	m.cursor = 0
	m.errMsg = ""
	es, err := os.ReadDir(m.cwd)
	if err != nil {
		m.entries = nil
		m.errMsg = err.Error()
		return
	}
	dirs := es[:0:0]
	for _, e := range es {
		if e.IsDir() {
			dirs = append(dirs, e)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
	m.entries = dirs
}

func (m browseModel) Init() tea.Cmd { return nil }

func (m browseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if len(m.entries) == 0 {
				return m, nil
			}
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.entries) - 1
			}
		case "down", "j":
			if len(m.entries) == 0 {
				return m, nil
			}
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case "right", "enter", "l":
			if len(m.entries) == 0 {
				return m, nil
			}
			next := filepath.Join(m.cwd, m.entries[m.cursor].Name())
			m.cwd = next
			m.refresh()
		case "left", "backspace", "h":
			if m.cwd == m.root {
				return m, nil
			}
			parent := filepath.Dir(m.cwd)
			prev := filepath.Base(m.cwd)
			m.cwd = parent
			m.refresh()
			for i, e := range m.entries {
				if e.Name() == prev {
					m.cursor = i
					break
				}
			}
		case "s", ".":
			m.chosen = m.cwd
			return m, tea.Quit
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			if len(m.entries) > 0 {
				m.cursor = len(m.entries) - 1
			}
		}
	}
	return m, nil
}

var (
	brHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	brPath   = lipgloss.NewStyle().Faint(true)
	brSel    = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	brDir    = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	brHint   = lipgloss.NewStyle().Faint(true).Italic(true)
	brErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func (m browseModel) View() string {
	rel := m.cwd
	if r, err := filepath.Rel(m.root, m.cwd); err == nil {
		if r == "." {
			rel = filepath.Base(m.root) + "/"
		} else {
			rel = filepath.Base(m.root) + "/" + r
		}
	}
	out := brHeader.Render("obmr browse") + "  " + brPath.Render(rel) + "\n\n"
	if m.errMsg != "" {
		out += brErr.Render(m.errMsg) + "\n"
	} else if len(m.entries) == 0 {
		out += brPath.Render("  (no subdirectories)") + "\n"
	} else {
		for i, e := range m.entries {
			line := brDir.Render(e.Name() + "/")
			cur := "  "
			if i == m.cursor {
				cur = "› "
				line = brSel.Render(e.Name() + "/")
			}
			out += cur + line + "\n"
		}
	}
	out += "\n" + brHint.Render(strings.Join([]string{
		"↑/↓ move",
		"→/enter descend",
		"← ascend",
		"s select",
		"q cancel",
	}, " • "))
	return out
}
