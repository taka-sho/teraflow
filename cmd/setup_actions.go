package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/actions"
	"github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/hooks"
)

func newSetupActionsCmd() *cobra.Command {
	var force bool
	var withHooks bool
	var hooksOnly bool
	cmd := &cobra.Command{
		Use:   "actions",
		Short: "Generate GitHub Actions workflow files",
		Long:  "Generate 16 teraflow workflow YAML files into .github/workflows/",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}

			cwd := filepath.Dir(filepath.Dir(configPath))
			targetDir := filepath.Join(cwd, ".github", "workflows")

			enableHooks := withHooks || hooksOnly
			hookCfg := hooks.HookConfig{}
			if enableHooks {
				cfg, err := config.Load(configPath)
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}
				hookCfg = hooks.NewParser().Parse(cfg)
			}

			planned := make([]string, 0, len(actions.WorkflowNames)+3)
			if !hooksOnly {
				planned = append(planned, actions.WorkflowNames...)
			}
			if enableHooks {
				planned = append(planned, actions.HookWorkflowNames(hookCfg)...)
			}

			if !force {
				for _, name := range planned {
					path := filepath.Join(targetDir, name+".yml")
					if _, err := os.Stat(path); err == nil {
						return fmt.Errorf("workflow %s already exists; use --force to overwrite", name+".yml")
					}
				}
			}

			if !hooksOnly {
				if err := actions.GenerateWorkflows(targetDir); err != nil {
					return fmt.Errorf("generate workflows: %w", err)
				}
			}

			if enableHooks {
				if err := actions.GenerateHookWorkflows(hookCfg, targetDir); err != nil {
					return fmt.Errorf("generate hook workflows: %w", err)
				}
			}

			names := make([]string, 0, len(planned))
			if !hooksOnly {
				templateNames, err := actions.ListTemplates()
				if err != nil {
					return fmt.Errorf("list workflows: %w", err)
				}
				names = append(names, templateNames...)
			}
			if enableHooks {
				for _, name := range actions.HookWorkflowNames(hookCfg) {
					names = append(names, name+".yml")
				}
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"status":    "generated",
					"directory": targetDir,
					"count":     len(names),
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Generated %d workflow files in %s\n", len(names), targetDir)
			for _, name := range names {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", name)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing workflow files")
	cmd.Flags().BoolVar(&withHooks, "hooks", false, "Also generate workflows from hooks section in config")
	cmd.Flags().BoolVar(&hooksOnly, "hooks-only", false, "Generate only workflows derived from hooks section in config")
	return cmd
}
