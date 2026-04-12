package discovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTemplateEmptyPath(t *testing.T) {
	tmpl, err := LoadTemplate("")
	if err != nil {
		t.Fatalf("empty path should return default: %v", err)
	}
	if len(tmpl.Categories) == 0 {
		t.Fatal("default template should have categories")
	}
}

func TestLoadTemplateInvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.yml")
	if err := os.WriteFile(path, []byte(":::invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadTemplate(path)
	if err == nil {
		t.Fatal("should error on invalid YAML")
	}
}

func TestLoadTemplateEmptyCategories(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "empty.yml")
	if err := os.WriteFile(path, []byte("version: '1'\ncategories: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadTemplate(path)
	if err == nil {
		t.Fatal("should error on empty categories")
	}
}

func TestLoadTemplateNoFile(t *testing.T) {
	_, err := LoadTemplate("/nonexistent/file.yml")
	if err == nil {
		t.Fatal("should error on missing file")
	}
}

func TestLoadTemplateNoVersion(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "no-ver.yml")
	data := `categories:
  scope:
    - id: scope.q1
      question: Q1
      category: scope
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpl, err := LoadTemplate(path)
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Version != "1" {
		t.Fatalf("version = %q, want 1", tmpl.Version)
	}
}

func TestBuildInitialTreeAllCategories(t *testing.T) {
	tmpl := DefaultTemplate()
	tree := BuildInitialTree(tmpl, nil)
	if len(tree) == 0 {
		t.Fatal("expected non-empty tree with all categories")
	}
	// Check that all nodes have pending status
	for _, node := range tree {
		if node.Status != StatusPending {
			t.Fatalf("node %s has status %s, want pending", node.ID, node.Status)
		}
	}
}

func TestBuildInitialTreeWithLLMNilGenerator(t *testing.T) {
	tmpl := DefaultTemplate()
	tree, err := BuildInitialTreeWithLLM(context.Background(), nil, "body", tmpl, TreeBuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) == 0 {
		t.Fatal("nil generator should use fallback")
	}
}

func TestBuildInitialTreeWithLLMInvalidJSON(t *testing.T) {
	tmpl := DefaultTemplate()
	llm := stubGenerator{output: "not json"}
	tree, err := BuildInitialTreeWithLLM(context.Background(), llm, "body", tmpl, TreeBuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) == 0 {
		t.Fatal("invalid JSON should use fallback")
	}
}

func TestBuildInitialTreeWithLLMEmptyResult(t *testing.T) {
	tmpl := DefaultTemplate()
	llm := stubGenerator{output: `[{"id":"","question":""}]`}
	tree, err := BuildInitialTreeWithLLM(context.Background(), llm, "body", tmpl, TreeBuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// All items filtered out, should fallback
	if len(tree) == 0 {
		t.Fatal("should fallback when LLM returns empty valid items")
	}
}

func TestBuildInitialTreeWithLLMDefaultCategory(t *testing.T) {
	tmpl := DefaultTemplate()
	llm := stubGenerator{output: `[{"id":"q1","question":"What?"}]`}
	tree, err := BuildInitialTreeWithLLM(context.Background(), llm, "body", tmpl, TreeBuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) != 1 {
		t.Fatalf("len = %d, want 1", len(tree))
	}
	if tree[0].Category != "functional" {
		t.Fatalf("default category = %q, want functional", tree[0].Category)
	}
}
