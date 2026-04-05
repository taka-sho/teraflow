package hooks

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutorExecuteAllDryRunActions(t *testing.T) {
	e := NewExecutor(t.TempDir(), true)
	ctx := HookContext{
		Event:        EventDiscussionComment,
		Author:       "alice",
		Category:     "requirements",
		Labels:       []string{"ai:request"},
		Paths:        []string{"docs/spec.md"},
		Input:        "please summarize this discussion",
		DiscussionID: "123",
	}
	actions := []HookAction{
		{Action: "respond", Skill: "requirements"},
		{Action: "summarize"},
		{Action: "index_update"},
		{Action: "summary_update"},
		{Action: "generate"},
	}

	result := e.ExecuteAll(ctx, actions)
	if result.Event != EventDiscussionComment {
		t.Fatalf("unexpected event: %s", result.Event)
	}
	if !result.DryRun {
		t.Fatal("expected dry-run=true")
	}
	if len(result.Matched) != len(actions) || len(result.Results) != len(actions) {
		t.Fatalf("unexpected result sizes: matched=%d results=%d", len(result.Matched), len(result.Results))
	}

	for i, ar := range result.Results {
		if !ar.Success {
			t.Fatalf("action[%d] failed unexpectedly: %+v", i, ar)
		}
		if ar.Action != "generate" && !strings.Contains(ar.Message, "would run:") {
			t.Fatalf("expected dry-run message for action[%d], got: %q", i, ar.Message)
		}
	}
}

func TestExecutorExecuteGenerateStub(t *testing.T) {
	e := NewExecutor(t.TempDir(), false)
	result := e.Execute(HookContext{Event: EventPush}, HookAction{Action: "generate"})
	if !result.Success {
		t.Fatalf("expected generate stub success, got: %+v", result)
	}
	if !strings.Contains(result.Message, "stub (Phase 5)") {
		t.Fatalf("unexpected generate message: %q", result.Message)
	}
}

func TestExecutorExecuteIndexUpdateDryRun(t *testing.T) {
	t.Parallel()
	e := NewExecutor(t.TempDir(), true)

	got := e.executeIndexUpdate()
	if !got.Success {
		t.Fatalf("executeIndexUpdate() success = false: %+v", got)
	}
	if !strings.Contains(got.Message, "would run:") {
		t.Fatalf("executeIndexUpdate() message = %q, want dry-run message", got.Message)
	}
}

func TestExecutorExecuteIndexUpdateSuccess(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteFile(t, root+"/docs/req.md", `---
codd:
  node_id: req:test
  title: Test
---
body`)

	e := NewExecutor(root, false)
	got := e.executeIndexUpdate()
	if !got.Success {
		t.Fatalf("executeIndexUpdate() success = false: %+v", got)
	}
	if !strings.Contains(got.Message, "index updated:") {
		t.Fatalf("executeIndexUpdate() message = %q", got.Message)
	}
}

func TestExecutorExecuteSummaryUpdateDryRun(t *testing.T) {
	t.Parallel()
	e := NewExecutor(t.TempDir(), true)

	got := e.executeSummaryUpdate()
	if !got.Success {
		t.Fatalf("executeSummaryUpdate() success = false: %+v", got)
	}
	if !strings.Contains(got.Message, "would run:") {
		t.Fatalf("executeSummaryUpdate() message = %q, want dry-run message", got.Message)
	}
}

func TestExecutorExecuteSummaryUpdateLoadIndexFailed(t *testing.T) {
	t.Parallel()
	e := NewExecutor(t.TempDir(), false)

	got := e.executeSummaryUpdate()
	if got.Success {
		t.Fatalf("executeSummaryUpdate() success = true, want false: %+v", got)
	}
	if !strings.Contains(got.Message, "load index failed:") {
		t.Fatalf("executeSummaryUpdate() message = %q", got.Message)
	}
}

func TestExecutorExecuteSummaryUpdateLoadConfigFailed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteFile(t, root+"/.teraflow/index.yml", "version: \"1\"\nentries: []\n")

	e := NewExecutor(root, false)
	got := e.executeSummaryUpdate()
	if got.Success {
		t.Fatalf("executeSummaryUpdate() success = true, want false: %+v", got)
	}
	if !strings.Contains(got.Message, "load config failed:") {
		t.Fatalf("executeSummaryUpdate() message = %q", got.Message)
	}
}

func TestExecutorExecuteSummaryUpdateCreateProviderFailed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteFile(t, root+"/.teraflow/index.yml", "version: \"1\"\nentries: []\n")
	mustWriteFile(t, root+"/.github/teraflow.yml", `version: "1"
project:
  name: test
  description: test
  repository: test
ai:
  default_provider: custom
agent:
  provider: anthropic
  model: claude-haiku-4-5-20251001
  max_tokens: 500
  timeout: 60
  custom_command: ""
  fallback: ""
  trust_level: low
  rate_limit:
    max_calls_per_hour: 10
`)

	e := NewExecutor(root, false)
	got := e.executeSummaryUpdate()
	if got.Success {
		t.Fatalf("executeSummaryUpdate() success = true, want false: %+v", got)
	}
	if !strings.Contains(got.Message, "create provider failed:") {
		t.Fatalf("executeSummaryUpdate() message = %q", got.Message)
	}
}

func TestExecutorExecuteSummaryUpdateSuccess(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWriteFile(t, root+"/.teraflow/index.yml", "version: \"1\"\nentries: []\n")
	mustWriteFile(t, root+"/.github/teraflow.yml", `version: "1"
project:
  name: test
  description: test
  repository: test
ai:
  default_provider: claude-code
agent:
  provider: anthropic
  model: claude-haiku-4-5-20251001
  max_tokens: 500
  timeout: 60
  custom_command: ""
  fallback: ""
  trust_level: low
  rate_limit:
    max_calls_per_hour: 10
`)

	e := NewExecutor(root, false)
	got := e.executeSummaryUpdate()
	if !got.Success {
		t.Fatalf("executeSummaryUpdate() success = false: %+v", got)
	}
	if !strings.Contains(got.Message, "summary updated: updated=0 skipped=0") {
		t.Fatalf("executeSummaryUpdate() message = %q", got.Message)
	}
}

func TestExecutorRunCommandActionDryRun(t *testing.T) {
	t.Parallel()
	e := NewExecutor(t.TempDir(), true)

	got := e.runCommandAction("respond", []string{"agent", "assign", "--type", "requirements"})
	if !got.Success {
		t.Fatalf("runCommandAction() success = false: %+v", got)
	}
	if !strings.Contains(got.Message, "would run: teraflow agent assign") {
		t.Fatalf("runCommandAction() message = %q", got.Message)
	}
}

func TestExecutorRunCommandActionCommandFailed(t *testing.T) {
	t.Parallel()
	orig := hookExecCommandContext
	t.Cleanup(func() { hookExecCommandContext = orig })
	hookExecCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "echo failure >&2; exit 1")
	}

	e := NewExecutor(t.TempDir(), false)
	got := e.runCommandAction("respond", []string{"agent", "assign"})
	if got.Success {
		t.Fatalf("runCommandAction() success = true, want false: %+v", got)
	}
	if !strings.Contains(got.Message, "command failed: failure") {
		t.Fatalf("runCommandAction() message = %q", got.Message)
	}
}

func mustWriteFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
