package hooks

import (
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
