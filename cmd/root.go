package cmd

import "github.com/spf13/cobra"

// Execute runs the CLI root command.
func Execute(version string) error {
	return newRootCmd(version).Execute()
}

func newRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "teraflow",
		Short:         "teraflow project lifecycle CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.Version = version
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.PersistentFlags().String("config", ".github/teraflow.yml", "Config file path")
	rootCmd.PersistentFlags().String("format", "text", "Output format: text|json")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")

	rootCmd.AddCommand(newInitCmd())
	return rootCmd
}
