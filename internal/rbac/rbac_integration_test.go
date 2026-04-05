package rbac_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/rbac"
)

func TestRBACIntegrationConfigLoadAndAdminCheck(t *testing.T) {
	bin := buildTeraflowBinary(t)
	projectDir, cfgPath := setupRBACProject(t, bin)
	_ = projectDir

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.RBAC.Enabled {
		t.Fatal("expected rbac.enabled=true")
	}
	if cfg.RBAC.AdminRole != "admin" {
		t.Fatalf("unexpected admin role: %q", cfg.RBAC.AdminRole)
	}

	engine := rbac.NewEngine(cfg.RBAC)
	if !engine.IsAdmin("alice") {
		t.Fatal("expected alice to be admin")
	}
	if engine.IsAdmin("bob") {
		t.Fatal("expected bob to be non-admin")
	}
}

func TestRBACIntegrationListAndCheckCommands(t *testing.T) {
	bin := buildTeraflowBinary(t)
	projectDir, cfgPath := setupRBACProject(t, bin)

	listOut, err := runCLI(bin, projectDir, "rbac", "list", "--config", cfgPath)
	if err != nil {
		t.Fatalf("rbac list failed: %v\n%s", err, listOut)
	}
	if !strings.Contains(listOut, "admin ★ (admin_role)") {
		t.Fatalf("expected admin marker in rbac list output, got:\n%s", listOut)
	}

	checkOut, err := runCLI(bin, projectDir,
		"rbac", "check",
		"--config", cfgPath,
		"--user", "alice",
		"--action", "gate.approve.planning",
		"--format", "json",
	)
	if err != nil {
		t.Fatalf("rbac check failed: %v\n%s", err, checkOut)
	}
	if !strings.Contains(checkOut, `"allowed":true`) {
		t.Fatalf("expected allowed=true json, got:\n%s", checkOut)
	}
}

func TestRBACIntegrationUserFlagOnGateStagePhase(t *testing.T) {
	bin := buildTeraflowBinary(t)
	projectDir, cfgPath := setupRBACProject(t, bin)

	gateOut, err := runCLI(bin, projectDir,
		"gate", "approve", "planning",
		"--config", cfgPath,
		"--user", "alice",
	)
	if err != nil {
		t.Fatalf("gate approve failed: %v\n%s", err, gateOut)
	}
	if !strings.Contains(gateOut, "Gate approved: planning") {
		t.Fatalf("unexpected gate output:\n%s", gateOut)
	}

	stageOut, err := runCLI(bin, projectDir,
		"stage", "advance",
		"--config", cfgPath,
		"--yes",
		"--force",
		"--reason", "integration test",
		"--user", "alice",
	)
	if err != nil {
		t.Fatalf("stage advance failed: %v\n%s", err, stageOut)
	}
	if !strings.Contains(stageOut, "Advanced: release") {
		t.Fatalf("unexpected stage output:\n%s", stageOut)
	}

	phaseOut, err := runCLI(bin, projectDir,
		"phase", "complete",
		"--config", cfgPath,
		"--force",
		"--reason", "integration test",
		"--user", "alice",
	)
	if err != nil {
		t.Fatalf("phase complete failed: %v\n%s", err, phaseOut)
	}
	if !strings.Contains(phaseOut, "Phase advanced: requirements -> basic_design") {
		t.Fatalf("unexpected phase output:\n%s", phaseOut)
	}
}

func TestRBACIntegrationApplyFromIssueFlagExists(t *testing.T) {
	bin := buildTeraflowBinary(t)
	projectDir, cfgPath := setupRBACProject(t, bin)
	_ = cfgPath

	helpOut, err := runCLI(bin, projectDir, "rbac", "apply", "--help")
	if err != nil {
		t.Fatalf("rbac apply --help failed: %v\n%s", err, helpOut)
	}
	if !strings.Contains(helpOut, "--from-issue") {
		t.Fatalf("expected --from-issue flag in help output, got:\n%s", helpOut)
	}
}

func buildTeraflowBinary(t *testing.T) string {
	t.Helper()
	repoRoot := testRepoRoot(t)
	binPath := filepath.Join(t.TempDir(), "teraflow")

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=auto")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, string(out))
	}
	return binPath
}

func setupRBACProject(t *testing.T, bin string) (string, string) {
	t.Helper()

	projectDir := t.TempDir()
	if out, err := runCLI(bin, testRepoRoot(t), "init", "--path", projectDir, "--name", "rbac-integration", "--non-interactive"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	cfgPath := filepath.Join(projectDir, ".github", "teraflow.yml")
	cfg := `version: "1"

project:
  name: "rbac-integration"
  description: ""
  repository: ""

confirmation:
  trigger: "確定"
  req_trigger: "要求確定"

ai:
  default_provider: anthropic

harness:
  score_threshold: 70
  auto_issue: false

rbac:
  enabled: true
  github_enforcement: true
  admin_role: "admin"
  roles:
    - name: admin
      members: ["alice"]
      permissions: ["*"]
    - name: dev
      members: ["bob"]
      permissions: ["gate.approve.planning"]

gate_rules:
  planning:
    approver_role: admin
    conditions:
      - type: manual_approval

constraints:
  - type: required_docs
    value: docs/required.md
    severity: block
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "docs", "required.md"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write required doc: %v", err)
	}

	return projectDir, cfgPath
}

func runCLI(bin, dir string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=auto")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
