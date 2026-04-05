package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/state"
)

func TestInitCmdNonInteractiveSuccess(t *testing.T) {
	tmp := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	command := newInitCmd()
	var buf bytes.Buffer
	command.SetOut(&buf)
	command.SetErr(&buf)
	command.SetArgs([]string{"--non-interactive", "--name", "test-project-cmd"})

	if err := command.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, ".github", "teraflow.yml")); err != nil {
		t.Fatal("teraflow.yml not created")
	}
}

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

	loaded, err := state.LoadState(filepath.Join(tmp, ".github", "teraflow.yml"))
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}
	if len(loaded.SLCPJCF.Processes) != 8 {
		t.Fatalf("expected 8 default SLCP-JCF processes, got %d", len(loaded.SLCPJCF.Processes))
	}
}

func TestRunInitDifferentStage(t *testing.T) {
	tmp := t.TempDir()
	opts := initOptions{Name: "sample-project", Stage: "release", NonInteractive: true}

	if err := runInit(opts, tmp); err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	statePath := filepath.Join(tmp, ".github", "project-state.yml")
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("failed reading project-state.yml: %v", err)
	}
	if !strings.Contains(string(data), `current_stage: release`) {
		t.Fatalf("project-state.yml should contain release stage, got:\n%s", string(data))
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

func TestRunInitDefaultsNameAndStage(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "demo-project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	if err := runInit(initOptions{}, projectDir); err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	cfgData, err := os.ReadFile(filepath.Join(projectDir, ".github", "teraflow.yml"))
	if err != nil {
		t.Fatalf("read teraflow.yml: %v", err)
	}
	if !strings.Contains(string(cfgData), `name: "demo-project"`) {
		t.Fatalf("expected default name from cwd, got:\n%s", string(cfgData))
	}

	stateData, err := os.ReadFile(filepath.Join(projectDir, ".github", "project-state.yml"))
	if err != nil {
		t.Fatalf("read project-state.yml: %v", err)
	}
	if !strings.Contains(string(stateData), `current_stage: initial_development`) {
		t.Fatalf("expected default stage, got:\n%s", string(stateData))
	}
}

func TestRunInitCreatesMissingDirectory(t *testing.T) {
	tmp := t.TempDir()
	missingDir := filepath.Join(tmp, "new-project")

	if err := runInit(initOptions{Name: "sample-project"}, missingDir); err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(missingDir, ".github", "teraflow.yml")); err != nil {
		t.Fatalf("expected teraflow.yml in created directory: %v", err)
	}
}

func TestRunInitFailsWhenCreateDirFails(t *testing.T) {
	tmp := t.TempDir()
	blockingFile := filepath.Join(tmp, "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("create blocking file: %v", err)
	}

	err := runInit(initOptions{Name: "x"}, blockingFile)
	if err == nil {
		t.Fatal("expected error when cwd path is not a directory")
	}
}
