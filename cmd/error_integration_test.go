package cmd

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tferrors "github.com/taka-sho/teraflow/internal/errors"
)

func TestIntegrationGateApprovePermissionDeniedAppError(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, `version: "1"
rbac:
  enabled: true
  roles:
    - name: "dev"
      members: ["another-user"]
      permissions: ["gate.approve.detailed_design"]
gate_rules:
  detailed_design:
    conditions:
      - type: manual_approval
`)
	writeCmdTestFile(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "initial_development"
phases:
  current: "requirements"
slcp_jcf:
  processes:
    - name: "ソフトウェア設計プロセス"
      status: "in_progress"
`)

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "gate", "approve", "detailed_design", "--user", "octocat"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected permission denied error")
	}

	var appErr *tferrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.Code != tferrors.CodeTFRB01 {
		t.Fatalf("code=%s, want %s", appErr.Code, tferrors.CodeTFRB01)
	}
	if appErr.ExitCode != 3 {
		t.Fatalf("exit code=%d, want 3", appErr.ExitCode)
	}
}

func TestIntegrationAgentAssignMissingAPIKeyAppError(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, `version: "1"
agent:
  provider: openai
`)
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "agent", "assign", "--type", "review", "check this code"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected missing API key error")
	}

	var appErr *tferrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.Code != tferrors.CodeTFAI01 {
		t.Fatalf("code=%s, want %s", appErr.Code, tferrors.CodeTFAI01)
	}
	if appErr.ExitCode != 4 {
		t.Fatalf("exit code=%d, want 4", appErr.ExitCode)
	}
}

func TestIntegrationJSONErrorOutput(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, `version: "1"
rbac:
  enabled: true
  roles:
    - name: "dev"
      members: ["another-user"]
      permissions: ["gate.approve.detailed_design"]
gate_rules:
  detailed_design:
    conditions:
      - type: manual_approval
`)
	writeCmdTestFile(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "initial_development"
phases:
  current: "requirements"
slcp_jcf:
  processes:
    - name: "ソフトウェア設計プロセス"
      status: "in_progress"
`)

	repoRoot := filepath.Clean(filepath.Join(".."))
	run := exec.Command("go", "run", ".", "--format", "json", "--config", configPath, "gate", "approve", "detailed_design", "--user", "octocat")
	run.Dir = repoRoot

	var stderr bytes.Buffer
	run.Stderr = &stderr
	run.Stdout = &bytes.Buffer{}
	err := run.Run()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	if _, ok := err.(*exec.ExitError); !ok {
		t.Fatalf("expected ExitError, got %T (%v)", err, err)
	}
	got := stderr.String()
	if !strings.Contains(got, `"error_code":"TF-RB01"`) {
		t.Fatalf("expected TF-RB01 JSON output, got: %s", got)
	}
	if !strings.Contains(got, `"exit_code":3`) {
		t.Fatalf("expected exit_code=3 in JSON output, got: %s", got)
	}
}

func TestIntegrationDoctorErrorContainsCode(t *testing.T) {
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"doctor", "error", "TF-RB01"})

	if err := root.Execute(); err != nil {
		t.Fatalf("doctor error failed: %v", err)
	}
	if !strings.Contains(out.String(), "TF-RB01") {
		t.Fatalf("expected TF-RB01 in output, got: %s", out.String())
	}
}
