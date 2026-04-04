package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/state"
)

func TestReworkCreate(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	command := newRootCmd("test")
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"rework", "create", "--group", "group-auth", "--target-phase", "basic_design", "--reason", "API仕様変更", "--config", cfgPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("rework create execute error: %v", err)
	}

	logPath := filepath.Join(tmp, ".teraflow", "rework-log.yml")
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("rework log should exist: %v", err)
	}

	log, err := state.LoadReworkLog(cfgPath)
	if err != nil {
		t.Fatalf("LoadReworkLog error: %v", err)
	}
	if len(log.Reworks) != 1 {
		t.Fatalf("expected 1 rework, got %d", len(log.Reworks))
	}
	if log.Reworks[0].ID != "rw-001" {
		t.Fatalf("unexpected rework id: %s", log.Reworks[0].ID)
	}
	if log.Reworks[0].Status != "open" {
		t.Fatalf("unexpected status: %s", log.Reworks[0].Status)
	}
}

func TestReworkCreateMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		errMsg  string
	}{
		{
			name:   "missing group",
			args:   []string{"rework", "create", "--target-phase", "basic_design", "--reason", "reason"},
			errMsg: "--group is required",
		},
		{
			name:   "missing target-phase",
			args:   []string{"rework", "create", "--group", "group-a", "--reason", "reason"},
			errMsg: "--target-phase is required",
		},
		{
			name:   "missing reason",
			args:   []string{"rework", "create", "--group", "group-a", "--target-phase", "basic_design"},
			errMsg: "--reason is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
			mustWrite(t, cfgPath, "version: \"1\"\n")

			command := newRootCmd("test")
			var out bytes.Buffer
			command.SetOut(&out)
			command.SetErr(&out)
			command.SetArgs(append(tc.args, "--config", cfgPath))

			err := command.Execute()
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.errMsg) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestReworkList(t *testing.T) {
	t.Run("no entries", func(t *testing.T) {
		tmp := t.TempDir()
		cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
		mustWrite(t, cfgPath, "version: \"1\"\n")

		command := newRootCmd("test")
		var out bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&out)
		command.SetArgs([]string{"rework", "list", "--config", cfgPath})

		if err := command.Execute(); err != nil {
			t.Fatalf("rework list execute error: %v", err)
		}
		if !strings.Contains(out.String(), "No reworks recorded.") {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})

	t.Run("with entries", func(t *testing.T) {
		tmp := t.TempDir()
		cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
		mustWrite(t, cfgPath, "version: \"1\"\n")
		if err := state.AppendRework(cfgPath, state.ReworkEntry{
			ID:          "rw-001",
			Group:       "group-auth",
			TargetPhase: "basic_design",
			Reason:      "API仕様変更",
			CreatedAt:   "2026-01-01T00:00:00",
			Status:      "open",
		}); err != nil {
			t.Fatalf("AppendRework error: %v", err)
		}

		command := newRootCmd("test")
		var out bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&out)
		command.SetArgs([]string{"rework", "list", "--config", cfgPath})

		if err := command.Execute(); err != nil {
			t.Fatalf("rework list execute error: %v", err)
		}
		if !strings.Contains(out.String(), "rw-001") || !strings.Contains(out.String(), "group-auth") {
			t.Fatalf("expected row in output, got:\n%s", out.String())
		}
	})
}
