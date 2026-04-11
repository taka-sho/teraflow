package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/graphbridge"
	indexpkg "github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/validate"
)

func newValidateCmd() *cobra.Command {
	var full bool
	var phase string
	var nodeID string
	var level int

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate CoDD consistency (schema/reference/phase/graph)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if level < 1 || level > 4 {
				return fmt.Errorf("--level must be 1..4")
			}
			if phase != "" && nodeID != "" {
				return fmt.Errorf("--phase and --node cannot be used together")
			}

			projectRoot, err := projectRootFromCmd(cmd)
			if err != nil {
				return err
			}
			idx, err := indexpkg.NewBuilder(projectRoot).LoadIndex()
			if err != nil {
				return err
			}

			validator := validate.NewValidator(idx, graphbridge.New(projectRoot), projectRoot)
			validator.SetLevel(level)

			var result validate.ValidationResult
			switch {
			case strings.TrimSpace(nodeID) != "":
				result = validator.ValidateArtifact(strings.TrimSpace(nodeID))
			case strings.TrimSpace(phase) != "":
				result = validator.ValidatePhaseTransition(strings.TrimSpace(phase), strings.TrimSpace(phase))
			case full:
				result = validator.ValidatePhaseTransition("", "")
			default:
				result = validator.ValidatePhaseTransition("", "")
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				payload := map[string]any{
					"valid":         result.Valid,
					"error_count":   len(result.Errors),
					"warning_count": len(result.Warnings),
					"errors":        result.Errors,
					"warnings":      result.Warnings,
				}
				if err := writeJSON(cmd, payload); err != nil {
					return err
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Valid: %t\n", result.Valid)
				fmt.Fprintf(cmd.OutOrStdout(), "Errors: %d\n", len(result.Errors))
				fmt.Fprintf(cmd.OutOrStdout(), "Warnings: %d\n", len(result.Warnings))
				for _, e := range sortErrors(result.Errors) {
					fmt.Fprintf(cmd.OutOrStdout(), "[ERROR][L%d] %s %s\n", e.Level, e.NodeID, e.Message)
				}
				for _, w := range sortWarnings(result.Warnings) {
					fmt.Fprintf(cmd.OutOrStdout(), "[WARN][L%d] %s %s\n", w.Level, w.NodeID, w.Message)
				}
			}

			if len(result.Errors) > 0 {
				return newExitCode(2)
			}
			if len(result.Warnings) > 0 {
				return newExitCode(1)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&full, "full", false, "Validate full project")
	cmd.Flags().StringVar(&phase, "phase", "", "Validate a specific phase")
	cmd.Flags().StringVar(&nodeID, "node", "", "Validate a single node_id")
	cmd.Flags().IntVar(&level, "level", 4, "Maximum validation level (1-4)")
	return cmd
}

func sortErrors(in []validate.ValidationError) []validate.ValidationError {
	out := append([]validate.ValidationError(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Level == out[j].Level {
			if out[i].NodeID == out[j].NodeID {
				return out[i].ErrorType < out[j].ErrorType
			}
			return out[i].NodeID < out[j].NodeID
		}
		return out[i].Level < out[j].Level
	})
	return out
}

func sortWarnings(in []validate.ValidationWarning) []validate.ValidationWarning {
	out := append([]validate.ValidationWarning(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Level == out[j].Level {
			if out[i].NodeID == out[j].NodeID {
				return out[i].WarningType < out[j].WarningType
			}
			return out[i].NodeID < out[j].NodeID
		}
		return out[i].Level < out[j].Level
	})
	return out
}
