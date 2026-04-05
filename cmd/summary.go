package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/agent"
	cfgpkg "github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/index"
)

func newSummaryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Manage AI summaries for indexed documents",
	}
	cmd.AddCommand(newSummaryUpdateCmd())
	cmd.AddCommand(newSummaryShowCmd())
	return cmd
}

func newSummaryUpdateCmd() *cobra.Command {
	var force bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update summary cache from .codd/index.yml",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			rootDir := projectRootFromConfig(configPath)
			idx, err := index.NewBuilder(rootDir).LoadIndex()
			if err != nil {
				var pe *os.PathError
				if errors.As(err, &pe) && errors.Is(pe.Err, os.ErrNotExist) {
					fmt.Fprintln(cmd.ErrOrStderr(), "E7001: index not found. Run `teraflow index build` first")
					return errors.New("E7001: index not found")
				}
				return err
			}

			provider, err := resolveSummaryProvider(configPath)
			if err != nil {
				return err
			}
			s := index.NewSummarizer(provider, filepath.Join(rootDir, ".teraflow", "summaries"))

			if dryRun {
				need := make([]string, 0, len(idx.Entries))
				for _, entry := range idx.Entries {
					if force || !s.IsCacheValid(entry) {
						need = append(need, entry.NodeID)
					}
				}
				if format == "json" {
					return writeJSON(cmd, map[string]any{"needs_update": need, "count": len(need)})
				}
				if len(need) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No summary updates required")
					return nil
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Need update: %d\n", len(need))
				for _, nodeID := range need {
					fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", nodeID)
				}
				return nil
			}

			if force {
				for _, entry := range idx.Entries {
					_ = os.Remove(filepath.Join(rootDir, ".teraflow", "summaries", summaryFileName(entry.NodeID)))
				}
			}

			updated, skipped, err := s.UpdateAll(cmd.Context(), idx)
			if err != nil {
				return err
			}

			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"updated": updated,
					"skipped": skipped,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "updated %d, skipped %d\n", updated, skipped)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force regenerate all summaries")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show entries that require updates")
	return cmd
}

func newSummaryShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <node_id>",
		Short: "Show cached summary for a node",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			rootDir := projectRootFromConfig(configPath)
			s := index.NewSummarizer(nil, filepath.Join(rootDir, ".teraflow", "summaries"))
			nodeID := args[0]
			summary, err := s.LoadSummary(nodeID)
			if err != nil {
				return err
			}

			if format == "json" {
				return writeJSON(cmd, map[string]string{
					"node_id": nodeID,
					"summary": summary,
				})
			}

			fmt.Fprintln(cmd.OutOrStdout(), summary)
			return nil
		},
	}
	return cmd
}

func resolveSummaryProvider(configPath string) (agent.Provider, error) {
	var cfg *cfgpkg.TeraflowConfig
	loadedCfg, err := cfgpkg.Load(configPath)
	if err == nil {
		cfg = loadedCfg
	}

	providerName, model := cfgpkg.ResolveProviderForType(cfg, "requirements")
	return agent.NewProviderFromConfig(agent.ProviderConfig{
		Provider: providerName,
		Model:    model,
	})
}

func projectRootFromConfig(configPath string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(configPath), ".."))
}

func summaryFileName(nodeID string) string {
	safe := strings.ReplaceAll(nodeID, ":", "--")
	safe = strings.ReplaceAll(safe, "/", "-")
	return safe + ".txt"
}
