package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/skill"
)

func TestAssembleIncludeExcludeAndTokenLimit(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	summaryDir := filepath.Join(root, ".teraflow", "summaries")
	if err := os.MkdirAll(summaryDir, 0o755); err != nil {
		t.Fatalf("mkdir summaries: %v", err)
	}

	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}

	docAPath := filepath.Join(docsDir, "alpha.md")
	docBPath := filepath.Join(docsDir, "beta.md")
	if err := os.WriteFile(docAPath, []byte(strings.Repeat("A", 2000)), 0o644); err != nil {
		t.Fatalf("write doc A: %v", err)
	}
	if err := os.WriteFile(docBPath, []byte("short"), 0o644); err != nil {
		t.Fatalf("write doc B: %v", err)
	}

	if err := os.WriteFile(filepath.Join(summaryDir, "req_alpha.txt"), []byte("summary alpha"), 0o644); err != nil {
		t.Fatalf("write summary A: %v", err)
	}
	if err := os.WriteFile(filepath.Join(summaryDir, "req_beta.txt"), []byte("summary beta"), 0o644); err != nil {
		t.Fatalf("write summary B: %v", err)
	}

	idx := &index.Index{Entries: []index.Entry{
		{
			NodeID:    "req:alpha",
			Title:     "Alpha",
			Path:      "docs/alpha.md",
			Tags:      []string{"important"},
			UpdatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			NodeID:    "req:beta",
			Title:     "Beta",
			Path:      "docs/beta.md",
			Tags:      []string{"important"},
			UpdatedAt: time.Now().Add(-3 * time.Hour),
		},
	}}

	assembler := NewAssembler(idx, summaryDir, root)
	result, err := assembler.Assemble(skill.ContextCfg{
		Include:          []string{"docs/*.md"},
		Exclude:          []string{"docs/beta.md"},
		MaxContextTokens: 50,
	}, "focus req:alpha", "important")
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if len(result.IncludedDocs) != 1 || result.IncludedDocs[0] != "req:alpha" {
		t.Fatalf("IncludedDocs = %#v, want [req:alpha]", result.IncludedDocs)
	}
	if !strings.Contains(result.Context, "summary alpha") {
		t.Fatalf("Context should contain alpha summary, got: %q", result.Context)
	}
	if strings.Contains(result.Context, "summary beta") {
		t.Fatalf("Context should exclude beta summary, got: %q", result.Context)
	}
	if result.TotalTokens > 50 {
		t.Fatalf("TotalTokens = %d, should be <= 50", result.TotalTokens)
	}
}

func TestTrimToTokensNoTrimWhenWithinLimit(t *testing.T) {
	t.Parallel()
	text := "short text for trim"
	maxTokens := EstimateTokens(text)

	got := trimToTokens(text, maxTokens)
	if got != text {
		t.Fatalf("trimToTokens() = %q, want %q", got, text)
	}
}

func TestTrimToTokensTrimWhenOverLimit(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("alpha beta ", 200)
	maxTokens := 20

	got := trimToTokens(text, maxTokens)
	if got == "" {
		t.Fatal("trimToTokens() returned empty string")
	}
	if EstimateTokens(got) > maxTokens {
		t.Fatalf("trimToTokens() tokens = %d, want <= %d", EstimateTokens(got), maxTokens)
	}
	if len(got) >= len(text) {
		t.Fatalf("trimToTokens() did not trim: len(got)=%d len(text)=%d", len(got), len(text))
	}
}
