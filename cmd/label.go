package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var ghLookPath = exec.LookPath
var ghExecCommand = exec.Command

type labelDefinition struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description"`
}

type labelsConfig struct {
	Labels []labelDefinition `yaml:"labels"`
}

type labelSyncOptions struct {
	DryRun bool
	Force  bool
}

func newLabelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "label",
		Short: "Manage GitHub labels",
	}

	cmd.AddCommand(newLabelListCmd())
	cmd.AddCommand(newLabelSyncCmd())
	return cmd
}

func newLabelListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured labels",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}

			labels, fromConfig, err := loadLabels(configPath)
			if err != nil {
				return err
			}

			if !fromConfig {
				fmt.Fprintln(cmd.OutOrStdout(), "Default teraflow labels:")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Configured teraflow labels:")
			}
			for _, label := range labels {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-22s — %s\n", label.Name, label.Description)
			}
			return nil
		},
	}

	return cmd
}

func newLabelSyncCmd() *cobra.Command {
	opts := &labelSyncOptions{}
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync labels to GitHub repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ensureGHAuthenticated(); err != nil {
				return err
			}

			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			labels, _, err := loadLabels(configPath)
			if err != nil {
				return err
			}

			for _, label := range labels {
				if opts.DryRun {
					fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] gh label create %q --color %q --description %q\n", label.Name, label.Color, label.Description)
					if opts.Force {
						fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] gh label edit %q --color %q --description %q\n", label.Name, label.Color, label.Description)
					}
					continue
				}

				if err := createOrUpdateLabel(label, opts.Force); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show commands without executing")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Overwrite existing labels")
	return cmd
}

func createOrUpdateLabel(label labelDefinition, force bool) error {
	createErr := runGHCommand("label", "create", label.Name, "--color", label.Color, "--description", label.Description)
	if createErr == nil {
		fmt.Fprintf(os.Stdout, "✓ Created label: %s\n", label.Name)
		return nil
	}

	if !strings.Contains(strings.ToLower(createErr.Error()), "already exists") {
		return fmt.Errorf("E5003: GitHub API error: %w", createErr)
	}

	if !force {
		fmt.Fprintf(os.Stdout, "• Label exists: %s (use --force to update)\n", label.Name)
		return nil
	}

	if err := runGHCommand("label", "edit", label.Name, "--color", label.Color, "--description", label.Description); err != nil {
		return fmt.Errorf("E5003: GitHub API error: %w", err)
	}
	fmt.Fprintf(os.Stdout, "✓ Updated label: %s\n", label.Name)
	return nil
}

func loadLabels(configPath string) ([]labelDefinition, bool, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return defaultLabels(), false, nil
	}

	var cfg labelsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, false, fmt.Errorf("E0003: Configuration file is invalid: %w", err)
	}
	if len(cfg.Labels) == 0 {
		return defaultLabels(), false, nil
	}

	out := make([]labelDefinition, 0, len(cfg.Labels))
	for _, label := range cfg.Labels {
		if strings.TrimSpace(label.Name) == "" {
			continue
		}
		if strings.TrimSpace(label.Color) == "" {
			label.Color = "1D76DB"
		}
		out = append(out, label)
	}
	if len(out) == 0 {
		return defaultLabels(), false, nil
	}
	return out, true, nil
}

func defaultLabels() []labelDefinition {
	return []labelDefinition{
		{Name: "stage:initial-dev", Color: "0E8A16", Description: "初期開発ステージ"},
		{Name: "stage:release", Color: "5319E7", Description: "リリースステージ"},
		{Name: "phase:requirements", Color: "1D76DB", Description: "要件定義フェーズ"},
		{Name: "phase:basic-design", Color: "0052CC", Description: "基本設計フェーズ"},
		{Name: "phase:detailed-design", Color: "5319E7", Description: "詳細設計フェーズ"},
		{Name: "phase:implementation", Color: "0E8A16", Description: "実装フェーズ"},
		{Name: "phase:testing", Color: "FBCA04", Description: "テストフェーズ"},
		{Name: "phase:integration", Color: "B60205", Description: "結合テストフェーズ"},
		{Name: "type:rework", Color: "D93F0B", Description: "手戻り"},
		{Name: "type:incident", Color: "B60205", Description: "障害"},
		{Name: "type:confirmed", Color: "0E8A16", Description: "確定"},
	}
}

func ensureGHAuthenticated() error {
	if _, err := ghLookPath("gh"); err != nil {
		return fmt.Errorf("E5001: GitHub CLI (gh) is not installed.")
	}

	authCmd := ghExecCommand("gh", "auth", "status")
	var stderr bytes.Buffer
	authCmd.Stdout = os.Stdout
	authCmd.Stderr = &stderr
	if err := authCmd.Run(); err != nil {
		return fmt.Errorf("E5002: GitHub CLI is not authenticated. Run: gh auth login")
	}
	return nil
}

func runGHCommand(args ...string) error {
	ghCmd := ghExecCommand("gh", args...)
	ghCmd.Stdin = os.Stdin
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr
	return ghCmd.Run()
}
