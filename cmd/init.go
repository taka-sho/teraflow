package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type initOptions struct {
	Name           string
	Stage          string
	NonInteractive bool
}

func newInitCmd() *cobra.Command {
	opts := &initOptions{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a teraflow project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.NonInteractive && strings.TrimSpace(opts.Name) == "" {
				return fmt.Errorf("--name is required when --non-interactive is set")
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			return runInit(*opts, cwd)
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "Project name")
	cmd.Flags().StringVar(&opts.Stage, "stage", "initial_development", "Starting stage")
	cmd.Flags().BoolVar(&opts.NonInteractive, "non-interactive", false, "Run without prompts")
	return cmd
}

func runInit(opts initOptions, cwd string) error {
	if opts.Stage == "" {
		opts.Stage = "initial_development"
	}

	cfgPath := filepath.Join(cwd, ".github", "teraflow.yml")
	if _, err := os.Stat(cfgPath); err == nil {
		return fmt.Errorf("project already initialized: %s exists", cfgPath)
	}

	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = filepath.Base(cwd)
	}

	targets := map[string]string{
		filepath.Join(cwd, ".github", "teraflow.yml"):                       renderTeraflowYAML(name),
		filepath.Join(cwd, ".github", "project-state.yml"):                  renderProjectStateYAML(name, opts.Stage),
		filepath.Join(cwd, "docs", "shared", "01_requirements", "index.md"): "# Requirements\n\n- Add requirements here.\n",
		filepath.Join(cwd, "docs", "golden-principles.md"):                  defaultGoldenPrinciples,
	}

	for path, content := range targets {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}

	fmt.Println("Initialized teraflow project files:")
	for path := range targets {
		rel, _ := filepath.Rel(cwd, path)
		fmt.Printf("  - %s\n", rel)
	}
	return nil
}

func renderTeraflowYAML(name string) string {
	return fmt.Sprintf(`version: "1"

project:
  name: %q
  description: ""
  repository: ""

confirmation:
  trigger: "確定"
  req_trigger: "要求確定"

ai:
  default_provider: anthropic

harness:
  score_threshold: 70
  auto_issue: false
`, name)
}

func renderProjectStateYAML(name, stage string) string {
	return fmt.Sprintf(`project:
  name: %q

lifecycle:
  current_stage: %q

phases:
  current: "requirements"
`, name, stage)
}

const defaultGoldenPrinciples = `# Golden Principles

1. Keep requirements traceable from issue to implementation.
2. Prefer small, reversible changes.
3. Verify with tests before merge.
4. Document decisions in ADRs.
`
