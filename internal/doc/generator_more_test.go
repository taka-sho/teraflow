package doc

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestOutputDir(t *testing.T) {
	tests := []struct {
		name       string
		reqOutput  string
		structured map[string]interface{}
		want       string
	}{
		{
			name:       "request output provided",
			reqOutput:  "custom/output",
			structured: map[string]interface{}{"category": "requirements"},
			want:       "custom/output",
		},
		{
			name:       "category from structured",
			reqOutput:  "",
			structured: map[string]interface{}{"category": "requirements"},
			want:       "docs/requirements",
		},
		{
			name:       "no output or category",
			reqOutput:  "",
			structured: map[string]interface{}{},
			want:       "docs",
		},
		{
			name:       "empty category",
			reqOutput:  "",
			structured: map[string]interface{}{"category": ""},
			want:       "docs",
		},
		{
			name:       "nil structured",
			reqOutput:  "",
			structured: nil,
			want:       "docs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := outputDir(tt.reqOutput, tt.structured)
			if got != tt.want {
				t.Fatalf("outputDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripCodeFence(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no fence", `{"key":"val"}`, `{"key":"val"}`},
		{"json fence", "```json\n{\"key\":\"val\"}\n```", `{"key":"val"}`},
		{"plain fence", "```\n{\"key\":\"val\"}\n```", `{"key":"val"}`},
		{"no trailing fence", "```json\n{\"key\":\"val\"}", `{"key":"val"}`},
		{"empty string", "", ""},
		{"just text", "hello", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripCodeFence(tt.input)
			if got != tt.want {
				t.Fatalf("stripCodeFence() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMaxTokens(t *testing.T) {
	g := NewGenerator(&mockProvider{}, "/tmp", true)
	if got := g.maxTokens(); got != 8192 {
		t.Fatalf("default maxTokens = %d, want 8192", got)
	}
	g.maxTokensCfg = 4096
	if got := g.maxTokens(); got != 4096 {
		t.Fatalf("custom maxTokens = %d, want 4096", got)
	}
}

func TestAsString(t *testing.T) {
	if got := asString("hello"); got != "hello" {
		t.Fatalf("asString(string) = %q", got)
	}
	if got := asString(123); got != "" {
		t.Fatalf("asString(int) = %q, want empty", got)
	}
	if got := asString(nil); got != "" {
		t.Fatalf("asString(nil) = %q, want empty", got)
	}
}

func TestAsStringSlice(t *testing.T) {
	got := asStringSlice([]interface{}{"a", "b", "", "c"})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got := asStringSlice("not a slice"); got != nil {
		t.Fatalf("non-slice should return nil, got %v", got)
	}
	if got := asStringSlice(nil); got != nil {
		t.Fatalf("nil should return nil, got %v", got)
	}
	// Test non-string items
	got = asStringSlice([]interface{}{123, "x"})
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (non-string filtered)", len(got))
	}
}

func TestBuildBodyEmptySections(t *testing.T) {
	body := buildBody("", nil)
	if !strings.Contains(body, "（構造化に失敗しました") {
		t.Fatalf("empty body should show failure message: %s", body)
	}
}

func TestBuildBodyNoSummary(t *testing.T) {
	body := buildBody("", []DocSection{
		{Heading: "Heading1", Body: "content"},
	})
	if strings.Contains(body, "# 概要") {
		t.Fatalf("should not have summary heading: %s", body)
	}
	if !strings.Contains(body, "## Heading1") {
		t.Fatalf("missing heading: %s", body)
	}
}

func TestBuildBodySkipsEmptyHeadingAndBody(t *testing.T) {
	body := buildBody("s", []DocSection{
		{Heading: "", Body: ""},
		{Heading: "H", Body: "B"},
	})
	// Empty heading+body sections should be skipped
	if strings.Contains(body, "## 詳細") {
		t.Fatalf("empty section should be skipped: %s", body)
	}
}

func TestBuildBodyDefaultHeading(t *testing.T) {
	body := buildBody("s", []DocSection{
		{Heading: "", Body: "some content"},
	})
	if !strings.Contains(body, "## 詳細") {
		t.Fatalf("empty heading should default to 詳細: %s", body)
	}
}

func TestBuildDocumentFallbacks(t *testing.T) {
	g := NewGenerator(&mockProvider{}, "/tmp", true)
	doc := g.buildDocument(
		&DiscussionData{Number: 5, Title: "Fallback Title", CreatedAt: ""},
		map[string]interface{}{},
	)
	if !strings.Contains(doc.NodeID, "discussion:5") {
		t.Fatalf("NodeID = %q, want fallback", doc.NodeID)
	}
	if doc.Title != "Fallback Title" {
		t.Fatalf("Title = %q, want Fallback Title", doc.Title)
	}
	if doc.Status != "draft" {
		t.Fatalf("Status = %q, want draft", doc.Status)
	}
}

func TestParseSectionsNil(t *testing.T) {
	got := parseSections(nil)
	if got != nil {
		t.Fatalf("parseSections(nil) = %v, want nil", got)
	}
}

func TestParseSectionsWrongType(t *testing.T) {
	got := parseSections("not a slice")
	if got != nil {
		t.Fatalf("parseSections(string) = %v, want nil", got)
	}
}

func TestParseSectionsEmptyStringFiltered(t *testing.T) {
	got := parseSections([]interface{}{"", "  ", "valid"})
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
}

func TestParseGitHubRepository(t *testing.T) {
	tests := []struct {
		name    string
		remote  string
		owner   string
		repo    string
		wantErr bool
	}{
		{"https", "https://github.com/acme/rocket.git", "acme", "rocket", false},
		{"ssh", "git@github.com:acme/rocket.git", "acme", "rocket", false},
		{"ssh with scheme", "ssh://git@github.com/acme/rocket.git", "acme", "rocket", false},
		{"no .git suffix", "https://github.com/acme/rocket", "acme", "rocket", false},
		{"empty", "", "", "", true},
		{"invalid", "notaurl", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseGitHubRepository(tt.remote)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if owner != tt.owner || repo != tt.repo {
					t.Fatalf("got %s/%s, want %s/%s", owner, repo, tt.owner, tt.repo)
				}
			}
		})
	}
}

func TestWriteFileNilDoc(t *testing.T) {
	g := NewGenerator(&mockProvider{}, t.TempDir(), false)
	_, err := g.writeFile(nil, "")
	if err == nil {
		t.Fatal("nil doc should return error")
	}
}

func TestWriteFileRelativeOutputDir(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{}, root, false)
	path, err := g.writeFile(&CoDDDocument{
		NodeID: "test-doc",
		Title:  "Test",
		Status: "draft",
		Body:   "# Test\n\ncontent",
	}, "relative/path")
	if err != nil {
		t.Fatalf("writeFile() error = %v", err)
	}
	if !strings.Contains(path, "relative/path") {
		t.Fatalf("path = %q, should contain relative/path", path)
	}
}

func TestGenerateEmptyProjectRoot(t *testing.T) {
	g := NewGenerator(&mockProvider{}, "", false)
	_, err := g.Generate(context.TODO(), GenerateRequest{DiscussionID: "1"})
	if err == nil {
		t.Fatal("empty projectRoot should error")
	}
}

func TestGenerateWithProjectRootOverride(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{output: `{"node_id":"x","title":"T","summary":"s","sections":[],"status":"draft"}`}, "", true)

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

	res, err := g.Generate(context.Background(), GenerateRequest{
		DiscussionID: "1",
		ProjectRoot:  root,
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if res.Document == nil {
		t.Fatal("document should not be nil")
	}
}

func TestGenerateStructurizeError(t *testing.T) {
	root := t.TempDir()
	provider := &mockProvider{err: errors.New("ai failed")}
	g := NewGenerator(provider, root, true)

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

	_, err := g.Generate(context.Background(), GenerateRequest{DiscussionID: "1"})
	if err == nil {
		t.Fatal("structurize error should propagate")
	}
}

func TestStructurizeNilDiscussion(t *testing.T) {
	g := NewGenerator(&mockProvider{}, t.TempDir(), true)
	_, err := g.structurize(context.Background(), nil)
	if err == nil {
		t.Fatal("nil discussion should error")
	}
}

func TestStructurizeInvalidJSON(t *testing.T) {
	g := NewGenerator(&mockProvider{output: "not json"}, t.TempDir(), true)
	_, err := g.structurize(context.Background(), &DiscussionData{Title: "T", Body: "B"})
	if err == nil {
		t.Fatal("invalid JSON should error")
	}
}

func TestFetchDiscussionInvalidID(t *testing.T) {
	g := NewGenerator(&mockProvider{}, t.TempDir(), false)
	_, err := g.fetchDiscussion(context.Background(), "notanumber")
	if err == nil {
		t.Fatal("non-numeric ID should error")
	}
}

func TestFetchDiscussionEmptyTitle(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{}, root, false)

	orig := fetchCommandContext
	t.Cleanup(func() { fetchCommandContext = orig })
	fetchCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "git" {
			return exec.CommandContext(ctx, "sh", "-c", "echo https://github.com/acme/rocket.git")
		}
		return exec.CommandContext(ctx, "sh", "-c", `cat <<'JSON'
{"data":{"repository":{"discussion":{"title":"","body":"","createdAt":"","labels":{"nodes":[]},"comments":{"nodes":[]}}}}}
JSON`)
	}

	_, err := g.fetchDiscussion(context.Background(), "99")
	if err == nil {
		t.Fatal("empty title should error (not found)")
	}
}

func TestFetchDiscussionCommandFail(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{}, root, false)

	orig := fetchCommandContext
	t.Cleanup(func() { fetchCommandContext = orig })
	fetchCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "git" {
			return exec.CommandContext(ctx, "sh", "-c", "echo https://github.com/acme/rocket.git")
		}
		return exec.CommandContext(ctx, "sh", "-c", "echo fail >&2; exit 1")
	}

	_, err := g.fetchDiscussion(context.Background(), "1")
	if err == nil {
		t.Fatal("command failure should error")
	}
}

