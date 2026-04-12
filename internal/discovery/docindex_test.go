package discovery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateDocIndex(t *testing.T) {
	tmp := t.TempDir()
	docsDir := filepath.Join(tmp, "docs")
	if err := os.MkdirAll(filepath.Join(docsDir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	authDoc := "# Auth Guide\n\n## JWT\nUse short-lived tokens.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "auth.md"), []byte(authDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	apiDoc := "# API Guide\n\n## Endpoints\nList endpoint contracts.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "sub", "api.md"), []byte(apiDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "README.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}

	idx, err := GenerateDocIndex(context.Background(), docsDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if idx == nil {
		t.Fatal("index is nil")
	}
	if idx.Version != "1" {
		t.Fatalf("version = %q, want 1", idx.Version)
	}
	if idx.Generator == "" {
		t.Fatal("generator should not be empty")
	}
	if len(idx.Docs) != 2 {
		t.Fatalf("docs = %d, want 2", len(idx.Docs))
	}
	if idx.Docs[0].Path != "auth.md" || idx.Docs[1].Path != "sub/api.md" {
		t.Fatalf("paths not sorted as expected: %+v", []string{idx.Docs[0].Path, idx.Docs[1].Path})
	}
	if idx.Docs[0].Title == "" || idx.Docs[0].TotalTokens <= 0 {
		t.Fatalf("unexpected doc metadata: %+v", idx.Docs[0])
	}
}

func TestLoadDocIndex(t *testing.T) {
	tmp := t.TempDir()
	indexPath := filepath.Join(tmp, "doc-index.yaml")
	content := `generated_at: "2026-04-12T00:00:00Z"
generator: "test"
docs:
  - path: "auth.md"
    title: "Auth Guide"
    summary: "JWT"
    sections:
      - heading: "JWT"
        line_start: 1
        line_end: 3
        token_estimate: 40
    total_tokens: 40
`
	if err := os.WriteFile(indexPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	idx, err := LoadDocIndex(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if idx.Version != "1" {
		t.Fatalf("version = %q, want default 1", idx.Version)
	}
	if len(idx.Docs) != 1 || idx.Docs[0].Path != "auth.md" {
		t.Fatalf("unexpected docs: %+v", idx.Docs)
	}
}

func TestReadDocChunksRespectsBudget(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	docsDir := filepath.Join(tmp, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Doc\n\n## A\nalpha\n\n## B\nbeta\n"
	if err := os.WriteFile(filepath.Join(docsDir, "a.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []DocEntry{{
		Path: "a.md",
		Sections: []DocSection{
			{Heading: "A", LineStart: 3, LineEnd: 4, TokenEstimate: 120},
			{Heading: "B", LineStart: 6, LineEnd: 7, TokenEstimate: 120},
		},
	}}

	chunks, err := ReadDocChunks(entries, 150)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("chunks = %d, want 1", len(chunks))
	}
	if chunks[0].Section != "A" {
		t.Fatalf("section = %s, want A", chunks[0].Section)
	}
}

func TestReadDocChunksGracefulDegradation(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll("docs", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("docs/existing.md", []byte("# Existing\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []DocEntry{
		{
			Path: "missing.md",
			Sections: []DocSection{
				{Heading: "Missing", LineStart: 1, LineEnd: 2, TokenEstimate: 30},
			},
		},
		{
			Path: "existing.md",
			Sections: []DocSection{
				{Heading: "Existing", LineStart: 1, LineEnd: 3, TokenEstimate: 30},
			},
		},
	}

	chunks, err := ReadDocChunks(entries, 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("chunks = %d, want 1", len(chunks))
	}
	if chunks[0].Path != "existing.md" {
		t.Fatalf("path = %s, want existing.md", chunks[0].Path)
	}
}

func TestFormatDocContext(t *testing.T) {
	formatted := FormatDocContext([]DocChunk{{
		Path: "docs/auth.md", Section: "JWT", Content: "Use short expiry", Tokens: 20,
	}})
	if !strings.Contains(formatted, "## 既存ドキュメント参照") {
		t.Fatalf("missing heading in formatted context: %s", formatted)
	}
	if !strings.Contains(formatted, "docs/auth.md / JWT") {
		t.Fatalf("missing chunk header in formatted context: %s", formatted)
	}
}
