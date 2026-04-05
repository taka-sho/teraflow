package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/rbac"
	"github.com/taka-sho/teraflow/internal/state"
)

var slcpJCFProcessLabels = []string{
	"企画プロセス",
	"要件定義プロセス",
	"システム設計プロセス",
	"ソフトウェア設計プロセス",
	"ソフトウェア構築プロセス",
	"ソフトウェアテストプロセス",
	"システム結合テスト",
	"運用・保守プロセス",
}

var slcpJCFProcessAliases = map[string]string{
	"企画":                    "企画プロセス",
	"要件定義":                  "要件定義プロセス",
	"システム設計":                "システム設計プロセス",
	"ソフトウェア設計":              "ソフトウェア設計プロセス",
	"構築":                    "ソフトウェア構築プロセス",
	"テスト":                   "ソフトウェアテストプロセス",
	"リリース":                  "システム結合テスト",
	"結合テスト":                 "システム結合テスト",
	"運用保守":                  "運用・保守プロセス",
	"planning":              "企画プロセス",
	"requirements":          "要件定義プロセス",
	"system_design":         "システム設計プロセス",
	"software_design":       "ソフトウェア設計プロセス",
	"implementation":        "ソフトウェア構築プロセス",
	"testing":               "ソフトウェアテストプロセス",
	"integration_test":      "システム結合テスト",
	"operation_maintenance": "運用・保守プロセス",
}

func newProcessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "process",
		Short: "Manage SLCP-JCF process state",
		RunE:  runProcessList,
	}

	cmd.AddCommand(newProcessListCmd())
	cmd.AddCommand(newProcessStartCmd())
	cmd.AddCommand(newProcessCompleteCmd())
	return cmd
}

func newProcessListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show SLCP-JCF process progress mapped from current stage/phase",
		RunE:  runProcessList,
	}
}

func newProcessStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <process_name>",
		Short: "Start a process",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			processName, err := normalizeProcessName(args[0])
			if err != nil {
				return err
			}
			if err := checkProcessPermission(configPath, "process.start.*"); err != nil {
				return err
			}

			s, err := loadStateOrNotProjectErr(configPath)
			if err != nil {
				return err
			}
			ensureSLCPJCFState(s)
			index := findSLCPIndexByName(processName)
			if index < 0 {
				return fmt.Errorf("unknown process: %s", processName)
			}
			if s.SLCPJCF.Processes[index].Status == "completed" {
				return fmt.Errorf("process already completed: %s", processName)
			}
			now := time.Now().Format(time.RFC3339)
			s.SLCPJCF.Processes[index].Status = "in_progress"
			s.SLCPJCF.Processes[index].StartedAt = now
			s.SLCPJCF.CurrentProcess = processName
			s.SLCPJCF.UpdatedAt = now
			return state.SaveState(configPath, s)
		},
	}
}

func newProcessCompleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "complete <process_name>",
		Short: "Complete a process",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			processName, err := normalizeProcessName(args[0])
			if err != nil {
				return err
			}
			if err := checkProcessPermission(configPath, "process.complete.*"); err != nil {
				return err
			}

			s, err := loadStateOrNotProjectErr(configPath)
			if err != nil {
				return err
			}
			ensureSLCPJCFState(s)
			index := findSLCPIndexByName(processName)
			if index < 0 {
				return fmt.Errorf("unknown process: %s", processName)
			}
			if s.SLCPJCF.Processes[index].Status != "in_progress" {
				return fmt.Errorf("process must be in_progress: %s", processName)
			}
			now := time.Now().Format(time.RFC3339)
			s.SLCPJCF.Processes[index].Status = "completed"
			s.SLCPJCF.Processes[index].CompletedAt = now
			if index+1 < len(s.SLCPJCF.Processes) {
				s.SLCPJCF.CurrentProcess = s.SLCPJCF.Processes[index+1].Name
			} else {
				s.SLCPJCF.CurrentProcess = ""
			}
			s.SLCPJCF.UpdatedAt = now
			return state.SaveState(configPath, s)
		},
	}
}

func runProcessList(cmd *cobra.Command, _ []string) error {
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
	fmt.Fprintf(cmd.OutOrStdout(), "現在のプロセス: %s\n", currentProcess)
	fmt.Fprintf(cmd.OutOrStdout(), "次のゲート: %s\n", nextGate)
	fmt.Fprintln(cmd.OutOrStdout(), "")
	fmt.Fprintln(cmd.OutOrStdout(), "teraflow stage/phase の現状:")
	fmt.Fprintf(cmd.OutOrStdout(), "  Stage: %s\n", s.Lifecycle.CurrentStage)
	fmt.Fprintf(cmd.OutOrStdout(), "  Phase: %s\n", phaseWithProgress(s.Phases.Current))
	return nil
}

func ensureSLCPJCFState(s *state.ProjectState) {
	if len(s.SLCPJCF.Processes) == 0 {
		s.SLCPJCF.Processes = state.DefaultSLCPJCFProcesses()
	}
	if s.SLCPJCF.CurrentProcess != "" {
		return
	}
	for _, p := range s.SLCPJCF.Processes {
		if p.Status == "in_progress" {
			s.SLCPJCF.CurrentProcess = p.Name
			return
		}
	}
	for _, p := range s.SLCPJCF.Processes {
		if p.Status != "completed" {
			s.SLCPJCF.CurrentProcess = p.Name
			return
		}
	}
}

func normalizeProcessName(input string) (string, error) {
	normalized := strings.TrimSpace(input)
	if normalized == "" {
		return "", errors.New("process name is required")
	}
	if canonical, ok := slcpJCFProcessAliases[normalized]; ok {
		return canonical, nil
	}
	for _, name := range slcpJCFProcessLabels {
		if normalized == name {
			return name, nil
		}
	}
	return "", fmt.Errorf("unknown process: %s", input)
}

func checkProcessPermission(configPath, permission string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	if !cfg.RBAC.Enabled {
		return nil
	}
	user, err := rbac.CurrentUser()
	if err != nil {
		return err
	}
	engine := rbac.NewEngine(cfg.RBAC)
	if engine.CheckPermission(user, permission) {
		return nil
	}
	return fmt.Errorf("permission denied: user %q lacks %s", user, permission)
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
	if normalized, err := normalizeProcessName(name); err == nil {
		name = normalized
	}
	for i, label := range slcpJCFProcessLabels {
		if name == label {
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
