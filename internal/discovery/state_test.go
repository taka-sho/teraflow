package discovery

import (
	"os"
	"path/filepath"
	"strings"
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

func TestSessionStateAddBranchAndLookup(t *testing.T) {
	s := NewSessionState(20, "add branch")
	if err := s.AddBranch("", Branch{ID: "root", Question: "root", Status: "  ANSWERED "}); err != nil {
		t.Fatalf("add root branch: %v", err)
	}
	if err := s.AddBranch("root", Branch{ID: "child", Question: "child", Status: "resolved"}); err != nil {
		t.Fatalf("add child branch: %v", err)
	}

	root := s.FindBranch("root")
	if root == nil || len(root.Children) != 1 {
		t.Fatalf("expected child on root, got %+v", root)
	}
	if root.Children[0].Status != StatusAnswered {
		t.Fatalf("resolved status should normalize to answered, got %q", root.Children[0].Status)
	}
}

func TestSessionStateAddBranchErrors(t *testing.T) {
	s := NewSessionState(21, "add branch errors")
	if err := s.AddBranch("", Branch{ID: "", Question: "x"}); err == nil {
		t.Fatal("expected missing branch id error")
	}
	if err := s.AddBranch("", Branch{ID: "dup", Question: "x"}); err != nil {
		t.Fatalf("unexpected add branch error: %v", err)
	}
	if err := s.AddBranch("", Branch{ID: "dup", Question: "x"}); err == nil {
		t.Fatal("expected duplicate branch id error")
	}
	if err := s.AddBranch("missing", Branch{ID: "child", Question: "x"}); err == nil {
		t.Fatal("expected missing parent error")
	}
}

func TestSessionStateMarkSkippedBlockedAndPendingBranches(t *testing.T) {
	s := NewSessionState(22, "marks")
	s.Tree = []Branch{
		{ID: "a", Question: "A", Status: StatusPending},
		{ID: "b", Question: "B", Status: StatusPending},
	}

	if err := s.MarkSkipped("a", "not needed"); err != nil {
		t.Fatalf("mark skipped: %v", err)
	}
	if s.FindBranch("a").Status != StatusSkipped {
		t.Fatalf("expected skipped status, got %q", s.FindBranch("a").Status)
	}

	if err := s.MarkBlocked("b"); err != nil {
		t.Fatalf("mark blocked: %v", err)
	}
	if s.FindBranch("b").Status != StatusBlocked {
		t.Fatalf("expected blocked status, got %q", s.FindBranch("b").Status)
	}

	if err := s.MarkSkipped("missing", "x"); err == nil {
		t.Fatal("expected missing branch error on skip")
	}
	if err := s.MarkBlocked("missing"); err == nil {
		t.Fatal("expected missing branch error on blocked")
	}

	pending := s.PendingBranches()
	if len(pending) != 0 {
		t.Fatalf("pending should be empty, got %d", len(pending))
	}
}

func TestSessionStateDependenciesSatisfiedAndCompletion(t *testing.T) {
	s := NewSessionState(23, "deps")
	s.Tree = []Branch{
		{ID: "a", Question: "A", Status: StatusAnswered},
		{ID: "b", Question: "B", Status: StatusSkipped},
		{ID: "c", Question: "C", Status: StatusPending, DependsOn: []string{"a", "b"}},
	}

	if !s.DependenciesSatisfied([]string{"a", "b"}) {
		t.Fatal("dependencies should be satisfied by answered/skipped")
	}
	if s.DependenciesSatisfied([]string{"a", "missing"}) {
		t.Fatal("dependencies should fail for missing branch")
	}

	if s.IsComplete() {
		t.Fatal("state with pending branch should not be complete")
	}
	if err := s.MarkAnswered("c", "done", "agent"); err != nil {
		t.Fatalf("mark answered: %v", err)
	}
	if !s.IsComplete() {
		t.Fatal("all branches resolved/skipped should be complete")
	}
	if report := s.CompletionReport(); !strings.Contains(report, "要件探索完了") {
		t.Fatalf("unexpected completion report: %q", report)
	}
}

func TestSessionStateLoadParseAndNilSave(t *testing.T) {
	tmp := t.TempDir()
	badPath := filepath.Join(tmp, "bad.yaml")
	if err := os.WriteFile(badPath, []byte(":\n bad: ["), 0o644); err != nil {
		t.Fatalf("write bad yaml: %v", err)
	}

	if _, err := LoadSessionState(badPath); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error from bad yaml, got: %v", err)
	}

	var nilState *SessionState
	if err := nilState.Save(filepath.Join(tmp, "x.yaml")); err == nil {
		t.Fatal("expected nil session state error")
	}
}
