package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/actions"
)

func newSetupActionsCmd() *cobra.Command {
	var force bool
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

			if !force {
				for _, name := range actions.WorkflowNames {
					path := filepath.Join(targetDir, name+".yml")
					if _, err := os.Stat(path); err == nil {
						return fmt.Errorf("workflow %s already exists; use --force to overwrite", name+".yml")
					}
				}
			}

			if err := actions.GenerateWorkflows(targetDir); err != nil {
				return fmt.Errorf("generate workflows: %w", err)
			}

			names, err := actions.ListTemplates()
			if err != nil {
				return fmt.Errorf("list workflows: %w", err)
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
	return cmd
}
