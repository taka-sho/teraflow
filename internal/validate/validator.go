package validate

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/taka-sho/teraflow/internal/graph"
	"github.com/taka-sho/teraflow/internal/graphbridge"
	"github.com/taka-sho/teraflow/internal/index"
)

var nodeIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9:_-]*$`)

var validReviewRequired = map[string]struct{}{
	"auto":    {},
	"review":  {},
	"approve": {},
}

var validImpactBands = map[string]struct{}{
	"green": {},
	"amber": {},
	"gray":  {},
}

// Validator validates CoDD graph/index consistency in 4 levels.
type Validator struct {
	idx         *index.Index
	graphBridge *graphbridge.Bridge
	projectRoot string
	maxLevel    int
}

func NewValidator(idx *index.Index, bridge *graphbridge.Bridge, projectRoot string) *Validator {
	return &Validator{
		idx:         idx,
		graphBridge: bridge,
		projectRoot: projectRoot,
		maxLevel:    4,
	}
}

func (v *Validator) SetLevel(level int) {
	if level >= 1 && level <= 4 {
		v.maxLevel = level
	}
}

// ValidatePhaseTransition validates consistency focused on phase transition constraints.
func (v *Validator) ValidatePhaseTransition(from, to string) ValidationResult {
	result := ValidationResult{Valid: true}
	if v.idx == nil {
		result.addError(1, "", "missing_index", "index is nil", "validate")
		return result
	}

	if v.maxLevel >= 1 {
		for _, entry := range v.idx.Entries {
			if from != "" && entry.Phase != from {
				continue
			}
			v.validateSchema(entry, &result)
		}
	}
	if v.maxLevel >= 2 {
		v.validateReferences("", &result)
	}
	if v.maxLevel >= 3 {
		v.validatePhase(from, to, &result)
	}
	if v.maxLevel >= 4 {
		v.validateGraph("", &result)
	}
	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result
}

// ValidateArtifact validates consistency focused on a single node.
func (v *Validator) ValidateArtifact(nodeID string) ValidationResult {
	result := ValidationResult{Valid: true}
	if v.idx == nil {
		result.addError(1, nodeID, "missing_index", "index is nil", "validate")
		return result
	}

	entry := v.idx.FindByNodeID(nodeID)
	if entry == nil {
		result.addError(1, nodeID, "missing_node", "node not found in index", "validate")
		return result
	}

	if v.maxLevel >= 1 {
		v.validateSchema(*entry, &result)
	}
	if v.maxLevel >= 2 {
		v.validateReferences(nodeID, &result)
	}
	if v.maxLevel >= 3 {
		v.validateArtifactPhase(*entry, &result)
	}
	if v.maxLevel >= 4 {
		v.validateGraph(nodeID, &result)
	}
	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result
}

func (v *Validator) validateSchema(entry index.Entry, result *ValidationResult) {
	if strings.TrimSpace(entry.NodeID) == "" {
		result.addError(1, entry.NodeID, "missing_node_id", "node_id is required", "schema")
	}
	if strings.TrimSpace(entry.Title) == "" {
		result.addError(1, entry.NodeID, "missing_title", "title is required", "schema")
	}
	if strings.TrimSpace(entry.Path) == "" {
		result.addError(1, entry.NodeID, "missing_path", "path is required", "schema")
	}
	if entry.NodeID != "" && !nodeIDPattern.MatchString(strings.ToLower(entry.NodeID)) {
		result.addError(1, entry.NodeID, "invalid_node_id", "node_id must match ^[a-z0-9][a-z0-9:_-]*$", "schema")
	}

	review := strings.ToLower(strings.TrimSpace(entry.ReviewRequired))
	if review != "" {
		if _, ok := validReviewRequired[review]; !ok {
			result.addError(1, entry.NodeID, "invalid_review_required", "review_required must be one of auto|review|approve", "schema")
		}
	}

	impact := strings.ToLower(strings.TrimSpace(entry.ChangeImpact))
	if impact != "" {
		if _, ok := validImpactBands[impact]; !ok {
			result.addError(1, entry.NodeID, "invalid_change_impact", "change_impact must be one of green|amber|gray", "schema")
		}
	}
}

func (v *Validator) validateReferences(scopeNodeID string, result *ValidationResult) {
	entryByID := make(map[string]index.Entry, len(v.idx.Entries))
	for _, e := range v.idx.Entries {
		entryByID[e.NodeID] = e
	}

	for _, entry := range v.idx.Entries {
		if scopeNodeID != "" && entry.NodeID != scopeNodeID {
			continue
		}

		for _, dep := range entry.DependsOn {
			if _, ok := entryByID[dep]; !ok {
				result.addError(2, entry.NodeID, "missing_dependency", fmt.Sprintf("depends_on node not found: %s", dep), "reference")
			}
		}

		for _, module := range entry.Modules {
			if module == "" {
				continue
			}
			if filepath.IsAbs(module) || strings.Contains(module, "..") {
				result.addError(2, entry.NodeID, "invalid_module_path", fmt.Sprintf("invalid module path: %s", module), "reference")
				continue
			}
			if v.projectRoot != "" {
				abs := filepath.Join(v.projectRoot, filepath.FromSlash(module))
				if !strings.HasPrefix(abs, filepath.Clean(v.projectRoot)+string(filepath.Separator)) {
					result.addError(2, entry.NodeID, "invalid_module_path", fmt.Sprintf("module escapes project root: %s", module), "reference")
				}
			}
		}

		for _, target := range entry.Verifies {
			targetEntry, ok := entryByID[target]
			if !ok {
				result.addError(2, entry.NodeID, "missing_verifies_target", fmt.Sprintf("verifies target not found: %s", target), "reference")
				continue
			}
			if !slices.Contains(targetEntry.VerifiedBy, entry.NodeID) {
				result.addError(2, entry.NodeID, "verifies_mismatch", fmt.Sprintf("%s verifies %s but target lacks verified_by link", entry.NodeID, target), "reference")
			}
		}

		for _, source := range entry.VerifiedBy {
			sourceEntry, ok := entryByID[source]
			if !ok {
				result.addError(2, entry.NodeID, "missing_verified_by_source", fmt.Sprintf("verified_by source not found: %s", source), "reference")
				continue
			}
			if !slices.Contains(sourceEntry.Verifies, entry.NodeID) {
				result.addError(2, entry.NodeID, "verified_by_mismatch", fmt.Sprintf("%s lists verified_by %s but source lacks verifies link", entry.NodeID, source), "reference")
			}
		}
	}
}

func (v *Validator) validatePhase(from, to string, result *ValidationResult) {
	readyStatuses := map[string]struct{}{
		"approved":    {},
		"implemented": {},
		"tested":      {},
		"confirmed":   {},
		"completed":   {},
	}

	for _, entry := range v.idx.Entries {
		if from != "" && entry.Phase != from {
			continue
		}
		if strings.TrimSpace(entry.Phase) == "" {
			continue
		}

		status := strings.ToLower(strings.TrimSpace(entry.Status))
		if status == "" {
			result.addWarning(3, entry.NodeID, "missing_status", "phase artifact has no status", "phase")
			continue
		}
		if from != "" && to != "" && from != to {
			if _, ok := readyStatuses[status]; !ok {
				result.addError(3, entry.NodeID, "status_not_ready", fmt.Sprintf("phase transition %s->%s requires approved/implemented/tested/confirmed/completed status", from, to), "phase")
			}
		}
	}
}

func (v *Validator) validateArtifactPhase(entry index.Entry, result *ValidationResult) {
	if strings.TrimSpace(entry.Phase) == "" {
		return
	}
	status := strings.ToLower(strings.TrimSpace(entry.Status))
	if status == "" {
		result.addWarning(3, entry.NodeID, "missing_status", "phase artifact has no status", "phase")
	}
}

func (v *Validator) validateGraph(scopeNodeID string, result *ValidationResult) {
	analyzer := graph.NewAnalyzer(v.idx)
	checkResult := analyzer.Check()
	for _, issue := range checkResult.Issues {
		if scopeNodeID != "" && issue.NodeID != scopeNodeID && issue.TargetID != scopeNodeID {
			continue
		}
		if issue.Severity == graph.SeverityError {
			result.addError(4, issue.NodeID, issue.Type, issue.Message, "graph")
		} else {
			result.addWarning(4, issue.NodeID, issue.Type, issue.Message, "graph")
		}
	}

	if v.graphBridge == nil {
		result.addWarning(4, scopeNodeID, "graphrag_unavailable", "GraphRAG bridge is nil; skipped semantic consistency check", "graphrag")
		return
	}
	if !v.graphBridge.Available() {
		result.addWarning(4, scopeNodeID, "graphrag_unavailable", "GraphRAG module not installed; level 4 semantic checks skipped", "graphrag")
		return
	}

	resp, err := v.graphBridge.Execute(graphbridge.Request{
		Command: "check",
		Args: map[string]any{
			"graph_path": filepath.Join(v.projectRoot, ".teraflow", "graphrag", "graph.graphml"),
		},
	})
	if err != nil {
		result.addWarning(4, scopeNodeID, "graphrag_check_failed", fmt.Sprintf("GraphRAG check skipped: %v", err), "graphrag")
		return
	}

	issues, _ := resp.Data["issues"].([]any)
	for _, item := range issues {
		issue, _ := item.(map[string]any)
		nodeID, _ := issue["node_id"].(string)
		targetID, _ := issue["target_id"].(string)
		if scopeNodeID != "" && nodeID != scopeNodeID && targetID != scopeNodeID {
			continue
		}
		severity, _ := issue["severity"].(string)
		issueType, _ := issue["type"].(string)
		message, _ := issue["message"].(string)
		source, _ := issue["source"].(string)
		if source == "" {
			source = "graphrag"
		}
		if strings.EqualFold(severity, "error") {
			result.addError(4, nodeID, issueType, message, source)
		} else {
			result.addWarning(4, nodeID, issueType, message, source)
		}
	}
}
