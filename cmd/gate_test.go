package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGateApproveSuccess(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, `version: "1"
rbac:
  enabled: false
gate_rules:
  detailed_design:
    conditions:
      - type: document_exists
        value: "docs/design"
      - type: manual_approval
`)
	writeCmdTestFile(t, filepath.Join(tmp, "docs", "design", "spec.md"), "# spec")
	writeCmdTestFile(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "initial_development"
phases:
  current: "detailed_design"
slcp_jcf:
  processes:
    - name: "ソフトウェア設計プロセス"
      status: "in_progress"
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "gate", "approve", "detailed_design"})

	if err := root.Execute(); err != nil {
		t.Fatalf("gate approve failed: %v", err)
	}
	if !strings.Contains(out.String(), "Gate approved: detailed_design") {
		t.Fatalf("unexpected output: %s", out.String())
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".github", "project-state.yml"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !strings.Contains(string(data), "status: completed") {
		t.Fatalf("expected status completed, got:\n%s", string(data))
	}
}

func TestGateApprovePermissionDenied(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, `version: "1"
rbac:
  enabled: true
  roles:
    - name: "dev"
      members: ["another-user"]
      permissions: ["gate.approve.requirements"]
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
	root.SetArgs([]string{"--config", configPath, "gate", "approve", "detailed_design"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected permission denied error")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGateApproveConditionNotMet(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, `version: "1"
rbac:
  enabled: false
gate_rules:
  detailed_design:
    conditions:
      - type: document_exists
        value: "docs/design"
`)
	writeCmdTestFile(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "initial_development"
phases:
  current: "detailed_design"
slcp_jcf:
  processes:
    - name: "ソフトウェア設計プロセス"
      status: "in_progress"
`)

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "gate", "approve", "detailed_design"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected gate condition error")
	}
	if !strings.Contains(err.Error(), "gate conditions not met") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadGateRuleNotFound(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, `version: "1"
gate_rules:
  planning:
    conditions:
      - type: manual_approval
`)

	_, err := loadGateRule(configPath, "testing")
	if err == nil {
		t.Fatal("expected gate rule not found error")
	}
	if !strings.Contains(err.Error(), "gate rule not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadGateRuleValueFallback(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, `version: "1"
gate_rules:
  detailed_design:
    approver_role: "qa"
    conditions:
      - type: document_exists
        path: "docs/design"
      - type: test_coverage
        threshold: 80
`)

	rule, err := loadGateRule(configPath, "detailed_design")
	if err != nil {
		t.Fatalf("loadGateRule failed: %v", err)
	}
	if got := rule.ApproverRole; got != "qa" {
		t.Fatalf("unexpected approver role: %q", got)
	}
	if len(rule.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(rule.Conditions))
	}
	if got := rule.Conditions[0].Value; got != "docs/design" {
		t.Fatalf("unexpected path fallback value: %q", got)
	}
	if got := rule.Conditions[1].Value; got != "80" {
		t.Fatalf("unexpected threshold fallback value: %q", got)
	}
}

func TestGateApproveWithUserFlag(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, `version: "1"
rbac:
  enabled: true
  roles:
    - name: "approver"
      members: ["octocat"]
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
  current: "detailed_design"
slcp_jcf:
  processes:
    - name: "ソフトウェア設計プロセス"
      status: "in_progress"
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{
		"--config", configPath,
		"gate", "approve", "detailed_design",
		"--user", "octocat",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("gate approve with --user failed: %v", err)
	}
	if !strings.Contains(out.String(), "Gate approved: detailed_design") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
