package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInitCreatesExpectedFiles(t *testing.T) {
	tmp := t.TempDir()
	opts := initOptions{Name: "sample-project", Stage: "initial_development", NonInteractive: true}

	if err := runInit(opts, tmp); err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	expected := []string{
		".github/teraflow.yml",
		".github/project-state.yml",
		"docs/shared/01_requirements/index.md",
		"docs/golden-principles.md",
	}

	for _, rel := range expected {
		path := filepath.Join(tmp, rel)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file missing: %s (%v)", rel, err)
		}
	}

	cfg, err := os.ReadFile(filepath.Join(tmp, ".github", "teraflow.yml"))
	if err != nil {
		t.Fatalf("failed reading teraflow.yml: %v", err)
	}
	if !strings.Contains(string(cfg), `name: "sample-project"`) {
		t.Fatalf("teraflow.yml should contain project name")
	}
}

func TestRunInitFailsWhenAlreadyInitialized(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".github"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".github", "teraflow.yml"), []byte("version: \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := runInit(initOptions{Name: "x"}, tmp)
	if err == nil {
		t.Fatal("expected error when project already initialized")
	}
}

func TestInitCommandRequiresNameInNonInteractiveMode(t *testing.T) {
	command := newInitCmd()
	var stderr bytes.Buffer
	command.SetErr(&stderr)
	command.SetOut(&stderr)
	command.SetArgs([]string{"--non-interactive"})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
