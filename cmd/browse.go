package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

// humanSize formats a byte count like "12K", "3.4M", "1.2G".
func humanSize(n int64) string {
	const k = 1024
	if n < k {
		return fmt.Sprintf("%dB", n)
	}
	units := []string{"K", "M", "G", "T", "P"}
	val := float64(n) / k
	idx := 0
	for val >= k && idx < len(units)-1 {
		val /= k
		idx++
	}
	if val < 10 {
		return fmt.Sprintf("%.1f%s", val, units[idx])
	}
	return fmt.Sprintf("%.0f%s", val, units[idx])
}

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
  space or .     select the current folder
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

			// `outd` runs `obmr browse` in $(...), so stdout is a pipe and
			// lipgloss's default profile (sniffed from stdout) would strip
			// all colors. Force it to detect from stderr — the TUI's
			// real output.
			lipgloss.SetColorProfile(termenv.NewOutput(os.Stderr).Profile)

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
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(searchDir, name)
		st, err := os.Stat(full)
		if err != nil || !st.IsDir() {
			continue
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		base := dirPart
		if dirPart == "" {
			base = "out/"
		}
		candidate := base + name
		// Collapse single-child chains.
		full = filepath.Join(benchDir, candidate)
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
	files   []fileRow // up to fileLimit; trailing row with name="..." if more
	cursor  int
	chosen  string
	errMsg  string
	width   int
}

type fileRow struct {
	name    string
	size    int64
	modTime time.Time
	more    bool // true for the "..." sentinel row
}

const fileLimit = 10

func (m *browseModel) refresh() {
	m.cursor = 0
	m.errMsg = ""
	es, err := os.ReadDir(m.cwd)
	if err != nil {
		m.entries = nil
		m.errMsg = err.Error()
		return
	}
	// Show readable symlinks (e.g. `filter_type-manual` -> `.f876a054`) and
	// hide their hash targets. Plain dirs without a sibling symlink still
	// show. Anything starting with "." is hidden either way.
	dirs := es[:0:0]
	var files []fileRow
	for _, e := range es {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(m.cwd, name)
		st, err := os.Stat(full)
		if err != nil {
			continue
		}
		if st.IsDir() {
			dirs = append(dirs, e)
		} else {
			files = append(files, fileRow{name: name, size: st.Size(), modTime: st.ModTime()})
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	m.entries = dirs
	if len(files) > fileLimit {
		files = append(files[:fileLimit:fileLimit], fileRow{name: "...", more: true})
	}
	m.files = files
}

func (m browseModel) Init() tea.Cmd { return nil }

func (m browseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
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
		case " ", ".":
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
	brCrumb  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	brCrumbS = lipgloss.NewStyle().Faint(true)
	brFile   = lipgloss.NewStyle().Faint(true)
	// lsd-ish category palette. Bold + hex so they survive theme remaps
	// and remain visible on light/dark backgrounds. Fallbacks to ANSI
	// palette codes for terminals without truecolor.
	brFileData = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#b58900", Dark: "#e5c07b"})
	brFileArch = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#dc322f", Dark: "#e06c75"})
	brFileCode = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#198844", Dark: "#98c379"})
	brFileImg  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#9024a0", Dark: "#c678dd"})
	brFileDoc  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#586e75", Dark: "#abb2bf"})
	brHint     = lipgloss.NewStyle().Faint(true).Italic(true)
	brErr      = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	brMeta     = lipgloss.NewStyle().Faint(true)
)

// styleForFile picks a color by extension, lsd-style.
func styleForFile(name string) lipgloss.Style {
	if name == "..." {
		return brFile
	}
	ext := strings.ToLower(filepath.Ext(name))
	// Strip a trailing compression suffix to classify the underlying type.
	switch ext {
	case ".gz", ".bz2", ".xz", ".zst":
		inner := strings.ToLower(filepath.Ext(strings.TrimSuffix(name, ext)))
		if inner != "" {
			ext = inner
		} else {
			return brFileArch
		}
	}
	switch ext {
	case ".h5", ".h5ad", ".hdf5", ".csv", ".tsv", ".json", ".yaml", ".yml", ".parquet", ".feather", ".rds":
		return brFileData
	case ".zip", ".tar", ".tgz", ".7z":
		return brFileArch
	case ".py", ".r", ".sh", ".go", ".rs", ".js", ".ts", ".c", ".cpp", ".h", ".hpp":
		return brFileCode
	case ".png", ".jpg", ".jpeg", ".pdf", ".svg", ".gif":
		return brFileImg
	case ".md", ".txt", ".log", ".out", ".rst":
		return brFileDoc
	}
	return brFile
}

func (m browseModel) View() string {
	crumbs := []string{filepath.Base(m.root)}
	if r, err := filepath.Rel(m.root, m.cwd); err == nil && r != "." {
		crumbs = append(crumbs, strings.Split(r, string(filepath.Separator))...)
	}
	rendered := make([]string, len(crumbs))
	for i, c := range crumbs {
		rendered[i] = brCrumb.Render(c)
	}
	sep := " " + brCrumbS.Render("›") + " "
	sepW := lipgloss.Width(sep)
	headerText := brHeader.Render("obmr browse")
	headerW := lipgloss.Width(headerText)
	indent := strings.Repeat(" ", headerW+2)
	line := headerText + "  "
	lineW := headerW + 2
	var crumbLines []string
	for i, r := range rendered {
		addW := lipgloss.Width(r)
		if i > 0 {
			addW += sepW
		}
		if m.width > 0 && i > 0 && lineW+addW > m.width {
			crumbLines = append(crumbLines, line)
			line = indent
			lineW = len(indent)
		}
		if i > 0 {
			line += sep
			lineW += sepW
		}
		line += r
		lineW += lipgloss.Width(r)
	}
	crumbLines = append(crumbLines, line)
	out := strings.Join(crumbLines, "\n") + "\n\n"
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
	// File area: always reserve fileLimit rows so the cursor/footer
	// don't jump when navigating between dirs with different file counts.
	out += "\n"
	nameW := 0
	for _, f := range m.files {
		if l := len(f.name); l > nameW {
			nameW = l
		}
	}
	for i := 0; i < fileLimit; i++ {
		if i >= len(m.files) {
			out += "\n"
			continue
		}
		f := m.files[i]
		if f.more {
			out += "  " + brFile.Render(f.name) + "\n"
			continue
		}
		pad := strings.Repeat(" ", nameW-len(f.name))
		meta := fmt.Sprintf("  %8s  %s",
			humanSize(f.size), f.modTime.Local().Format("2006-01-02 15:04"))
		out += "  " + styleForFile(f.name).Render(f.name) + pad +
			brMeta.Render(meta) + "\n"
	}
	out += "\n" + brHint.Render(strings.Join([]string{
		"↑/↓ move",
		"→/enter descend",
		"← ascend",
		"space select",
		"q cancel",
	}, " • "))
	return out
}
