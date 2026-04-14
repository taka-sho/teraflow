package discovery

import "testing"

func TestAnalyzeGapFulfillment(t *testing.T) {
	tmpl := RequirementTemplate{Items: []TemplateItem{
		{ID: "project_overview", Category: TemplateCategoryRequired},
		{ID: "security_requirements", Category: TemplateCategoryRequired},
		{ID: "decision_makers", Category: TemplateCategoryRecommended},
	}}
	state := NewSessionState(1, "gap")
	state.Tree = []Branch{
		{ID: "project_overview", Status: StatusAnswered, Answer: "Web受発注を効率化し、運用工数を50%削減する"},
		{ID: "security_requirements", Status: StatusAnswered, Answer: "未定"},
		{ID: "decision_makers", Status: StatusAnswered, Answer: "POと事業責任者"},
	}

	got := AnalyzeGap(state, tmpl)

	if got.RequiredTotal != 2 || got.RequiredFulfilled != 1 {
		t.Fatalf("required fulfillment = %d/%d, want 1/2", got.RequiredFulfilled, got.RequiredTotal)
	}
	if got.RecommendedTotal != 1 || got.RecommendedFilled != 1 {
		t.Fatalf("recommended fulfillment = %d/%d, want 1/1", got.RecommendedFilled, got.RecommendedTotal)
	}
	if got.Fulfillment["project_overview"] != true {
		t.Fatal("project_overview should be fulfilled")
	}
	if got.Fulfillment["security_requirements"] != false {
		t.Fatal("security_requirements should be unfulfilled")
	}
	if len(got.MissingRequired) != 1 || got.MissingRequired[0].ID != "security_requirements" {
		t.Fatalf("missing required mismatch: %+v", got.MissingRequired)
	}
}

func TestAnalyzeGapMissingNodes(t *testing.T) {
	tmpl := RequirementTemplate{Items: []TemplateItem{
		{ID: "project_overview", Category: TemplateCategoryRequired},
		{ID: "decision_makers", Category: TemplateCategoryRecommended},
	}}
	state := NewSessionState(1, "gap")

	got := AnalyzeGap(state, tmpl)
	if got.RequiredFulfilled != 0 || got.RecommendedFilled != 0 {
		t.Fatalf("unexpected fulfilled counts: %+v", got)
	}
	if len(got.MissingRequired) != 1 || len(got.MissingRecommended) != 1 {
		t.Fatalf("missing lists mismatch: %+v", got)
	}
}

func TestAnalyzeGapShortAnswerIsUnfulfilled(t *testing.T) {
	tmpl := RequirementTemplate{Items: []TemplateItem{{ID: "project_overview", Category: TemplateCategoryRequired}}}
	state := NewSessionState(1, "gap")
	state.Tree = []Branch{{ID: "project_overview", Status: StatusAnswered, Answer: "短い"}}

	got := AnalyzeGap(state, tmpl)
	if got.RequiredFulfilled != 0 {
		t.Fatalf("short answer should be unfulfilled: %+v", got)
	}
}
