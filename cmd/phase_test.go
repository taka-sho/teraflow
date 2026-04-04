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
