package cmd

import "github.com/spf13/cobra"

func newSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup GitHub Actions workflows and templates",
	}

	cmd.AddCommand(newSetupActionsCmd())
	cmd.AddCommand(newSetupTemplatesCmd())
	return cmd
}
