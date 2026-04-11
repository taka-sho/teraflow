package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/graphbridge"
	indexpkg "github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/pipeline"
)

func newImpactCmd() *cobra.Command {
	var depth int

	cmd := &cobra.Command{
		Use:   "impact <node-id>",
		Short: "Analyze downstream impact from a changed CoDD node",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if depth < 1 {
				return fmt.Errorf("--depth must be >= 1")
			}

			projectRoot, err := projectRootFromCmd(cmd)
			if err != nil {
				return err
			}
			idx, err := indexpkg.NewBuilder(projectRoot).LoadIndex()
			if err != nil {
				return err
			}

			analyzer := pipeline.NewImpactAnalyzer(idx, graphbridge.New(projectRoot), projectRoot)
			result, err := analyzer.Analyze(context.Background(), args[0], depth)
			if err != nil {
				return err
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, result)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Changed: %s\n", result.ChangedNode)
			fmt.Fprintf(cmd.OutOrStdout(), "Affected: %d (gray=%d amber=%d green=%d)\n", result.Summary["total"], result.Summary["gray"], result.Summary["amber"], result.Summary["green"])
			for _, node := range result.AffectedNodes {
				fmt.Fprintf(cmd.OutOrStdout(), "- %s band=%s distance=%d source=%s reason=%s\n", node.NodeID, node.Band, node.Distance, node.Source, node.Reason)
			}
			for _, w := range result.Warnings {
				fmt.Fprintf(cmd.OutOrStdout(), "Warning: %s\n", w)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&depth, "depth", 2, "Traversal depth")
	return cmd
}
