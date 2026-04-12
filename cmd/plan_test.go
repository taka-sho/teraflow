package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanWaves(t *testing.T) {
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"plan", "waves", "--phase", "basic-design"})
	if err := root.Execute(); err != nil {
		t.Fatalf("plan waves failed: %v", err)
	}
	if !strings.Contains(out.String(), "Wave 1") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestPlanPhasesStatus(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), "project:\n  name: test\nlifecycle:\n  current_stage: initial_development\nphases:\n  current: basic-design\n")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "plan", "phases"})
	if err := root.Execute(); err != nil {
		t.Fatalf("plan phases failed: %v", err)
	}
	if !strings.Contains(out.String(), "Current phase: basic-design") {
		t.Fatalf("unexpected phases output: %s", out.String())
	}

	out.Reset()
	root = newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "plan", "status"})
	if err := root.Execute(); err != nil {
		t.Fatalf("plan status failed: %v", err)
	}
	if !strings.Contains(out.String(), "\"current_phase\":\"basic-design\"") {
		t.Fatalf("unexpected status output: %s", out.String())
	}
}

func TestPlanWavesErrorsAndJSON(t *testing.T) {
	root := newRootCmd("test")
	root.SetArgs([]string{"plan", "waves"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "--phase is required") {
		t.Fatalf("expected missing phase error, got: %v", err)
	}

	root = newRootCmd("test")
	root.SetArgs([]string{"--format", "json", "plan", "waves", "--phase", "unknown-phase"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "no wave definitions") {
		t.Fatalf("expected unknown phase error, got: %v", err)
	}

	var out bytes.Buffer
	root = newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--format", "json", "plan", "waves", "--phase", "basic-design"})
	if err := root.Execute(); err != nil {
		t.Fatalf("plan waves json failed: %v", err)
	}
	if !strings.Contains(out.String(), "\"phase\":\"basic-design\"") || !strings.Contains(out.String(), "\"waves\"") {
		t.Fatalf("unexpected plan waves json: %s", out.String())
	}
}

func TestPlanPhasesStatusMissingState(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "plan", "phases"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected missing state error")
	}

	root = newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "plan", "status"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected missing state error")
	}
}
