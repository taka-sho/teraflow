package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
