package discovery

import (
	"testing"
)

func makeEntry(action, fieldID, source, repo string, details map[string]string) HistoryEntry {
	return HistoryEntry{
		Action:     action,
		FieldID:    fieldID,
		Source:     source,
		Repository: repo,
		Details:    details,
	}
}

func TestFindRepeatedAdditions(t *testing.T) {
	tests := []struct {
		name      string
		entries   []HistoryEntry
		threshold int
		wantIDs   []string
	}{
		{
			name: "3 repos → recommended",
			entries: []HistoryEntry{
				makeEntry("add", "custom_field", "cli", "repo1", map[string]string{"name": "Custom"}),
				makeEntry("add", "custom_field", "cli", "repo2", map[string]string{"name": "Custom"}),
				makeEntry("add", "custom_field", "cli", "repo3", map[string]string{"name": "Custom"}),
			},
			threshold: 3,
			wantIDs:   []string{"custom_field"},
		},
		{
			name: "2 repos only → not recommended",
			entries: []HistoryEntry{
				makeEntry("add", "custom_field", "cli", "repo1", nil),
				makeEntry("add", "custom_field", "cli", "repo2", nil),
			},
			threshold: 3,
			wantIDs:   nil,
		},
		{
			name: "recommend_accept excluded",
			entries: []HistoryEntry{
				makeEntry("add", "custom_field", "recommend_accept", "repo1", nil),
				makeEntry("add", "custom_field", "recommend_accept", "repo2", nil),
				makeEntry("add", "custom_field", "recommend_accept", "repo3", nil),
			},
			threshold: 3,
			wantIDs:   nil,
		},
		{
			name: "empty repo excluded",
			entries: []HistoryEntry{
				makeEntry("add", "custom_field", "cli", "", nil),
				makeEntry("add", "custom_field", "cli", "", nil),
				makeEntry("add", "custom_field", "cli", "", nil),
			},
			threshold: 3,
			wantIDs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := TemplateHistory{Entries: tt.entries}
			recs := FindRepeatedAdditions(history, tt.threshold)
			if len(tt.wantIDs) == 0 {
				if len(recs) != 0 {
					t.Errorf("want 0 recs, got %d", len(recs))
				}
				return
			}
			if len(recs) != len(tt.wantIDs) {
				t.Fatalf("want %d recs, got %d", len(tt.wantIDs), len(recs))
			}
			got := make(map[string]bool)
			for _, r := range recs {
				got[r.FieldID] = true
			}
			for _, id := range tt.wantIDs {
				if !got[id] {
					t.Errorf("missing field %q in recommendations", id)
				}
			}
		})
	}
}

func TestFindDeleteReaddPatterns(t *testing.T) {
	tests := []struct {
		name    string
		entries []HistoryEntry
		wantN   int
		wantID  string
	}{
		{
			name: "remove then add → delete_warning",
			entries: []HistoryEntry{
				{Timestamp: "2026-01-01T00:00:00Z", Action: "remove", FieldID: "f1", Repository: "repo1"},
				{Timestamp: "2026-01-02T00:00:00Z", Action: "add", FieldID: "f1", Repository: "repo1", Details: map[string]string{"name": "Field1"}},
			},
			wantN:  1,
			wantID: "f1",
		},
		{
			name: "add then remove, no readd → no warning",
			entries: []HistoryEntry{
				{Timestamp: "2026-01-01T00:00:00Z", Action: "add", FieldID: "f1", Repository: "repo1"},
				{Timestamp: "2026-01-02T00:00:00Z", Action: "remove", FieldID: "f1", Repository: "repo1"},
			},
			wantN: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := TemplateHistory{Entries: tt.entries}
			recs := FindDeleteReaddPatterns(history)
			if len(recs) != tt.wantN {
				t.Fatalf("want %d recs, got %d", tt.wantN, len(recs))
			}
			if tt.wantN > 0 {
				if recs[0].FieldID != tt.wantID {
					t.Errorf("want fieldID %q, got %q", tt.wantID, recs[0].FieldID)
				}
				if recs[0].Type != RecommendationTypeDeleteWarning {
					t.Errorf("want type delete_warning, got %q", recs[0].Type)
				}
				if recs[0].Confidence != 0.8 {
					t.Errorf("want confidence 0.8, got %f", recs[0].Confidence)
				}
			}
		})
	}
}

func TestFilterAlreadyInTemplate(t *testing.T) {
	current := RequirementTemplate{
		Items: []TemplateItem{
			{ID: "existing_field", Name: "Existing"},
		},
	}
	recs := []Recommendation{
		{FieldID: "existing_field"},
		{FieldID: "new_field"},
	}
	result := FilterAlreadyInTemplate(recs, current)
	if len(result) != 1 {
		t.Fatalf("want 1 rec, got %d", len(result))
	}
	if result[0].FieldID != "new_field" {
		t.Errorf("want new_field, got %q", result[0].FieldID)
	}
}

func TestAnalyzePatterns(t *testing.T) {
	t.Run("empty history → empty recommendations", func(t *testing.T) {
		result := AnalyzePatterns(TemplateHistory{}, 3)
		if len(result.Recommendations) != 0 {
			t.Errorf("want 0 recs, got %d", len(result.Recommendations))
		}
		if result.Timestamp == "" {
			t.Error("want non-empty timestamp")
		}
	})

	t.Run("pattern1 + pattern2 integration", func(t *testing.T) {
		entries := []HistoryEntry{
			// pattern1: repeated add across 3 repos
			makeEntry("add", "f1", "cli", "repo1", map[string]string{"name": "F1"}),
			makeEntry("add", "f1", "cli", "repo2", map[string]string{"name": "F1"}),
			makeEntry("add", "f1", "cli", "repo3", map[string]string{"name": "F1"}),
			// pattern2: remove → add in same repo
			{Timestamp: "2026-01-01T00:00:00Z", Action: "remove", FieldID: "f2", Repository: "repo1"},
			{Timestamp: "2026-01-02T00:00:00Z", Action: "add", FieldID: "f2", Repository: "repo1"},
		}
		result := AnalyzePatterns(TemplateHistory{Entries: entries}, 3)
		if len(result.Recommendations) < 2 {
			t.Errorf("want >=2 recs, got %d", len(result.Recommendations))
		}
		// sorted by confidence descending: f1 has 3/(3*2)=0.5, f2 has 0.8 → f2 first
		if result.Recommendations[0].FieldID != "f2" {
			t.Errorf("want f2 first (higher confidence), got %q", result.Recommendations[0].FieldID)
		}
	})
}
