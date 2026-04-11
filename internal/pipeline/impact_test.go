package pipeline

import (
	"context"
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
