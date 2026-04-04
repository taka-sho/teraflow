package cmd

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLabelList(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.AddCommand(newLabelCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"label", "list", "--config", cfgPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("label list failed: %v", err)
	}

	got := out.String()
	checks := []string{
		"Default teraflow labels:",
		"stage:initial-dev",
		"stage:release",
		"phase:requirements",
		"phase:basic-design",
		"phase:detailed-design",
		"phase:implementation",
		"phase:testing",
		"phase:integration",
		"type:rework",
		"type:incident",
		"type:confirmed",
	}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestLabelListCustom(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, cfgPath, "version: \"1\"\n")

	// Create a labels config file at the configPath (teraflow.yml location)
	labelsConfigPath := filepath.Join(tmp, ".teraflow", "labels.yml")
	writeCmdTestFile(t, labelsConfigPath, `labels:
  - name: "custom-label"
    color: "FF0000"
    description: "A custom label"
`)

	// loadLabels takes configPath, not a separate labels path
	// Use configPath to point to labels.yml directly
	labels, fromConfig, err := loadLabels(labelsConfigPath)
	if err != nil {
		t.Fatalf("loadLabels failed: %v", err)
	}
	if !fromConfig {
		t.Fatal("expected fromConfig=true for custom labels")
	}
	if len(labels) != 1 || labels[0].Name != "custom-label" {
		t.Fatalf("unexpected labels: %+v", labels)
	}
}

func TestLabelListInvalidConfig(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "invalid.yml")
	writeCmdTestFile(t, cfgPath, ":\n  bad: [\ninvalid yaml\n")

	_, _, err := loadLabels(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "E0003") {
		t.Fatalf("expected E0003 error, got: %v", err)
	}
}

func TestLabelListCustomLabelInLabelListCmd(t *testing.T) {
	tmp := t.TempDir()
	// Create labels.yml at configPath location so loadLabels finds it
	labelsYml := filepath.Join(tmp, "labels.yml")
	writeCmdTestFile(t, labelsYml, `labels:
  - name: "my-label"
    color: "00FF00"
    description: "My custom label"
`)

	root := newRootCmd("test")
	root.AddCommand(newLabelCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	// Use --config pointing to labels.yml so loadLabels finds it
	root.SetArgs([]string{"label", "list", "--config", labelsYml})

	if err := root.Execute(); err != nil {
		t.Fatalf("label list with custom config failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Configured teraflow labels:") {
		t.Fatalf("expected Configured label header, got:\n%s", got)
	}
	if !strings.Contains(got, "my-label") {
		t.Fatalf("expected my-label in output, got:\n%s", got)
	}
}

func TestLabelSyncDryRun(t *testing.T) {
	oldLookPath := ghLookPath
	oldExecCommand := ghExecCommand

	ghLookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 0")
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
		ghExecCommand = oldExecCommand
	})

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.AddCommand(newLabelCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"label", "sync", "--dry-run", "--config", cfgPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("label sync --dry-run failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "[dry-run]") {
		t.Fatalf("expected dry-run output, got:\n%s", got)
	}
}

func TestLabelSyncDryRunForce(t *testing.T) {
	oldLookPath := ghLookPath
	oldExecCommand := ghExecCommand

	ghLookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 0")
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
		ghExecCommand = oldExecCommand
	})

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.AddCommand(newLabelCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"label", "sync", "--dry-run", "--force", "--config", cfgPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("label sync --dry-run --force failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "[dry-run] gh label edit") {
		t.Fatalf("expected dry-run edit output, got:\n%s", got)
	}
}

func TestRunGHCommandSuccess(t *testing.T) {
	oldExecCommand := ghExecCommand
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 0")
	}
	t.Cleanup(func() { ghExecCommand = oldExecCommand })

	err := runGHCommand("test", "arg")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestCreateOrUpdateLabelSuccess(t *testing.T) {
	oldExecCommand := ghExecCommand
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 0")
	}
	t.Cleanup(func() { ghExecCommand = oldExecCommand })

	label := labelDefinition{Name: "test-label", Color: "FF0000", Description: "Test"}
	err := createOrUpdateLabel(label, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestCreateOrUpdateLabelAlreadyExists(t *testing.T) {
	oldExecCommand := ghExecCommand
	call := 0
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		call++
		if call == 1 {
			return exec.Command("sh", "-c", "echo 'already exists' >&2; exit 1")
		}
		return exec.Command("sh", "-c", "exit 0")
	}
	t.Cleanup(func() { ghExecCommand = oldExecCommand })

	label := labelDefinition{Name: "test-label", Color: "FF0000", Description: "Test"}
	err := createOrUpdateLabel(label, true) // force=true to trigger edit
	if err != nil {
		t.Fatalf("expected no error for already-exists+force, got: %v", err)
	}
}

func TestCreateOrUpdateLabelCreateFails(t *testing.T) {
	oldExecCommand := ghExecCommand
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 1")
	}
	t.Cleanup(func() { ghExecCommand = oldExecCommand })

	label := labelDefinition{Name: "test-label", Color: "FF0000", Description: "Test"}
	err := createOrUpdateLabel(label, false)
	if err == nil {
		t.Fatal("expected error when create fails")
	}
	if !strings.Contains(err.Error(), "E5003") {
		t.Fatalf("expected E5003 error, got: %v", err)
	}
}

func TestEnsureGHAuthenticatedNotAuthenticated(t *testing.T) {
	oldLookPath := ghLookPath
	oldExecCommand := ghExecCommand

	ghLookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 1")
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
		ghExecCommand = oldExecCommand
	})

	err := ensureGHAuthenticated()
	if err == nil {
		t.Fatal("expected error for unauthenticated gh")
	}
	if !strings.Contains(err.Error(), "E5002") {
		t.Fatalf("expected E5002 error, got: %v", err)
	}
}

func TestEnsureGHAuthenticatedSuccess(t *testing.T) {
	oldLookPath := ghLookPath
	oldExecCommand := ghExecCommand

	ghLookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 0")
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
		ghExecCommand = oldExecCommand
	})

	err := ensureGHAuthenticated()
	if err != nil {
		t.Fatalf("expected no error for authenticated gh, got: %v", err)
	}
}

func TestLabelSyncNoGH(t *testing.T) {
	oldLookPath := ghLookPath
	ghLookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
	})

	root := newRootCmd("test")
	root.AddCommand(newLabelCmd())
	root.SetArgs([]string{"label", "sync"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "E5001") {
		t.Fatalf("expected E5001, got: %v", err)
	}
}
