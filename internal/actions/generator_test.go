package actions_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/actions"
	"github.com/taka-sho/teraflow/internal/hooks"
)

func TestGenerateHookWorkflowsDiscussion(t *testing.T) {
	tmp := t.TempDir()
	cfg := hooks.HookConfig{
		hooks.EventDiscussionCreated: []hooks.HookAction{{Action: "respond"}},
		hooks.EventDiscussionComment: []hooks.HookAction{{Action: "respond"}},
		hooks.EventConfirmation:      []hooks.HookAction{{Action: "summarize"}},
	}

	if err := actions.GenerateHookWorkflows(cfg, tmp); err != nil {
		t.Fatalf("GenerateHookWorkflows: %v", err)
	}

	path := filepath.Join(tmp, "teraflow-hooks-discussion.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected discussion workflow: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "name: teraflow-hooks-discussion") {
		t.Fatalf("missing workflow name: %s", s)
	}
	if !strings.Contains(s, "discussion_comment:") {
		t.Fatalf("missing discussion_comment trigger: %s", s)
	}
	if !strings.Contains(s, `teraflow hook run "on_confirmation"`) {
		t.Fatalf("missing confirmation hook run: %s", s)
	}
}

func TestGenerateHookWorkflowsPushAndPR(t *testing.T) {
	tmp := t.TempDir()
	cfg := hooks.HookConfig{
		hooks.EventPush:     []hooks.HookAction{{Action: "index_update"}},
		hooks.EventPROpened: []hooks.HookAction{{Action: "respond", Skill: "review"}},
	}

	if err := actions.GenerateHookWorkflows(cfg, tmp); err != nil {
		t.Fatalf("GenerateHookWorkflows: %v", err)
	}

	pushData, err := os.ReadFile(filepath.Join(tmp, "teraflow-hooks-push.yml"))
	if err != nil {
		t.Fatalf("expected push workflow: %v", err)
	}
	if !strings.Contains(string(pushData), `teraflow hook run "on_push"`) {
		t.Fatalf("missing on_push hook run: %s", string(pushData))
	}

	prData, err := os.ReadFile(filepath.Join(tmp, "teraflow-hooks-pr.yml"))
	if err != nil {
		t.Fatalf("expected pr workflow: %v", err)
	}
	if !strings.Contains(string(prData), "pull_request:") {
		t.Fatalf("missing pull_request trigger: %s", string(prData))
	}
	if !strings.Contains(string(prData), `teraflow hook run "on_pr_opened"`) {
		t.Fatalf("missing on_pr_opened hook run: %s", string(prData))
	}
}

func TestGenerateHookWorkflowsEmptyConfig(t *testing.T) {
	tmp := t.TempDir()
	if err := actions.GenerateHookWorkflows(hooks.HookConfig{}, tmp); err != nil {
		t.Fatalf("GenerateHookWorkflows with empty config should not fail: %v", err)
	}

	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no generated workflows, got %d", len(entries))
	}
}

func TestHookWorkflowNames(t *testing.T) {
	cfg := hooks.HookConfig{
		hooks.EventDiscussionComment: []hooks.HookAction{{Action: "respond"}},
		hooks.EventPush:              []hooks.HookAction{{Action: "index_update"}},
	}

	got := actions.HookWorkflowNames(cfg)
	if len(got) != 2 {
		t.Fatalf("expected 2 names, got %d (%v)", len(got), got)
	}
	if got[0] != "teraflow-hooks-discussion" {
		t.Fatalf("unexpected first workflow name: %v", got)
	}
	if got[1] != "teraflow-hooks-push" {
		t.Fatalf("unexpected second workflow name: %v", got)
	}
}
