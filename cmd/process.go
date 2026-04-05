package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/state"
)

var slcpJCFProcessLabels = []string{
	"企画",
	"要件定義",
	"システム設計",
	"ソフトウェア設計",
	"構築",
	"テスト",
	"リリース",
	"運用保守",
}

func newProcessCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "process",
		Short: "Show SLCP-JCF process progress mapped from current stage/phase",
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

			currentIndex := mapCurrentProcessIndex(s)
			currentProcess := slcpJCFProcessLabels[currentIndex]
			nextGate := "最終プロセス到達"
			if currentIndex+1 < len(slcpJCFProcessLabels) {
				nextGate = slcpJCFProcessLabels[currentIndex+1] + "フェーズへの移行"
			}

			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"current_process": currentProcess,
					"next_gate":       nextGate,
					"stage":           s.Lifecycle.CurrentStage,
					"phase":           s.Phases.Current,
					"process_index":   currentIndex + 1,
					"process_total":   len(slcpJCFProcessLabels),
				})
			}

			fmt.Fprintln(cmd.OutOrStdout(), "SLCP-JCF プロセス状況")
			fmt.Fprintln(cmd.OutOrStdout(), "========================")
			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintln(cmd.OutOrStdout(), renderProcessTimeline(currentIndex))
			fmt.Fprintf(cmd.OutOrStdout(), "↑ 現在ここ: %s\n", currentProcess)
			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintf(cmd.OutOrStdout(), "現在のプロセス: %sプロセス\n", currentProcess)
			fmt.Fprintf(cmd.OutOrStdout(), "次のゲート: %s\n", nextGate)
			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintln(cmd.OutOrStdout(), "teraflow stage/phase の現状:")
			fmt.Fprintf(cmd.OutOrStdout(), "  Stage: %s\n", s.Lifecycle.CurrentStage)
			fmt.Fprintf(cmd.OutOrStdout(), "  Phase: %s\n", phaseWithProgress(s.Phases.Current))
			return nil
		},
	}
}

func mapCurrentProcessIndex(s *state.ProjectState) int {
	if s.SLCPJCF.CurrentProcess != "" {
		if i := findSLCPIndexByName(s.SLCPJCF.CurrentProcess); i >= 0 {
			return i
		}
	}

	switch s.Lifecycle.CurrentStage {
	case "release":
		return 6
	case "operation", "maintenance", "continuous_improvement", "retirement":
		return 7
	}

	switch s.Phases.Current {
	case "requirements":
		return 1
	case "basic_design":
		return 2
	case "detailed_design":
		return 3
	case "implementation":
		return 4
	case "testing":
		return 5
	case "integration_test":
		return 6
	default:
		return 0
	}
}

func findSLCPIndexByName(name string) int {
	for i, label := range slcpJCFProcessLabels {
		if name == label || name == label+"プロセス" {
			return i
		}
	}
	return -1
}

func renderProcessTimeline(currentIndex int) string {
	parts := make([]string, 0, len(slcpJCFProcessLabels))
	for i, label := range slcpJCFProcessLabels {
		status := "⏳"
		if i < currentIndex {
			status = "✅"
		}
		if i == currentIndex {
			status = "🔄"
		}
		parts = append(parts, fmt.Sprintf("%s %s", label, status))
	}
	return strings.Join(parts, " → ")
}

func phaseWithProgress(phase string) string {
	for i, p := range availablePhases {
		if p == phase {
			return fmt.Sprintf("%s (%d/%d)", phase, i+1, len(availablePhases))
		}
	}
	return phase
}
