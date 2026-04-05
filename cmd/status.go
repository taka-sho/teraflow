package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/state"
)

func newStatusCmd() *cobra.Command {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show current project stage and phase",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			role, err := cmd.Flags().GetString("role")
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
			if cmd.Flags().Changed("role") {
				// `--role <value>` can be parsed as an optional-argument flag where
				// the value is left in positional args on some parse paths.
				if (role == "" || role == "auto") && len(args) > 0 {
					role = args[0]
					args = args[1:]
				}
				if len(args) > 0 {
					return fmt.Errorf("status does not accept positional arguments: %s", strings.Join(args, ", "))
				}
				if role == "" || role == "auto" {
					role, err = loadLocalRole(configPath)
					if err != nil {
						return err
					}
				}
				return showRoleNextActions(cmd.OutOrStdout(), role, configPath, s)
			}

			if format == "json" {
				return writeJSON(cmd, map[string]string{
					"stage": s.Lifecycle.CurrentStage,
					"phase": s.Phases.Current,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Stage:  %s\n", s.Lifecycle.CurrentStage)
			fmt.Fprintf(cmd.OutOrStdout(), "Phase:  %s\n", s.Phases.Current)
			return nil
		},
	}
	statusCmd.Flags().StringP("role", "r", "", "show next actions for role: pm|dev|qa|release_mgr (no value: use local role)")
	if flag := statusCmd.Flags().Lookup("role"); flag != nil {
		flag.NoOptDefVal = "auto"
	}
	return statusCmd
}

func showRoleNextActions(w io.Writer, role, configPath string, s *state.ProjectState) error {
	currentStage := s.Lifecycle.CurrentStage
	currentPhase := s.Phases.Current
	projectRoot := filepath.Dir(filepath.Dir(configPath))
	ensureSLCPJCFState(s)
	currentProcess := s.SLCPJCF.CurrentProcess
	currentIndex := findSLCPIndexByName(currentProcess)
	if currentIndex < 0 {
		currentIndex = mapCurrentProcessIndex(s)
		currentProcess = slcpJCFProcessLabels[currentIndex]
	}
	currentStatus := slcpProcessStatus(s, currentIndex)
	nextProcessName, nextProcessStatus := nextSLCPProcess(s, currentIndex)

	switch strings.ToLower(role) {
	case "pm":
		fmt.Fprintln(w, "PM の次のアクション:")
		fmt.Fprintln(w, "========================")
		fmt.Fprintf(w, "現在のステージ: %s / フェーズ: %s\n\n", currentStage, currentPhase)
		fmt.Fprintln(w, "今すぐやるべきこと:")
		action := 1
		if currentStatus == "in_progress" {
			fmt.Fprintf(w, "  %d. 現プロセス[%s]のゲート承認が必要\n", action, currentProcess)
			action++
		}
		if missing := missingRequiredDocs(projectRoot, currentProcess); len(missing) > 0 {
			fmt.Fprintf(w, "  %d. ゲート前提ドキュメント未作成: %s\n", action, strings.Join(missing, ", "))
			action++
		}
		if nextProcessName != "" && nextProcessStatus == "not_started" {
			fmt.Fprintf(w, "  %d. 次のプロセス[%s]への移行準備\n", action, nextProcessName)
			action++
		}
		if action == 1 {
			fmt.Fprintln(w, "  1. 現状維持（未対応アクションなし）")
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "次のマイルストーン:")
		if nextProcessName != "" {
			fmt.Fprintf(w, "  - %s 完了ゲート\n", nextProcessName)
		} else {
			fmt.Fprintln(w, "  - 最終プロセス完了")
		}
		return nil
	case "dev":
		reworkCount, err := unresolvedReworkCount(configPath)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, "開発者 の次のアクション:")
		fmt.Fprintln(w, "==============================")
		fmt.Fprintf(w, "現在のステージ: %s / フェーズ: %s\n\n", currentStage, currentPhase)
		fmt.Fprintln(w, "今すぐやるべきこと:")
		fmt.Fprintf(w, "  1. 現phase[%s]の作業を完了せよ\n", currentPhase)
		if reworkCount > 0 {
			fmt.Fprintf(w, "  2. リワーク[%d]件を対処せよ\n", reworkCount)
		} else {
			fmt.Fprintln(w, "  2. 未解決リワークなし")
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "ブロッカー:")
		if reworkCount > 0 {
			fmt.Fprintf(w, "  - リワーク対応待ち: %d件\n", reworkCount)
		} else {
			fmt.Fprintln(w, "  - なし")
		}
		return nil
	case "qa":
		done, total := qaProgress(s)
		testingStarted := currentPhase == "testing" || currentPhase == "integration_test" || done > 0 || slcpProcessStatusByName(s, "ソフトウェアテストプロセス") == "in_progress"
		fmt.Fprintln(w, "QA の次のアクション:")
		fmt.Fprintln(w, "==========================")
		fmt.Fprintf(w, "現在のステージ: %s / フェーズ: %s\n\n", currentStage, currentPhase)
		fmt.Fprintln(w, "今すぐやるべきこと:")
		if !testingStarted {
			fmt.Fprintln(w, "  1. テスト計画書を作成せよ")
			fmt.Fprintln(w, "  2. テストフェーズ開始待ち")
		} else {
			fmt.Fprintf(w, "  1. テスト実行中 [%d/%d件完了]\n", done, total)
			fmt.Fprintln(w, "  2. テスト結果を記録・共有せよ")
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "次の担当フェーズ:")
		if currentPhase == "integration_test" {
			fmt.Fprintln(w, "  - リリース判定")
		} else {
			fmt.Fprintln(w, "  - テストフェーズ")
		}
		return nil
	case "release_mgr":
		fmt.Fprintln(w, "Release Manager の次のアクション:")
		fmt.Fprintln(w, "==================================")
		fmt.Fprintf(w, "現在のステージ: %s / フェーズ: %s\n\n", currentStage, currentPhase)
		fmt.Fprintln(w, "今すぐやるべきこと:")
		fmt.Fprintln(w, "  1. リリース判定の最終レビュー")
		fmt.Fprintln(w, "  2. ステージ遷移可否の確認")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "次のマイルストーン:")
		fmt.Fprintln(w, "  - release フェーズ完了承認")
		return nil
	default:
		return fmt.Errorf("invalid role: %q (expected: pm|dev|qa|release_mgr)", role)
	}
}

