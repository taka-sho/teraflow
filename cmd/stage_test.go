package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)


func TestStageList(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "stage", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("stage list failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "-> initial_development") {
		t.Fatalf("current stage marker not found: %s", got)
	}
	if !strings.Contains(got, "  retirement") {
		t.Fatalf("retirement stage not listed: %s", got)
	}
}

func TestAskForConfirmation(t *testing.T) {
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)

	root.SetIn(strings.NewReader("y\n"))
	got, err := askForConfirmation(root, "Proceed? ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected true for 'y'")
	}

	root.SetIn(strings.NewReader("n\n"))
	got, err = askForConfirmation(root, "Proceed? ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected false for 'n'")
	}
}

func TestLoadLifecycleStartedAt(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	// state file has no started_at — should return empty string without error
	got, err := loadLifecycleStartedAt(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}

	// write a state with started_at
	stateContent := `project:
  name: "test"
lifecycle:
  current_stage: "initial_development"
  started_at: "2026-04-05"
phases:
  current: "requirements"
`
	statePath := filepath.Join(tmp, ".github", "project-state.yml")
	if err := os.WriteFile(statePath, []byte(stateContent), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}
	got, err = loadLifecycleStartedAt(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2026-04-05" {
		t.Fatalf("expected 2026-04-05, got %q", got)
	}
}

func TestStageListJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "stage", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("stage list --format json failed: %v", err)
	}
	if !strings.Contains(out.String(), `"stages"`) {
		t.Fatalf("expected JSON with stages: %s", out.String())
	}
}

func TestStageAdvance(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "stage", "advance", "--yes"})

	if err := root.Execute(); err != nil {
		t.Fatalf("stage advance failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".github", "project-state.yml"))
	if err != nil {
		t.Fatalf("read project-state: %v", err)
	}
	if !strings.Contains(string(data), "current_stage: release") {
		t.Fatalf("stage not advanced in state file:\n%s", string(data))
	}
}
