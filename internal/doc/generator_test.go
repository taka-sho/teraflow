package doc

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type mockProvider struct {
	output string
	err    error
}

func (m *mockProvider) Complete(_ context.Context, _, _ string, _ int) (string, int, error) {
	if m.err != nil {
		return "", 0, m.err
	}
	return m.output, 123, nil
}

func (m *mockProvider) Name() string {
	return "mock"
}

func TestFetchDiscussion(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{}, root, false)

	orig := fetchCommandContext
	t.Cleanup(func() { fetchCommandContext = orig })

	fetchCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		t.Helper()
		if name == "git" {
			return exec.CommandContext(ctx, "sh", "-c", "echo git@github.com:acme/rocket.git")
		}
		if name == "gh" {
			return exec.CommandContext(ctx, "sh", "-c", `cat <<'JSON'
{"data":{"repository":{"discussion":{"title":"Need docs","body":"Please generate","createdAt":"2026-04-06T00:00:00Z","labels":{"nodes":[{"name":"requirements"}]},"comments":{"nodes":[{"author":{"login":"alice"},"body":"Looks good","createdAt":"2026-04-06T01:00:00Z","isAnswer":true,"replies":{"nodes":[{"author":{"login":"bob"},"body":"I agree","createdAt":"2026-04-06T01:30:00Z"}]}}]}}}}}
JSON`)
		}
		return exec.CommandContext(ctx, name, args...)
	}

	disc, err := g.fetchDiscussion(context.Background(), "42")
	if err != nil {
		t.Fatalf("fetchDiscussion() error = %v", err)
	}
	if disc.Number != 42 {
		t.Fatalf("number = %d, want 42", disc.Number)
	}
	if disc.Title != "Need docs" {
		t.Fatalf("title = %q", disc.Title)
	}
	if len(disc.Comments) != 1 || disc.Comments[0].Author != "alice" {
		t.Fatalf("unexpected comments: %+v", disc.Comments)
	}
	if len(disc.Comments[0].Replies) != 1 || disc.Comments[0].Replies[0].Author != "bob" {
		t.Fatalf("unexpected replies: %+v", disc.Comments[0].Replies)
	}
}

func TestGenerateDryRun(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{output: `{"node_id":"req:auth","title":"Auth","category":"requirements","summary":"s","sections":["a"],"depends_on":["x"],"status":"review"}`}, root, true)

	orig := fetchCommandContext
	t.Cleanup(func() { fetchCommandContext = orig })
	fetchCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "git" {
			return exec.CommandContext(ctx, "sh", "-c", "echo https://github.com/acme/rocket.git")
		}
		return exec.CommandContext(ctx, "sh", "-c", `cat <<'JSON'
{"data":{"repository":{"discussion":{"title":"T","body":"B","createdAt":"2026-04-06T00:00:00Z","labels":{"nodes":[]},"comments":{"nodes":[]}}}}}
JSON`)
	}

	res, err := g.Generate(context.Background(), GenerateRequest{DiscussionID: "1"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if res.Document == nil {
		t.Fatal("document is nil")
	}
	if res.FilePath != "" {
		t.Fatalf("FilePath = %q, want empty on dry-run", res.FilePath)
	}
	if res.IndexUpdate {
		t.Fatal("IndexUpdate = true, want false on dry-run")
	}
}

func TestBuildDocument(t *testing.T) {
	g := NewGenerator(&mockProvider{}, t.TempDir(), true)
	doc := g.buildDocument(&DiscussionData{Number: 7, Title: "Fallback", CreatedAt: "2026-04-06T00:00:00Z"}, map[string]interface{}{
		"node_id":    "design:api",
		"title":      "API design",
		"depends_on": []interface{}{"req:auth", "req:user"},
		"status":     "confirmed",
		"summary":    "Summary text",
		"sections": []interface{}{
			map[string]interface{}{"heading": "背景・課題", "body": "現状の整理"},
			map[string]interface{}{"heading": "機能要件", "body": "要件の詳細"},
		},
	})

	if doc.NodeID != "design:api" {
		t.Fatalf("NodeID = %q", doc.NodeID)
	}
	if doc.Source != "discussion:#7" {
		t.Fatalf("Source = %q", doc.Source)
	}
	if len(doc.DependsOn) != 2 {
		t.Fatalf("DependsOn = %+v", doc.DependsOn)
	}
	if !strings.Contains(doc.Body, "Summary text") {
		t.Fatalf("Body missing summary: %s", doc.Body)
	}
	if len(doc.Sections) != 2 || doc.Sections[0].Heading != "背景・課題" {
		t.Fatalf("Sections unexpected: %+v", doc.Sections)
	}
}

func TestParseSections(t *testing.T) {
	tests := []struct {
		name string
		raw  interface{}
		want int
	}{
		{
			name: "object array",
			raw: []interface{}{
				map[string]interface{}{"heading": "h1", "body": "b1"},
				map[string]interface{}{"heading": "h2", "body": "b2"},
			},
			want: 2,
		},
		{
			name: "string array fallback",
			raw:  []interface{}{"h1", "h2"},
			want: 2,
		},
		{
			name: "mixed",
			raw: []interface{}{
				map[string]interface{}{"heading": "h1", "body": "b1"},
				"h2",
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSections(tt.raw)
			if len(got) != tt.want {
				t.Fatalf("len(parseSections()) = %d, want %d", len(got), tt.want)
			}
		})
	}
}

func TestBuildBodyWithSections(t *testing.T) {
	body := buildBody("summary", []DocSection{
		{Heading: "背景・課題", Body: "課題本文"},
		{Heading: "機能要件", Body: "要件本文"},
	})
	if !strings.Contains(body, "# 概要") {
		t.Fatalf("missing summary heading: %s", body)
	}
	if !strings.Contains(body, "## 背景・課題") || !strings.Contains(body, "課題本文") {
		t.Fatalf("missing section content: %s", body)
	}
}

func TestBuildStructurizePromptIncludesReplies(t *testing.T) {
	g := NewGenerator(&mockProvider{}, t.TempDir(), true)
	prompt := g.buildStructurizePrompt(&DiscussionData{
		Title:     "Discussion title",
		Body:      "body text",
		CreatedAt: "2026-04-08T00:00:00Z",
		Labels:    []string{"requirements"},
		Comments: []Comment{
			{
				Author:    "alice",
				Body:      "comment body",
				CreatedAt: "2026-04-08T00:01:00Z",
				Replies: []Reply{
					{Author: "bob", Body: "reply body", CreatedAt: "2026-04-08T00:02:00Z"},
				},
			},
		},
	})
	if !strings.Contains(prompt, "reply body") {
		t.Fatalf("prompt must include replies: %s", prompt)
	}
}

func TestWriteFile(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{}, root, false)

	path, err := g.writeFile(&CoDDDocument{
		NodeID:    "req-auth",
		Title:     "Auth",
		DependsOn: []string{"base"},
		Status:    "draft",
		Source:    "discussion:#1",
		CreatedAt: "2026-04-06T00:00:00Z",
		UpdatedAt: "2026-04-06T00:00:00Z",
		Body:      "# Body\n\ncontent",
	}, filepath.Join(root, "docs", "requirements"))
	if err != nil {
		t.Fatalf("writeFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "codd:") || !strings.Contains(got, "node_id: req-auth") {
		t.Fatalf("frontmatter missing: %s", got)
	}
	if !strings.Contains(got, "# Body") {
		t.Fatalf("body missing: %s", got)
	}
}

func TestGeneratorHelpersAndErrors(t *testing.T) {
	g := NewGenerator(&mockProvider{}, t.TempDir(), false)
	if g.maxTokens() != 8192 {
		t.Fatalf("default max tokens mismatch: %d", g.maxTokens())
	}
	g.maxTokensCfg = 512
	if g.maxTokens() != 512 {
		t.Fatalf("configured max tokens mismatch: %d", g.maxTokens())
	}

	gNoRoot := NewGenerator(&mockProvider{}, "", false)
	if _, err := gNoRoot.Generate(context.Background(), GenerateRequest{}); err == nil || !strings.Contains(err.Error(), "projectRoot is required") {
		t.Fatalf("expected projectRoot required error, got: %v", err)
	}

	if out := outputDir("", map[string]interface{}{"category": "requirements"}); out != filepath.Join("docs", "requirements") {
		t.Fatalf("unexpected outputDir category path: %q", out)
	}
	if out := outputDir("custom", nil); out != "custom" {
		t.Fatalf("unexpected outputDir override: %q", out)
	}

	if got := asStringSlice([]interface{}{"a", " ", 3, "b"}); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("unexpected asStringSlice: %+v", got)
	}
	if got := stripCodeFence("```json\n{\"x\":1}\n```"); got != "{\"x\":1}" {
		t.Fatalf("stripCodeFence json mismatch: %q", got)
	}
}

