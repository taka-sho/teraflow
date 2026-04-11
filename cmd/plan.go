package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/state"
	"github.com/taka-sho/teraflow/internal/wave"
)

func newPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show V-model execution plan and state",
	}
	cmd.AddCommand(newPlanWavesCmd())
	cmd.AddCommand(newPlanPhasesCmd())
	cmd.AddCommand(newPlanStatusCmd())
	return cmd
}

func newPlanWavesCmd() *cobra.Command {
	var phase string

	cmd := &cobra.Command{
		Use:   "waves",
		Short: "Show wave execution plan for a phase",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(phase) == "" {
				return fmt.Errorf("--phase is required")
			}
			defs := wave.DefaultWaveDefinitions(phase)
			if len(defs) == 0 {
				return fmt.Errorf("no wave definitions for phase %q", phase)
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, map[string]any{"phase": phase, "waves": defs})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Phase: %s\n", phase)
			for _, d := range defs {
				fmt.Fprintf(cmd.OutOrStdout(), "- Wave %d: %s template=%s review=%s\n", d.Number, d.Name, d.Template, d.ReviewReq)
				if len(d.DependsOn) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "  depends_on: %v\n", d.DependsOn)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&phase, "phase", "", "Target phase")
	return cmd
}

func newPlanPhasesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "phases",
		Short: "Show current phase from project-state",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			s, err := state.LoadState(configPath)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"current_phase": s.Phases.Current,
					"lifecycle":     s.Lifecycle.CurrentStage,
					"project":       s.Project.Name,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Project: %s\n", s.Project.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "Lifecycle: %s\n", s.Lifecycle.CurrentStage)
			fmt.Fprintf(cmd.OutOrStdout(), "Current phase: %s\n", s.Phases.Current)
			return nil
		},
	}
}

func newPlanStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show V-model overall status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			s, err := state.LoadState(configPath)
			if err != nil {
				return err
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"project":       s.Project.Name,
				"lifecycle":     s.Lifecycle.CurrentStage,
				"current_phase": s.Phases.Current,
				"slcp_jcf":      s.SLCPJCF,
			}
			if format == "json" {
				return writeJSON(cmd, payload)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Project: %s\n", s.Project.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "Lifecycle: %s\n", s.Lifecycle.CurrentStage)
			fmt.Fprintf(cmd.OutOrStdout(), "Current phase: %s\n", s.Phases.Current)
			if len(s.SLCPJCF.Processes) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Processes:")
				for _, p := range s.SLCPJCF.Processes {
					fmt.Fprintf(cmd.OutOrStdout(), "- %s: %s\n", p.Name, p.Status)
				}
			}
			return nil
		},
	}
}
