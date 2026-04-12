package pipeline

import (
	"context"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/index"
)

func TestImpactAnalyzerFallback(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "req:a", DependsOn: nil},
		{NodeID: "design:b", DependsOn: []string{"req:a"}},
		{NodeID: "impl:c", DependsOn: []string{"design:b"}},
		{NodeID: "test:d", DependsOn: []string{"impl:c"}},
	}}

	analyzer := NewImpactAnalyzer(idx, nil, t.TempDir())
	result, err := analyzer.Analyze(context.Background(), "req:a", 3)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if got, want := result.Summary["total"], 3; got != want {
		t.Fatalf("total = %d, want %d", got, want)
	}
	if got, want := result.Summary["gray"], 1; got != want {
		t.Fatalf("gray = %d, want %d", got, want)
	}
	if got, want := result.Summary["amber"], 2; got != want {
		t.Fatalf("amber = %d, want %d", got, want)
	}
	if got, want := result.Summary["green"], 0; got != want {
		t.Fatalf("green = %d, want %d", got, want)
	}
}

func TestImpactAnalyzerDeleteEscalatesSeverity(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "req:a", Status: "deleted"},
		{NodeID: "design:b", DependsOn: []string{"req:a"}},
		{NodeID: "impl:c", DependsOn: []string{"design:b"}},
		{NodeID: "test:d", DependsOn: []string{"impl:c"}},
	}}

	analyzer := NewImpactAnalyzer(idx, nil, t.TempDir())
	result, err := analyzer.Analyze(context.Background(), "req:a", 3)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if got, want := result.ChangeType, "delete"; got != want {
		t.Fatalf("change_type = %q, want %q", got, want)
	}
	if got, want := result.Summary["gray"], 2; got != want {
		t.Fatalf("gray = %d, want %d", got, want)
	}
	if got, want := result.Summary["amber"], 1; got != want {
		t.Fatalf("amber = %d, want %d", got, want)
	}
}

func TestImpactAnalyzerValidationErrors(t *testing.T) {
	analyzer := NewImpactAnalyzer(nil, nil, t.TempDir())
	if _, err := analyzer.AnalyzeWithChangeType(context.Background(), "x", 1, "modify"); err == nil || !strings.Contains(err.Error(), "index is required") {
		t.Fatalf("expected index required error, got: %v", err)
	}

	idx := &index.Index{Entries: []index.Entry{{NodeID: "req:a"}}}
	analyzer = NewImpactAnalyzer(idx, nil, t.TempDir())
	if _, err := analyzer.AnalyzeWithChangeType(context.Background(), "", 1, "modify"); err == nil {
		t.Fatal("expected changed node id required error")
	}
	if _, err := analyzer.AnalyzeWithChangeType(context.Background(), "req:a", 0, "modify"); err == nil {
		t.Fatal("expected depth validation error")
	}
	if _, err := analyzer.AnalyzeWithChangeType(context.Background(), "missing", 1, "modify"); err == nil {
		t.Fatal("expected node not found error")
	}
}

func TestParseBridgeImpactAndHelpers(t *testing.T) {
	nodes := parseBridgeImpact(map[string]any{
		"affected_nodes": []any{
			map[string]any{
				"node_id":   "design:a",
				"depth":     float64(2),
				"edge_type": "RELATED_TO",
				"source":    "graph",
			},
			map[string]any{
				"node_id":     "impl:b",
				"depth":       1,
				"change_type": "delete",
			},
			map[string]any{},
			"bad",
		},
	}, "modify")
	if len(nodes) != 2 {
		t.Fatalf("expected 2 parsed nodes, got %d", len(nodes))
	}
	if nodes[0].Reason == "" || nodes[1].Band == "" {
		t.Fatalf("unexpected parsed nodes: %+v", nodes)
	}

	if got := toInt(int64(7)); got != 7 {
		t.Fatalf("toInt int64 mismatch: %d", got)
	}
	if got := toInt("bad"); got != 0 {
		t.Fatalf("toInt fallback mismatch: %d", got)
	}

	if got := inferChangeType(nil); got != "modify" {
		t.Fatalf("inferChangeType nil mismatch: %q", got)
	}
	if got := inferChangeType(&index.Entry{Status: "new"}); got != "add" {
		t.Fatalf("inferChangeType add mismatch: %q", got)
	}
	if got := inferChangeType(&index.Entry{Status: "removed"}); got != "delete" {
		t.Fatalf("inferChangeType delete mismatch: %q", got)
	}

	if got := normalizeChangeType("created"); got != "add" {
		t.Fatalf("normalizeChangeType created mismatch: %q", got)
	}
	if got := normalizeChangeType("remove"); got != "delete" {
		t.Fatalf("normalizeChangeType remove mismatch: %q", got)
	}
	if got := normalizeChangeType("other"); got != "modify" {
		t.Fatalf("normalizeChangeType fallback mismatch: %q", got)
	}
}

func TestClassifyBandByDistance(t *testing.T) {
	if got := classifyBandByDistance(0, "modify"); got != "gray" {
		t.Fatalf("distance floor gray mismatch: %q", got)
	}
	if got := classifyBandByDistance(2, "add"); got != "amber" {
		t.Fatalf("add distance=2 expected amber, got %q", got)
	}
	if got := classifyBandByDistance(5, "modify"); got != "green" {
		t.Fatalf("modify distance=5 expected green, got %q", got)
	}
}
