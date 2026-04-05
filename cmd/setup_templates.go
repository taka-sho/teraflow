package cmd

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	cfgpkg "github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/templates"
	"gopkg.in/yaml.v3"
)

func newSetupTemplatesCmd() *cobra.Command {
	var force bool
	var sync bool
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Generate GitHub Issue and Discussion templates",
		Long:  "Generate Issue templates and Discussion category files",
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

			var syncResult *discussionCategorySyncResult
			if sync {
				syncResult, err = syncDiscussionCategories(configPath)
				if err != nil {
					return fmt.Errorf("sync discussion categories: %w", err)
				}
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				out := map[string]any{
					"status":               "generated",
					"issue_templates":      issueCount,
					"discussion_templates": discussionCount,
				}
				if syncResult != nil {
					out["sync"] = map[string]any{
						"skipped":          syncResult.Skipped,
						"skip_reason":      syncResult.SkipReason,
						"repository_owner": syncResult.RepositoryOwner,
						"repository_name":  syncResult.RepositoryName,
						"existing":         syncResult.Existing,
						"missing":          syncResult.Missing,
					}
				}
				return writeJSON(cmd, out)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Generated %d issue templates in %s\n", issueCount, issueDir)
			fmt.Fprintf(cmd.OutOrStdout(), "Generated %d discussion templates in %s\n", discussionCount, discussionDir)
			if syncResult != nil {
				printDiscussionCategorySyncResult(cmd, syncResult)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing template files")
	cmd.Flags().BoolVar(&sync, "sync", false, "Check Discussion categories and show creation guide")
	return cmd
}

type discussionCategoryConfig struct {
	Categories []discussionCategoryDefinition `yaml:"categories"`
}

type discussionCategoryDefinition struct {
	Name        string `yaml:"name" json:"name"`
	Emoji       string `yaml:"emoji" json:"emoji"`
	Description string `yaml:"description" json:"description"`
	Format      string `yaml:"format" json:"format"`
}

type discussionCategorySyncResult struct {
	Skipped         bool                           `json:"skipped"`
	SkipReason      string                         `json:"skip_reason,omitempty"`
	RepositoryOwner string                         `json:"repository_owner,omitempty"`
	RepositoryName  string                         `json:"repository_name,omitempty"`
	Existing        []discussionCategoryDefinition `json:"existing,omitempty"`
	Missing         []discussionCategoryDefinition `json:"missing,omitempty"`
}

func loadDiscussionCategoryConfig() (*discussionCategoryConfig, error) {
	data, err := templates.DiscussionCategoryFS.ReadFile("discussions/categories.yml")
	if err != nil {
		return nil, fmt.Errorf("read categories.yml: %w", err)
	}

	var cfg discussionCategoryConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse categories.yml: %w", err)
	}
	return &cfg, nil
}

func syncDiscussionCategories(configPath string) (*discussionCategorySyncResult, error) {
	cfg, err := loadDiscussionCategoryConfig()
	if err != nil {
		return nil, err
	}

	if _, err := ghLookPath("gh"); err != nil {
		return &discussionCategorySyncResult{
			Skipped:    true,
			SkipReason: "gh CLI is not installed",
		}, nil
	}

	owner, repo, err := resolveRepositoryInfo(configPath)
	if err != nil {
		return nil, err
	}

	current, err := fetchDiscussionCategories(owner, repo)
	if err != nil {
		return &discussionCategorySyncResult{
			Skipped:         true,
			SkipReason:      fmt.Sprintf("failed to query GitHub discussion categories: %v", err),
			RepositoryOwner: owner,
			RepositoryName:  repo,
		}, nil
	}

	existingByName := make(map[string]struct{}, len(current))
	for _, category := range current {
		existingByName[category.Name] = struct{}{}
	}

	result := &discussionCategorySyncResult{
		RepositoryOwner: owner,
		RepositoryName:  repo,
	}
	for _, category := range cfg.Categories {
		if _, ok := existingByName[category.Name]; ok {
			result.Existing = append(result.Existing, category)
			continue
		}
		result.Missing = append(result.Missing, category)
	}
	return result, nil
}

func printDiscussionCategorySyncResult(cmd *cobra.Command, result *discussionCategorySyncResult) {
	if result.Skipped {
		fmt.Fprintf(cmd.OutOrStdout(), "Discussion category check skipped: %s\n", result.SkipReason)
		return
	}

	for _, category := range result.Existing {
		fmt.Fprintf(cmd.OutOrStdout(), "  ✅ %s %s\n", category.Name, category.Emoji)
	}
	for _, category := range result.Missing {
		fmt.Fprintf(cmd.OutOrStdout(), "  ❌ %s %s\n", category.Name, category.Emoji)
	}

	if len(result.Missing) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "All configured Discussion categories are present.")
		return
	}

	fmt.Fprintln(cmd.OutOrStdout(), "\nThe following Discussion categories need to be created manually:")
	for _, category := range result.Missing {
		fmt.Fprintf(cmd.OutOrStdout(), "  ❌ %s %s\n", category.Name, category.Emoji)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "\nTo create them, go to:")
	fmt.Fprintf(cmd.OutOrStdout(), "  https://github.com/%s/%s/settings/discussions\n", result.RepositoryOwner, result.RepositoryName)
	fmt.Fprintln(cmd.OutOrStdout(), "Then add each category with the following settings:")
	for _, category := range result.Missing {
		fmt.Fprintf(cmd.OutOrStdout(), "  - Name: %s\n", category.Name)
		fmt.Fprintf(cmd.OutOrStdout(), "  - Emoji: %s\n", category.Emoji)
		fmt.Fprintf(cmd.OutOrStdout(), "  - Description: %s\n", category.Description)
		fmt.Fprintf(cmd.OutOrStdout(), "  - Format: %s\n", discussionFormatLabel(category.Format))
	}
}

