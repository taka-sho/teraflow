package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
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
				// With NoOptDefVal on a string flag, `--role pm` may parse as
				// role="auto" and args=["pm"]. Treat the first positional arg
				// as the explicit role value in that case.
				if role == "auto" && len(args) > 0 {
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
				return showRoleNextActions(cmd.OutOrStdout(), role, s)
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

func showRoleNextActions(w io.Writer, role string, s *state.ProjectState) error {
	currentStage := s.Lifecycle.CurrentStage
	currentPhase := s.Phases.Current

	switch strings.ToLower(role) {
	case "pm":
		fmt.Fprintln(w, "PM の次のアクション:")
		fmt.Fprintln(w, "========================")
		fmt.Fprintf(w, "現在のステージ: %s / フェーズ: %s\n\n", currentStage, currentPhase)
		fmt.Fprintln(w, "今すぐやるべきこと:")
		fmt.Fprintln(w, "  1. 設計レビューの承認（ゲート承認権限が必要）")
		fmt.Fprintln(w, "  2. テスト計画のレビュー依頼")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "次のマイルストーン:")
		fmt.Fprintln(w, "  - 詳細設計フェーズ完了ゲート")
		return nil
	case "dev":
		fmt.Fprintln(w, "開発者 の次のアクション:")
		fmt.Fprintln(w, "==============================")
		fmt.Fprintf(w, "現在のステージ: %s / フェーズ: %s\n\n", currentStage, currentPhase)
		fmt.Fprintln(w, "今すぐやるべきこと:")
		fmt.Fprintln(w, "  1. 詳細設計書の作成")
		fmt.Fprintln(w, "  2. インタフェース定義の確定")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "ブロッカー:")
		fmt.Fprintln(w, "  - なし")
		return nil
	case "qa":
		fmt.Fprintln(w, "QA の次のアクション:")
		fmt.Fprintln(w, "==========================")
		fmt.Fprintf(w, "現在のステージ: %s / フェーズ: %s\n\n", currentStage, currentPhase)
		fmt.Fprintln(w, "今すぐやるべきこと:")
		fmt.Fprintln(w, "  1. テスト計画書の作成開始")
		fmt.Fprintln(w, "  2. テスト環境の準備")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "次の担当フェーズ:")
		fmt.Fprintln(w, "  - テストフェーズ（現在: 3フェーズ先）")
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
