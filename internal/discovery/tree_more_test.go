package discovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTemplateEmptyPath(t *testing.T) {
	tmpl, err := LoadRequirementTemplate("")
	if err != nil {
		t.Fatalf("empty path should return default: %v", err)
	}
	if len(tmpl.Items) != 31 {
		t.Fatalf("default template should have 31 items, got %d", len(tmpl.Items))
	}
}

func TestLoadTemplateInvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.yml")
	if err := os.WriteFile(path, []byte(":::invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadRequirementTemplate(path)
	if err == nil {
		t.Fatal("should error on invalid YAML")
	}
}

func TestLoadTemplateNoItems(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "empty.yml")
	if err := os.WriteFile(path, []byte("version: '1'\nitems: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadRequirementTemplate(path)
	if err == nil {
		t.Fatal("should error on empty items")
	}
}

func TestLoadTemplateNoFile(t *testing.T) {
	_, err := LoadRequirementTemplate("/nonexistent/file.yml")
	if err == nil {
		t.Fatal("should error on missing file")
	}
}

func TestLoadTemplateNoVersion(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "no-ver.yml")
	data := `items:
  - id: custom_id
    name: カスタム
    category: required
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpl, err := LoadRequirementTemplate(path)
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Version != "1" {
		t.Fatalf("version = %q, want 1", tmpl.Version)
	}
}

func TestLoadTemplatePrefersDotTeraflowYml(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(".teraflow", 0o755); err != nil {
		t.Fatal(err)
	}

	yml := `items:
  - id: from_yml
    name: from yml
    category: required
`
	yaml := `items:
  - id: from_yaml
    name: from yaml
    category: required
`
	if err := os.WriteFile(".teraflow/requirement-template.yml", []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".teraflow/requirement-template.yaml", []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpl, err := LoadRequirementTemplate("")
	if err != nil {
		t.Fatal(err)
	}
	if len(tmpl.Items) != 1 || tmpl.Items[0].ID != "from_yml" {
		t.Fatalf("expected .yml to win, got %+v", tmpl.Items)
	}
}

func TestLoadTemplateLegacySections(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "template.yml")
	data := `version: "1"
template:
  sections:
    - id: overview
      title: 概要
      priority: required
      fields:
        - id: overview.purpose
          label: 目的
          priority: required
          hint: 目的を教えてください
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpl, err := LoadRequirementTemplate(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(tmpl.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(tmpl.Items))
	}
	if tmpl.Items[0].ID != "overview.purpose" {
		t.Fatalf("id = %q", tmpl.Items[0].ID)
	}
}

func TestBuildInitialTreeWithLLMNilGenerator(t *testing.T) {
	tmpl := NewDefaultTemplate()
	tree, err := BuildInitialTreeWithLLM(context.Background(), nil, "body", tmpl, TreeBuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) == 0 {
		t.Fatal("nil generator should use fallback")
	}
}

func TestBuildInitialTreeWithLLMInvalidJSON(t *testing.T) {
	tmpl := NewDefaultTemplate()
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
	tmpl := NewDefaultTemplate()
	llm := stubGenerator{output: `[{"id":"","question":""}]`}
	tree, err := BuildInitialTreeWithLLM(context.Background(), llm, "body", tmpl, TreeBuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) == 0 {
		t.Fatal("should fallback when LLM returns empty valid items")
	}
}

func TestBuildInitialTreeWithLLMDefaultCategory(t *testing.T) {
	tmpl := NewDefaultTemplate()
	llm := stubGenerator{output: `[{"id":"q1","question":"What?"}]`}
	tree, err := BuildInitialTreeWithLLM(context.Background(), llm, "body", tmpl, TreeBuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) != 1 {
		t.Fatalf("len = %d, want 1", len(tree))
	}
	if tree[0].Category != TemplateCategoryRecommended {
		t.Fatalf("default category = %q, want recommended", tree[0].Category)
	}
}
