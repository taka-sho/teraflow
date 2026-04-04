package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhaseList(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "phase", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("phase list failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "-> requirements") {
		t.Fatalf("current phase marker not found: %s", got)
	}
	if !strings.Contains(got, "  integration_test") {
		t.Fatalf("integration_test phase not listed: %s", got)
	}
}

func TestContains(t *testing.T) {
	if !contains([]string{"a", "b", "c"}, "b") {
		t.Fatal("expected true")
	}
	if contains([]string{"a", "b", "c"}, "d") {
		t.Fatal("expected false")
	}
	if contains(nil, "x") {
		t.Fatal("expected false for nil slice")
	}
}

func TestPhaseListJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "phase", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("phase list --format json failed: %v", err)
	}
	if !strings.Contains(out.String(), `"phases"`) {
		t.Fatalf("expected JSON with phases: %s", out.String())
	}
}

func TestPhaseComplete(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "phase", "complete"})

	if err := root.Execute(); err != nil {
		t.Fatalf("phase complete failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".github", "project-state.yml"))
	if err != nil {
		t.Fatalf("read project-state: %v", err)
	}
	if !strings.Contains(string(data), "current: basic_design") {
		t.Fatalf("phase not advanced in state file:\n%s", string(data))
	}
}

func TestPhaseStart(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "phase", "start", "implementation"})

	if err := root.Execute(); err != nil {
		t.Fatalf("phase start failed: %v", err)
	}

	if !strings.Contains(out.String(), "Phase started: implementation") {
		t.Fatalf("expected Phase started message, got: %s", out.String())
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".github", "project-state.yml"))
	if err != nil {
		t.Fatalf("read project-state: %v", err)
	}
	if !strings.Contains(string(data), "current: implementation") {
		t.Fatalf("phase not updated in state file:\n%s", string(data))
	}
}

func TestPhaseStartInvalid(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "phase", "start", "invalid_phase"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid phase")
	}
	if !strings.Contains(err.Error(), "invalid phase") {
		t.Fatalf("expected invalid phase error, got: %v", err)
	}
}

func TestPhaseStartJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "phase", "start", "basic_design"})

	if err := root.Execute(); err != nil {
		t.Fatalf("phase start --format json failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"status"`) {
		t.Fatalf("expected JSON with status: %s", got)
	}
	if !strings.Contains(got, `"basic_design"`) {
		t.Fatalf("expected phase name in JSON: %s", got)
	}
}

func TestPhaseCompleteJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "phase", "complete"})

	if err := root.Execute(); err != nil {
		t.Fatalf("phase complete --format json failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"status"`) {
		t.Fatalf("expected JSON with status: %s", got)
	}
	if !strings.Contains(got, `"advanced"`) {
		t.Fatalf("expected advanced status in JSON: %s", got)
	}
}

func TestPhaseCompleteAllPhases(t *testing.T) {
	tmp := t.TempDir()
	// integration_test is the last phase
	configPath := setupTestProjectState(t, tmp, "initial_development", "integration_test")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "phase", "complete"})

	if err := root.Execute(); err != nil {
		t.Fatalf("phase complete on last phase failed: %v", err)
	}

	if !strings.Contains(out.String(), "All phases completed.") {
		t.Fatalf("expected All phases completed message, got: %s", out.String())
	}
}

func TestPhaseCompleteAllPhasesJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "integration_test")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "phase", "complete"})

	if err := root.Execute(); err != nil {
		t.Fatalf("phase complete all --format json failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"completed"`) {
		t.Fatalf("expected completed in JSON: %s", got)
	}
}
