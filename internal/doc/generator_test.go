package doc

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
{"data":{"repository":{"discussion":{"title":"Need docs","body":"Please generate","createdAt":"2026-04-06T00:00:00Z","labels":{"nodes":[{"name":"requirements"}]},"comments":{"nodes":[{"author":{"login":"alice"},"body":"Looks good","createdAt":"2026-04-06T01:00:00Z","isAnswer":true}]}}}}}
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
		"sections":   []interface{}{"sec1", "sec2"},
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

func TestPostDiscussionComment(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{}, root, false)

	orig := postCommentCommandContext
	t.Cleanup(func() { postCommentCommandContext = orig })

	postCommentCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		t.Helper()
		if name != "gh" {
			t.Fatalf("unexpected command: %s", name)
		}
		cmdline := strings.Join(args, " ")
		if !strings.Contains(cmdline, "addDiscussionComment") {
			t.Fatalf("mutation not found in args: %v", args)
		}
		if !strings.Contains(cmdline, "id=D_kwDOEXAMPLE") {
			t.Fatalf("discussion id not passed: %v", args)
		}
		if !strings.Contains(cmdline, "PR**: #123") {
			t.Fatalf("PR number missing in comment body: %v", args)
		}
		return exec.CommandContext(ctx, "sh", "-c", `echo '{"data":{"addDiscussionComment":{"comment":{"id":"X"}}}}'`)
	}

	err := g.PostDiscussionComment(context.Background(), "D_kwDOEXAMPLE", &GenerateResult{
		Document: &CoDDDocument{NodeID: "req:auth"},
		FilePath: "docs/requirements/req-auth.md",
		PRBranch: "123",
	})
	if err != nil {
		t.Fatalf("PostDiscussionComment() error = %v", err)
	}
}

func TestPostDiscussionCommentCommandFailed(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{}, root, false)

	orig := postCommentCommandContext
	t.Cleanup(func() { postCommentCommandContext = orig })
	postCommentCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "echo failed >&2; exit 1")
	}

	err := g.PostDiscussionComment(context.Background(), "D_kwDOEXAMPLE", &GenerateResult{
		Document: &CoDDDocument{NodeID: "req:auth"},
		FilePath: "docs/requirements/req-auth.md",
		PRBranch: "123",
	})
	if err == nil || !strings.Contains(err.Error(), "post discussion comment failed: failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
