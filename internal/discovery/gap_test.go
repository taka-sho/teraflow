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

func TestCanAutoConfirm(t *testing.T) {
	t.Run("all required fulfilled", func(t *testing.T) {
		result := GapAnalysisResult{
			RequiredTotal:     2,
			RequiredFulfilled: 2,
		}
		ok, missing := result.CanAutoConfirm()
		if !ok {
			t.Fatal("CanAutoConfirm() should be true when all required items are fulfilled")
		}
		if len(missing) != 0 {
			t.Fatalf("missing should be empty, got %+v", missing)
		}
	})

	t.Run("missing required remains", func(t *testing.T) {
		result := GapAnalysisResult{
			RequiredTotal:     2,
			RequiredFulfilled: 1,
			MissingRequired: []TemplateItem{
				{ID: "security_requirements", Name: "セキュリティ要件"},
			},
		}
		ok, missing := result.CanAutoConfirm()
		if ok {
			t.Fatal("CanAutoConfirm() should be false when required items are missing")
		}
		if len(missing) != 1 || missing[0].ID != "security_requirements" {
			t.Fatalf("missing mismatch: %+v", missing)
		}
	})
}

func TestBuildConfirmWarningMessage(t *testing.T) {
	msg := BuildConfirmWarningMessage([]string{"プロジェクトの目的", "セキュリティ要件"})
	want := "⚠️ 未充足の必須項目があります: プロジェクトの目的, セキュリティ要件。それでも確定する場合はコメントを続けてください。"
	if msg != want {
		t.Fatalf("warning message mismatch:\n got: %q\nwant: %q", msg, want)
	}

	empty := BuildConfirmWarningMessage([]string{" ", ""})
	if empty != "" {
		t.Fatalf("expected empty message for blank labels, got %q", empty)
	}
}
