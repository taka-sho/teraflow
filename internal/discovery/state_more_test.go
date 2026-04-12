package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsComplete(t *testing.T) {
	tests := []struct {
		name string
		tree []Branch
		want bool
	}{
		{
			name: "all answered",
			tree: []Branch{
				{ID: "a", Status: StatusAnswered},
				{ID: "b", Status: StatusSkipped},
			},
			want: true,
		},
		{
			name: "has pending",
			tree: []Branch{
				{ID: "a", Status: StatusAnswered},
				{ID: "b", Status: StatusPending},
			},
			want: false,
		},
		{
			name: "has blocked",
			tree: []Branch{
				{ID: "a", Status: StatusAnswered},
				{ID: "b", Status: StatusBlocked},
			},
			want: false,
		},
		{
			name: "empty tree",
			tree: []Branch{},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSessionState(1, "test")
			s.Tree = tt.tree
			if got := s.IsComplete(); got != tt.want {
				t.Fatalf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsCompleteNilReceiver(t *testing.T) {
	var s *SessionState
	if s.IsComplete() != false {
		t.Fatal("nil receiver should return false")
	}
}

func TestCompletionReport(t *testing.T) {
	s := NewSessionState(1, "test")
	s.Tree = []Branch{
		{ID: "a", Status: StatusAnswered},
		{ID: "b", Status: StatusSkipped},
		{ID: "c", Status: StatusPending},
	}
	report := s.CompletionReport()
	if report == "" {
		t.Fatal("report should not be empty")
	}

	var nilState *SessionState
	if got := nilState.CompletionReport(); got != "" {
		t.Fatalf("nil CompletionReport() = %q, want empty", got)
	}
}

func TestAddBranch(t *testing.T) {
	tests := []struct {
		name     string
		parentID string
		branch   Branch
		wantErr  bool
	}{
		{
			name:     "add to root",
			parentID: "",
			branch:   Branch{ID: "new1", Question: "Q1", Status: StatusPending},
		},
		{
			name:     "add to parent",
			parentID: "root",
			branch:   Branch{ID: "child1", Question: "Q2", Status: StatusPending},
		},
		{
			name:     "empty id",
			parentID: "",
			branch:   Branch{ID: "", Question: "Q"},
			wantErr:  true,
		},
		{
			name:     "duplicate id",
			parentID: "",
			branch:   Branch{ID: "root", Question: "dup"},
			wantErr:  true,
		},
		{
			name:     "parent not found",
			parentID: "nonexistent",
			branch:   Branch{ID: "orphan", Question: "Q"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSessionState(1, "test")
			s.Tree = []Branch{{ID: "root", Question: "Root", Status: StatusPending}}

			err := s.AddBranch(tt.parentID, tt.branch)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AddBranch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddBranchNilReceiver(t *testing.T) {
	var s *SessionState
	err := s.AddBranch("", Branch{ID: "x"})
	if err == nil {
		t.Fatal("nil receiver should return error")
	}
}

func TestMarkSkipped(t *testing.T) {
	s := NewSessionState(1, "test")
	s.Tree = []Branch{
		{ID: "a", Question: "Q", Status: StatusPending},
	}

	if err := s.MarkSkipped("a", "out of scope"); err != nil {
		t.Fatalf("MarkSkipped() error = %v", err)
	}
	node := s.FindBranch("a")
	if node.Status != StatusSkipped {
		t.Fatalf("status = %s, want skipped", node.Status)
	}
	if node.SkipReason != "out of scope" {
		t.Fatalf("skip_reason = %q", node.SkipReason)
	}
	if node.ResolvedBy != "user" {
		t.Fatalf("resolved_by = %q, want user", node.ResolvedBy)
	}
}

func TestMarkSkippedNotFound(t *testing.T) {
	s := NewSessionState(1, "test")
	if err := s.MarkSkipped("nonexistent", "reason"); err == nil {
		t.Fatal("should error for nonexistent branch")
	}
}

func TestMarkBlocked(t *testing.T) {
	s := NewSessionState(1, "test")
	s.Tree = []Branch{
		{ID: "a", Question: "Q", Status: StatusPending},
	}

	if err := s.MarkBlocked("a"); err != nil {
		t.Fatalf("MarkBlocked() error = %v", err)
	}
	node := s.FindBranch("a")
	if node.Status != StatusBlocked {
		t.Fatalf("status = %s, want blocked", node.Status)
	}
}

func TestMarkBlockedNotFound(t *testing.T) {
	s := NewSessionState(1, "test")
	if err := s.MarkBlocked("nonexistent"); err == nil {
		t.Fatal("should error for nonexistent branch")
	}
}

func TestPendingBranches(t *testing.T) {
	s := NewSessionState(1, "test")
	s.Tree = []Branch{
		{ID: "b", Status: StatusPending},
		{ID: "a", Status: StatusPending},
		{ID: "c", Status: StatusAnswered},
	}
	pending := s.PendingBranches()
	if len(pending) != 2 {
		t.Fatalf("pending count = %d, want 2", len(pending))
	}
	if pending[0].ID != "a" || pending[1].ID != "b" {
		t.Fatalf("pending not sorted: %v, %v", pending[0].ID, pending[1].ID)
	}
}

func TestPendingBranchesNilReceiver(t *testing.T) {
	var s *SessionState
	if got := s.PendingBranches(); got != nil {
		t.Fatalf("nil receiver should return nil, got %v", got)
	}
}

func TestFindBranchNilReceiver(t *testing.T) {
	var s *SessionState
	if got := s.FindBranch("x"); got != nil {
		t.Fatal("nil receiver should return nil")
	}
}

func TestFindBranchInChildren(t *testing.T) {
	s := NewSessionState(1, "test")
	s.Tree = []Branch{
		{ID: "parent", Children: []Branch{
			{ID: "child", Question: "Q"},
		}},
	}
	if got := s.FindBranch("child"); got == nil {
		t.Fatal("should find child branch")
	}
}

func TestNormalizeStatusResolved(t *testing.T) {
	if got := normalizeStatus("resolved"); got != StatusAnswered {
		t.Fatalf("normalizeStatus(resolved) = %q, want answered", got)
	}
	if got := normalizeStatus("PENDING"); got != StatusPending {
		t.Fatalf("normalizeStatus(PENDING) = %q, want pending", got)
	}
	if got := normalizeStatus(""); got != StatusPending {
		t.Fatalf("normalizeStatus('') = %q, want pending", got)
	}
	if got := normalizeStatus("unknown_status"); got != StatusPending {
		t.Fatalf("normalizeStatus(unknown) = %q, want pending", got)
	}
}

func TestSaveNilReceiver(t *testing.T) {
	var s *SessionState
	if err := s.Save("/tmp/test.yaml"); err == nil {
		t.Fatal("nil receiver should return error")
	}
}

func TestSaveVersionAndCreatedAtDefaults(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "state.yaml")

	s := &SessionState{
		DiscussionNumber: 1,
		Title:            "test",
		Tree:             []Branch{},
	}
	if err := s.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if s.Version != "1" {
		t.Fatalf("version = %q, want 1", s.Version)
	}
	if s.CreatedAt == "" {
		t.Fatal("created_at should be set")
	}
}

func TestLoadSessionStateInvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.yaml")
	if err := os.WriteFile(path, []byte(":::invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSessionState(path)
	if err == nil {
		t.Fatal("should error on invalid YAML")
	}
}

func TestLoadSessionStateNotFound(t *testing.T) {
	_, err := LoadSessionState("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("should error on missing file")
	}
}

func TestRecalculateSummaryNilReceiver(t *testing.T) {
	var s *SessionState
	s.RecalculateSummary() // should not panic
}

func TestUpdateBlockedByDependenciesNilReceiver(t *testing.T) {
	var s *SessionState
	s.UpdateBlockedByDependencies() // should not panic
}

func TestMarkAnsweredNotFound(t *testing.T) {
	s := NewSessionState(1, "test")
	if err := s.MarkAnswered("nonexistent", "answer", "user"); err == nil {
		t.Fatal("should error for nonexistent branch")
	}
}

func TestMarkAnsweredDefaultResolvedBy(t *testing.T) {
	s := NewSessionState(1, "test")
	s.Tree = []Branch{{ID: "a", Status: StatusPending}}
	if err := s.MarkAnswered("a", "yes", ""); err != nil {
		t.Fatal(err)
	}
	if s.FindBranch("a").ResolvedBy != "user" {
		t.Fatalf("resolvedBy = %q, want user", s.FindBranch("a").ResolvedBy)
	}
}

func TestWalkBranchesNestedChildren(t *testing.T) {
	nodes := []Branch{
		{ID: "a", Children: []Branch{
			{ID: "b", Children: []Branch{
				{ID: "c"},
			}},
		}},
	}
	var ids []string
	walkBranches(nodes, func(b *Branch) {
		ids = append(ids, b.ID)
	})
	if len(ids) != 3 {
		t.Fatalf("walked %d nodes, want 3", len(ids))
	}
}

func TestNextQuestionsDefaultLimit(t *testing.T) {
	s := NewSessionState(1, "test")
	s.Tree = []Branch{
		{ID: "a", Status: StatusPending},
		{ID: "b", Status: StatusPending},
	}
	// limit <= 0 defaults to 1
	next := s.NextQuestions(0)
	if len(next) != 1 {
		t.Fatalf("next len = %d, want 1 (default limit)", len(next))
	}
}

func TestNextQuestionsNilReceiver(t *testing.T) {
	var s *SessionState
	if got := s.NextQuestions(5); got != nil {
		t.Fatalf("nil receiver should return nil, got %v", got)
	}
}
