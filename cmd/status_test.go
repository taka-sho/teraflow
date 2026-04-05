package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
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
	if !strings.Contains(got, "設計レビューの承認") {
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
	if !strings.Contains(got, "詳細設計書の作成") {
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
	if !strings.Contains(got, "テスト計画書の作成開始") {
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
