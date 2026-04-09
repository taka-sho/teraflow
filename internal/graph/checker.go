package graph

import (
	"fmt"
	"strings"

	"github.com/taka-sho/teraflow/internal/index"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Issue struct {
	Severity Severity `json:"severity"`
	Type     string   `json:"type"`
	Message  string   `json:"message"`
	NodeID   string   `json:"node_id,omitempty"`
	TargetID string   `json:"target_id,omitempty"`
}

type CheckResult struct {
	Issues []Issue `json:"issues"`
	OK     bool    `json:"ok"`
}

func (a *Analyzer) Check() *CheckResult {
	result := &CheckResult{Issues: make([]Issue, 0)}
	if a == nil || a.idx == nil {
		result.OK = true
		return result
	}

	entryByID := make(map[string]int, len(a.idx.Entries))
	statusByID := make(map[string]string, len(a.idx.Entries))
	for i, entry := range a.idx.Entries {
		entryByID[entry.NodeID] = i
		statusByID[entry.NodeID] = normalizeStatus(entry.Status)
	}

	for _, entry := range a.idx.Entries {
		sourceStatus := normalizeStatus(entry.Status)
		for _, dep := range entry.DependsOn {
			if _, ok := entryByID[dep]; !ok {
				result.Issues = append(result.Issues, Issue{
					Severity: SeverityError,
					Type:     "broken_ref",
					Message:  fmt.Sprintf("%s depends_on missing node %s", entry.NodeID, dep),
					NodeID:   entry.NodeID,
					TargetID: dep,
				})
				continue
			}

			targetStatus := statusByID[dep]
			if sourceStatus == "confirmed" && (targetStatus == "draft" || targetStatus == "review") {
				result.Issues = append(result.Issues, Issue{
					Severity: SeverityWarning,
					Type:     "status_conflict",
					Message:  fmt.Sprintf("confirmed node %s depends on %s node %s", entry.NodeID, targetStatus, dep),
					NodeID:   entry.NodeID,
					TargetID: dep,
				})
			}
		}
	}

	result.Issues = append(result.Issues, findCycles(a.idx.Entries)...)
	result.OK = len(result.Issues) == 0
	return result
}

func findCycles(entries []index.Entry) []Issue {
	issues := make([]Issue, 0)
	adj := make(map[string][]string, len(entries))
	nodeSet := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		nodeSet[entry.NodeID] = struct{}{}
	}
	for _, entry := range entries {
		for _, dep := range entry.DependsOn {
			if _, ok := nodeSet[dep]; ok {
				adj[entry.NodeID] = append(adj[entry.NodeID], dep)
			}
		}
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)
	state := make(map[string]int, len(adj))
	stack := make([]string, 0, len(adj))
	seenCycle := make(map[string]struct{})

	var dfs func(string)
	dfs = func(node string) {
		state[node] = gray
		stack = append(stack, node)

		for _, next := range adj[node] {
			switch state[next] {
			case white:
				dfs(next)
			case gray:
				cycle := cyclePath(stack, next)
				if len(cycle) == 0 {
					continue
				}
				key := strings.Join(cycle, "->")
				if _, ok := seenCycle[key]; ok {
					continue
				}
				seenCycle[key] = struct{}{}
				issues = append(issues, Issue{
					Severity: SeverityError,
					Type:     "cyclic_dep",
					Message:  "cyclic dependency: " + strings.Join(cycle, " -> "),
					NodeID:   next,
				})
			}
		}

		stack = stack[:len(stack)-1]
		state[node] = black
	}

	for _, entry := range entries {
		if state[entry.NodeID] == white {
			dfs(entry.NodeID)
		}
	}

	return issues
}

func cyclePath(stack []string, start string) []string {
	idx := -1
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == start {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}
	cycle := append([]string(nil), stack[idx:]...)
	cycle = append(cycle, start)
	return cycle
}
