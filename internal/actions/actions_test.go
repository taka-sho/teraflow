package actions_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/actions"
)

func TestGenerateWorkflows(t *testing.T) {
	tmp := t.TempDir()
	if err := actions.GenerateWorkflows(tmp); err != nil {
		t.Fatalf("GenerateWorkflows: %v", err)
	}

	for _, name := range actions.WorkflowNames {
		path := filepath.Join(tmp, name+".yml")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected workflow %s: %v", path, err)
		}
	}
}

func TestListTemplates(t *testing.T) {
	names, err := actions.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(names) != 17 {
		t.Fatalf("expected 17 templates, got %d", len(names))
	}
}

func TestGenerateWorkflowsMkdirError(t *testing.T) {
	err := actions.GenerateWorkflows("/dev/null/teraflow-workflows")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create workflows directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateWorkflowsWriteError(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, actions.WorkflowNames[0]+".yml"), 0o755); err != nil {
		t.Fatalf("mkdir conflict dir: %v", err)
	}

	err := actions.GenerateWorkflows(tmp)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "write workflow") {
		t.Fatalf("unexpected error: %v", err)
	}
}
