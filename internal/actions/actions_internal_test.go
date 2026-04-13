package actions

import (
	"embed"
	"testing"
)

func TestListTemplatesReadDirError(t *testing.T) {
	original := templateFS
	templateFS = embed.FS{}
	defer func() { templateFS = original }()

	_, err := ListTemplates()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateWorkflowsWalkDirError(t *testing.T) {
	original := templateFS
	templateFS = embed.FS{}
	defer func() { templateFS = original }()

	err := GenerateWorkflows(t.TempDir(), "v9.9.9")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderWorkflowTemplateReplacesTeraflowVersion(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "no spaces",
			content:  `go install "github.com/taka-sho/teraflow@{{.TeraflowVersion}}"`,
			expected: `go install "github.com/taka-sho/teraflow@v9.9.9"`,
		},
		{
			name:     "with spaces",
			content:  `go install "github.com/taka-sho/teraflow@{{ .TeraflowVersion }}"`,
			expected: `go install "github.com/taka-sho/teraflow@v9.9.9"`,
		},
		{
			name:     "with tabs",
			content:  "go install \"github.com/taka-sho/teraflow@{{\t.TeraflowVersion\t}}\"",
			expected: `go install "github.com/taka-sho/teraflow@v9.9.9"`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := renderWorkflowTemplate(tc.content, "v9.9.9")
			if got != tc.expected {
				t.Fatalf("unexpected render result:\nwant: %q\n got: %q", tc.expected, got)
			}
		})
	}
}

func TestRenderWorkflowTemplatePreservesGitHubExpressions(t *testing.T) {
	in := `if [ "${{ github.event_name }}" = "discussion_comment" ]; then echo ok; fi`
	got := renderWorkflowTemplate(in, "v9.9.9")
	if got != in {
		t.Fatalf("github expression should remain unchanged:\nwant: %q\n got: %q", in, got)
	}
}
