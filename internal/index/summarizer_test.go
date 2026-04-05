package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type mockProvider struct {
	calls  int
	output string
}

func (m *mockProvider) Complete(_ context.Context, _, userPrompt string, _ int) (string, int, error) {
	m.calls++
	if m.output != "" {
		return m.output, len(userPrompt), nil
	}
	return fmt.Sprintf("summary: %s", userPrompt), len(userPrompt), nil
}

func (m *mockProvider) Name() string { return "mock" }

func TestIsCacheValid(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	s := NewSummarizer(&mockProvider{}, tmp)

	entry := Entry{NodeID: "req:slcp-jcf", ContentHash: "sha256:abc"}
	if s.IsCacheValid(entry) {
		t.Fatal("cache should be invalid when file does not exist")
	}

	path := filepath.Join(tmp, "req--slcp-jcf.txt")
	if err := os.WriteFile(path, []byte("# hash:sha256:abc\nhello\n"), 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	if !s.IsCacheValid(entry) {
		t.Fatal("expected cache to be valid")
	}

	if err := os.WriteFile(path, []byte("# hash:sha256:def\nhello\n"), 0o644); err != nil {
		t.Fatalf("write cache mismatch: %v", err)
	}
	if s.IsCacheValid(entry) {
		t.Fatal("cache should be invalid when hash differs")
	}
}

func TestSummarizeCacheHitAndMiss(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	mp := &mockProvider{output: "cached summary"}
	s := NewSummarizer(mp, tmp)

	entry := Entry{NodeID: "req:node/1", ContentHash: "sha256:xyz"}

	summary, err := s.Summarize(context.Background(), entry, "first content")
	if err != nil {
		t.Fatalf("summarize miss: %v", err)
	}
	if summary != "cached summary" {
		t.Fatalf("unexpected summary: %q", summary)
	}
	if mp.calls != 1 {
		t.Fatalf("expected 1 provider call, got %d", mp.calls)
	}

	summary, err = s.Summarize(context.Background(), entry, "changed content is ignored on cache hit")
	if err != nil {
		t.Fatalf("summarize hit: %v", err)
	}
	if summary != "cached summary" {
		t.Fatalf("unexpected cached summary: %q", summary)
	}
	if mp.calls != 1 {
		t.Fatalf("cache hit should not call provider, got %d calls", mp.calls)
	}
}

func TestUpdateAllCounts(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	sourceDir := filepath.Join(tmp, "docs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}

	file1 := filepath.Join(sourceDir, "a.md")
	file2 := filepath.Join(sourceDir, "b.md")
	if err := os.WriteFile(file1, []byte("A content"), 0o644); err != nil {
		t.Fatalf("write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("B content"), 0o644); err != nil {
		t.Fatalf("write file2: %v", err)
	}

	idx := &Index{Entries: []Entry{
		{NodeID: "req:a", FilePath: file1, ContentHash: "sha256:a"},
		{NodeID: "req:b", FilePath: file2, ContentHash: "sha256:b"},
	}}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "req--b.txt"), []byte("# hash:sha256:b\nexisting\n"), 0o644); err != nil {
		t.Fatalf("write cache b: %v", err)
	}

	mp := &mockProvider{output: "new summary"}
	s := NewSummarizer(mp, cacheDir)

	updated, skipped, err := s.UpdateAll(context.Background(), idx)
	if err != nil {
		t.Fatalf("update all: %v", err)
	}
	if updated != 1 || skipped != 1 {
		t.Fatalf("expected updated=1 skipped=1, got updated=%d skipped=%d", updated, skipped)
	}
	if mp.calls != 1 {
		t.Fatalf("expected provider call once, got %d", mp.calls)
	}
}
