package cmd

import "github.com/spf13/cobra"

const (
	groupBasics = "basics"
	groupGit    = "git"
	groupBench  = "bench"
)

func Root() *cobra.Command {
	root := &cobra.Command{
		Use:           "obmr",
		Short:         "omnibenchmark monorepo helper",
		Long:          "obmr manages an omnibenchmark and its module repos as a workspace of sibling clones.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddGroup(
		&cobra.Group{ID: groupBasics, Title: "Basics:"},
		&cobra.Group{ID: groupGit, Title: "Git fan-out:"},
		&cobra.Group{ID: groupBench, Title: "Benchmarker actions:"},
	)

	use := newUseCmd()
	initC := newInitCmd()
	status := newStatusCmd()
	runC := newRunCmd()
	dev := newDevCmd()
	list := newListCmd()
	for _, c := range []*cobra.Command{use, initC, status, runC, dev, list} {
		c.GroupID = groupBasics
	}

	checkout := newCheckoutCmd()
	push := newPushCmd()
	pull := newPullCmd()
	foreach := newForeachCmd()
	trim := newTrimCmd()
	for _, c := range []*cobra.Command{checkout, push, pull, foreach, trim} {
		c.GroupID = groupGit
	}

	plan := newPlanCmd()
	plan.GroupID = groupBench

	root.AddCommand(
		use, initC, status, runC, dev, list,
		checkout, push, pull, foreach, trim,
		plan,
		newConfigCmd(),
	)
	return root
}
