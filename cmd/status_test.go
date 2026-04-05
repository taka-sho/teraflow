package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/state"
)

func TestStatusCmd(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "status"})

	if err := root.Execute(); err != nil {
		t.Fatalf("status command failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Stage:  initial_development") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "Phase:  requirements") {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestStatusCmdJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "integration_test")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "status"})

	if err := root.Execute(); err != nil {
		t.Fatalf("status --format json failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"stage"`) || !strings.Contains(got, "release") {
		t.Fatalf("unexpected JSON output: %s", got)
	}
}

func TestStatusCmdNoProject(t *testing.T) {
	tmp := t.TempDir()
	configPath := tmp + "/.github/teraflow.yml"

	root := newRootCmd("test")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"--config", configPath, "status"})

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

func TestStatusCmdRolePM(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "development", "design")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "status", "--role", "pm"})

	if err := root.Execute(); err != nil {
		t.Fatalf("status --role pm failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "PM の次のアクション:") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "現在のステージ: development / フェーズ: design") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "今すぐやるべきこと:") {
		t.Fatalf("unexpected output: %s", got)
	}
	if strings.TrimSpace(got) == "" {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestStatusCmdRoleDev(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "development", "design")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "status", "--role", "dev"})

	if err := root.Execute(); err != nil {
		t.Fatalf("status --role dev failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "開発者 の次のアクション:") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "現phase[design]の作業を完了せよ") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "ブロッカー:") {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestStatusCmdRoleQA(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "development", "design")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "status", "--role", "qa"})

	if err := root.Execute(); err != nil {
		t.Fatalf("status --role qa failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "QA の次のアクション:") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "今すぐやるべきこと:") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "次の担当フェーズ:") {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestStatusCmdRoleInvalid(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "development", "design")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "status", "--role", "ops"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid role") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatusCmdRoleAutoFromLocalConfig(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "development", "design")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "user-config.yml"), "user:\n  role: pm\n")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "status", "--role"})

	if err := root.Execute(); err != nil {
		t.Fatalf("status --role(auto) failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "PM の次のアクション:") {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestStatusCmdRoleAutoFromLocalConfigMissing(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "development", "design")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "status", "--role"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "role is not set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatusCmdRoleRejectsPositionalArgs(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "development", "design")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "status", "--role", "pm", "extra"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status does not accept positional arguments") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatusCmdRoleReleaseManager(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "release", "integration_test")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "status", "--role", "release_mgr"})

	if err := root.Execute(); err != nil {
		t.Fatalf("status --role release_mgr failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Release Manager の次のアクション:") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "リリース判定の最終レビュー") {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestMissingRequiredDocs(t *testing.T) {
	root := t.TempDir()

	if got := missingRequiredDocs(root, "要件定義プロセス"); len(got) != 1 || got[0] != "docs/requirements" {
		t.Fatalf("unexpected missing docs for requirements process: %v", got)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "requirements"), 0o755); err != nil {
		t.Fatalf("mkdir docs/requirements: %v", err)
	}
	if got := missingRequiredDocs(root, "要件定義プロセス"); len(got) != 0 {
		t.Fatalf("expected no missing docs after creating requirements, got %v", got)
	}

	if got := missingRequiredDocs(root, "システム設計プロセス"); len(got) != 1 || got[0] != "docs/design" {
		t.Fatalf("unexpected missing docs for design process: %v", got)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "design"), 0o755); err != nil {
		t.Fatalf("mkdir docs/design: %v", err)
	}
	if got := missingRequiredDocs(root, "システム設計プロセス"); len(got) != 0 {
		t.Fatalf("expected no missing docs after creating design docs, got %v", got)
	}

	missing := missingRequiredDocs(root, "ソフトウェアテストプロセス")
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing docs for testing process, got %v", missing)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "testing"), 0o755); err != nil {
		t.Fatalf("mkdir docs/testing: %v", err)
	}
	mustWrite(t, filepath.Join(root, "docs", "test-plan.md"), "# test plan\n")
	if got := missingRequiredDocs(root, "ソフトウェアテストプロセス"); len(got) != 0 {
		t.Fatalf("expected no missing docs for testing process, got %v", got)
	}

	if got := missingRequiredDocs(root, "unknown"); got != nil {
		t.Fatalf("expected nil for unknown process, got %v", got)
	}
}

func TestUnresolvedReworkCount(t *testing.T) {
	tmp := t.TempDir()
	configPath := setupTestProjectState(t, tmp, "development", "implementation")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "rework-log.yml"), `reworks:
  - id: rw-1
    group: qa
    target_phase: testing
    reason: open item
    created_at: "2026-04-06T00:00:00Z"
    status: open
  - id: rw-2
    group: qa
    target_phase: testing
    reason: approved item
    created_at: "2026-04-06T00:00:00Z"
    status: approved
  - id: rw-3
    group: qa
    target_phase: testing
    reason: rejected item
    created_at: "2026-04-06T00:00:00Z"
    status: rejected
  - id: rw-4
    group: qa
    target_phase: testing
    reason: pending item
    created_at: "2026-04-06T00:00:00Z"
    status: pending
`)

	count, err := unresolvedReworkCount(configPath)
	if err != nil {
		t.Fatalf("unresolvedReworkCount returned error: %v", err)
	}
	if count != 2 {
		t.Fatalf("unexpected unresolved rework count: got %d want 2", count)
	}
}

func TestSLCPProcessStatus(t *testing.T) {
	s := &state.ProjectState{
		SLCPJCF: state.SLCPJCFState{
			Processes: []state.SLCPJCFProcess{
				{Name: "企画プロセス", Status: ""},
				{Name: "要件定義プロセス", Status: "in_progress"},
			},
		},
	}

	if got := slcpProcessStatus(s, -1); got != "not_started" {
		t.Fatalf("unexpected status for negative index: %s", got)
	}
	if got := slcpProcessStatus(s, 0); got != "not_started" {
		t.Fatalf("unexpected status for empty status: %s", got)
	}
	if got := slcpProcessStatus(s, 1); got != "in_progress" {
		t.Fatalf("unexpected status for in-progress process: %s", got)
	}
	if got := slcpProcessStatus(s, 3); got != "not_started" {
		t.Fatalf("unexpected status for out-of-range index: %s", got)
	}
}

func TestNextSLCPProcess(t *testing.T) {
	s := &state.ProjectState{
		SLCPJCF: state.SLCPJCFState{
			Processes: []state.SLCPJCFProcess{
				{Name: "企画プロセス", Status: "completed"},
				{Name: "要件定義プロセス", Status: ""},
			},
		},
	}

	name, status := nextSLCPProcess(s, 0)
	if name != "要件定義プロセス" || status != "not_started" {
		t.Fatalf("unexpected next process: %s/%s", name, status)
	}

	name, status = nextSLCPProcess(s, 1)
	if name != "" || status != "" {
		t.Fatalf("expected empty next process at tail, got %s/%s", name, status)
	}
}

func TestQAProgress(t *testing.T) {
	processes := state.DefaultSLCPJCFProcesses()
	processes[findSLCPIndexByName("ソフトウェアテストプロセス")].Status = "completed"
	processes[findSLCPIndexByName("システム結合テスト")].Status = "in_progress"

	s := &state.ProjectState{
		SLCPJCF: state.SLCPJCFState{
			Processes: processes,
		},
	}

	done, total := qaProgress(s)
	if done != 1 || total != 2 {
		t.Fatalf("unexpected qa progress: %d/%d", done, total)
	}
}
