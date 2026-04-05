package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	indexpkg "github.com/taka-sho/teraflow/internal/index"
	tracepkg "github.com/taka-sho/teraflow/internal/trace"
)

func newTraceCmd() *cobra.Command {
	var direction string
	var depth int

	cmd := &cobra.Command{
		Use:   "trace <node_id>",
		Short: "Trace upstream/downstream dependency graph from index",
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

			dir, err := parseTraceDirection(direction)
			if err != nil {
				return err
			}
			if depth <= 0 {
				return fmt.Errorf("depth must be >= 1")
			}

			projectRoot := projectRootFromConfig(configPath)
			idx, err := indexpkg.NewBuilder(projectRoot).LoadIndex()
			if err != nil {
				return err
			}

			nodeID := args[0]
			resolver := tracepkg.NewResolver(idx)
			result, err := resolver.Resolve(nodeID, dir)
			if err != nil {
				return err
			}
			result.Nodes = filterTraceNodesByDepth(result.Nodes, depth)

			if format == "json" {
				return writeJSON(cmd, result)
			}

			rootTitle := findTraceNodeTitle(idx, nodeID)
			if dir == tracepkg.Up || dir == tracepkg.Both {
				up, err := resolver.Resolve(nodeID, tracepkg.Up)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "▲ Upstream (depends on):")
				renderTraceNodesText(cmd, filterTraceNodesByDepth(up.Nodes, depth))
				fmt.Fprintln(cmd.OutOrStdout())
			}

			fmt.Fprintf(cmd.OutOrStdout(), "● %s%s\n", nodeID, formatTraceNodeTitle(rootTitle))

			if dir == tracepkg.Down || dir == tracepkg.Both {
				down, err := resolver.Resolve(nodeID, tracepkg.Down)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprintln(cmd.OutOrStdout(), "▼ Downstream (depended by):")
				renderTraceNodesText(cmd, filterTraceNodesByDepth(down.Nodes, depth))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&direction, "direction", string(tracepkg.Both), "Trace direction: up|down|both")
	cmd.Flags().IntVar(&depth, "depth", 10, "Maximum traversal depth")
	return cmd
}

func parseTraceDirection(raw string) (tracepkg.Direction, error) {
	switch raw {
	case string(tracepkg.Up):
		return tracepkg.Up, nil
	case string(tracepkg.Down):
		return tracepkg.Down, nil
	case "", string(tracepkg.Both):
		return tracepkg.Both, nil
	default:
		return "", fmt.Errorf("invalid direction: %s", raw)
	}
}

func filterTraceNodesByDepth(nodes []tracepkg.TraceNode, maxDepth int) []tracepkg.TraceNode {
	if maxDepth <= 0 {
		return nil
	}
	filtered := make([]tracepkg.TraceNode, 0, len(nodes))
	for _, node := range nodes {
		if node.Depth <= maxDepth {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

func renderTraceNodesText(cmd *cobra.Command, nodes []tracepkg.TraceNode) {
	if len(nodes) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "  (none)")
		return
	}
	for _, node := range nodes {
		indent := strings.Repeat("   ", node.Depth)
		fmt.Fprintf(cmd.OutOrStdout(), "%s└─ %s%s\n", indent, node.NodeID, formatTraceNodeTitle(node.Title))
	}
}

func findTraceNodeTitle(idx *indexpkg.Index, nodeID string) string {
	if idx == nil {
		return ""
	}
	for _, entry := range idx.Entries {
		if entry.NodeID == nodeID {
			return entry.Title
		}
	}
	return ""
}

func formatTraceNodeTitle(title string) string {
	if strings.TrimSpace(title) == "" {
		return ""
	}
	return " (" + title + ")"
}
