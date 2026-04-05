package cmd

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/config"
)

func TestRbacListShowsAdminMark(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, cfgPath, `
version: "1"
rbac:
  enabled: true
  admin_role: "admin"
  roles:
    - name: admin
      members: ["taka-sho"]
      permissions: ["*"]
    - name: pm
      members: ["pm-user1"]
      permissions: ["gate.approve.*", "stage.advance", "rbac.apply"]
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"rbac", "list", "--config", cfgPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("rbac list failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "admin ★ (admin_role)") {
		t.Fatalf("expected admin marker, got:\n%s", got)
	}
	if !strings.Contains(got, "permissions: *") {
		t.Fatalf("expected permissions output, got:\n%s", got)
	}
}

func TestRbacCheckAllowAndDeny(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, cfgPath, `
version: "1"
rbac:
  enabled: true
  roles:
    - name: admin
      members: ["alice"]
      permissions: ["*"]
    - name: pm
      members: ["bob"]
      permissions: ["gate.approve.*"]
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"rbac", "check", "--config", cfgPath, "--user", "alice", "--action", "gate.approve.planning", "--format", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("rbac check allow failed: %v", err)
	}
	if !strings.Contains(out.String(), `"allowed":true`) {
		t.Fatalf("expected allowed=true json, got:\n%s", out.String())
	}

	root = newRootCmd("test")
	out.Reset()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"rbac", "check", "--config", cfgPath, "--user", "bob", "--action", "rbac.apply"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected deny error")
	}
}

func TestRbacCheckRequiresUserWhenGitHubEnforcementEnabled(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, cfgPath, `
version: "1"
rbac:
  enabled: true
  github_enforcement: true
  roles:
    - name: admin
      members: ["alice"]
      permissions: ["*"]
`)

	root := newRootCmd("test")
	root.SetArgs([]string{"rbac", "check", "--config", cfgPath, "--action", "gate.approve.planning"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --user is missing under github_enforcement")
	}
	if !strings.Contains(err.Error(), "--user flag is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRbacApplyAddChangeRemoveAndLastAdminProtection(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, cfgPath, `
version: "1"
rbac:
  enabled: true
  admin_role: "admin"
  roles:
    - name: admin
      members: ["admin-user"]
      permissions: ["*"]
    - name: pm
      members: []
      permissions: ["rbac.apply"]
    - name: qa
      members: []
      permissions: []
`)

	oldExec := rbacExecCommand
	oldLookPath := rbacLookPath
	t.Cleanup(func() {
		rbacExecCommand = oldExec
		rbacLookPath = oldLookPath
	})

	rbacLookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	issueBody := `### 操作
add（ロール追加）

### 対象ユーザー
bob

### ロール
pm
`
	rbacExecCommand = func(name string, args ...string) *exec.Cmd {
		if name == "git" && len(args) == 3 && args[0] == "config" && args[1] == "--get" && args[2] == "remote.origin.url" {
			return exec.Command("sh", "-c", "printf 'git@github.com:acme/teraflow.git\\n'")
		}
		if name == "gh" && len(args) >= 1 && args[0] == "api" {
			return exec.Command("sh", "-c", "cat <<'EOF'\n"+issueBody+"\nEOF")
		}
		return exec.Command("sh", "-c", "exit 0")
	}

	root := newRootCmd("test")
	root.SetArgs([]string{"rbac", "apply", "--config", cfgPath, "--from-issue", "10", "--user", "admin-user"})
	if err := root.Execute(); err != nil {
		t.Fatalf("rbac apply add failed: %v", err)
	}
	assertRoleHasMember(t, cfgPath, "pm", "bob")

	issueBody = `### 操作
change（ロール変更）

### 対象ユーザー
bob

### ロール
qa

### 変更元ロール（change時のみ）
pm
`
	root = newRootCmd("test")
	root.SetArgs([]string{"rbac", "apply", "--config", cfgPath, "--from-issue", "11", "--user", "admin-user"})
	if err := root.Execute(); err != nil {
		t.Fatalf("rbac apply change failed: %v", err)
	}
	assertRoleHasMember(t, cfgPath, "qa", "bob")
	assertRoleLacksMember(t, cfgPath, "pm", "bob")

	issueBody = `### 操作
remove（ロール削除）

### 対象ユーザー
bob

### ロール
qa
`
	root = newRootCmd("test")
	root.SetArgs([]string{"rbac", "apply", "--config", cfgPath, "--from-issue", "12", "--user", "admin-user"})
	if err := root.Execute(); err != nil {
		t.Fatalf("rbac apply remove failed: %v", err)
	}
	assertRoleLacksMember(t, cfgPath, "qa", "bob")

	issueBody = `### 操作
remove（ロール削除）

### 対象ユーザー
admin-user

### ロール
admin
`
	root = newRootCmd("test")
	root.SetArgs([]string{"rbac", "apply", "--config", cfgPath, "--from-issue", "13", "--user", "admin-user"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected last-admin protection error")
	}
	assertRoleHasMember(t, cfgPath, "admin", "admin-user")
}

func assertRoleHasMember(t *testing.T, cfgPath, role, member string) {
	t.Helper()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	for _, r := range cfg.RBAC.Roles {
		if r.Name != role {
			continue
		}
		for _, m := range r.Members {
			if m == member {
				return
			}
		}
		t.Fatalf("expected member %s in role %s", member, role)
	}
	t.Fatalf("role not found: %s", role)
}

func assertRoleLacksMember(t *testing.T, cfgPath, role, member string) {
	t.Helper()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	for _, r := range cfg.RBAC.Roles {
		if r.Name != role {
			continue
		}
		for _, m := range r.Members {
			if m == member {
				t.Fatalf("expected member %s to be absent in role %s", member, role)
			}
		}
		return
	}
	t.Fatalf("role not found: %s", role)
}
