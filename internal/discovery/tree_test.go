package discovery

import (
	"context"
	"errors"
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

func TestNewDefaultTemplateCounts(t *testing.T) {
	tmpl := NewDefaultTemplate()
	if got := len(tmpl.Items); got != 31 {
		t.Fatalf("items = %d, want 31", got)
	}

	counts := map[string]int{}
	for _, item := range tmpl.Items {
		counts[item.Category]++
	}
	if counts[TemplateCategoryRequired] != 14 {
		t.Fatalf("required = %d, want 14", counts[TemplateCategoryRequired])
	}
	if counts[TemplateCategoryRecommended] != 10 {
		t.Fatalf("recommended = %d, want 10", counts[TemplateCategoryRecommended])
	}
	if counts[TemplateCategoryOptional] != 7 {
		t.Fatalf("optional = %d, want 7", counts[TemplateCategoryOptional])
	}
}

func TestBuildInitialTreeFromTemplate(t *testing.T) {
	tmpl := NewDefaultTemplate()
	tree := BuildInitialTreeFromTemplate(tmpl)
	if len(tree) != len(tmpl.Items) {
		t.Fatalf("tree len = %d, want %d", len(tree), len(tmpl.Items))
	}

	for i, node := range tree {
		if node.ID != tmpl.Items[i].ID {
			t.Fatalf("node[%d].id = %q, want %q", i, node.ID, tmpl.Items[i].ID)
		}
		if node.Status != StatusPending {
			t.Fatalf("status = %s, want pending", node.Status)
		}
		if strings.TrimSpace(node.Question) == "" {
			t.Fatalf("question should not be empty for id=%s", node.ID)
		}
	}
}

func TestBuildInitialTreeWithLLM(t *testing.T) {
	tmpl := NewDefaultTemplate()
	llm := stubGenerator{output: `[
  {"id":"project_overview","question":"概要は？","category":"required"},
  {"id":"nfr.security","question":"セキュリティ要件は？","category":"recommended"}
]`}

	tree, err := BuildInitialTreeWithLLM(context.Background(), llm, "discussion", tmpl, TreeBuildOptions{})
	if err != nil {
		t.Fatalf("build with llm: %v", err)
	}
	if len(tree) != 2 {
		t.Fatalf("tree len = %d, want 2", len(tree))
	}
	if tree[0].ID != "project_overview" {
		t.Fatalf("unexpected first id: %s", tree[0].ID)
	}
}

func TestBuildInitialTreeWithLLMFallback(t *testing.T) {
	tmpl := NewDefaultTemplate()
	tree, err := BuildInitialTreeWithLLM(context.Background(), stubGenerator{err: errors.New("boom")}, "discussion", tmpl, TreeBuildOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tree) == 0 {
		t.Fatal("expected fallback tree")
	}
}

func TestBuildInitialTreeWithLLMIncludesDocContext(t *testing.T) {
	tmpl := NewDefaultTemplate()
	var prompt string
	llm := stubGenerator{
		output: `[{"id":"q1","question":"Q1","category":"required"}]`,
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
