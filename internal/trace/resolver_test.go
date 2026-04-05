package trace

import (
	"fmt"
	"testing"

	"github.com/taka-sho/teraflow/internal/index"
)

func TestResolverResolveUpDirection(t *testing.T) {
	idx := makeIndex(
		entry("A", "Node A", "docs/a.md", "B"),
		entry("B", "Node B", "docs/b.md", "C"),
		entry("C", "Node C", "docs/c.md"),
	)

	resolver := NewResolver(idx)
	result, err := resolver.Resolve("A", Up)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if result.RootNodeID != "A" {
		t.Fatalf("RootNodeID = %q, want %q", result.RootNodeID, "A")
	}
	if result.Direction != Up {
		t.Fatalf("Direction = %q, want %q", result.Direction, Up)
	}

	assertNodeIDsWithDepth(t, result.Nodes, []TraceNode{
		{NodeID: "B", Depth: 1},
		{NodeID: "C", Depth: 2},
	})
}

func TestResolverResolveDownDirection(t *testing.T) {
	idx := makeIndex(
		entry("A", "Node A", "docs/a.md", "B"),
		entry("B", "Node B", "docs/b.md", "C"),
		entry("C", "Node C", "docs/c.md"),
	)

	resolver := NewResolver(idx)
	result, err := resolver.Resolve("C", Down)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	assertNodeIDsWithDepth(t, result.Nodes, []TraceNode{
		{NodeID: "B", Depth: 1},
		{NodeID: "A", Depth: 2},
	})
}

func TestResolverResolveBothDirection(t *testing.T) {
	idx := makeIndex(
		entry("A", "Node A", "docs/a.md", "B"),
		entry("B", "Node B", "docs/b.md", "C"),
		entry("C", "Node C", "docs/c.md"),
	)

	resolver := NewResolver(idx)
	result, err := resolver.Resolve("B", Both)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	assertNodeIDsWithDepth(t, result.Nodes, []TraceNode{
		{NodeID: "A", Depth: 1},
		{NodeID: "C", Depth: 1},
	})
}

func TestResolverResolveCycleDetection(t *testing.T) {
	idx := makeIndex(
		entry("A", "Node A", "docs/a.md", "B"),
		entry("B", "Node B", "docs/b.md", "A"),
	)

	resolver := NewResolver(idx)
	result, err := resolver.Resolve("A", Up)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	assertNodeIDsWithDepth(t, result.Nodes, []TraceNode{
		{NodeID: "B", Depth: 1},
	})
}

func TestResolverResolveNodeNotFound(t *testing.T) {
	idx := makeIndex(entry("A", "Node A", "docs/a.md"))
	resolver := NewResolver(idx)

	_, err := resolver.Resolve("missing", Up)
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}
}

func TestResolverResolveMaxDepthLimit(t *testing.T) {
	entries := make([]index.Entry, 0, 12)
	for i := 0; i <= 11; i++ {
		nodeID := fmt.Sprintf("N%d", i)
		dependsOn := []string{}
		if i < 11 {
			dependsOn = append(dependsOn, fmt.Sprintf("N%d", i+1))
		}
		entries = append(entries, entry(nodeID, nodeID, fmt.Sprintf("docs/%s.md", nodeID), dependsOn...))
	}

	resolver := NewResolver(makeIndex(entries...))
	result, err := resolver.Resolve("N0", Up)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if got, want := len(result.Nodes), 10; got != want {
		t.Fatalf("len(Nodes) = %d, want %d", got, want)
	}
	if got, want := result.Nodes[len(result.Nodes)-1].NodeID, "N10"; got != want {
		t.Fatalf("last node = %q, want %q", got, want)
	}
	if got, want := result.Nodes[len(result.Nodes)-1].Depth, 10; got != want {
		t.Fatalf("last depth = %d, want %d", got, want)
	}
}

func makeIndex(entries ...index.Entry) *index.Index {
	return &index.Index{Entries: entries}
}

func entry(nodeID, title, path string, dependsOn ...string) index.Entry {
	return index.Entry{
		NodeID:    nodeID,
		Title:     title,
		Path:      path,
		DependsOn: dependsOn,
	}
}

func assertNodeIDsWithDepth(t *testing.T, got []TraceNode, want []TraceNode) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(nodes) = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i].NodeID != want[i].NodeID {
			t.Fatalf("nodes[%d].NodeID = %q, want %q", i, got[i].NodeID, want[i].NodeID)
		}
		if got[i].Depth != want[i].Depth {
			t.Fatalf("nodes[%d].Depth = %d, want %d", i, got[i].Depth, want[i].Depth)
		}
	}
}