func discussionFormatLabel(format string) string {
	switch strings.ToUpper(strings.TrimSpace(format)) {
	case "OPEN":
		return "Open-ended discussion"
	case "ANNOUNCEMENT":
		return "Announcements"
	case "QANDA":
		return "Question and answer"
	default:
		if strings.TrimSpace(format) == "" {
			return "Open-ended discussion"
		}
		return format
	}
}

func resolveRepositoryInfo(configPath string) (string, string, error) {
	ghCmd := ghExecCommand("gh", "repo", "view", "--json", "owner,name")
	var stderr bytes.Buffer
	ghCmd.Stderr = &stderr
	out, err := ghCmd.Output()
	if err == nil {
		var response struct {
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
			Name string `json:"name"`
		}
		if unmarshalErr := json.Unmarshal(out, &response); unmarshalErr == nil {
			if response.Owner.Login != "" && response.Name != "" {
				return response.Owner.Login, response.Name, nil
			}
		}
	}

	cfg, cfgErr := cfgpkg.Load(configPath)
	if cfgErr == nil {
		owner, repo, parseErr := parseGitHubRepository(cfg.Project.Repository)
		if parseErr == nil {
			return owner, repo, nil
		}
	}

	if err != nil {
		return "", "", fmt.Errorf("resolve repository (gh failed: %v; stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return "", "", fmt.Errorf("resolve repository from config failed")
}

func parseGitHubRepository(repository string) (string, string, error) {
	normalized := strings.TrimSpace(repository)
	if normalized == "" {
		return "", "", fmt.Errorf("empty repository")
	}

	normalized = strings.TrimPrefix(normalized, "https://github.com/")
	normalized = strings.TrimPrefix(normalized, "http://github.com/")
	normalized = strings.TrimPrefix(normalized, "ssh://git@github.com/")
	normalized = strings.TrimPrefix(normalized, "git@github.com:")
	normalized = strings.TrimPrefix(normalized, "github.com/")
	normalized = strings.TrimSuffix(normalized, ".git")
	normalized = strings.Trim(normalized, "/")

	parts := strings.Split(normalized, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("repository must be in owner/repo format")
	}
	return parts[0], parts[1], nil
}

func fetchDiscussionCategories(owner, repo string) ([]discussionCategoryDefinition, error) {
	const query = "query($owner: String!, $repo: String!) { repository(owner: $owner, name: $repo) { discussionCategories(first: 25) { nodes { name emoji description } } } }"

	ghCmd := ghExecCommand("gh", "api", "graphql",
		"-f", "query="+query,
		"-f", "owner="+owner,
		"-f", "repo="+repo,
	)
	var stderr bytes.Buffer
	ghCmd.Stderr = &stderr
	out, err := ghCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}

	var response struct {
		Data struct {
			Repository struct {
				DiscussionCategories struct {
					Nodes []discussionCategoryDefinition `json:"nodes"`
				} `json:"discussionCategories"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &response); err != nil {
		return nil, fmt.Errorf("parse GraphQL response: %w", err)
	}
	return response.Data.Repository.DiscussionCategories.Nodes, nil
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
