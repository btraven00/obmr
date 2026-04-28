package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/btraven00/obmr/internal/benchmark"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

func newCdCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "cd [module-id]",
		Short: "Pick a module directory (interactive TUI; or pass a module-id to print its path)",
		Long: `Without args, opens an interactive picker over the modules in the current benchmark.
With a module-id, prints that module's absolute path directly (non-interactive).

Use ↑/↓ or j/k to move, Enter to select, q/Esc to cancel.

The chosen absolute path is printed on stdout; the TUI itself runs on stderr,
so wrap with a shell function to actually change directory. Run ` + "`obmr shell-init`" + `
or ` + "`obmr shell-install`" + ` to set up the wrapper.
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			plan, err := resolvePlan("")
			if err != nil {
				return err
			}
			if detectMode(plan) != "dev" {
				return fmt.Errorf("`obmr cd` only operates in dev mode (no %s found); run `obmr dev` first",
					filepath.Base(localYAMLPathFromCanonical(plan)))
			}
			localPlan := localYAMLPathFromCanonical(plan)
			f, err := benchmark.Load(localPlan)
			if err != nil {
				return err
			}
			benchDir, err := filepath.Abs(filepath.Dir(localPlan))
			if err != nil {
				return err
			}

			items := make([]cdItem, 0)
			items = append(items, cdItem{
				stage: "",
				id:    "<bench root>",
				path:  benchDir,
			})
			for _, s := range f.Stages {
				for _, mod := range s.Modules {
					p := mod.Repository.URL
					if !filepath.IsAbs(p) {
						p = filepath.Join(benchDir, p)
					}
					items = append(items, cdItem{
						stage: s.ID,
						id:    mod.ID,
						path:  p,
					})
				}
			}

			if len(args) == 1 {
				want := args[0]
				for _, it := range items {
					if it.id == want {
						fmt.Println(it.path)
						return nil
					}
				}
				return fmt.Errorf("no module with id %q (try `obmr list`)", want)
			}

			// `ocd` consumes our stdout via $(...), so force lipgloss to
			// detect colors from stderr (the TUI output) instead.
			lipgloss.SetColorProfile(termenv.NewOutput(os.Stderr).Profile)

			m := cdModel{items: items, chosen: -1}
			p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithAltScreen())
			final, err := p.Run()
			if err != nil {
				return err
			}
			fm := final.(cdModel)
			if fm.chosen < 0 {
				return nil
			}
			fmt.Println(fm.items[fm.chosen].path)
			return nil
		},
	}
	c.ValidArgsFunction = func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		plan, err := resolvePlan("")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if detectMode(plan) != "dev" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		f, err := benchmark.Load(localYAMLPathFromCanonical(plan))
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var ids []string
		for _, s := range f.Stages {
			for _, m := range s.Modules {
				ids = append(ids, m.ID)
			}
		}
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
	return c
}

type cdItem struct {
	stage string
	id    string
	path  string
}

type cdModel struct {
	items  []cdItem
	cursor int
	chosen int
}

func (m cdModel) Init() tea.Cmd { return nil }

func (m cdModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.chosen = -1
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.items) - 1
			}
		case "down", "j", "tab":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case "shift+tab":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.items) - 1
			}
		case "enter":
			m.chosen = m.cursor
			return m, tea.Quit
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			m.cursor = len(m.items) - 1
		}
	}
	return m, nil
}

var (
	cdHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	cdStage  = lipgloss.NewStyle().Faint(true).Width(18)
	cdID     = lipgloss.NewStyle().Width(20)
	cdPath   = lipgloss.NewStyle().Faint(true)
	cdSel    = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	cdHint   = lipgloss.NewStyle().Faint(true).Italic(true)
)

func (m cdModel) View() string {
	out := cdHeader.Render("obmr cd — pick a module directory") + "\n\n"
	for i, it := range m.items {
		cursor := "  "
		row := cdStage.Render(it.stage) + cdID.Render(it.id) + cdPath.Render(it.path)
		if i == m.cursor {
			cursor = "› "
			row = cdSel.Render(row)
		}
		out += cursor + row + "\n"
	}
	out += "\n" + cdHint.Render("↑/↓ or j/k to move • enter to select • q/esc to cancel")
	return out
}
