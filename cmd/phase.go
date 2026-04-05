package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/state"
)

var availablePhases = []string{
	"requirements",
	"basic_design",
	"detailed_design",
	"implementation",
	"testing",
	"integration_test",
}

func newPhaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "phase",
		Short: "Manage project phases",
	}
	cmd.AddCommand(newPhaseListCmd())
	cmd.AddCommand(newPhaseStartCmd())
	cmd.AddCommand(newPhaseCompleteCmd())
	return cmd
}

func newPhaseListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available phases",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			s, err := loadStateOrNotProjectErr(configPath)
			if err != nil {
				return err
			}

			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"current": s.Phases.Current,
					"phases":  availablePhases,
				})
			}

			for _, p := range availablePhases {
				prefix := "  "
				if p == s.Phases.Current {
					prefix = "->"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", prefix, p)
			}
			return nil
		},
	}
}

func newPhaseStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start a specific phase",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			target := args[0]
			if !contains(availablePhases, target) {
				return fmt.Errorf("invalid phase: %s", target)
			}

			s, err := loadStateOrNotProjectErr(configPath)
			if err != nil {
				return err
			}
			prev := s.Phases.Current
			s.Phases.Current = target
			if err := state.SaveState(configPath, s); err != nil {
				return err
			}

			if format == "json" {
				return writeJSON(cmd, map[string]string{
					"status": "started",
					"from":   prev,
					"to":     target,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Phase started: %s\n", target)
			return nil
		},
	}
}

func newPhaseCompleteCmd() *cobra.Command {
	var force bool
	var reason string
	cmd := &cobra.Command{
		Use:   "complete",
		Short: "Complete current phase and move to next",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			s, err := loadStateOrNotProjectErr(configPath)
			if err != nil {
				return err
			}
			// TODO(cmd_108/w2-f): Auto-sync in-progress SLCP-JCF process on phase complete.
			current := s.Phases.Current
			next, ok := nextValue(availablePhases, current)
			if !ok {
				if format == "json" {
					return writeJSON(cmd, map[string]any{
						"status":  "completed",
						"phase":   current,
						"message": "All phases completed.",
					})
				}
				fmt.Fprintln(cmd.OutOrStdout(), "All phases completed.")
				return nil
			}

			if err := runConstraintGuard(cmd, configPath, "phase.complete", force, reason); err != nil {
				return err
			}

			s.Phases.Current = next
			if err := state.SaveState(configPath, s); err != nil {
				return err
			}

			if format == "json" {
				return writeJSON(cmd, map[string]string{
					"status": "advanced",
					"from":   current,
					"to":     next,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Phase advanced: %s -> %s\n", current, next)
			if next == "integration_test" {
				fmt.Fprintln(cmd.OutOrStdout(), "All phases completed.")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Override constraint checks")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for --force")
	return cmd
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
