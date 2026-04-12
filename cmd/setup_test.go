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

	"github.com/taka-sho/teraflow/internal/actions"
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
	expected := len(actions.WorkflowNames)
	if len(entries) != expected {
		t.Fatalf("expected %d workflows, got %d", expected, len(entries))
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

func TestSetupActionsWithHooks(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\nhooks:\n  on_pr_opened:\n    - action: respond\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"setup", "actions", "--config", cfgPath, "--hooks"})
	if err := root.Execute(); err != nil {
		t.Fatalf("setup actions --hooks: %v", err)
	}

	workflowDir := filepath.Join(tmp, ".github", "workflows")
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		t.Fatalf("workflow dir not created: %v", err)
	}
	expected := len(actions.WorkflowNames) + 1
	if len(entries) != expected {
		t.Fatalf("expected %d workflows (%d default + 1 hook), got %d", expected, expected-1, len(entries))
	}
	if _, err := os.Stat(filepath.Join(workflowDir, "teraflow-hooks-pr.yml")); err != nil {
		t.Fatalf("expected hook workflow file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workflowDir, "teraflow-hooks-push.yml")); !os.IsNotExist(err) {
		t.Fatalf("push hook workflow should not be generated, err=%v", err)
	}
}

func TestSetupActionsHooksOnly(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\nhooks:\n  on_pr_opened:\n    - action: respond\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"setup", "actions", "--config", cfgPath, "--hooks-only"})
	if err := root.Execute(); err != nil {
		t.Fatalf("setup actions --hooks-only: %v", err)
	}

	workflowDir := filepath.Join(tmp, ".github", "workflows")
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		t.Fatalf("workflow dir not created: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 hook workflow, got %d", len(entries))
	}
	if entries[0].Name() != "teraflow-hooks-pr.yml" {
		t.Fatalf("unexpected generated workflow: %s", entries[0].Name())
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
	if len(issueEntries) != 7 {
		t.Fatalf("expected 7 issue templates, got %d", len(issueEntries))
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

func TestDiscussionFormatLabel(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "open", input: "OPEN", expect: "Open-ended discussion"},
		{name: "announcement", input: "announcement", expect: "Announcements"},
		{name: "qanda", input: " QANDA ", expect: "Question and answer"},
		{name: "empty", input: "   ", expect: "Open-ended discussion"},
		{name: "unknown", input: "POLL", expect: "POLL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := discussionFormatLabel(tt.input); got != tt.expect {
				t.Fatalf("discussionFormatLabel(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestFetchDiscussionCategories(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		oldExec := ghExecCommand
		ghExecCommand = func(name string, args ...string) *exec.Cmd {
			return exec.Command("sh", "-c", "echo 'api failed' 1>&2; exit 1")
		}
		t.Cleanup(func() { ghExecCommand = oldExec })

		_, err := fetchDiscussionCategories("acme", "rocket")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "api failed") {
			t.Fatalf("expected stderr to be included, got: %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		oldExec := ghExecCommand
		ghExecCommand = func(name string, args ...string) *exec.Cmd {
			return exec.Command("sh", "-c", `printf '{"data":{"repository":{"discussionCategories":{"nodes":[{"name":"General","emoji":"💬","description":"Talk"}]}}}}'`)
		}
		t.Cleanup(func() { ghExecCommand = oldExec })

		categories, err := fetchDiscussionCategories("acme", "rocket")
		if err != nil {
			t.Fatalf("fetchDiscussionCategories returned error: %v", err)
		}
		if len(categories) != 1 {
			t.Fatalf("expected 1 category, got %d", len(categories))
		}
		if categories[0].Name != "General" || categories[0].Emoji != "💬" {
			t.Fatalf("unexpected category: %+v", categories[0])
		}
	})
}

func TestSyncDiscussionCategories(t *testing.T) {
	t.Run("skip when gh missing", func(t *testing.T) {
		oldLookPath := ghLookPath
		ghLookPath = func(file string) (string, error) { return "", errors.New("missing") }
		t.Cleanup(func() { ghLookPath = oldLookPath })

		tmp := t.TempDir()
		cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
		mustWrite(t, cfgPath, "version: \"1\"\n")

		result, err := syncDiscussionCategories(cfgPath)
		if err != nil {
			t.Fatalf("syncDiscussionCategories returned error: %v", err)
		}
		if !result.Skipped || result.SkipReason != "gh CLI is not installed" {
			t.Fatalf("unexpected skip result: %+v", result)
		}
	})

	t.Run("classify existing and missing", func(t *testing.T) {
		cfg, err := loadDiscussionCategoryConfig()
		if err != nil {
			t.Fatalf("loadDiscussionCategoryConfig: %v", err)
		}
		if len(cfg.Categories) == 0 {
			t.Fatal("expected embedded categories")
		}
		existingCategory := cfg.Categories[0]
		repoViewJSON := filepath.Join(t.TempDir(), "repo-view.json")
		apiJSON := filepath.Join(t.TempDir(), "api.json")
		mustWrite(t, repoViewJSON, `{"owner":{"login":"acme"},"name":"rocket"}`)
		mustWrite(t, apiJSON, `{"data":{"repository":{"discussionCategories":{"nodes":[{"name":"`+existingCategory.Name+`","emoji":"`+existingCategory.Emoji+`","description":"`+existingCategory.Description+`"}]}}}}`)

		oldLookPath := ghLookPath
		oldExec := ghExecCommand
		ghLookPath = func(file string) (string, error) { return "/usr/bin/gh", nil }
		ghExecCommand = func(name string, args ...string) *exec.Cmd {
			if len(args) >= 3 && args[0] == "repo" && args[1] == "view" {
				return exec.Command("cat", repoViewJSON)
			}
			if len(args) >= 2 && args[0] == "api" && args[1] == "graphql" {
				return exec.Command("cat", apiJSON)
			}
			return exec.Command("sh", "-c", "exit 1")
		}
		t.Cleanup(func() {
			ghLookPath = oldLookPath
			ghExecCommand = oldExec
		})

		tmp := t.TempDir()
		cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
		mustWrite(t, cfgPath, "version: \"1\"\nproject:\n  repository: \"acme/rocket\"\n")

		result, err := syncDiscussionCategories(cfgPath)
		if err != nil {
			t.Fatalf("syncDiscussionCategories returned error: %v", err)
		}
		if result.Skipped {
			t.Fatalf("unexpected skipped result: %+v", result)
		}
		if result.RepositoryOwner != "acme" || result.RepositoryName != "rocket" {
			t.Fatalf("unexpected repository: %+v", result)
		}
		if len(result.Existing) != 1 || result.Existing[0].Name != existingCategory.Name {
			t.Fatalf("unexpected existing categories: %+v", result.Existing)
		}
		if len(result.Missing) != len(cfg.Categories)-1 {
			t.Fatalf("unexpected missing count: got %d want %d", len(result.Missing), len(cfg.Categories)-1)
		}
	})
}

func TestResolveRepositoryInfoReturnsErrorWhenGHAndConfigFail(t *testing.T) {
	oldExec := ghExecCommand
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'gh unavailable' 1>&2; exit 1")
	}
	t.Cleanup(func() { ghExecCommand = oldExec })

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\nproject:\n  repository: \"invalid\"\n")

	_, _, err := resolveRepositoryInfo(cfgPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "resolve repository") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrintDiscussionCategorySyncResult(t *testing.T) {
	t.Run("skipped", func(t *testing.T) {
		root := newRootCmd("test")
		var out bytes.Buffer
		root.SetOut(&out)

		printDiscussionCategorySyncResult(root, &discussionCategorySyncResult{
			Skipped:    true,
			SkipReason: "gh not installed",
		})
		if !strings.Contains(out.String(), "Discussion category check skipped: gh not installed") {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})

	t.Run("all present", func(t *testing.T) {
		root := newRootCmd("test")
		var out bytes.Buffer
		root.SetOut(&out)
		result := &discussionCategorySyncResult{
			RepositoryOwner: "acme",
			RepositoryName:  "rocket",
			Existing: []discussionCategoryDefinition{
				{Name: "General", Emoji: "💬"},
			},
		}

		printDiscussionCategorySyncResult(root, result)
		got := out.String()
		if !strings.Contains(got, "✅ General 💬") {
			t.Fatalf("unexpected output: %s", got)
		}
		if !strings.Contains(got, "All configured Discussion categories are present.") {
			t.Fatalf("unexpected output: %s", got)
		}
	})

	t.Run("missing", func(t *testing.T) {
		root := newRootCmd("test")
		var out bytes.Buffer
		root.SetOut(&out)
		result := &discussionCategorySyncResult{
			RepositoryOwner: "acme",
			RepositoryName:  "rocket",
			Missing: []discussionCategoryDefinition{
				{Name: "Q&A", Emoji: "❓", Description: "Ask anything", Format: "QANDA"},
			},
		}

		printDiscussionCategorySyncResult(root, result)
		got := out.String()
		if !strings.Contains(got, "https://github.com/acme/rocket/settings/discussions") {
			t.Fatalf("unexpected output: %s", got)
		}
		if !strings.Contains(got, "- Format: Question and answer") {
			t.Fatalf("unexpected output: %s", got)
		}
	})
}
