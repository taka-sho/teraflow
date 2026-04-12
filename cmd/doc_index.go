package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/discovery"
	"gopkg.in/yaml.v3"
)

var docIndexGenerateFn = discovery.GenerateDocIndex

func newDocIndexCmd() *cobra.Command {
	var outputPath string
	var docsDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Build .teraflow/doc-index.yaml from docs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			projectRoot, err := projectRootFromCmd(cmd)
			if err != nil {
				return err
			}

			resolvedDocsDir := docsDir
			if !filepath.IsAbs(resolvedDocsDir) {
				resolvedDocsDir = filepath.Join(projectRoot, resolvedDocsDir)
			}

			resolvedOutputPath := outputPath
			if !filepath.IsAbs(resolvedOutputPath) {
				resolvedOutputPath = filepath.Join(projectRoot, resolvedOutputPath)
			}

			if !force {
				if _, err := os.Stat(resolvedOutputPath); err == nil {
					return fmt.Errorf("doc index already exists: %s (use --force to overwrite)", resolvedOutputPath)
				} else if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("check output file: %w", err)
				}
			}

			idx, err := docIndexGenerateFn(cmd.Context(), resolvedDocsDir, nil)
			if err != nil {
				return fmt.Errorf("generate doc index: %w", err)
			}

			data, err := yaml.Marshal(idx)
			if err != nil {
				return fmt.Errorf("marshal doc index: %w", err)
			}
			if err := os.MkdirAll(filepath.Dir(resolvedOutputPath), 0o755); err != nil {
				return fmt.Errorf("prepare output directory: %w", err)
			}
			if err := os.WriteFile(resolvedOutputPath, data, 0o644); err != nil {
				return fmt.Errorf("write doc index: %w", err)
			}

			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"status":       "ok",
					"path":         resolvedOutputPath,
					"docs_count":   len(idx.Docs),
					"generated_at": idx.GeneratedAt,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Doc index generated: %s\n", resolvedOutputPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Docs indexed: %d\n", len(idx.Docs))
			return nil
		},
	}

	cmd.Flags().StringVar(&docsDir, "dir", "docs", "Directory to scan for markdown documents")
	cmd.Flags().StringVar(&outputPath, "output", ".teraflow/doc-index.yaml", "Output path for doc index YAML")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing output file")
	return cmd
}
