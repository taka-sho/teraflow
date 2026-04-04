package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatusCmd(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "status"})

	if err := root.Execute(); err != nil {
		t.Fatalf("status command failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Stage:  initial_development") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "Phase:  requirements") {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestStatusCmdNoProject(t *testing.T) {
	tmp := t.TempDir()
	configPath := tmp + "/.github/teraflow.yml"

	root := newRootCmd("test")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"--config", configPath, "status"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "E0001: Not a teraflow project.") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "E0001: Not a teraflow project.") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}
