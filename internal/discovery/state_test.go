package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionStateSaveLoadAndSummary(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".teraflow", "discovery", "discussion-42.yaml")

	s := NewSessionState(42, "ユーザー認証要件")
	s.Tree = []Branch{
		{ID: "scope.platform", Question: "対象プラットフォーム", Category: "scope", Status: StatusAnswered, Answer: "Web + API"},
		{ID: "nfr.performance", Question: "レスポンスタイム", Category: "non_functional", Status: StatusPending},
		{ID: "risk.security", Question: "セキュリティ", Category: "risk", Status: StatusSkipped, SkipReason: "Phase 2で検討"},
	}

	if err := s.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadSessionState(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Summary.Total != 3 {
		t.Fatalf("summary total = %d, want 3", loaded.Summary.Total)
	}
	if loaded.Summary.Answered != 1 {
		t.Fatalf("summary answered = %d, want 1", loaded.Summary.Answered)
	}
	if loaded.Summary.ProgressPercent != 66 {
		t.Fatalf("summary progress = %d, want 66", loaded.Summary.ProgressPercent)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("state file missing: %v", err)
	}
}

func TestSessionStateDependencyBlockingAndRelease(t *testing.T) {
	s := NewSessionState(12, "dependency")
	s.Tree = []Branch{
		{ID: "scope.platform", Question: "platform", Category: "scope", Status: StatusPending},
		{ID: "scope.mobile.auth", Question: "mobile auth", Category: "functional", Status: StatusPending, DependsOn: []string{"scope.platform"}},
	}

	s.UpdateBlockedByDependencies()
	child := s.FindBranch("scope.mobile.auth")
	if child == nil {
		t.Fatal("child not found")
	}
	if child.Status != StatusBlocked {
		t.Fatalf("child status = %s, want blocked", child.Status)
	}

	if err := s.MarkAnswered("scope.platform", "Web + Mobile", "user"); err != nil {
		t.Fatalf("mark answered: %v", err)
	}
	if child.Status != StatusPending {
		t.Fatalf("child status after resolve = %s, want pending", child.Status)
	}
}

func TestSessionStateNextQuestions(t *testing.T) {
	s := NewSessionState(13, "next")
	s.Tree = []Branch{
		{ID: "a", Question: "A", Status: StatusPending},
		{ID: "b", Question: "B", Status: StatusPending, DependsOn: []string{"a"}},
		{ID: "c", Question: "C", Status: StatusPending},
	}
	s.UpdateBlockedByDependencies()

	next := s.NextQuestions(10)
	if len(next) != 2 {
		t.Fatalf("next len = %d, want 2", len(next))
	}
	if next[0].ID != "a" || next[1].ID != "c" {
		t.Fatalf("unexpected next ids: %v, %v", next[0].ID, next[1].ID)
	}

	if err := s.MarkAnswered("a", "ok", "user"); err != nil {
		t.Fatalf("mark answered: %v", err)
	}
	next = s.NextQuestions(10)
	if len(next) != 2 {
		t.Fatalf("next len after answer = %d, want 2", len(next))
	}
}
