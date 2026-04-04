package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/state"
)

type dashboardSummary struct {
	Project struct {
		Name string `json:"name"`
	} `json:"project"`
	Lifecycle struct {
		CurrentStage string `json:"current_stage"`
	} `json:"lifecycle"`
	Phases struct {
		Current string `json:"current"`
	} `json:"phases"`
	Rework struct {
		Total    int `json:"total"`
		Open     int `json:"open"`
		Resolved int `json:"resolved"`
	} `json:"rework"`
	RecentActivity []state.ReworkEntry `json:"recent_activity"`
}

func newDashboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Show project dashboard",
	}

	cmd.AddCommand(newDashboardShowCmd())
	return cmd
}

func newDashboardShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show project dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			s, err := state.LoadState(configPath)
			if err != nil {
				var pe *os.PathError
				if errors.As(err, &pe) && errors.Is(pe.Err, os.ErrNotExist) {
					fmt.Fprintln(cmd.ErrOrStderr(), notProjectError)
					return errors.New(notProjectError)
				}
				return err
			}

			log, err := state.LoadReworkLog(configPath)
			if err != nil {
				return err
			}

			summary := buildDashboardSummary(s, log)
			if format == "json" {
				return writeJSON(cmd, summary)
			}

			printDashboardText(cmd, summary)
			return nil
		},
	}
}

func buildDashboardSummary(s *state.ProjectState, log *state.ReworkLog) dashboardSummary {
	summary := dashboardSummary{}
	summary.Project.Name = s.Project.Name
	summary.Lifecycle.CurrentStage = s.Lifecycle.CurrentStage
	summary.Phases.Current = s.Phases.Current
	summary.Rework.Total = len(log.Reworks)

	recent := append([]state.ReworkEntry(nil), log.Reworks...)
	sort.SliceStable(recent, func(i, j int) bool {
		return recent[i].CreatedAt > recent[j].CreatedAt
	})

	for _, rw := range recent {
		if strings.EqualFold(rw.Status, "open") {
			summary.Rework.Open++
		} else {
			summary.Rework.Resolved++
		}
	}

	if len(recent) > 5 {
		recent = recent[:5]
	}
	summary.RecentActivity = recent
	return summary
}

func printDashboardText(cmd *cobra.Command, summary dashboardSummary) {
	fmt.Fprintln(cmd.OutOrStdout(), "╔══════════════════════════════════════╗")
	fmt.Fprintln(cmd.OutOrStdout(), "║  teraflow Dashboard                  ║")
	fmt.Fprintln(cmd.OutOrStdout(), "╚══════════════════════════════════════╝")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Project:  %s\n", summary.Project.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Stage:    %s\n", summary.Lifecycle.CurrentStage)
	fmt.Fprintf(cmd.OutOrStdout(), "Phase:    %s\n", summary.Phases.Current)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "── Rework Summary ──────────────────────")
	fmt.Fprintf(cmd.OutOrStdout(), "Total:    %d  Open: %d  Resolved: %d\n", summary.Rework.Total, summary.Rework.Open, summary.Rework.Resolved)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "── Recent Activity ─────────────────────")
	if len(summary.RecentActivity) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "(no rework entries)")
		return
	}
	for _, rw := range summary.RecentActivity {
		fmt.Fprintf(cmd.OutOrStdout(), "- %s  %s  %s  %s\n", rw.CreatedAt, rw.ID, rw.Status, rw.Reason)
	}
}