func slcpProcessStatus(s *state.ProjectState, index int) string {
	if index < 0 || index >= len(s.SLCPJCF.Processes) {
		return "not_started"
	}
	status := s.SLCPJCF.Processes[index].Status
	if status == "" {
		return "not_started"
	}
	return status
}

func nextSLCPProcess(s *state.ProjectState, currentIndex int) (string, string) {
	nextIndex := currentIndex + 1
	if nextIndex < 0 || nextIndex >= len(s.SLCPJCF.Processes) {
		return "", ""
	}
	p := s.SLCPJCF.Processes[nextIndex]
	status := p.Status
	if status == "" {
		status = "not_started"
	}
	return p.Name, status
}

func slcpProcessStatusByName(s *state.ProjectState, name string) string {
	i := findSLCPIndexByName(name)
	return slcpProcessStatus(s, i)
}

func missingRequiredDocs(projectRoot, processName string) []string {
	var candidates []string
	switch processName {
	case "要件定義プロセス":
		candidates = []string{"docs/requirements"}
	case "システム設計プロセス", "ソフトウェア設計プロセス":
		candidates = []string{"docs/design"}
	case "ソフトウェアテストプロセス", "システム結合テスト":
		candidates = []string{"docs/test-plan.md", "docs/testing"}
	default:
		return nil
	}

	missing := make([]string, 0, len(candidates))
	for _, rel := range candidates {
		if _, err := os.Stat(filepath.Join(projectRoot, rel)); errors.Is(err, os.ErrNotExist) {
			missing = append(missing, rel)
		}
	}
	return missing
}

func unresolvedReworkCount(configPath string) (int, error) {
	log, err := state.LoadReworkLog(configPath)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, rw := range log.Reworks {
		if strings.ToLower(rw.Status) != "approved" && strings.ToLower(rw.Status) != "rejected" {
			count++
		}
	}
	return count, nil
}

func qaProgress(s *state.ProjectState) (int, int) {
	targets := []string{"ソフトウェアテストプロセス", "システム結合テスト"}
	done := 0
	for _, name := range targets {
		if slcpProcessStatusByName(s, name) == "completed" {
			done++
		}
	}
	return done, len(targets)
}
