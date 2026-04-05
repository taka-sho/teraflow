package context

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/skill"
)

func TestIndexBuildToContextAssembleIntegration(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join("testdata", "integration")

	builder := index.NewBuilder(projectRoot)
	idx, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(idx.Entries) == 0 {
		t.Fatal("Build() returned empty index entries")
	}

	assembler := NewAssembler(idx, filepath.Join(projectRoot, ".teraflow", "summaries"), projectRoot)
	cfg := skill.ContextCfg{
		Include:          []string{"docs/**/*.md", "docs/*.md"},
		Exclude:          []string{"docs/guide/beta.md"},
		MaxContextTokens: 80,
	}

	result, err := assembler.Assemble(cfg, "Please focus req:alpha", "alpha guide context")
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if len(result.IncludedDocs) == 0 {
		t.Fatal("IncludedDocs should not be empty")
	}
	for _, nodeID := range result.IncludedDocs {
		if nodeID == "req:beta" {
			t.Fatalf("excluded node should not be included: %v", result.IncludedDocs)
		}
	}
	if !containsString(result.IncludedDocs, "req:alpha") {
		t.Fatalf("IncludedDocs should contain req:alpha, got: %v", result.IncludedDocs)
	}
	if !strings.Contains(result.Context, "Alpha summary") {
		t.Fatalf("Context should contain alpha summary, got: %q", result.Context)
	}
	if strings.Contains(result.Context, "Beta summary") {
		t.Fatalf("Context should not contain beta summary, got: %q", result.Context)
	}
	if result.TotalTokens > cfg.MaxContextTokens {
		t.Fatalf("TotalTokens = %d, want <= %d", result.TotalTokens, cfg.MaxContextTokens)
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
