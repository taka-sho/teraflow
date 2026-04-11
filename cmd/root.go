package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	tferrors "github.com/taka-sho/teraflow/internal/errors"
)

// Execute runs the CLI root command.
func Execute(version string) error {
	rootCmd := newRootCmd(version)
	err := rootCmd.Execute()
	if err != nil {
		var appErr *tferrors.AppError
		if errors.As(err, &appErr) {
			format, _ := rootCmd.PersistentFlags().GetString("format")
			if format == "json" {
				_ = json.NewEncoder(os.Stderr).Encode(appErr.ToJSON())
			} else {
				fmt.Fprintln(os.Stderr, "Error:", appErr.Error())
			}
			os.Exit(appErr.ExitCode)
		}
		var exitErr exitCodeCarrier
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "Error:", err.Error())
		os.Exit(1)
	}
	return nil
}

func newRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "teraflow",
		Short:         "teraflow project lifecycle CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.Version = version
	// Keep version output as plain text for scripts that parse `teraflow version`.
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.PersistentFlags().String("config", ".github/teraflow.yml", "Config file path")
	rootCmd.PersistentFlags().String("format", "text", "Output format: text|json")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")

	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newScanCmd())
	rootCmd.AddCommand(newReworkCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newProcessCmd())
	rootCmd.AddCommand(newStageCmd())
	rootCmd.AddCommand(newPhaseCmd())
	rootCmd.AddCommand(newGateCmd())
	rootCmd.AddCommand(newAuditCmd())
	rootCmd.AddCommand(newIncidentCmd())
	rootCmd.AddCommand(newChangelogCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newScheduleCmd())
	rootCmd.AddCommand(newDashboardCmd())
	rootCmd.AddCommand(newLabelCmd())
	rootCmd.AddCommand(newDiscussionCmd())
	rootCmd.AddCommand(newSetupCmd())
	rootCmd.AddCommand(newAgentCmd())
	rootCmd.AddCommand(newSkillCmd())
	rootCmd.AddCommand(newIndexCmd())
	rootCmd.AddCommand(newSummaryCmd())
	rootCmd.AddCommand(newRbacCmd())
	rootCmd.AddCommand(newHookCmd())
	rootCmd.AddCommand(newTraceCmd())
	rootCmd.AddCommand(newDocCmd())
	rootCmd.AddCommand(newGraphCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newImpactCmd())
	rootCmd.AddCommand(newGenerateCmd())
	rootCmd.AddCommand(newPlanCmd())
	rootCmd.AddCommand(newImplementCmd())
	return rootCmd
}
