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
