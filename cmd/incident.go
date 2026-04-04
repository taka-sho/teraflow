package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/state"
)

type incidentCreateOptions struct {
	Title       string
	Severity    string
	Description string
}

type incidentCloseOptions struct {
	ID string
}

func newIncidentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "incident",
		Short: "Manage incidents",
	}

	cmd.AddCommand(newIncidentCreateCmd())
	cmd.AddCommand(newIncidentListCmd())
	cmd.AddCommand(newIncidentCloseCmd())
	return cmd
}

func newIncidentCreateCmd() *cobra.Command {
	opts := &incidentCreateOptions{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an incident record",
		RunE: func(cmd *cobra.Command, args []string) error {
			title := strings.TrimSpace(opts.Title)
			if title == "" {
				return fmt.Errorf("--title is required")
			}

			severity := strings.TrimSpace(opts.Severity)
			switch severity {
			case "critical", "major", "minor":
			default:
				return fmt.Errorf("--severity must be one of: critical, major, minor")
			}

			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}

			log, err := state.LoadIncidentLog(configPath)
			if err != nil {
				return err
			}

			entry := state.IncidentEntry{
				ID:          fmt.Sprintf("inc-%03d", len(log.Incidents)+1),
				Title:       title,
				Severity:    severity,
				Description: opts.Description,
				CreatedAt:   time.Now().Format("2006-01-02T15:04:05"),
				Status:      "open",
			}

			log.Incidents = append(log.Incidents, entry)
			if err := state.SaveIncidentLog(configPath, log); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Incident recorded: %s [%s] %s\n", entry.ID, entry.Severity, entry.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Title, "title", "", "Incident title")
	cmd.Flags().StringVar(&opts.Severity, "severity", "major", "Severity: critical|major|minor")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Incident description")
	return cmd
}

func newIncidentListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List incidents",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			log, err := state.LoadIncidentLog(configPath)
			if err != nil {
				return err
			}

			if len(log.Incidents) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No incidents recorded.")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "ID       Severity   Status   Title")
			fmt.Fprintln(cmd.OutOrStdout(), "──────   ────────   ──────   ─────")
			for _, incident := range log.Incidents {
				fmt.Fprintf(cmd.OutOrStdout(), "%-8s %-10s %-8s %s\n", incident.ID, incident.Severity, incident.Status, incident.Title)
			}
			return nil
		},
	}

	return cmd
}

func newIncidentCloseCmd() *cobra.Command {
	opts := &incidentCloseOptions{}
	cmd := &cobra.Command{
		Use:   "close",
		Short: "Close an incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			id := strings.TrimSpace(opts.ID)
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			log, err := state.LoadIncidentLog(configPath)
			if err != nil {
				return err
			}

			for i := range log.Incidents {
				if log.Incidents[i].ID == id {
					log.Incidents[i].Status = "closed"
					log.Incidents[i].ClosedAt = time.Now().Format("2006-01-02T15:04:05")
					if err := state.SaveIncidentLog(configPath, log); err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Incident %s closed.\n", id)
					return nil
				}
			}

			return fmt.Errorf("incident not found: %s", id)
		},
	}

	cmd.Flags().StringVar(&opts.ID, "id", "", "Incident ID")
	return cmd
}
