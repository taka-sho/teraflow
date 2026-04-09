package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	graphpkg "github.com/taka-sho/teraflow/internal/graph"
	indexpkg "github.com/taka-sho/teraflow/internal/index"
)

func newGraphCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Analyze and export CoDD dependency graph",
	}

	cmd.AddCommand(newGraphStatusCmd())
	cmd.AddCommand(newGraphCheckCmd())
	cmd.AddCommand(newGraphExportCmd())
	return cmd
}

func newGraphStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show graph stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			analyzer, err := loadGraphAnalyzer(cmd)
			if err != nil {
				return err
			}

			result := analyzer.Status()
			if format == "json" {
				return writeJSON(cmd, result)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Nodes: %d\n", result.TotalNodes)
			fmt.Fprintf(cmd.OutOrStdout(), "Edges: %d\n", result.TotalEdges)
			fmt.Fprintf(cmd.OutOrStdout(), "Isolated nodes: %d\n", result.IsolatedNodes)
			fmt.Fprintln(cmd.OutOrStdout(), "By status:")
			for status, count := range result.ByStatus {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s: %d\n", status, count)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "By tag:")
			for tag, count := range result.ByTag {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s: %d\n", tag, count)
			}
			return nil
		},
	}
}

func newGraphCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Validate graph consistency",
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			analyzer, err := loadGraphAnalyzer(cmd)
			if err != nil {
				return err
			}

			result := analyzer.Check()
			if format == "json" {
				return writeJSON(cmd, result)
			}

			if result.OK {
				fmt.Fprintln(cmd.OutOrStdout(), "OK: no issues found")
				return nil
			}
			for _, issue := range result.Issues {
				fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s", issue.Severity, issue.Type)
				if issue.NodeID != "" {
					fmt.Fprintf(cmd.OutOrStdout(), " node=%s", issue.NodeID)
				}
				if issue.TargetID != "" {
					fmt.Fprintf(cmd.OutOrStdout(), " target=%s", issue.TargetID)
				}
				fmt.Fprintf(cmd.OutOrStdout(), " - %s\n", issue.Message)
			}
			return nil
		},
	}
}

func newGraphExportCmd() *cobra.Command {
	var format string
	var outputPath string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export graph as Mermaid or DOT",
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "mermaid" && format != "dot" {
				return fmt.Errorf("unsupported export format: %s", format)
			}

			analyzer, err := loadGraphAnalyzer(cmd)
			if err != nil {
				return err
			}

			var out string
			switch format {
			case "mermaid":
				out = analyzer.ExportMermaid()
			case "dot":
				out = analyzer.ExportDOT()
			}

			if outputPath == "" {
				fmt.Fprint(cmd.OutOrStdout(), out)
				return nil
			}
			if err := os.WriteFile(outputPath, []byte(out), 0o644); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Exported graph to %s\n", outputPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "mermaid", "Export format: mermaid|dot")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output file path (default: stdout)")
	return cmd
}

func loadGraphAnalyzer(cmd *cobra.Command) (*graphpkg.Analyzer, error) {
	projectRoot, err := projectRootFromCmd(cmd)
	if err != nil {
		return nil, err
	}
	idx, err := indexpkg.NewBuilder(projectRoot).LoadIndex()
	if err != nil {
		return nil, err
	}
	return graphpkg.NewAnalyzer(idx), nil
}
