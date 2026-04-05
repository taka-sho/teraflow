package trace

import (
	"fmt"
	"sort"

	"github.com/taka-sho/teraflow/internal/index"
)

const maxDepth = 10

type Direction string

const (
	Up   Direction = "up"
	Down Direction = "down"
	Both Direction = "both"
)

type TraceResult struct {
	RootNodeID string
	Direction  Direction
	Nodes      []TraceNode
}

type TraceNode struct {
	NodeID   string
	Title    string
	Depth    int
	FilePath string
}

type Resolver struct {
	index *index.Index
}

func NewResolver(idx *index.Index) *Resolver {
	return &Resolver{index: idx}
}

func (r *Resolver) Resolve(nodeID string, dir Direction) (*TraceResult, error) {
	if r == nil || r.index == nil {
		return nil, fmt.Errorf("index is nil")
	}

	entriesByID := make(map[string]index.Entry, len(r.index.Entries))
	dependentsByID := make(map[string][]index.Entry, len(r.index.Entries))
	for _, entry := range r.index.Entries {
		entriesByID[entry.NodeID] = entry
		for _, dep := range entry.DependsOn {
			dependentsByID[dep] = append(dependentsByID[dep], entry)
		}
	}

	if _, ok := entriesByID[nodeID]; !ok {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	result := &TraceResult{RootNodeID: nodeID, Direction: dir}

	switch dir {
	case Up:
		result.Nodes = resolveUp(nodeID, entriesByID)
	case Down:
		result.Nodes = resolveDown(nodeID, dependentsByID)
	case Both:
		upNodes := resolveUp(nodeID, entriesByID)
		downNodes := resolveDown(nodeID, dependentsByID)
		result.Nodes = mergeNodes(upNodes, downNodes)
	default:
		return nil, fmt.Errorf("invalid direction: %s", dir)
	}

	sortTraceNodes(result.Nodes)
	return result, nil
}

func resolveUp(rootNodeID string, entriesByID map[string]index.Entry) []TraceNode {
	nodes := make([]TraceNode, 0, 16)
	visited := map[string]bool{rootNodeID: true}

	var walk func(currentNodeID string, depth int)
	walk = func(currentNodeID string, depth int) {
		if depth >= maxDepth {
			return
		}

		entry, ok := entriesByID[currentNodeID]
		if !ok {
			return
		}

		for _, depNodeID := range entry.DependsOn {
			if depNodeID == "" || visited[depNodeID] {
				continue
			}
			depEntry, ok := entriesByID[depNodeID]
			if !ok {
				continue
			}

			visited[depNodeID] = true
			nodes = append(nodes, TraceNode{
				NodeID:   depEntry.NodeID,
				Title:    depEntry.Title,
				Depth:    depth + 1,
				FilePath: depEntry.Path,
			})
			walk(depNodeID, depth+1)
		}
	}

	walk(rootNodeID, 0)
	return nodes
}

func resolveDown(rootNodeID string, dependentsByID map[string][]index.Entry) []TraceNode {
	nodes := make([]TraceNode, 0, 16)
	visited := map[string]bool{rootNodeID: true}

	var walk func(currentNodeID string, depth int)
	walk = func(currentNodeID string, depth int) {
		if depth >= maxDepth {
			return
		}

		for _, dependent := range dependentsByID[currentNodeID] {
			if dependent.NodeID == "" || visited[dependent.NodeID] {
				continue
			}

			visited[dependent.NodeID] = true
			nodes = append(nodes, TraceNode{
				NodeID:   dependent.NodeID,
				Title:    dependent.Title,
				Depth:    depth + 1,
				FilePath: dependent.Path,
			})
			walk(dependent.NodeID, depth+1)
		}
	}

	walk(rootNodeID, 0)
	return nodes
}

func mergeNodes(primary []TraceNode, secondary []TraceNode) []TraceNode {
	merged := make([]TraceNode, 0, len(primary)+len(secondary))
	seen := make(map[string]int, len(primary)+len(secondary))

	for _, node := range primary {
		seen[node.NodeID] = len(merged)
		merged = append(merged, node)
	}

	for _, node := range secondary {
		if i, ok := seen[node.NodeID]; ok {
			if node.Depth < merged[i].Depth {
				merged[i].Depth = node.Depth
			}
			if merged[i].Title == "" {
				merged[i].Title = node.Title
			}
			if merged[i].FilePath == "" {
				merged[i].FilePath = node.FilePath
			}
			continue
		}
		seen[node.NodeID] = len(merged)
		merged = append(merged, node)
	}

	return merged
}

func sortTraceNodes(nodes []TraceNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Depth == nodes[j].Depth {
			return nodes[i].NodeID < nodes[j].NodeID
		}
		return nodes[i].Depth < nodes[j].Depth
	})
}
