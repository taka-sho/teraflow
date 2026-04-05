package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/state"
)

func TestProcessListCmd(t *testing.T) {
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

func TestProcessListCmdJSON(t *testing.T) {
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
	if !strings.Contains(got, `"current_process"`) || !strings.Contains(got, "システム結合テスト") {
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

func TestProcessStartAndComplete(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "process", "start", "要件定義"})
	if err := root.Execute(); err != nil {
		t.Fatalf("process start failed: %v", err)
	}

	s, err := state.LoadState(configPath)
	if err != nil {
		t.Fatalf("load state after start: %v", err)
	}
	idx := findSLCPIndexByName("要件定義プロセス")
	if got := s.SLCPJCF.Processes[idx].Status; got != "in_progress" {
		t.Fatalf("expected in_progress, got %s", got)
	}
	if s.SLCPJCF.Processes[idx].StartedAt == "" {
		t.Fatal("expected started_at to be set")
	}
	if s.SLCPJCF.CurrentProcess != "要件定義プロセス" {
		t.Fatalf("unexpected current process: %s", s.SLCPJCF.CurrentProcess)
	}

	root = newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "process", "complete", "要件定義プロセス"})
	if err := root.Execute(); err != nil {
		t.Fatalf("process complete failed: %v", err)
	}

	s, err = state.LoadState(configPath)
	if err != nil {
		t.Fatalf("load state after complete: %v", err)
	}
	if got := s.SLCPJCF.Processes[idx].Status; got != "completed" {
		t.Fatalf("expected completed, got %s", got)
	}
	if s.SLCPJCF.Processes[idx].CompletedAt == "" {
		t.Fatal("expected completed_at to be set")
	}
	if s.SLCPJCF.CurrentProcess != "システム設計プロセス" {
		t.Fatalf("unexpected next current process: %s", s.SLCPJCF.CurrentProcess)
	}
}

func TestProcessStartPermissionDenied(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")
	writeCmdTestFile(t, configPath, `
version: "1"
rbac:
  enabled: true
  roles:
    - name: blocked-role
      members:
        - no-such-user
      permissions:
        - process.start.*
`)

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "process", "start", "planning"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected permission denied error")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSLCPJCFLabelsMatchDefaultStateNames(t *testing.T) {
	defaults := state.DefaultSLCPJCFProcesses()
	if len(defaults) != len(slcpJCFProcessLabels) {
		t.Fatalf("mismatched process length: defaults=%d labels=%d", len(defaults), len(slcpJCFProcessLabels))
	}
	for i, p := range defaults {
		if p.Name != slcpJCFProcessLabels[i] {
			t.Fatalf("label mismatch at %d: default=%s label=%s", i, p.Name, slcpJCFProcessLabels[i])
		}
	}
}
