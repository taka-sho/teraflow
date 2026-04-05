package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/agent"
	cfgpkg "github.com/taka-sho/teraflow/internal/config"
	docpkg "github.com/taka-sho/teraflow/internal/doc"
	indexpkg "github.com/taka-sho/teraflow/internal/index"
	"gopkg.in/yaml.v3"
)

type docGenerator interface {
	Generate(context.Context, docpkg.GenerateRequest) (*docpkg.GenerateResult, error)
	PostDiscussionComment(context.Context, string, *docpkg.GenerateResult) error
}

var docLoadConfig = cfgpkg.Load
var docResolveProviderForType = cfgpkg.ResolveProviderForType
var docNewProviderFromConfig = agent.NewProviderFromConfig
var docNewGenerator = func(provider agent.Provider, projectRoot string, dryRun bool) docGenerator {
	return docpkg.NewGenerator(provider, projectRoot, dryRun)
}

func newDocCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "doc", Short: "Manage CoDD documents"}
	cmd.AddCommand(newDocGenerateCmd())
	cmd.AddCommand(newDocListCmd())
	return cmd
}

func newDocGenerateCmd() *cobra.Command {
	var discussion string
	var dryRun bool
	var outputDir string
	var createPR bool

	cmd := &cobra.Command{
		Use:   "generate --discussion <N>",
		Short: "Generate CoDD document from GitHub Discussion",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			projectRoot, err := projectRootFromCmd(cmd)
			if err != nil {
				return err
			}

			cfg, err := docLoadConfig(configPath)
			if err != nil {
				return err
			}
			providerName, model := docResolveProviderForType(cfg, "requirements")
			provider, err := docNewProviderFromConfig(agent.ProviderConfig{
				Provider: providerName,
				Model:    model,
			})
			if err != nil {
				return fmt.Errorf("create provider: %w", err)
			}

			generator := docNewGenerator(provider, projectRoot, dryRun)
			res, err := generator.Generate(cmd.Context(), docpkg.GenerateRequest{
				DiscussionID: discussion,
				ConfigPath:   configPath,
				ProjectRoot:  projectRoot,
				DryRun:       dryRun,
				OutputDir:    outputDir,
			})
			if err != nil {
				return err
			}

			if dryRun {
				preview, err := renderDocPreview(res.Document)
				if err != nil {
					return err
				}
				fmt.Fprint(cmd.OutOrStdout(), preview)
				return nil
			}

			relPath := res.FilePath
			if p, relErr := filepath.Rel(projectRoot, res.FilePath); relErr == nil {
				relPath = filepath.ToSlash(p)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Generated: %s\n", relPath)
			if res.IndexUpdate {
				fmt.Fprintln(cmd.OutOrStdout(), "Index updated: .teraflow/index.yml")
			}

			if createPR {
				branch, prNumber, err := createDocPR(projectRoot, discussion, relPath)
				if err != nil {
					return err
				}
				res.PRBranch = prNumber
				if err := generator.PostDiscussionComment(cmd.Context(), discussion, res); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "PR created from branch: %s\n", branch)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&discussion, "discussion", "", "GitHub discussion number")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview generated content without writing files")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory for generated document")
	cmd.Flags().BoolVar(&createPR, "create-pr", false, "Create branch/commit/push and open PR")
	_ = cmd.MarkFlagRequired("discussion")
	return cmd
}

