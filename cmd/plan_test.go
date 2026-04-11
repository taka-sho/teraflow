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
