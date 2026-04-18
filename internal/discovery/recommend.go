package discovery

import (
	"sort"
	"time"
)

type RecommendationType string

const (
	RecommendationTypePromoteToDefault RecommendationType = "promote_to_default"
	RecommendationTypeDeleteWarning    RecommendationType = "delete_warning"
)

type Recommendation struct {
	Type       RecommendationType `yaml:"type" json:"type"`
	FieldID    string             `yaml:"field_id" json:"field_id"`
	Name       string             `yaml:"name" json:"name"`
	Category   string             `yaml:"category" json:"category"`
	Reason     string             `yaml:"reason" json:"reason"`
	Source     string             `yaml:"source" json:"source"`
	Confidence float64            `yaml:"confidence" json:"confidence"`
	RepoCount  int                `yaml:"repo_count,omitempty" json:"repo_count,omitempty"`
}

type RecommendationResult struct {
	Recommendations []Recommendation `yaml:"recommendations" json:"recommendations"`
	AnalyzedRepos   int              `yaml:"analyzed_repos" json:"analyzed_repos"`
	Timestamp       string           `yaml:"timestamp" json:"timestamp"`
}

// DefaultThreshold returns the default minimum repo count for recommendations.
func DefaultThreshold() int {
	return 3
}

// AnalyzePatterns runs all recommendation patterns and returns deduplicated, sorted results.
func AnalyzePatterns(history TemplateHistory, threshold int) RecommendationResult {
	recs1 := FindRepeatedAdditions(history, threshold)
	recs2 := FindDeleteReaddPatterns(history)

	merged := make(map[string]Recommendation)
	for _, r := range recs1 {
		merged[r.FieldID] = r
	}
	for _, r := range recs2 {
		if existing, ok := merged[r.FieldID]; ok {
			if r.Confidence > existing.Confidence {
				merged[r.FieldID] = r
			}
		} else {
			merged[r.FieldID] = r
		}
	}

	result := make([]Recommendation, 0, len(merged))
	for _, r := range merged {
		result = append(result, r)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Confidence > result[j].Confidence
	})

	repoSet := make(map[string]struct{})
	for _, e := range history.Entries {
		if e.Repository != "" {
			repoSet[e.Repository] = struct{}{}
		}
	}

	return RecommendationResult{
		Recommendations: result,
		AnalyzedRepos:   len(repoSet),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}
}

// FindRepeatedAdditions finds fields added across multiple repositories.
func FindRepeatedAdditions(history TemplateHistory, threshold int) []Recommendation {
	fieldRepos := make(map[string]map[string]bool)
	fieldMeta := make(map[string]HistoryEntry)

	for _, e := range history.Entries {
		if e.Action != "add" {
			continue
		}
		if e.Source == "recommend_accept" {
			continue
		}
		if e.Repository == "" {
			continue
		}
		if _, ok := fieldRepos[e.FieldID]; !ok {
			fieldRepos[e.FieldID] = make(map[string]bool)
		}
		fieldRepos[e.FieldID][e.Repository] = true
		fieldMeta[e.FieldID] = e
	}

	var recs []Recommendation
	for fieldID, repos := range fieldRepos {
		if len(repos) < threshold {
			continue
		}
		entry := fieldMeta[fieldID]
		name := entry.Details["name"]
		if name == "" {
			name = fieldID
		}
		confidence := float64(len(repos)) / float64(threshold*2)
		if confidence > 1.0 {
			confidence = 1.0
		}
		recs = append(recs, Recommendation{
			Type:       RecommendationTypePromoteToDefault,
			FieldID:    fieldID,
			Name:       name,
			Category:   entry.Details["category"],
			Reason:     "repeated additions across multiple repositories",
			Source:     "history_analysis",
			Confidence: confidence,
			RepoCount:  len(repos),
		})
	}
	return recs
}

// FindDeleteReaddPatterns finds fields that were removed then re-added in the same repo.
func FindDeleteReaddPatterns(history TemplateHistory) []Recommendation {
	entries := make([]HistoryEntry, len(history.Entries))
	copy(entries, history.Entries)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp < entries[j].Timestamp
	})

	// track removed state per (repo, fieldID)
	type repoField struct{ repo, fieldID string }
	removed := make(map[repoField]bool)
	seen := make(map[repoField]bool)

	var recs []Recommendation
	for _, e := range entries {
		key := repoField{e.Repository, e.FieldID}
		if e.Action == "remove" {
			removed[key] = true
		} else if e.Action == "add" && removed[key] && !seen[key] {
			seen[key] = true
			name := e.Details["name"]
			if name == "" {
				name = e.FieldID
			}
			recs = append(recs, Recommendation{
				Type:       RecommendationTypeDeleteWarning,
				FieldID:    e.FieldID,
				Name:       name,
				Category:   e.Details["category"],
				Reason:     "field was removed then re-added",
				Source:     "history_analysis",
				Confidence: 0.8,
			})
		}
	}
	return recs
}

// FilterAlreadyInTemplate removes recommendations for fields already in the current template.
func FilterAlreadyInTemplate(recs []Recommendation, current RequirementTemplate) []Recommendation {
	existing := make(map[string]struct{}, len(current.Items))
	for _, item := range current.Items {
		existing[item.ID] = struct{}{}
	}

	result := recs[:0:0]
	for _, r := range recs {
		if _, ok := existing[r.FieldID]; !ok {
			result = append(result, r)
		}
	}
	return result
}