func newDocListCmd() *cobra.Command {
	var category string
	var statusFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List CoDD documents from index",
		RunE: func(cmd *cobra.Command, _ []string) error {
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			projectRoot, err := projectRootFromCmd(cmd)
			if err != nil {
				return err
			}

			builder := indexpkg.NewBuilder(projectRoot)
			idx, err := builder.LoadIndex()
			if err != nil {
				return err
			}

			rows := make([]docListRow, 0, len(idx.Entries))
			for _, entry := range idx.Entries {
				if !matchesCategory(entry.Path, category) {
					continue
				}

				status, err := readDocStatus(projectRoot, entry.Path)
				if err != nil {
					return err
				}
				if statusFilter != "" && !strings.EqualFold(status, statusFilter) {
					continue
				}

				rows = append(rows, docListRow{
					NodeID:    entry.NodeID,
					Title:     entry.Title,
					Status:    status,
					DependsOn: entry.DependsOn,
				})
			}

			if format == "json" {
				return writeJSON(cmd, rows)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "NODE_ID\tTITLE\tSTATUS\tDEPENDS_ON")
			for _, row := range rows {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", row.NodeID, row.Title, row.Status, strings.Join(row.DependsOn, ","))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "Filter by document category (docs/<category>)")
	cmd.Flags().StringVar(&statusFilter, "status", "", "Filter by document status")
	return cmd
}

type docListRow struct {
	NodeID    string   `json:"node_id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	DependsOn []string `json:"depends_on,omitempty"`
}

func renderDocPreview(document *docpkg.CoDDDocument) (string, error) {
	if document == nil {
		return "", fmt.Errorf("generated document is nil")
	}

	type frontmatter struct {
		CoDD *docpkg.CoDDDocument `yaml:"codd"`
	}

	content, err := yaml.Marshal(frontmatter{CoDD: document})
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(content)
	b.WriteString("---\n\n")
	b.WriteString(document.Body)
	b.WriteString("\n")
	return b.String(), nil
}

func matchesCategory(relPath, category string) bool {
	if strings.TrimSpace(category) == "" {
		return true
	}
	prefix := filepath.ToSlash(filepath.Join("docs", category)) + "/"
	return strings.HasPrefix(filepath.ToSlash(relPath), prefix)
}

func readDocStatus(projectRoot, relPath string) (string, error) {
	absPath := filepath.Join(projectRoot, filepath.FromSlash(relPath))
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", relPath, err)
	}

	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return "", nil
	}
	parts := strings.SplitN(text, "\n---\n", 2)
	if len(parts) < 2 {
		return "", nil
	}

	var raw struct {
		CoDD struct {
			Status string `yaml:"status"`
		} `yaml:"codd"`
	}
	if err := yaml.Unmarshal([]byte(strings.TrimPrefix(parts[0], "---\n")), &raw); err != nil {
		return "", fmt.Errorf("parse frontmatter %s: %w", relPath, err)
	}
	return strings.TrimSpace(raw.CoDD.Status), nil
}

func createDocPR(projectRoot, discussion, generatedRelPath string) (string, string, error) {
	if _, err := rbacLookPath("gh"); err != nil {
		return "", "", fmt.Errorf("E5001: GitHub CLI (gh) is not installed")
	}

	branch := fmt.Sprintf("doc/discussion-%s-%d", discussion, time.Now().Unix())
	if _, err := runExternalCommand(projectRoot, "git", "checkout", "-b", branch); err != nil {
		return "", "", fmt.Errorf("create branch: %w", err)
	}
	if generatedRelPath != "" {
		if _, err := runExternalCommand(projectRoot, "git", "add", generatedRelPath); err != nil {
			return "", "", fmt.Errorf("git add generated doc: %w", err)
		}
	}
	if _, err := runExternalCommand(projectRoot, "git", "add", ".teraflow/index.yml"); err != nil {
		return "", "", fmt.Errorf("git add index: %w", err)
	}

	msg := fmt.Sprintf("feat(doc): generate CoDD from discussion #%s", discussion)
	if _, err := runExternalCommand(projectRoot, "git", "commit", "-m", msg); err != nil {
		return "", "", fmt.Errorf("git commit: %w", err)
	}
	if _, err := runExternalCommand(projectRoot, "git", "push", "-u", "origin", branch); err != nil {
		return "", "", fmt.Errorf("git push: %w", err)
	}

	title := fmt.Sprintf("feat(doc): generated CoDD from discussion #%s", discussion)
	body := fmt.Sprintf("Generated by `teraflow doc generate --discussion %s --create-pr`.", discussion)
	prURLBytes, err := runExternalCommand(projectRoot, "gh", "pr", "create", "--title", title, "--body", body, "--base", "main")
	if err != nil {
		return "", "", fmt.Errorf("gh pr create: %w", err)
	}

	prNumber := extractPRNumberFromPRURL(string(prURLBytes))
	if prNumber == "" {
		return "", "", fmt.Errorf("gh pr create: could not parse PR number")
	}

	return branch, prNumber, nil
}

func extractPRNumberFromPRURL(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	idx := strings.LastIndex(v, "/")
	if idx < 0 || idx == len(v)-1 {
		return ""
	}
	last := v[idx+1:]
	for _, ch := range last {
		if ch < '0' || ch > '9' {
			return ""
		}
	}
	return last
}
