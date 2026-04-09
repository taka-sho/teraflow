package graph

import (
	"strings"

	"github.com/taka-sho/teraflow/internal/index"
)

// Analyzer computes graph stats and consistency checks over the CoDD index.
type Analyzer struct {
	idx *index.Index
}

func NewAnalyzer(idx *index.Index) *Analyzer {
	return &Analyzer{idx: idx}
}

type StatusResult struct {
	TotalNodes    int            `json:"total_nodes"`
	TotalEdges    int            `json:"total_edges"`
	IsolatedNodes int            `json:"isolated_nodes"`
	ByStatus      map[string]int `json:"by_status"`
	ByTag         map[string]int `json:"by_tag"`
}

func (a *Analyzer) Status() *StatusResult {
	result := &StatusResult{
		ByStatus: make(map[string]int),
		ByTag:    make(map[string]int),
	}
	if a == nil || a.idx == nil {
		return result
	}

	result.TotalNodes = len(a.idx.Entries)
	incoming := make(map[string]int, len(a.idx.Entries))

	for _, entry := range a.idx.Entries {
		result.TotalEdges += len(entry.DependsOn)
		for _, dep := range entry.DependsOn {
			incoming[dep]++
		}

		status := normalizeStatus(entry.Status)
		result.ByStatus[status]++

		for _, tag := range entry.Tags {
			if strings.TrimSpace(tag) == "" {
				continue
			}
			result.ByTag[tag]++
		}
	}

	for _, entry := range a.idx.Entries {
		if len(entry.DependsOn) == 0 && incoming[entry.NodeID] == 0 {
			result.IsolatedNodes++
		}
	}

	return result
}

func normalizeStatus(status string) string {
	if strings.TrimSpace(status) == "" {
		return "unknown"
	}
	return strings.ToLower(strings.TrimSpace(status))
}
