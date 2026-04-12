package discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stubGenerator struct {
	output string
	err    error
	got    *string
}

func (s stubGenerator) Generate(_ context.Context, prompt string) (string, error) {
	if s.got != nil {
		*s.got = prompt
	}
	if s.err != nil {
		return "", s.err
	}
	return s.output, nil
}

func TestDefaultTemplateAndBuildInitialTree(t *testing.T) {
	tmpl := DefaultTemplate()
	tree := BuildInitialTree(tmpl, []string{"scope", "functional"})
	if len(tree) == 0 {
		t.Fatal("expected non-empty tree")
	}
	for _, node := range tree {
		if node.Status != StatusPending {
			t.Fatalf("status = %s, want pending", node.Status)
		}
		if strings.TrimSpace(node.ID) == "" || strings.TrimSpace(node.Question) == "" {
			t.Fatalf("invalid node: %+v", node)
		}
	}
}

func TestLoadTemplate(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "template.yml")
	data := `version: "1"
categories:
  scope:
    - id: scope.platform
      question: 対象プラットフォームは？
      category: scope
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	tmpl, err := LoadTemplate(path)
	if err != nil {
		t.Fatalf("load template: %v", err)
	}
	if len(tmpl.Categories["scope"]) != 1 {
		t.Fatalf("scope count = %d, want 1", len(tmpl.Categories["scope"]))
	}
}

func TestBuildInitialTreeWithLLM(t *testing.T) {
	tmpl := DefaultTemplate()
	llm := stubGenerator{output: `[
  {"id":"scope.target_users","question":"対象ユーザーは？","category":"scope","recommendation":"管理者"},
  {"id":"nfr.performance","question":"性能要件は？","category":"non_functional"}
]`}

	tree, err := BuildInitialTreeWithLLM(context.Background(), llm, "discussion", tmpl, TreeBuildOptions{})
	if err != nil {
		t.Fatalf("build with llm: %v", err)
	}
	if len(tree) != 2 {
		t.Fatalf("tree len = %d, want 2", len(tree))
	}
	if tree[0].ID != "scope.target_users" {
		t.Fatalf("unexpected first id: %s", tree[0].ID)
	}
}

func TestBuildInitialTreeWithLLMFallback(t *testing.T) {
	tmpl := DefaultTemplate()
	tree, err := BuildInitialTreeWithLLM(context.Background(), stubGenerator{err: errors.New("boom")}, "discussion", tmpl, TreeBuildOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tree) == 0 {
		t.Fatal("expected fallback tree")
	}
}

func TestBuildInitialTreeWithLLMIncludesDocContext(t *testing.T) {
	tmpl := DefaultTemplate()
	var prompt string
	llm := stubGenerator{
		output: `[{"id":"q1","question":"Q1","category":"scope"}]`,
		got:    &prompt,
	}

	_, err := BuildInitialTreeWithLLM(
		context.Background(),
		llm,
		"discussion body",
		tmpl,
		TreeBuildOptions{DocContext: "## auth.md\nJWT expires in 15m"},
	)
	if err != nil {
		t.Fatalf("build with llm: %v", err)
	}
	if !strings.Contains(prompt, "【既存文書コンテキスト】") {
		t.Fatalf("prompt should include doc context section: %s", prompt)
	}
	if !strings.Contains(prompt, "JWT expires in 15m") {
		t.Fatalf("prompt should include doc context payload: %s", prompt)
	}
	if !strings.Contains(prompt, "【投稿内容】") {
		t.Fatalf("prompt should include discussion section: %s", prompt)
	}
}
