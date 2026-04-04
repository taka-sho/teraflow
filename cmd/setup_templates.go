package cmd

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/templates"
)

func newSetupTemplatesCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Generate GitHub Issue and Discussion templates",
		Long:  "Generate 6 Issue templates and 3 Discussion category files",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			cwd := filepath.Dir(filepath.Dir(configPath))

			issueDir := filepath.Join(cwd, ".github", "ISSUE_TEMPLATE")
			discussionDir := filepath.Join(cwd, ".github", "DISCUSSION_TEMPLATE")

			issueCount, err := copyEmbedFS(templates.IssueFS, "issues", issueDir, force)
			if err != nil {
				return fmt.Errorf("generate issue templates: %w", err)
			}
			discussionCount, err := copyEmbedFS(templates.DiscussionFS, "discussions", discussionDir, force)
			if err != nil {
				return fmt.Errorf("generate discussion templates: %w", err)
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"status":               "generated",
					"issue_templates":      issueCount,
					"discussion_templates": discussionCount,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Generated %d issue templates in %s\n", issueCount, issueDir)
			fmt.Fprintf(cmd.OutOrStdout(), "Generated %d discussion templates in %s\n", discussionCount, discussionDir)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing template files")
	return cmd
}

func copyEmbedFS(efs embed.FS, srcDir, destDir string, force bool) (int, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return 0, fmt.Errorf("create directory %s: %w", destDir, err)
	}

	count := 0
	if err := fs.WalkDir(efs, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		dest := filepath.Join(destDir, d.Name())
		if !force {
			if _, err := os.Stat(dest); err == nil {
				return fmt.Errorf("file %s already exists; use --force to overwrite", dest)
			}
		}

		data, err := efs.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return err
		}
		count++
		return nil
	}); err != nil {
		return count, err
	}

	return count, nil
}