func TestFetchDiscussionInvalidJSON(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{}, root, false)

	orig := fetchCommandContext
	t.Cleanup(func() { fetchCommandContext = orig })
	fetchCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "git" {
			return exec.CommandContext(ctx, "sh", "-c", "echo https://github.com/acme/rocket.git")
		}
		return exec.CommandContext(ctx, "sh", "-c", "echo 'not json'")
	}

	_, err := g.fetchDiscussion(context.Background(), "1")
	if err == nil {
		t.Fatal("invalid JSON should error")
	}
}

func TestResolveRepositoryFail(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{}, root, false)

	orig := fetchCommandContext
	t.Cleanup(func() { fetchCommandContext = orig })
	fetchCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "exit 1")
	}

	_, err := g.fetchDiscussion(context.Background(), "1")
	if err == nil {
		t.Fatal("resolve repository fail should error")
	}
}

func TestGenerateFetchError(t *testing.T) {
	root := t.TempDir()
	g := NewGenerator(&mockProvider{}, root, true)

	orig := fetchCommandContext
	t.Cleanup(func() { fetchCommandContext = orig })
	fetchCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "exit 1")
	}

	_, err := g.Generate(context.Background(), GenerateRequest{DiscussionID: "1"})
	if err == nil {
		t.Fatal("fetch error should propagate")
	}
}

func TestStripCodeFenceJSONPrefix(t *testing.T) {
	// JSON prefix case-insensitive
	got := stripCodeFence("```JSON\n{\"key\":\"val\"}\n```")
	if got != `{"key":"val"}` {
		t.Fatalf("got %q", got)
	}
}
