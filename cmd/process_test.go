package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestProcessCmd(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "detailed_design")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "process"})

	if err := root.Execute(); err != nil {
		t.Fatalf("process command failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "SLCP-JCF プロセス状況") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "現在のプロセス: ソフトウェア設計プロセス") {
		t.Fatalf("unexpected current process output: %s", got)
	}
	if !strings.Contains(got, "Stage: initial_development") {
		t.Fatalf("unexpected stage output: %s", got)
	}
	if !strings.Contains(got, "Phase: detailed_design") {
		t.Fatalf("unexpected phase output: %s", got)
	}
}

func TestProcessCmdJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "integration_test")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "process"})

	if err := root.Execute(); err != nil {
		t.Fatalf("process --format json failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"current_process"`) || !strings.Contains(got, "リリース") {
		t.Fatalf("unexpected JSON output: %s", got)
	}
}

func TestProcessCmdNoProject(t *testing.T) {
	tmp := t.TempDir()
	configPath := tmp + "/.github/teraflow.yml"

	root := newRootCmd("test")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"--config", configPath, "process"})

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
