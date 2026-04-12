package discovery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectRelevantDocsPicksTopMatches(t *testing.T) {
	idx := DocIndex{Docs: []DocEntry{
		{Path: "docs/auth.md", Title: "Authentication", Summary: "JWT login", Keywords: []string{"jwt", "oauth"}},
		{Path: "docs/wave.md", Title: "Wave", Summary: "planning"},
	}}
	selected, err := SelectRelevantDocs(context.Background(), idx, "JWT token expiry for login")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) == 0 {
		t.Fatal("expected at least one selected document")
	}
	if selected[0].Path != "docs/auth.md" {
		t.Fatalf("top selected path = %s, want docs/auth.md", selected[0].Path)
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
