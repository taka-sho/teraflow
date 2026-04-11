package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/state"
)

type initOptions struct {
	Name           string
	Stage          string
	Path           string
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
			targetPath := strings.TrimSpace(opts.Path)
			if targetPath == "" {
				targetPath = cwd
			} else if !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(cwd, targetPath)
			}
			return runInit(*opts, targetPath)
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "Project name")
	cmd.Flags().StringVar(&opts.Stage, "stage", "initial_development", "Starting stage")
	cmd.Flags().StringVar(&opts.Path, "path", "", "Target directory to initialize")
	cmd.Flags().BoolVar(&opts.NonInteractive, "non-interactive", false, "Run without prompts")
	return cmd
}

func runInit(opts initOptions, cwd string) error {
	if opts.Stage == "" {
		opts.Stage = "initial_development"
	}

	createdDir := false
	dirInfo, err := os.Stat(cwd)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(cwd, 0o755); err != nil {
			return err
		}
		createdDir = true
	} else if !dirInfo.IsDir() {
		return fmt.Errorf("target path is not a directory: %s", cwd)
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
	if err := os.MkdirAll(filepath.Join(cwd, ".teraflow", "discovery"), 0o755); err != nil {
		return err
	}

	if err := state.SaveState(cfgPath, state.NewProjectState(name, opts.Stage)); err != nil {
		return err
	}

	if createdDir {
		fmt.Printf("ディレクトリを作成しました: %s\n", cwd)
	}
	fmt.Println("Initialized teraflow project files:")
	for path := range targets {
		rel, _ := filepath.Rel(cwd, path)
		fmt.Printf("  - %s\n", rel)
	}
	fmt.Println("  - .github/project-state.yml")
	fmt.Println("  - .teraflow/discovery/")
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

hooks:
  on_push:
    - action: index_update
    - action: summary_update
  on_discussion_created:
    - action: respond
      skill: requirements
  on_discussion_comment:
    - action: respond
      skill: requirements
`, name)
}

const defaultGoldenPrinciples = `# Golden Principles

1. Keep requirements traceable from issue to implementation.
2. Prefer small, reversible changes.
3. Verify with tests before merge.
4. Document decisions in ADRs.
`
