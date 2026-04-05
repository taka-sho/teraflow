package templates_test

import (
	"io/fs"
	"testing"

	"github.com/taka-sho/teraflow/internal/templates"
)

func TestIssueTemplatesEmbedded(t *testing.T) {
	entries, err := fs.ReadDir(templates.IssueFS, "issues")
	if err != nil {
		t.Fatalf("ReadDir issues: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected embedded issue templates")
	}
}

func TestDiscussionTemplatesEmbedded(t *testing.T) {
	entries, err := fs.ReadDir(templates.DiscussionFS, "discussions")
	if err != nil {
		t.Fatalf("ReadDir discussions: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected embedded discussion templates")
	}
}
