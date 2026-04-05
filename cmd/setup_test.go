package cmd

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/templates"
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
	if len(discussionEntries) != 7 {
		t.Fatalf("expected 7 discussion templates, got %d", len(discussionEntries))
	}
}

func TestSetupTemplatesAlreadyExistsError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".github", "ISSUE_TEMPLATE", "phase-start.yml"), "# existing\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"setup", "templates", "--config", cfgPath})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when template exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetupActionsAndTemplatesJSON(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--format", "json", "setup", "actions", "--config", cfgPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("setup actions --format json: %v", err)
	}
	if !strings.Contains(out.String(), `"status"`) || !strings.Contains(out.String(), `"generated"`) {
		t.Fatalf("expected JSON output, got: %s", out.String())
	}

	out.Reset()
	root = newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--format", "json", "setup", "templates", "--config", cfgPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("setup templates --format json: %v", err)
	}
	if !strings.Contains(out.String(), `"issue_templates"`) {
		t.Fatalf("expected templates JSON output, got: %s", out.String())
	}
}

func TestCopyEmbedFSErrors(t *testing.T) {
	tmp := t.TempDir()

	_, err := copyEmbedFS(templates.IssueFS, "missing-dir", filepath.Join(tmp, "out"), false)
	if err == nil {
		t.Fatal("expected error for missing source directory")
	}
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("expected path error, got: %T (%v)", err, err)
	}

	blockingFile := filepath.Join(tmp, "blocking-file")
	mustWrite(t, blockingFile, "x")
	_, err = copyEmbedFS(templates.IssueFS, "issues", filepath.Join(blockingFile, "subdir"), false)
	if err == nil {
		t.Fatal("expected mkdir error")
	}
	if !strings.Contains(err.Error(), "create directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetupActionsAndTemplatesInvalidFormat(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--format", "xml", "setup", "actions", "--config", cfgPath})
	if err := root.Execute(); err == nil {
		t.Fatal("expected invalid format error for setup actions")
	}

	root = newRootCmd("test")
	root.SetArgs([]string{"--format", "xml", "setup", "templates", "--config", cfgPath})
	if err := root.Execute(); err == nil {
		t.Fatal("expected invalid format error for setup templates")
	}
}

func TestSetupCommandsConfigFlagError(t *testing.T) {
	cmd := newSetupActionsCmd()
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected config flag error for setup actions")
	}

	cmd = newSetupTemplatesCmd()
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected config flag error for setup templates")
	}
}

func TestLoadDiscussionCategoryConfig(t *testing.T) {
	cfg, err := loadDiscussionCategoryConfig()
	if err != nil {
		t.Fatalf("loadDiscussionCategoryConfig failed: %v", err)
	}
	if len(cfg.Categories) == 0 {
		t.Fatal("expected categories to be loaded")
	}
	if cfg.Categories[0].Name == "" {
		t.Fatal("expected first category name")
	}
}

func TestSetupTemplatesSyncSkipsWhenGHMissing(t *testing.T) {
	oldLookPath := ghLookPath
	ghLookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}
	t.Cleanup(func() { ghLookPath = oldLookPath })

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"setup", "templates", "--sync", "--config", cfgPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("setup templates --sync should not fail when gh is missing: %v", err)
	}
	if !strings.Contains(out.String(), "Discussion category check skipped: gh CLI is not installed") {
		t.Fatalf("expected skip message, got: %s", out.String())
	}
}

func TestResolveRepositoryInfoFallsBackToConfig(t *testing.T) {
	oldExec := ghExecCommand
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 1")
	}
	t.Cleanup(func() { ghExecCommand = oldExec })

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\nproject:\n  repository: \"https://github.com/acme/rocket.git\"\n")

	owner, repo, err := resolveRepositoryInfo(cfgPath)
	if err != nil {
		t.Fatalf("resolveRepositoryInfo should fallback to config: %v", err)
	}
	if owner != "acme" || repo != "rocket" {
		t.Fatalf("unexpected owner/repo: %s/%s", owner, repo)
	}
}
