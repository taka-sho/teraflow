package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	graphpkg "github.com/taka-sho/teraflow/internal/graph"
	"github.com/taka-sho/teraflow/internal/graphbridge"
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
	cmd.AddCommand(newGraphBuildCmd())
	cmd.AddCommand(newGraphSearchCmd())
	cmd.AddCommand(newGraphImpactCmd())
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

func newGraphBuildCmd() *cobra.Command {
	var full bool
	var incremental bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build knowledge graph from CoDD documents",
		Long: `Build the GraphRAG knowledge graph by extracting entities from CoDD documents
and constructing a NetworkX graph stored in .teraflow/graphrag/.

Requires teraflow-graphrag Python module to be installed:
  cd graphrag && pip install -e .`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := projectRootFromCmd(cmd)
			if err != nil {
				return err
			}

			bridge := graphbridge.New(projectRoot)
			if !bridge.Available() {
				return fmt.Errorf("GraphRAG module not found; install with: cd graphrag && pip install -e ./")
			}

			if !incremental {
				full = true
			}

			resp, err := bridge.Execute(graphbridge.Request{
				Command: "build",
				Args: map[string]any{
					"project_root": projectRoot,
					"full":         full,
					"dry_run":      dryRun,
				},
			})
			if err != nil {
				return fmt.Errorf("graph build failed: %w", err)
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" && resp.Data != nil {
				return writeJSON(cmd, resp.Data)
			}

			if dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), "Dry-run completed.")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Graph built successfully.")
			}

			if resp.Data != nil {
				if payload, err := json.MarshalIndent(resp.Data, "", "  "); err == nil {
					fmt.Fprintln(cmd.OutOrStdout(), string(payload))
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&full, "full", false, "Force full rebuild (ignore incremental state)")
	cmd.Flags().BoolVar(&incremental, "incremental", true, "Use incremental build (default)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be built without executing")
	return cmd
}

