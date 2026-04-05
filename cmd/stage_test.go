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

func TestStageListNoProject(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "stage", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing project state")
	}
	if !strings.Contains(err.Error(), notProjectError) {
		t.Fatalf("expected notProjectError, got: %v", err)
	}
}

func TestStageListUnsupportedFormat(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "--format", "yaml", "stage", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected unsupported format error")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("unexpected error: %v", err)
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

func TestStageAdvanceWithPromptConfirmation(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader("y\n"))
	root.SetArgs([]string{"--config", configPath, "stage", "advance"})

	if err := root.Execute(); err != nil {
		t.Fatalf("stage advance with prompt failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Gate check: OK") {
		t.Fatalf("expected gate check output: %s", got)
	}
	if !strings.Contains(got, "Advanced: release") {
		t.Fatalf("expected advanced output: %s", got)
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".github", "project-state.yml"))
	if err != nil {
		t.Fatalf("read project-state: %v", err)
	}
	if !strings.Contains(string(data), "current_stage: release") {
		t.Fatalf("stage not advanced in state file:\n%s", string(data))
	}
}

func TestStageStatus(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "implementation")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "stage", "status"})

	if err := root.Execute(); err != nil {
		t.Fatalf("stage status failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Stage: release") {
		t.Fatalf("expected stage in output: %s", got)
	}
	if !strings.Contains(got, "Started:") {
		t.Fatalf("expected Started in output: %s", got)
	}
}

func TestStageStatusJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "implementation")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "stage", "status"})

	if err := root.Execute(); err != nil {
		t.Fatalf("stage status --format json failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"stage"`) {
		t.Fatalf("expected stage in JSON: %s", got)
	}
	if !strings.Contains(got, `"release"`) {
		t.Fatalf("expected release in JSON: %s", got)
	}
}

func TestStageStatusNoProject(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "stage", "status"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when project-state is missing")
	}
	if !strings.Contains(err.Error(), notProjectError) {
		t.Fatalf("expected notProjectError, got: %v", err)
	}
}

func TestStageStatusUnsupportedFormat(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "implementation")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "--format", "yaml", "stage", "status"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected unsupported format error")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStageAdvanceFinalStage(t *testing.T) {
	tmp := t.TempDir()
	// retirement is the last in stageAdvanceOrder
	configPath := setupTestProjectState(t, tmp, "retirement", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "stage", "advance"})

	if err := root.Execute(); err != nil {
		t.Fatalf("stage advance at final stage failed: %v", err)
	}

	if !strings.Contains(out.String(), "Already at final stage.") {
		t.Fatalf("expected final stage message: %s", out.String())
	}
}

func TestStageAdvanceFinalStageJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "retirement", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "stage", "advance"})

	if err := root.Execute(); err != nil {
		t.Fatalf("stage advance final --format json failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"no_change"`) {
		t.Fatalf("expected no_change in JSON: %s", got)
	}
}

func TestStageAdvanceJSONConfirmationRequired(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "stage", "advance"})

	if err := root.Execute(); err != nil {
		t.Fatalf("stage advance json no-yes failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"confirmation_required"`) {
		t.Fatalf("expected confirmation_required in JSON: %s", got)
	}
}

func TestStageAdvanceJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "stage", "advance", "--yes"})

	if err := root.Execute(); err != nil {
		t.Fatalf("stage advance --format json --yes failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"advanced"`) {
		t.Fatalf("expected advanced in JSON: %s", got)
	}
}

func TestStageAdvanceCancelled(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader("n\n"))
	root.SetArgs([]string{"--config", configPath, "stage", "advance"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for cancelled stage advance")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected cancelled error, got: %v", err)
	}
}

func TestStageAdvanceSaveError(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")
	statePath := filepath.Join(tmp, ".github", "project-state.yml")
	if err := os.Chmod(statePath, 0o400); err != nil {
		t.Fatalf("chmod state file: %v", err)
	}

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "stage", "advance", "--yes"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected save error")
	}
}

func TestLoadLifecycleStartedAtInvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, "version: \"1\"\n")
	// Write invalid YAML to state file
	statePath := filepath.Join(tmp, ".github", "project-state.yml")
	writeCmdTestFile(t, statePath, ":\n  bad: [\nbroken")

	_, err := loadLifecycleStartedAt(configPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML in state file")
	}
}

func TestLoadStateOrNotProjectErr_noProject(t *testing.T) {
	tmp := t.TempDir()
	// Config exists but no project-state.yml
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, "version: \"1\"\n")

	_, err := loadStateOrNotProjectErr(configPath)
	if err == nil {
		t.Fatal("expected error for missing state file")
	}
	if !strings.Contains(err.Error(), notProjectError) {
		t.Fatalf("expected notProjectError, got: %v", err)
	}
}

func TestNextValue(t *testing.T) {
	order := []string{"a", "b", "c"}

	got, ok := nextValue(order, "a")
	if !ok || got != "b" {
		t.Fatalf("expected b, got %q ok=%v", got, ok)
	}

	got, ok = nextValue(order, "c")
	if ok {
		t.Fatalf("expected false for last element, got %q", got)
	}

	_, ok = nextValue(order, "z")
	if ok {
		t.Fatal("expected false for missing element")
	}
}
