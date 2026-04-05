package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/state"
	"gopkg.in/yaml.v3"
)

var availableStages = []string{
	"initial_development",
	"release",
	"operation",
	"maintenance",
	"continuous_improvement",
	"retirement",
}

var stageAdvanceOrder = []string{
	"initial_development",
	"release",
	"operation",
	"maintenance",
	"retirement",
}

func newStageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stage",
		Short: "Manage project stages",
	}

	cmd.AddCommand(newStageListCmd())
	cmd.AddCommand(newStageStatusCmd())
	cmd.AddCommand(newStageAdvanceCmd())
	return cmd
}

func newStageListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available stages",
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
					"current": s.Lifecycle.CurrentStage,
					"stages":  availableStages,
				})
			}

			for _, st := range availableStages {
				prefix := "  "
				if st == s.Lifecycle.CurrentStage {
					prefix = "->"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", prefix, st)
			}
			return nil
		},
	}
}

func newStageStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current stage details",
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
			startedAt := "-"
			if val, err := loadLifecycleStartedAt(configPath); err == nil && strings.TrimSpace(val) != "" {
				startedAt = val
			}

			if format == "json" {
				return writeJSON(cmd, map[string]string{
					"stage":   s.Lifecycle.CurrentStage,
					"started": startedAt,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Stage: %s\n", s.Lifecycle.CurrentStage)
			fmt.Fprintf(cmd.OutOrStdout(), "Started: %s\n", startedAt)
			return nil
		},
	}
}

func newStageAdvanceCmd() *cobra.Command {
	var yes bool
	var force bool
	var reason string
	var user string
	cmd := &cobra.Command{
		Use:   "advance",
		Short: "Advance to next stage",
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
			// TODO(cmd_108/w2-f): Auto-sync in-progress SLCP-JCF process when stage/phase transitions are finalized.
			current := s.Lifecycle.CurrentStage

			next, ok := nextValue(stageAdvanceOrder, current)
			if !ok {
				if format == "json" {
					return writeJSON(cmd, map[string]any{
						"status":  "no_change",
						"current": current,
						"message": "Already at final stage.",
					})
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Already at final stage.")
				return nil
			}

			if err := runConstraintGuard(cmd, configPath, "stage.advance", force, reason, user); err != nil {
				return err
			}

			if format == "json" {
				if !yes {
					return writeJSON(cmd, map[string]any{
						"status": "confirmation_required",
						"from":   current,
						"to":     next,
					})
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Gate check: OK (Phase1: manual gates)")
			}

			if !yes {
				ok, err := askForConfirmation(cmd, fmt.Sprintf("Advance stage %s -> %s? (y/N): ", current, next))
				if err != nil {
					return err
				}
				if !ok {
					return errors.New("stage advance cancelled")
				}
			}

			s.Lifecycle.CurrentStage = next
			if err := state.SaveState(configPath, s); err != nil {
				return err
			}

			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"status": "advanced",
					"from":   current,
					"to":     next,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Advanced: %s\n", next)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&force, "force", false, "Override constraint checks")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for --force")
	cmd.Flags().StringVar(&user, "user", "", "GitHub username for RBAC check")
	return cmd
}

func askForConfirmation(cmd *cobra.Command, prompt string) (bool, error) {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func loadStateOrNotProjectErr(configPath string) (*state.ProjectState, error) {
	s, err := state.LoadState(configPath)
	if err != nil {
		var pe *os.PathError
		if errors.As(err, &pe) && errors.Is(pe.Err, os.ErrNotExist) {
			return nil, errors.New(notProjectError)
		}
		return nil, err
	}
	return s, nil
}

func loadLifecycleStartedAt(configPath string) (string, error) {
	data, err := os.ReadFile(statePathFromConfig(configPath))
	if err != nil {
		return "", err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return "", err
	}
	lifecycle, ok := raw["lifecycle"].(map[string]any)
	if !ok {
		return "", nil
	}
	startedAt, ok := lifecycle["started_at"]
	if !ok {
		return "", nil
	}
	return fmt.Sprintf("%v", startedAt), nil
}

func nextValue(order []string, current string) (string, bool) {
	for i, v := range order {
		if v == current {
			if i+1 < len(order) {
				return order[i+1], true
			}
			return "", false
		}
	}
	return "", false
}