func newGraphSearchCmd() *cobra.Command {
	var mode string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search graph context using GraphRAG",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.TrimSpace(args[0])
			if query == "" {
				return fmt.Errorf("query cannot be empty")
			}
			mode = strings.ToLower(strings.TrimSpace(mode))
			if mode != "local" && mode != "global" {
				return fmt.Errorf("invalid mode: %s (expected local or global)", mode)
			}

			projectRoot, err := projectRootFromCmd(cmd)
			if err != nil {
				return err
			}
			bridge := graphbridge.New(projectRoot)
			if !bridge.Available() {
				return fmt.Errorf("GraphRAG module not found; install with: cd graphrag && pip install -e ./")
			}

			resp, err := bridge.Execute(graphbridge.Request{
				Command: "query",
				Args: map[string]any{
					"query":        query,
					"mode":         mode,
					"graph_path":   filepath.Join(projectRoot, ".teraflow", "graphrag", "graph.graphml"),
					"storage_path": filepath.Join(projectRoot, ".teraflow", "graphrag"),
				},
			})
			if err != nil {
				return fmt.Errorf("graph search failed: %w", err)
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, resp.Data)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Query: %s\n", query)
			fmt.Fprintf(cmd.OutOrStdout(), "Mode: %s\n", mode)
			if answer, ok := resp.Data["answer"].(string); ok && strings.TrimSpace(answer) != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Answer: %s\n", answer)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Sources:")
			sources, _ := resp.Data["sources"].([]any)
			if len(sources) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "  - (none)")
				return nil
			}
			for _, item := range sources {
				sourceMap, _ := item.(map[string]any)
				if len(sourceMap) == 0 {
					continue
				}
				id := firstString(sourceMap, "node_id", "community_id")
				label := firstString(sourceMap, "label", "summary")
				srcType := firstString(sourceMap, "source")
				if id == "" {
					id = "(unknown)"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s", id)
				if label != "" {
					fmt.Fprintf(cmd.OutOrStdout(), " %s", label)
				}
				if srcType != "" {
					fmt.Fprintf(cmd.OutOrStdout(), " [%s]", srcType)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "local", "Search mode: local|global")
	return cmd
}

func newGraphImpactCmd() *cobra.Command {
	var depth int

	cmd := &cobra.Command{
		Use:   "impact <node_id>",
		Short: "Analyze impact for a node using GraphRAG or CoDD fallback",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeID := strings.TrimSpace(args[0])
			if nodeID == "" {
				return fmt.Errorf("node_id cannot be empty")
			}
			if depth < 1 {
				return fmt.Errorf("depth must be >= 1")
			}

			projectRoot, err := projectRootFromCmd(cmd)
			if err != nil {
				return err
			}

			bridge := graphbridge.New(projectRoot)
			var data map[string]any

			if bridge.Available() {
				resp, err := bridge.Execute(graphbridge.Request{
					Command: "impact",
					Args: map[string]any{
						"node_id":          nodeID,
						"depth":            depth,
						"include_graphrag": true,
						"graph_path":       filepath.Join(projectRoot, ".teraflow", "graphrag", "graph.graphml"),
					},
				})
				if err != nil {
					return fmt.Errorf("graph impact failed: %w", err)
				}
				data = resp.Data
			} else {
				idx, err := indexpkg.NewBuilder(projectRoot).LoadIndex()
				if err != nil {
					return err
				}
				data = analyzeCoDDImpact(idx, nodeID, depth)
				data["warning"] = "graphrag not available, showing CoDD explicit dependencies only"
				fmt.Fprintln(cmd.ErrOrStderr(), data["warning"])
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, data)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Root: %s\n", firstString(data, "root_node_id", "root"))
			fmt.Fprintf(cmd.OutOrStdout(), "Total affected: %d\n", intFromMap(data, "total_count"))
			if warning, ok := data["warning"].(string); ok && warning != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Warning: %s\n", warning)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Affected nodes:")
			nodes, _ := data["affected_nodes"].([]any)
			if len(nodes) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "  - (none)")
				return nil
			}
			for _, item := range nodes {
				nodeMap, _ := item.(map[string]any)
				if len(nodeMap) == 0 {
					continue
				}
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"  - %s depth=%d edge=%s source=%s label=%s\n",
					firstString(nodeMap, "node_id"),
					intFromMap(nodeMap, "depth"),
					firstString(nodeMap, "edge_type"),
					firstString(nodeMap, "source"),
					firstString(nodeMap, "label"),
				)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&depth, "depth", 2, "Traversal depth")
	return cmd
}

func analyzeCoDDImpact(idx *indexpkg.Index, rootNodeID string, depth int) map[string]any {
	entryByID := make(map[string]indexpkg.Entry, len(idx.Entries))
	reverseDepends := make(map[string][]string, len(idx.Entries))
	for _, entry := range idx.Entries {
		entryByID[entry.NodeID] = entry
		for _, dep := range entry.DependsOn {
			reverseDepends[dep] = append(reverseDepends[dep], entry.NodeID)
		}
	}

	type queueItem struct {
		nodeID string
		depth  int
	}

	visited := map[string]bool{rootNodeID: true}
	queue := []queueItem{{nodeID: rootNodeID, depth: 0}}
	affected := make([]map[string]any, 0)

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if item.depth >= depth {
			continue
		}

		for _, dependentID := range reverseDepends[item.nodeID] {
			if visited[dependentID] {
				continue
			}
			visited[dependentID] = true
			entry := entryByID[dependentID]
			affected = append(affected, map[string]any{
				"node_id":   dependentID,
				"label":     entry.Title,
				"node_type": "Document",
				"depth":     item.depth + 1,
				"edge_type": "DEPENDS_ON",
				"source":    "codd",
			})
			queue = append(queue, queueItem{nodeID: dependentID, depth: item.depth + 1})
		}
	}

	return map[string]any{
		"root_node_id":   rootNodeID,
		"total_count":    len(affected),
		"affected_nodes": affected,
	}
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func intFromMap(m map[string]any, key string) int {
	raw, ok := m[key]
	if !ok {
		return 0
	}
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	default:
		return 0
	}
}