func TestParseGitHubRepositoryTable(t *testing.T) {
	cases := []struct {
		remote string
		owner  string
		repo   string
		ok     bool
	}{
		{"https://github.com/acme/rocket.git", "acme", "rocket", true},
		{"git@github.com:acme/rocket.git", "acme", "rocket", true},
		{"ssh://git@github.com/acme/rocket.git", "acme", "rocket", true},
		{"invalid", "", "", false},
	}
	for _, tc := range cases {
		owner, repo, err := parseGitHubRepository(tc.remote)
		if tc.ok {
			if err != nil || owner != tc.owner || repo != tc.repo {
				t.Fatalf("parseGitHubRepository(%q) owner=%q repo=%q err=%v", tc.remote, owner, repo, err)
			}
		} else if err == nil {
			t.Fatalf("expected parse failure for %q", tc.remote)
		}
	}
}

func TestResolveRepositoryErrorAndStructurizeErrors(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{}, root, false)

	orig := fetchCommandContext
	t.Cleanup(func() { fetchCommandContext = orig })
	fetchCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "echo fail 1>&2; exit 1")
	}
	if _, _, err := g.resolveRepository(); err == nil || !strings.Contains(err.Error(), "resolve repository failed") {
		t.Fatalf("expected resolve repository failed, got: %v", err)
	}

	if _, err := g.structurize(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "discussion is nil") {
		t.Fatalf("expected nil discussion error, got: %v", err)
	}

	g.provider = &mockProvider{err: errors.New("llm fail")}
	if _, err := g.structurize(context.Background(), &DiscussionData{Title: "x"}); err == nil || !strings.Contains(err.Error(), "structurize failed") {
		t.Fatalf("expected structurize provider error, got: %v", err)
	}

	g.provider = &mockProvider{output: "not-json"}
	if _, err := g.structurize(context.Background(), &DiscussionData{Title: "x"}); err == nil || !strings.Contains(err.Error(), "parse structured json") {
		t.Fatalf("expected parse structured json error, got: %v", err)
	}
}

func TestWriteFileErrors(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{}, root, false)

	if _, err := g.writeFile(nil, ""); err == nil {
		t.Fatal("expected nil document error")
	}

	blocked := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocked file: %v", err)
	}
	_, err := g.writeFile(&CoDDDocument{NodeID: "x", Body: "b"}, blocked)
	if err == nil || !strings.Contains(err.Error(), "create output directory") {
		t.Fatalf("expected mkdir output error, got: %v", err)
	}
}
