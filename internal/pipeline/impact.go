package pipeline

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/taka-sho/teraflow/internal/graphbridge"
	"github.com/taka-sho/teraflow/internal/index"
)

// ImpactAnalyzer computes downstream impact from CoDD dependency graph with optional GraphRAG enrichment.
type ImpactAnalyzer struct {
	idx         *index.Index
	graphBridge *graphbridge.Bridge
	projectRoot string
}

func NewImpactAnalyzer(idx *index.Index, bridge *graphbridge.Bridge, projectRoot string) *ImpactAnalyzer {
	return &ImpactAnalyzer{idx: idx, graphBridge: bridge, projectRoot: projectRoot}
}

type AffectedNode struct {
	NodeID   string `json:"node_id"`
	Band     string `json:"band"`
	Reason   string `json:"reason"`
	Distance int    `json:"distance"`
	Source   string `json:"source,omitempty"`
}

type ImpactResult struct {
	ChangedNode   string         `json:"changed_node"`
	AffectedNodes []AffectedNode `json:"affected_nodes"`
	RegenRequired []string       `json:"regen_required"`
	ReviewNeeded  []string       `json:"review_needed"`
	Summary       map[string]int `json:"summary"`
	Warnings      []string       `json:"warnings,omitempty"`
}

func (a *ImpactAnalyzer) Analyze(ctx context.Context, changedNodeID string, depth int) (*ImpactResult, error) {
	if a == nil || a.idx == nil {
		return nil, fmt.Errorf("index is required")
	}
	if strings.TrimSpace(changedNodeID) == "" {
		return nil, fmt.Errorf("changed node id is required")
	}
	if depth < 1 {
		return nil, fmt.Errorf("depth must be >= 1")
	}
	if a.idx.FindByNodeID(changedNodeID) == nil {
		return nil, fmt.Errorf("node not found: %s", changedNodeID)
	}

	result := &ImpactResult{ChangedNode: changedNodeID}
	affectedMap := map[string]AffectedNode{}

	fallback := a.analyzeByDependsOn(changedNodeID, depth)
	for _, node := range fallback {
		affectedMap[node.NodeID] = node
	}

	if a.graphBridge != nil && a.graphBridge.Available() {
		resp, err := a.graphBridge.Execute(graphbridge.Request{
			Command: "impact",
			Args: map[string]any{
				"node_id":          changedNodeID,
				"depth":            depth,
				"include_graphrag": true,
				"graph_path":       filepath.Join(a.projectRoot, ".teraflow", "graphrag", "graph.graphml"),
			},
		})
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("graphrag impact failed, using CoDD fallback: %v", err))
		} else {
			for _, node := range parseBridgeImpact(resp.Data) {
				existing, ok := affectedMap[node.NodeID]
				if !ok || node.Distance < existing.Distance {
					affectedMap[node.NodeID] = node
				}
			}
		}
	} else {
		result.Warnings = append(result.Warnings, "graphrag not available, using CoDD fallback")
	}

	result.AffectedNodes = make([]AffectedNode, 0, len(affectedMap))
	for _, node := range affectedMap {
		result.AffectedNodes = append(result.AffectedNodes, node)
		if node.Band == "gray" {
			result.RegenRequired = append(result.RegenRequired, node.NodeID)
		}
		if node.Band == "amber" {
			result.ReviewNeeded = append(result.ReviewNeeded, node.NodeID)
		}
	}

	sort.Slice(result.AffectedNodes, func(i, j int) bool {
		if result.AffectedNodes[i].Distance == result.AffectedNodes[j].Distance {
			return result.AffectedNodes[i].NodeID < result.AffectedNodes[j].NodeID
		}
		return result.AffectedNodes[i].Distance < result.AffectedNodes[j].Distance
	})
	sort.Strings(result.RegenRequired)
	sort.Strings(result.ReviewNeeded)

	result.Summary = map[string]int{
		"total": len(result.AffectedNodes),
		"gray":  len(result.RegenRequired),
		"amber": len(result.ReviewNeeded),
		"green": max(len(result.AffectedNodes)-len(result.RegenRequired)-len(result.ReviewNeeded), 0),
	}

	_ = ctx
	return result, nil
}

func (a *ImpactAnalyzer) analyzeByDependsOn(changedNodeID string, depth int) []AffectedNode {
	reverse := make(map[string][]string, len(a.idx.Entries))
	for _, entry := range a.idx.Entries {
		for _, dep := range entry.DependsOn {
			reverse[dep] = append(reverse[dep], entry.NodeID)
		}
	}

	type qItem struct {
		nodeID string
		depth  int
	}
	queue := []qItem{{nodeID: changedNodeID, depth: 0}}
	visited := map[string]bool{changedNodeID: true}
	out := make([]AffectedNode, 0)

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if item.depth >= depth {
			continue
		}
		for _, dep := range reverse[item.nodeID] {
			if visited[dep] {
				continue
			}
			visited[dep] = true
			d := item.depth + 1
			out = append(out, AffectedNode{
				NodeID:   dep,
				Band:     classifyBand(d),
				Reason:   fmt.Sprintf("depends_on chain from %s", changedNodeID),
				Distance: d,
				Source:   "codd",
			})
			queue = append(queue, qItem{nodeID: dep, depth: d})
		}
	}
	return out
}

func parseBridgeImpact(data map[string]any) []AffectedNode {
	nodes, _ := data["affected_nodes"].([]any)
	out := make([]AffectedNode, 0, len(nodes))
	for _, raw := range nodes {
		m, _ := raw.(map[string]any)
		if len(m) == 0 {
			continue
		}
		nodeID, _ := m["node_id"].(string)
		if nodeID == "" {
			continue
		}
		distance := toInt(m["depth"])
		source, _ := m["source"].(string)
		if source == "" {
			source = "graphrag"
		}
		reason := "graph impact"
		if edgeType, _ := m["edge_type"].(string); edgeType != "" {
			reason = "edge: " + edgeType
		}
		out = append(out, AffectedNode{
			NodeID:   nodeID,
			Band:     classifyBand(distance),
			Reason:   reason,
			Distance: distance,
			Source:   source,
		})
	}
	return out
}

func classifyBand(distance int) string {
	switch {
	case distance <= 1:
		return "gray"
	case distance == 2:
		return "amber"
	default:
		return "green"
	}
}

func toInt(raw any) int {
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}
