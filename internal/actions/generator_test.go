package actions_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/actions"
	"github.com/taka-sho/teraflow/internal/hooks"
)

func TestGenerateHookWorkflowsPROnly(t *testing.T) {
	tmp := t.TempDir()
	cfg := hooks.HookConfig{
		hooks.EventPush:     []hooks.HookAction{{Action: "index_update"}},
		hooks.EventPROpened: []hooks.HookAction{{Action: "respond", Skill: "review"}},
	}

	if err := actions.GenerateHookWorkflows(cfg, tmp, "v9.9.9"); err != nil {
		t.Fatalf("GenerateHookWorkflows: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, "teraflow-hooks-push.yml")); !os.IsNotExist(err) {
		t.Fatalf("push workflow should not be generated, err=%v", err)
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
	if err := actions.GenerateHookWorkflows(hooks.HookConfig{}, tmp, "v9.9.9"); err != nil {
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
		hooks.EventPROpened:          []hooks.HookAction{{Action: "respond"}},
	}

	got := actions.HookWorkflowNames(cfg)
	if len(got) != 1 {
		t.Fatalf("expected 1 name, got %d (%v)", len(got), got)
	}
	if got[0] != "teraflow-hooks-pr" {
		t.Fatalf("unexpected workflow name: %v", got)
	}
}
