package discovery

import "strings"

const minFulfilledAnswerLength = 8

// FulfillmentMap stores whether each template field is fulfilled.
type FulfillmentMap map[string]bool

// GapAnalysisResult represents fulfillment summary and missing fields.
type GapAnalysisResult struct {
	Fulfillment        FulfillmentMap `yaml:"fulfillment" json:"fulfillment"`
	RequiredTotal      int            `yaml:"required_total" json:"required_total"`
	RequiredFulfilled  int            `yaml:"required_fulfilled" json:"required_fulfilled"`
	RecommendedTotal   int            `yaml:"recommended_total" json:"recommended_total"`
	RecommendedFilled  int            `yaml:"recommended_fulfilled" json:"recommended_fulfilled"`
	MissingRequired    []TemplateItem `yaml:"missing_required" json:"missing_required"`
	MissingRecommended []TemplateItem `yaml:"missing_recommended" json:"missing_recommended"`
}

// AnalyzeGap compares a session state against a requirement template.
// Fulfillment is initially rule-based: answered + minimum content length.
func AnalyzeGap(state *SessionState, tmpl RequirementTemplate) GapAnalysisResult {
	result := GapAnalysisResult{Fulfillment: FulfillmentMap{}}
	if state == nil {
		return result
	}

	for _, item := range tmpl.Items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		category := normalizeTemplateCategory(item.Category)
		node := state.FindBranch(id)
		fulfilled := isNodeFulfilled(node)
		result.Fulfillment[id] = fulfilled

		switch category {
		case TemplateCategoryRequired:
			result.RequiredTotal++
			if fulfilled {
				result.RequiredFulfilled++
			} else {
				result.MissingRequired = append(result.MissingRequired, item)
			}
		case TemplateCategoryRecommended:
			result.RecommendedTotal++
			if fulfilled {
				result.RecommendedFilled++
			} else {
				result.MissingRecommended = append(result.MissingRecommended, item)
			}
		}
	}

	return result
}

func isNodeFulfilled(node *Branch) bool {
	if node == nil {
		return false
	}
	if normalizeStatus(node.Status) != StatusAnswered {
		return false
	}
	answer := strings.TrimSpace(node.Answer)
	if len([]rune(answer)) < minFulfilledAnswerLength {
		return false
	}
	lower := strings.ToLower(answer)
	for _, token := range []string{"未定", "n/a", "不明", "なし", "わから", "未回答"} {
		if strings.Contains(lower, token) {
			return false
		}
	}
	return true
}
