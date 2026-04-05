package index_test

import (
	"testing"

	"github.com/taka-sho/teraflow/internal/index"
)

func TestIndexFindByNodeID(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{{NodeID: "a"}, {NodeID: "b"}}}

	if idx.FindByNodeID("b") == nil {
		t.Fatal("FindByNodeID(b) = nil, want entry")
	}
	if idx.FindByNodeID("z") != nil {
		t.Fatal("FindByNodeID(z) != nil, want nil")
	}
}

func TestIndexFindByTags(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "a", Tags: []string{"core", "api"}},
		{NodeID: "b", Tags: []string{"docs"}},
		{NodeID: "c", Tags: []string{"infra"}},
	}}

	got := idx.FindByTags([]string{"docs", "security"})
	if len(got) != 1 || got[0].NodeID != "b" {
		t.Fatalf("FindByTags() = %#v, want node b", got)
	}
}

func TestIndexFindRelatedBFS(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "A", DependsOn: []string{"B", "C"}},
		{NodeID: "B", DependsOn: []string{"D"}},
		{NodeID: "C", DependsOn: []string{"D"}},
		{NodeID: "D", DependsOn: []string{"A"}}, // cycle
	}}

	got := idx.FindRelated("A", 2)
	if len(got) != 3 {
		t.Fatalf("FindRelated() len = %d, want 3", len(got))
	}
	if got[0].NodeID != "B" || got[1].NodeID != "C" || got[2].NodeID != "D" {
		t.Fatalf("FindRelated() order = [%s,%s,%s], want [B,C,D]", got[0].NodeID, got[1].NodeID, got[2].NodeID)
	}

	shallow := idx.FindRelated("A", 1)
	if len(shallow) != 2 {
		t.Fatalf("FindRelated(maxDepth=1) len = %d, want 2", len(shallow))
	}
}
