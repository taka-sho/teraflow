package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestDashboardShowJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	root.AddCommand(newDashboardCmd())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "dashboard", "show"})

	if err := root.Execute(); err != nil {
		t.Fatalf("dashboard show --format json failed: %v", err)
	}
	if !strings.Contains(out.String(), `"project"`) {
		t.Fatalf("expected JSON output: %s", out.String())
	}
}

func TestDashboardShow(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "operation", "testing")
	writeCmdTestFile(t, tmp+"/.teraflow/rework-log.yml", `reworks:
  - id: rw-001
    group: group-auth
    target_phase: basic_design
    reason: API change
    created_at: "2026-04-05T10:00:00"
    status: open
  - id: rw-002
    group: group-api
    target_phase: implementation
    reason: Fix integration
    created_at: "2026-04-05T11:00:00"
    status: resolved
`)

	root := newRootCmd("test")
	root.AddCommand(newDashboardCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "dashboard", "show"})

	if err := root.Execute(); err != nil {
		t.Fatalf("dashboard show failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "teraflow Dashboard") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "Project:  test") {
		t.Fatalf("project not shown: %s", got)
	}
	if !strings.Contains(got, "Stage:    operation") {
		t.Fatalf("stage not shown: %s", got)
	}
	if !strings.Contains(got, "Phase:    testing") {
		t.Fatalf("phase not shown: %s", got)
	}
	if !strings.Contains(got, "Total:    2  Open: 1  Resolved: 1") {
		t.Fatalf("summary not shown: %s", got)
	}
	if !strings.Contains(got, "rw-002") || !strings.Contains(got, "rw-001") {
		t.Fatalf("recent activity not shown: %s", got)
	}
	if strings.Index(got, "rw-002") > strings.Index(got, "rw-001") {
		t.Fatalf("recent activity not sorted desc by created_at: %s", got)
	}
}
