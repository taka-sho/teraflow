package rbac

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckPermissionAllowDeny(t *testing.T) {
	engine := NewEngine(RBACConfig{
		Enabled: true,
		Roles: []Role{
			{
				Name:        "pm",
				Members:     []string{"alice"},
				Permissions: []Permission{"gate.approve.planning", "*.read.*"},
			},
		},
	})

	if !engine.CheckPermission("alice", "gate.approve.planning") {
		t.Fatal("expected allow for exact permission")
	}
	if engine.CheckPermission("alice", "gate.approve.release") {
		t.Fatal("expected deny for missing permission")
	}
	if !engine.CheckPermission("alice", "process.read.status") {
		t.Fatal("expected allow for wildcard permission")
	}
}

func TestCheckPermissionWildcard(t *testing.T) {
	engine := NewEngine(RBACConfig{
		Enabled: true,
		Roles: []Role{
			{
				Name:        "release_mgr",
				Members:     []string{"bob"},
				Permissions: []Permission{"gate.approve.*"},
			},
		},
	})

	if !engine.CheckPermission("bob", "gate.approve.planning") {
		t.Fatal("expected wildcard allow")
	}
	if engine.CheckPermission("bob", "process.start.any") {
		t.Fatal("expected deny outside wildcard scope")
	}
}

func TestCheckPermissionEmptyConfigAndDisabled(t *testing.T) {
	enabledEmpty := NewEngine(RBACConfig{Enabled: true})
	if enabledEmpty.CheckPermission("alice", "gate.approve.planning") {
		t.Fatal("expected deny when enabled with empty config")
	}

	disabled := NewEngine(RBACConfig{Enabled: false})
	if !disabled.CheckPermission("alice", "gate.approve.planning") {
		t.Fatal("expected allow when rbac.enabled=false")
	}
}

func TestGetUserRoles(t *testing.T) {
	engine := NewEngine(RBACConfig{
		Enabled: true,
		Roles: []Role{
			{Name: "pm", Members: []string{"alice"}},
			{Name: "qa", Members: []string{"alice", "charlie"}},
		},
	})

	roles := engine.GetUserRoles("alice")
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(roles))
	}
}

func TestCurrentUserFromGitConfig(t *testing.T) {
	tmp := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = tmp
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	run("init")
	run("config", "user.name", "rbac-test-user")

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(filepath.Clean(tmp)); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	user, err := CurrentUser()
	if err != nil {
		t.Fatalf("CurrentUser returned error: %v", err)
	}
	if user != "rbac-test-user" {
		t.Fatalf("unexpected user: %q", user)
	}
}
