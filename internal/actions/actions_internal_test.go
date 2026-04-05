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

	err := GenerateWorkflows(t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
}
