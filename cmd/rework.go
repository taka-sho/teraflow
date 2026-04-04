package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/state"
)

type reworkCreateOptions struct {
	Group       string
	TargetPhase string
	Reason      string
}

func newReworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rework",
		Short: "Manage reworks",
	}

	cmd.AddCommand(newReworkCreateCmd())
	cmd.AddCommand(newReworkListCmd())
	return cmd
}

func newReworkCreateCmd() *cobra.Command {
	opts := &reworkCreateOptions{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new rework record",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Group == "" {
				return fmt.Errorf("--group is required")
			}
			if opts.TargetPhase == "" {
				return fmt.Errorf("--target-phase is required")
			}
			if opts.Reason == "" {
				return fmt.Errorf("--reason is required")
			}

			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			log, err := state.LoadReworkLog(configPath)
			if err != nil {
				return err
			}

			entry := state.ReworkEntry{
				ID:          fmt.Sprintf("rw-%03d", len(log.Reworks)+1),
				Group:       opts.Group,
				TargetPhase: opts.TargetPhase,
				Reason:      opts.Reason,
				CreatedAt:   time.Now().Format("2006-01-02T15:04:05"),
				Status:      "open",
			}

			if err := state.AppendRework(configPath, entry); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Rework recorded:")
			fmt.Fprintf(cmd.OutOrStdout(), "  ID:           %s\n", entry.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "  Group:        %s\n", entry.Group)
			fmt.Fprintf(cmd.OutOrStdout(), "  Target phase: %s\n", entry.TargetPhase)
			fmt.Fprintf(cmd.OutOrStdout(), "  Reason:       %s\n", entry.Reason)
			fmt.Fprintf(cmd.OutOrStdout(), "  Status:       %s\n", entry.Status)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Group, "group", "g", "", "Target group")
	cmd.Flags().StringVar(&opts.TargetPhase, "target-phase", "", "Target phase")
	cmd.Flags().StringVar(&opts.Reason, "reason", "", "Reason")
	return cmd
}

func newReworkListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List rework records",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			log, err := state.LoadReworkLog(configPath)
			if err != nil {
				return err
			}

			if len(log.Reworks) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No reworks recorded.")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "ID       Group        Target Phase   Status   Reason")
			fmt.Fprintln(cmd.OutOrStdout(), "──────   ─────────    ────────────   ──────   ──────")
			for _, rw := range log.Reworks {
				fmt.Fprintf(cmd.OutOrStdout(), "%-8s %-12s %-14s %-8s %s\n", rw.ID, rw.Group, rw.TargetPhase, rw.Status, rw.Reason)
			}

			return nil
		},
	}

	return cmd
}
