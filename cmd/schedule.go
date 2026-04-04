package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/state"
	"gopkg.in/yaml.v3"
)

type scheduleFile struct {
	Version   string `yaml:"version"`
	UpdatedAt string `yaml:"updated_at"`
	Stages    struct {
		Current      string `yaml:"current"`
		CurrentPhase string `yaml:"current_phase"`
	} `yaml:"stages"`
}

func newScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Show and update master schedule",
	}

	cmd.AddCommand(newScheduleShowCmd())
	cmd.AddCommand(newScheduleUpdateCmd())
	return cmd
}

func newScheduleShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show master schedule",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			path := schedulePathFromConfig(configPath)
			data, err := os.ReadFile(path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					fmt.Fprintln(cmd.OutOrStdout(), "No schedule defined yet. Run `teraflow init` to create one.")
					return nil
				}
				return fmt.Errorf("read master schedule: %w", err)
			}

			if format == "json" {
				var raw any
				if err := yaml.Unmarshal(data, &raw); err != nil {
					return fmt.Errorf("parse master schedule: %w", err)
				}
				return writeJSON(cmd, raw)
			}

			if len(data) > 0 {
				if _, err := cmd.OutOrStdout().Write(data); err != nil {
					return err
				}
				if data[len(data)-1] != '\n' {
					fmt.Fprintln(cmd.OutOrStdout())
				}
			}
			return nil
		},
	}
}

func newScheduleUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update master schedule with current stage and phase",
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

			schedule := scheduleFile{Version: "1"}
			path := schedulePathFromConfig(configPath)
			if data, readErr := os.ReadFile(path); readErr == nil {
				if err := yaml.Unmarshal(data, &schedule); err != nil {
					return fmt.Errorf("parse master schedule: %w", err)
				}
			} else if !errors.Is(readErr, os.ErrNotExist) {
				return fmt.Errorf("read master schedule: %w", readErr)
			}

			if schedule.Version == "" {
				schedule.Version = "1"
			}
			schedule.UpdatedAt = time.Now().Format("2006-01-02T15:04:05")
			schedule.Stages.Current = s.Lifecycle.CurrentStage
			schedule.Stages.CurrentPhase = s.Phases.Current

			data, err := yaml.Marshal(&schedule)
			if err != nil {
				return fmt.Errorf("marshal master schedule: %w", err)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("create schedule directory: %w", err)
			}
			if err := os.WriteFile(path, data, 0o644); err != nil {
				return fmt.Errorf("write master schedule: %w", err)
			}

			if format == "json" {
				return writeJSON(cmd, map[string]string{
					"status":        "updated",
					"stage":         s.Lifecycle.CurrentStage,
					"phase":         s.Phases.Current,
					"updated_at":    schedule.UpdatedAt,
					"schedule_path": path,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Schedule updated: stage=%s, phase=%s\n", s.Lifecycle.CurrentStage, s.Phases.Current)
			return nil
		},
	}
}

func schedulePathFromConfig(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "master-schedule.yml")
}
