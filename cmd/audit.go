package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type auditLogFile struct {
	Entries []auditLogEntry `yaml:"entries"`
}

type auditLogEntry struct {
	ID        string `yaml:"id,omitempty" json:"id,omitempty"`
	Timestamp string `yaml:"timestamp" json:"timestamp"`
	User      string `yaml:"user" json:"user"`
	Role      string `yaml:"role,omitempty" json:"role,omitempty"`
	Action    string `yaml:"action" json:"action"`
	Target    string `yaml:"target" json:"target"`
	Result    string `yaml:"result" json:"result"`
}

func newAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Manage audit logs",
	}
	cmd.AddCommand(newAuditListCmd())
	return cmd
}

func newAuditListCmd() *cobra.Command {
	var userFilter string
	var actionFilter string
	var sinceFilter string
	var output string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List audit log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}

			entries, err := loadAuditLogEntries(configPath)
			if err != nil {
				return err
			}

			filtered, err := filterAuditEntries(entries, userFilter, actionFilter, sinceFilter)
			if err != nil {
				return err
			}

			switch strings.ToLower(strings.TrimSpace(output)) {
			case "json":
				return writeJSON(cmd, map[string]any{
					"entries": filtered,
				})
			case "text", "":
			default:
				return fmt.Errorf("unsupported output: %s (expected: text|json)", output)
			}

			if len(filtered) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "📋 監査ログ")
				fmt.Fprintln(cmd.OutOrStdout(), "===========")
				fmt.Fprintln(cmd.OutOrStdout(), "(記録なし)")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "📋 監査ログ")
			fmt.Fprintln(cmd.OutOrStdout(), "===========")
			for _, e := range filtered {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"%s  %-8s %-16s %-14s %s\n",
					formatAuditTimestamp(e.Timestamp),
					emptyFallback(e.User, "-"),
					emptyFallback(e.Action, "-"),
					emptyFallback(e.Target, "-"),
					emptyFallback(e.Result, "-"),
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&userFilter, "user", "", "Filter by user")
	cmd.Flags().StringVar(&actionFilter, "action", "", "Filter by action")
	cmd.Flags().StringVar(&sinceFilter, "since", "", "Filter entries since date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&output, "output", "text", "Output format: text|json")
	return cmd
}

func loadAuditLogEntries(configPath string) ([]auditLogEntry, error) {
	path := auditLogPathFromConfig(configPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []auditLogEntry{}, nil
		}
		return nil, fmt.Errorf("load audit log: %w", err)
	}

	var log auditLogFile
	if err := yaml.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("parse audit log: %w", err)
	}
	if log.Entries == nil {
		log.Entries = []auditLogEntry{}
	}
	return log.Entries, nil
}

func filterAuditEntries(entries []auditLogEntry, user, action, since string) ([]auditLogEntry, error) {
	user = strings.TrimSpace(user)
	action = strings.TrimSpace(action)
	since = strings.TrimSpace(since)

	var sinceDate time.Time
	if since != "" {
		parsed, err := time.ParseInLocation("2006-01-02", since, time.Local)
		if err != nil {
			return nil, fmt.Errorf("--since must be YYYY-MM-DD: %w", err)
		}
		sinceDate = parsed
	}

	filtered := make([]auditLogEntry, 0, len(entries))
	for _, e := range entries {
		if user != "" && e.User != user {
			continue
		}
		if action != "" && e.Action != action {
			continue
		}
		if !sinceDate.IsZero() {
			entryDate, ok := parseAuditDate(e.Timestamp)
			if !ok {
				continue
			}
			if entryDate.Before(sinceDate) {
				continue
			}
		}
		filtered = append(filtered, e)
	}

	return filtered, nil
}

func auditLogPathFromConfig(configPath string) string {
	root := filepath.Dir(filepath.Dir(configPath))
	return filepath.Join(root, ".teraflow", "audit-log.yml")
}

func parseAuditDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), true
		}
	}
	return time.Time{}, false
}

func formatAuditTimestamp(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.Format("2006-01-02 15:04")
		}
	}
	if len(value) >= 16 {
		return value[:16]
	}
	return value
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
