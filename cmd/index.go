package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	indexpkg "github.com/taka-sho/teraflow/internal/index"
)

func newIndexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Build and inspect CoDD document index",
	}

	cmd.AddCommand(newIndexBuildCmd())
	cmd.AddCommand(newIndexStatusCmd())
	return cmd
}

func newIndexBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Scan docs and generate .teraflow/index.yml",
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			projectRoot, err := projectRootFromCmd(cmd)
			if err != nil {
				return err
			}

			builder := indexpkg.NewBuilder(projectRoot)

			var existing *indexpkg.Index
			if idx, loadErr := builder.LoadIndex(); loadErr == nil {
				existing = idx
			} else if !isIndexNotFound(loadErr) {
				return loadErr
			}

			idx, err := builder.Build()
			if err != nil {
				return err
			}
			if err := builder.Save(idx); err != nil {
				return err
			}

			added, updated := calcIndexDiff(existing, idx)
			indexPath := filepath.Join(projectRoot, ".teraflow", "index.yml")

			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"status":     "ok",
					"entries":    len(idx.Entries),
					"added":      added,
					"updated":    updated,
					"index_path": indexPath,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Index built: entries=%d added=%d updated=%d\n", len(idx.Entries), added, updated)
			fmt.Fprintf(cmd.OutOrStdout(), "Path: %s\n", indexPath)
			return nil
		},
	}
}

func newIndexStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show index status and summary coverage",
		RunE: func(cmd *cobra.Command, args []string) error {
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
				if isIndexNotFound(err) {
					if format == "json" {
						return writeJSON(cmd, map[string]any{"status": "missing"})
					}
					fmt.Fprintln(cmd.OutOrStdout(), "Index not found. Run `teraflow index build` first.")
					return nil
				}
				return err
			}

			summaryCount := 0
			for _, entry := range idx.Entries {
				if entry.SummaryAvailable {
					summaryCount++
				}
			}
			coverage := 0.0
			if len(idx.Entries) > 0 {
				coverage = float64(summaryCount) * 100 / float64(len(idx.Entries))
			}

			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"status":               "ok",
					"entries":              len(idx.Entries),
					"generated_at":         idx.GeneratedAt,
					"summary_available":    summaryCount,
					"summary_coverage_pct": coverage,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Entries: %d\n", len(idx.Entries))
			fmt.Fprintf(cmd.OutOrStdout(), "Generated: %s\n", idx.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"))
			fmt.Fprintf(cmd.OutOrStdout(), "Summary coverage: %d/%d (%.1f%%)\n", summaryCount, len(idx.Entries), coverage)
			return nil
		},
	}
}

func projectRootFromCmd(cmd *cobra.Command) (string, error) {
	configPath, err := configPathFromCmd(cmd)
	if err != nil {
		return "", err
	}
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return "", err
	}
	return filepath.Dir(filepath.Dir(absConfigPath)), nil
}

func calcIndexDiff(oldIdx, newIdx *indexpkg.Index) (added, updated int) {
	if newIdx == nil {
		return 0, 0
	}
	if oldIdx == nil {
		return len(newIdx.Entries), 0
	}

	oldByNode := make(map[string]indexpkg.Entry, len(oldIdx.Entries))
	for _, entry := range oldIdx.Entries {
		oldByNode[entry.NodeID] = entry
	}

	for _, entry := range newIdx.Entries {
		prev, ok := oldByNode[entry.NodeID]
		if !ok {
			added++
			continue
		}
		if prev.ContentHash != entry.ContentHash {
			updated++
		}
	}
	return added, updated
}

func isIndexNotFound(err error) bool {
	var pe *os.PathError
	return errors.As(err, &pe) && errors.Is(pe.Err, os.ErrNotExist)
}
