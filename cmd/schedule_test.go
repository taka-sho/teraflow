package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestScheduleShow_noFile(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "schedule", "show"})

	if err := root.Execute(); err != nil {
		t.Fatalf("schedule show failed: %v", err)
	}

	if !strings.Contains(out.String(), "No schedule defined yet. Run `teraflow init` to create one.") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestScheduleUpdate(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "implementation")

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "schedule", "update"})

	if err := root.Execute(); err != nil {
		t.Fatalf("schedule update failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".github", "master-schedule.yml"))
	if err != nil {
		t.Fatalf("read master schedule: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "current: release") {
		t.Fatalf("current stage not updated:\n%s", got)
	}
	if !strings.Contains(got, "current_phase: implementation") {
		t.Fatalf("current phase not updated:\n%s", got)
	}
	if !strings.Contains(got, "updated_at:") {
		t.Fatalf("updated_at not found:\n%s", got)
	}
}

func TestScheduleUpdateNoProject(t *testing.T) {
	tmp := t.TempDir()
	// Config exists but no project-state.yml
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "schedule", "update"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for no project state")
	}
	if !strings.Contains(err.Error(), notProjectError) {
		t.Fatalf("expected notProjectError, got: %v", err)
	}
}

func TestScheduleShowWithFile(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "implementation")

	// pre-create a schedule file
	schedulePath := filepath.Join(tmp, ".github", "master-schedule.yml")
	scheduleContent := "version: \"1\"\nupdated_at: \"2026-04-05T00:00:00\"\nstages:\n  current: release\n  current_phase: implementation\n"
	writeCmdTestFile(t, schedulePath, scheduleContent)

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "schedule", "show"})

	if err := root.Execute(); err != nil {
		t.Fatalf("schedule show with file failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "version") {
		t.Fatalf("expected schedule content in output: %s", got)
	}
}

func TestScheduleUpdateInvalidSchedule(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "operation", "testing")

	// Create an invalid master-schedule.yml
	schedulePath := filepath.Join(tmp, ".github", "master-schedule.yml")
	writeCmdTestFile(t, schedulePath, ":\n  bad: [\nbroken yaml\n")

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "schedule", "update"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid schedule YAML")
	}
	if !strings.Contains(err.Error(), "parse master schedule") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleShowJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "implementation")

	schedulePath := filepath.Join(tmp, ".github", "master-schedule.yml")
	scheduleContent := "version: \"1\"\nupdated_at: \"2026-04-05T00:00:00\"\nstages:\n  current: release\n  current_phase: implementation\n"
	writeCmdTestFile(t, schedulePath, scheduleContent)

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "schedule", "show"})

	if err := root.Execute(); err != nil {
		t.Fatalf("schedule show --format json failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"version"`) {
		t.Fatalf("expected JSON with version: %s", got)
	}
}

func TestScheduleUpdateJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "operation", "testing")

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "schedule", "update"})

	if err := root.Execute(); err != nil {
		t.Fatalf("schedule update --format json failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"updated"`) {
		t.Fatalf("expected updated status in JSON: %s", got)
	}
	if !strings.Contains(got, `"operation"`) {
		t.Fatalf("expected stage in JSON: %s", got)
	}
}

func TestScheduleUpdateWithScheduleReadError(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "operation", "testing")
	if err := os.MkdirAll(filepath.Join(tmp, ".github", "master-schedule.yml"), 0o755); err != nil {
		t.Fatalf("mkdir schedule path: %v", err)
	}

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())
	root.SetArgs([]string{"--config", configPath, "schedule", "update"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected read error for directory schedule path")
	}
	if !strings.Contains(err.Error(), "read master schedule") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleUpdateSetsDefaultVersion(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "implementation")
	writeCmdTestFile(t, filepath.Join(tmp, ".github", "master-schedule.yml"), "updated_at: \"2026-04-05T00:00:00\"\nstages:\n  current: release\n  current_phase: implementation\n")

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())
	root.SetArgs([]string{"--config", configPath, "schedule", "update"})

	if err := root.Execute(); err != nil {
		t.Fatalf("schedule update failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".github", "master-schedule.yml"))
	if err != nil {
		t.Fatalf("read schedule: %v", err)
	}
	if !strings.Contains(string(data), "version: \"1\"") {
		t.Fatalf("expected default version to be set, got:\n%s", string(data))
	}
}

func TestScheduleShowReadError(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "implementation")
	if err := os.MkdirAll(filepath.Join(tmp, ".github", "master-schedule.yml"), 0o755); err != nil {
		t.Fatalf("mkdir schedule path: %v", err)
	}

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())
	root.SetArgs([]string{"--config", configPath, "schedule", "show"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected read error for directory schedule path")
	}
	if !strings.Contains(err.Error(), "read master schedule") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleShowWriteError(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "implementation")
	writeCmdTestFile(t, filepath.Join(tmp, ".github", "master-schedule.yml"), "version: \"1\"\n")

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())
	root.SetOut(failingWriter{})
	root.SetErr(failingWriter{})
	root.SetArgs([]string{"--config", configPath, "schedule", "show"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected write error")
	}
	if !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleShowJSONInvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "implementation")
	writeCmdTestFile(t, filepath.Join(tmp, ".github", "master-schedule.yml"), ":\n  bad: [\ninvalid")

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())
	root.SetArgs([]string{"--config", configPath, "--format", "json", "schedule", "show"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid schedule YAML with json output")
	}
	if !strings.Contains(err.Error(), "parse master schedule") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleShowAddsTrailingNewline(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "implementation")
	writeCmdTestFile(t, filepath.Join(tmp, ".github", "master-schedule.yml"), "version: \"1\"")

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "schedule", "show"})

	if err := root.Execute(); err != nil {
		t.Fatalf("schedule show failed: %v", err)
	}
	got := out.String()
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("expected trailing newline, got: %q", got)
	}
}

func TestScheduleUpdateInvalidFormat(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "operation", "testing")

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())
	root.SetArgs([]string{"--config", configPath, "--format", "xml", "schedule", "update"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleUpdateInvalidStateYAML(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, "version: \"1\"\n")
	writeCmdTestFile(t, filepath.Join(tmp, ".github", "project-state.yml"), ":\n broken: [\n")

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())
	root.SetArgs([]string{"--config", configPath, "schedule", "update"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid project-state.yml")
	}
	if !strings.Contains(err.Error(), "parse project state") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleUpdateWriteError(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "operation", "testing")
	schedulePath := filepath.Join(tmp, ".github", "master-schedule.yml")
	writeCmdTestFile(t, schedulePath, "version: \"1\"\n")
	if err := os.Chmod(schedulePath, 0o444); err != nil {
		t.Fatalf("chmod schedule file: %v", err)
	}

	root := newRootCmd("test")
	root.AddCommand(newScheduleCmd())
	root.SetArgs([]string{"--config", configPath, "schedule", "update"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected write error")
	}
	if !strings.Contains(err.Error(), "write master schedule") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleUpdateConfigFlagError(t *testing.T) {
	cmd := newScheduleUpdateCmd()
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected config flag error")
	}
	if !strings.Contains(err.Error(), "flag accessed but not defined: config") {
		t.Fatalf("unexpected error: %v", err)
	}
}
