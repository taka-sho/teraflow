package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupActionsGenerates(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"setup", "actions", "--config", cfgPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("setup actions: %v", err)
	}

	workflowDir := filepath.Join(tmp, ".github", "workflows")
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		t.Fatalf("workflow dir not created: %v", err)
	}
	if len(entries) != 16 {
		t.Fatalf("expected 16 workflows, got %d", len(entries))
	}
}

func TestSetupActionsAlreadyExistsError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".github", "workflows", "teraflow-phase-transition.yml"), "# existing\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"setup", "actions", "--config", cfgPath})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when workflow exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetupActionsForce(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".github", "workflows", "teraflow-phase-transition.yml"), "# existing\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"setup", "actions", "--config", cfgPath, "--force"})
	if err := root.Execute(); err != nil {
		t.Fatalf("setup actions --force: %v", err)
	}
}

func TestSetupTemplatesGenerates(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"setup", "templates", "--config", cfgPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("setup templates: %v", err)
	}

	issueDir := filepath.Join(tmp, ".github", "ISSUE_TEMPLATE")
	issueEntries, err := os.ReadDir(issueDir)
	if err != nil {
		t.Fatalf("ISSUE_TEMPLATE dir not created: %v", err)
	}
	if len(issueEntries) != 6 {
		t.Fatalf("expected 6 issue templates, got %d", len(issueEntries))
	}

	discussionDir := filepath.Join(tmp, ".github", "DISCUSSION_TEMPLATE")
	discussionEntries, err := os.ReadDir(discussionDir)
	if err != nil {
		t.Fatalf("DISCUSSION_TEMPLATE dir not created: %v", err)
	}
	if len(discussionEntries) != 3 {
		t.Fatalf("expected 3 discussion templates, got %d", len(discussionEntries))
	}
}
