package wave

import (
	"sort"
	"strings"
)

const DefaultContextTokenLimit = 8000

// ContextDocument is one source document attached to wave generation input.
type ContextDocument struct {
	NodeID string `json:"node_id"`
	Path   string `json:"path"`
	Body   string `json:"body"`
}

// WaveContext is the final context injected into the prompt template.
type WaveContext struct {
	Phase          string            `json:"phase"`
	Wave           WaveDefinition    `json:"wave"`
	Inputs         []ContextDocument `json:"inputs"`
	RelatedSummary string            `json:"related_summary,omitempty"`
	ImpactSummary  string            `json:"impact_summary,omitempty"`
	TokenEstimate  int               `json:"token_estimate"`
}

func estimateTokens(s string) int {
	if s == "" {
		return 0
	}
	return len([]rune(s)) / 4
}

func trimToTokenBudget(text string, budget int) string {
	if budget <= 0 {
		return ""
	}
	runes := []rune(text)
	maxChars := budget * 4
	if len(runes) <= maxChars {
		return text
	}
	if maxChars <= 3 {
		return string(runes[:maxChars])
	}
	return string(runes[:maxChars-3]) + "..."
}

func compactLines(text string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}
	return strings.Join(lines[:maxLines], "\n")
}

func summarizeNodeIDs(nodeIDs []string) string {
	if len(nodeIDs) == 0 {
		return ""
	}
	unique := make(map[string]struct{}, len(nodeIDs))
	for _, node := range nodeIDs {
		n := strings.TrimSpace(node)
		if n == "" {
			continue
		}
		unique[n] = struct{}{}
	}
	list := make([]string, 0, len(unique))
	for n := range unique {
		list = append(list, n)
	}
	sort.Strings(list)
	return strings.Join(list, ", ")
}
