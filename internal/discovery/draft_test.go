package discovery

import (
	"strings"
	"testing"
)

func TestBuildDraftFromState(t *testing.T) {
	state := NewSessionState(7, "ユーザー認証要件")
	state.Tree = []Branch{
		{ID: "scope.platform", Question: "対象プラットフォームは？", Category: "scope", Status: StatusAnswered, Answer: "Web + API"},
		{ID: "func.error", Question: "エラー時の振る舞いは？", Category: "functional", Status: StatusPending},
		{ID: "risk.security", Question: "セキュリティ懸念は？", Category: "risk", Status: StatusSkipped, SkipReason: "後続フェーズ"},
	}
	state.RecalculateSummary()

	out, err := BuildDraftFromState(state, DraftMetadata{
		NodeID: "req-user-auth",
		Title:  "ユーザー認証要件",
	})
	if err != nil {
		t.Fatalf("build draft: %v", err)
	}

	checks := []string{
		"node_id: req-user-auth",
		"status: draft",
		"## スコープ",
		"対象プラットフォームは？: Web + API",
		"## 機能要件",
		"エラー時の振る舞いは？: **未確定**",
		"## リスク・未確定事項",
		"セキュリティ懸念は？: 後続フェーズ (skipped)",
	}
	for _, needle := range checks {
		if !strings.Contains(out, needle) {
			t.Fatalf("draft does not contain %q\n%s", needle, out)
		}
	}
}
