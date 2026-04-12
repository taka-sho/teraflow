package discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateDocIndexAndLoadDocIndex(t *testing.T) {
	docsDir := filepath.Join(t.TempDir(), "docs")
	mustWriteDoc(t, filepath.Join(docsDir, "requirements", "scope.md"), `---
codd:
  node_id: REQ-001
  title: Scope and Goals
  tags: [scope, acceptance]
---
# Scope and Goals

この文書は要件を定義する。

## Functional Requirements
ユーザーはCLIで実行できる。

## Risks
外部依存が増える。
`)
	mustWriteDoc(t, filepath.Join(docsDir, "design", "api.md"), `# API Design

## Endpoints
GET /health
`)
	mustWriteDoc(t, filepath.Join(docsDir, "adr", "0001.md"), `# ADR 0001

設計上の決定を記録する。
`)

	idx, err := GenerateDocIndex(context.Background(), docsDir, nil)
	if err != nil {
		t.Fatalf("GenerateDocIndex() error = %v", err)
	}
	if idx.Version != "1" {
		t.Fatalf("Version = %q, want 1", idx.Version)
	}
	if len(idx.Docs) != 3 {
		t.Fatalf("Docs len = %d, want 3", len(idx.Docs))
	}

	req := findDoc(t, idx.Docs, "requirements/scope.md")
	if req.Title != "Scope and Goals" {
		t.Fatalf("Title = %q, want Scope and Goals", req.Title)
	}
	if req.CoddNodeID != "REQ-001" {
		t.Fatalf("CoddNodeID = %q, want REQ-001", req.CoddNodeID)
	}
	if req.TotalTokens <= 0 {
		t.Fatalf("TotalTokens = %d, want > 0", req.TotalTokens)
	}
	if len(req.Sections) < 2 {
		t.Fatalf("Sections len = %d, want >= 2", len(req.Sections))
	}
	if !contains(req.Categories, "scope") {
		t.Fatalf("Categories = %v, want scope", req.Categories)
	}
	if !contains(req.Keywords, "acceptance") {
		t.Fatalf("Keywords = %v, want acceptance", req.Keywords)
	}

	data, err := yaml.Marshal(idx)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	indexPath := filepath.Join(t.TempDir(), "doc-index.yaml")
	if err := os.WriteFile(indexPath, data, 0o644); err != nil {
		t.Fatalf("write index file: %v", err)
	}

	loaded, err := LoadDocIndex(indexPath)
	if err != nil {
		t.Fatalf("LoadDocIndex() error = %v", err)
	}
	if len(loaded.Docs) != len(idx.Docs) {
		t.Fatalf("loaded docs len = %d, want %d", len(loaded.Docs), len(idx.Docs))
	}
}

func TestLoadDocIndexDefaultsVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doc-index.yaml")
	if err := os.WriteFile(path, []byte("generated_at: now\ndocs: []\n"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	idx, err := LoadDocIndex(path)
	if err != nil {
		t.Fatalf("LoadDocIndex() error = %v", err)
	}
	if idx.Version != "1" {
		t.Fatalf("Version = %q, want 1", idx.Version)
	}
}

func TestGenerateDocIndexValidationAndContext(t *testing.T) {
	if _, err := GenerateDocIndex(context.Background(), "", nil); err == nil {
		t.Fatal("expected error for empty docs dir")
	}

	docsDir := filepath.Join(t.TempDir(), "docs")
	mustWriteDoc(t, filepath.Join(docsDir, "a.md"), "# A\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := GenerateDocIndex(ctx, docsDir, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestDocIndexHelpers(t *testing.T) {
	meta := parseDocFrontmatter([]string{
		"---",
		"node_id: TOP-1",
		"title: Top Title",
		"tags: [risk, risk, dependency]",
		"---",
		"# Heading",
	})
	if meta.nodeID != "TOP-1" || meta.title != "Top Title" {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	if meta.bodyStart != 6 {
		t.Fatalf("bodyStart = %d, want 6", meta.bodyStart)
	}

	broken := parseDocFrontmatter([]string{"---", "tags: [", "---", "# H"})
	if broken.bodyStart != 4 {
		t.Fatalf("broken bodyStart = %d, want 4", broken.bodyStart)
	}

	if got := normalizeStringList([]any{" alpha ", "", nil, "alpha", "beta"}); len(got) != 2 {
		t.Fatalf("normalizeStringList(any) = %v, want 2 unique entries", got)
	}
	if got := normalizeStringList(" one "); len(got) != 1 || got[0] != "one" {
		t.Fatalf("normalizeStringList(string) = %v, want [one]", got)
	}
	if got := normalizeStringList(42); got != nil {
		t.Fatalf("normalizeStringList(int) = %v, want nil", got)
	}

	if got := firstHeading([]string{"text", " # Title "}, "# "); got != "Title" {
		t.Fatalf("firstHeading() = %q, want Title", got)
	}

	sections := collectSections([]string{"# T", "", "plain body"}, 1)
	if len(sections) != 1 || sections[0].Heading != "Document" {
		t.Fatalf("collectSections() = %+v, want single Document section", sections)
	}

	kw := collectKeywords([]DocSection{{Heading: "Document"}, {Heading: "Risks"}}, []string{"scope", "scope"})
	if !contains(kw, "scope") || !contains(kw, "Risks") {
		t.Fatalf("collectKeywords() = %v", kw)
	}

	cats := inferCategories([]string{"unknown"}, []string{"other"}, "docs/other.md")
	if len(cats) != 1 || cats[0] != "functional" {
		t.Fatalf("inferCategories default = %v, want [functional]", cats)
	}

	summary := buildSummary([]string{"# H", "", "```", "line", "```", strings.Repeat("x", 200)}, 1)
	if len([]rune(summary)) != 4 {
		t.Fatalf("buildSummary() should pick first prose line, got %q", summary)
	}
	if got := buildSummary([]string{"# H", strings.Repeat("x", 200)}, 1); len([]rune(got)) != 140 {
		t.Fatalf("buildSummary truncation len = %d, want 140", len([]rune(got)))
	}

	if estimateTokens("   ") != 0 {
		t.Fatalf("estimateTokens(empty) should be 0")
	}
	if estimateTokens("abc") < 1 {
		t.Fatalf("estimateTokens(short) should be >= 1")
	}

	uniq := uniqueStrings([]string{" a ", "a", "", "b"})
	if len(uniq) != 2 {
		t.Fatalf("uniqueStrings() = %v, want 2", uniq)
	}
}

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

func mustWriteDoc(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func findDoc(t *testing.T, docs []DocEntry, path string) DocEntry {
	t.Helper()
	for _, doc := range docs {
		if doc.Path == path {
			return doc
		}
	}
	t.Fatalf("doc %q not found in %v", path, docs)
	return DocEntry{}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
