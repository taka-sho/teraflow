package cmd

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/actions"
	"github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/hooks"
)

func newUpdateCmd() *cobra.Command {
	var targetVersion string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update teraflow workflow version pin in teraflow.yml and redeploy workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			oldVersion := configuredTeraflowVersion(cfg)
			newVersion := strings.TrimSpace(targetVersion)
			if newVersion == "" {
				newVersion, err = fetchLatestTeraflowReleaseTag()
				if err != nil {
					return err
				}
			}

			cfg.TeraflowVersion = newVersion
			if err := config.Save(configPath, cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			cwd := filepath.Dir(filepath.Dir(configPath))
			targetDir := filepath.Join(cwd, ".github", "workflows")

			if err := actions.GenerateWorkflows(targetDir, newVersion); err != nil {
				return fmt.Errorf("deploy workflows: %w", err)
			}

			hookCfg := hooks.NewParser().Parse(cfg)
			if err := actions.GenerateHookWorkflows(hookCfg, targetDir, newVersion); err != nil {
				return fmt.Errorf("deploy hook workflows: %w", err)
			}

			updatedFiles := []string{".github/teraflow.yml"}
			for _, name := range actions.WorkflowNames {
				updatedFiles = append(updatedFiles, filepath.ToSlash(filepath.Join(".github", "workflows", name+".yml")))
			}
			for _, name := range actions.HookWorkflowNames(hookCfg) {
				updatedFiles = append(updatedFiles, filepath.ToSlash(filepath.Join(".github", "workflows", name+".yml")))
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"status":        "updated",
					"old_version":   oldVersion,
					"new_version":   newVersion,
					"updated_files": updatedFiles,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updated teraflow version: %s -> %s\n", oldVersion, newVersion)
			fmt.Fprintln(cmd.OutOrStdout(), "Updated files:")
			for _, file := range updatedFiles {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", file)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&targetVersion, "version", "", "Target teraflow version tag (e.g. v0.5.5)")
	return cmd
}

func fetchLatestTeraflowReleaseTag() (string, error) {
	ghCmd := ghExecCommand("gh", "api", "repos/taka-sho/teraflow/releases/latest", "--jq", ".tag_name")
	var stderr bytes.Buffer
	ghCmd.Stderr = &stderr
	out, err := ghCmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("fetch latest release tag: %s", msg)
	}

	tag := strings.TrimSpace(string(out))
	if tag == "" || tag == "null" {
		return "", fmt.Errorf("fetch latest release tag: empty response")
	}
	return tag, nil
}
