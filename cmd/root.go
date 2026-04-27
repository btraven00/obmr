package cmd

import "github.com/spf13/cobra"

func Root() *cobra.Command {
	root := &cobra.Command{
		Use:           "obmr",
		Short:         "omnibenchmark monorepo helper",
		Long:          "obmr manages an omnibenchmark and its module repos as a workspace of sibling clones.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newUseCmd(),
		newListCmd(),
		newInitCmd(),
		newStatusCmd(),
		newForeachCmd(),
		newCheckoutCmd(),
		newPullCmd(),
		newPushCmd(),
		newDevCmd(),
		newPinCmd(),
		newTrimCmd(),
	)
	return root
}
