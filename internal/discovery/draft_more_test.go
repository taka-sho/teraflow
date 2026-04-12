package discovery

import (
	"strings"
	"testing"
)

func TestBuildDraftFromStateNilState(t *testing.T) {
	_, err := BuildDraftFromState(nil, DraftMetadata{})
	if err == nil {
		t.Fatal("nil state should return error")
	}
}

func TestBuildDraftFromStateDefaultMeta(t *testing.T) {
	state := NewSessionState(42, "Test Title")
	state.Tree = []Branch{
		{ID: "a", Question: "Q", Category: "scope", Status: StatusAnswered, Answer: "yes"},
	}
	out, err := BuildDraftFromState(state, DraftMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "discussion-42") {
		t.Fatalf("missing default node_id: %s", out)
	}
	if !strings.Contains(out, "Test Title") {
		t.Fatalf("missing title: %s", out)
	}
	if !strings.Contains(out, "discussion:#42") {
		t.Fatalf("missing source: %s", out)
	}
}

func TestFormatBranchLineAllStatuses(t *testing.T) {
	tests := []struct {
		name   string
		branch Branch
		want   string
	}{
		{
			name:   "answered with answer",
			branch: Branch{ID: "a", Question: "Q", Status: StatusAnswered, Answer: "yes"},
			want:   "Q: yes",
		},
		{
			name:   "answered empty answer",
			branch: Branch{ID: "a", Question: "Q", Status: StatusAnswered, Answer: ""},
			want:   "Q: （回答済み）",
		},
		{
			name:   "skipped with reason",
			branch: Branch{ID: "a", Question: "Q", Status: StatusSkipped, SkipReason: "later"},
			want:   "Q: later (skipped)",
		},
		{
			name:   "skipped empty reason",
			branch: Branch{ID: "a", Question: "Q", Status: StatusSkipped, SkipReason: ""},
			want:   "Q: 理由未記入 (skipped)",
		},
		{
			name:   "blocked",
			branch: Branch{ID: "a", Question: "Q", Status: StatusBlocked},
			want:   "Q: **保留（依存未解決）**",
		},
		{
			name:   "pending",
			branch: Branch{ID: "a", Question: "Q", Status: StatusPending},
			want:   "Q: **未確定**",
		},
		{
			name:   "empty question uses ID",
			branch: Branch{ID: "my-id", Question: "", Status: StatusPending},
			want:   "my-id: **未確定**",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBranchLine(tt.branch)
			if got != tt.want {
				t.Fatalf("formatBranchLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSectionHeadingByCategory(t *testing.T) {
	tests := []struct {
		category string
		want     string
	}{
		{"scope", "スコープ"},
		{"functional", "機能要件"},
		{"non_functional", "非機能要件"},
		{"acceptance", "受入条件"},
		{"risk", "リスク・未確定事項"},
		{"dependency", "依存関係・前提条件"},
		{"priority", "優先度・フェーズ"},
		{"unknown", "背景・課題"},
		{"", "背景・課題"},
		{"SCOPE", "スコープ"},
	}
	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := sectionHeadingByCategory(tt.category)
			if got != tt.want {
				t.Fatalf("sectionHeadingByCategory(%q) = %q, want %q", tt.category, got, tt.want)
			}
		})
	}
}

func TestRenderEmptySections(t *testing.T) {
	d := NewDraft(DraftMetadata{
		NodeID: "test",
		Title:  "Test",
	})
	out := d.Render(Progress{Total: 3, Answered: 1, ProgressPercent: 33})
	if !strings.Contains(out, "（未記入）") {
		t.Fatalf("empty sections should show placeholder: %s", out)
	}
}

func TestFlattenBranchesNested(t *testing.T) {
	nodes := []Branch{
		{ID: "a", Children: []Branch{
			{ID: "b", Children: []Branch{
				{ID: "c"},
			}},
		}},
		{ID: "d"},
	}
	flat := flattenBranches(nodes)
	if len(flat) != 4 {
		t.Fatalf("flattenBranches = %d items, want 4", len(flat))
	}
}

func TestNewDraftDefaultStatus(t *testing.T) {
	d := NewDraft(DraftMetadata{NodeID: "test", Title: "T"})
	if d.Meta.Status != "draft" {
		t.Fatalf("status = %q, want draft", d.Meta.Status)
	}
}

func TestNewDraftCustomStatus(t *testing.T) {
	d := NewDraft(DraftMetadata{NodeID: "test", Title: "T", Status: "review"})
	if d.Meta.Status != "review" {
		t.Fatalf("status = %q, want review", d.Meta.Status)
	}
}
