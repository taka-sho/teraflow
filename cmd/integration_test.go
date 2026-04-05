package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestIntegrationWave2Scenarios(t *testing.T) {
	t.Run("slcp-jcf process basic flow", func(t *testing.T) {
		tmp := t.TempDir()
		configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

		if !commandPathExists(newRootCmd("test"), "process", "list") ||
			!commandPathExists(newRootCmd("test"), "process", "start") ||
			!commandPathExists(newRootCmd("test"), "process", "complete") {
			t.Skip("requires Wave2 commands")
		}

		root := newRootCmd("test")
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)

		root.SetArgs([]string{"--config", configPath, "process", "list"})
		if err := root.Execute(); err != nil {
			t.Fatalf("process list failed: %v", err)
		}
		list1 := out.String()
		if !strings.Contains(list1, "SLCP-JCF プロセス状況") {
			t.Fatalf("unexpected list output: %s", list1)
		}
		if count := strings.Count(list1, "プロセス"); count < 8 {
			t.Fatalf("expected 8 processes in output, got count=%d output=%s", count, list1)
		}

		out.Reset()
		root = newRootCmd("test")
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs([]string{"--config", configPath, "process", "start", "企画プロセス"})
		if err := root.Execute(); err != nil {
			t.Fatalf("process start failed: %v", err)
		}

		out.Reset()
		root = newRootCmd("test")
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs([]string{"--config", configPath, "process", "complete", "企画プロセス"})
		if err := root.Execute(); err != nil {
			t.Fatalf("process complete failed: %v", err)
		}

		out.Reset()
		root = newRootCmd("test")
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs([]string{"--config", configPath, "process", "list"})
		if err := root.Execute(); err != nil {
			t.Fatalf("process list(2nd) failed: %v", err)
		}
		list2 := out.String()
		if !strings.Contains(list2, "現在のプロセス: 要件定義プロセス") {
			t.Fatalf("unexpected current process after complete: %s", list2)
		}
	})

	t.Run("audit log recording", func(t *testing.T) {
		tmp := t.TempDir()
		configPath := setupTestProjectState(t, tmp, "initial_development", "requirements")

		if !commandPathExists(newRootCmd("test"), "process", "start") ||
			!commandPathExists(newRootCmd("test"), "audit", "list") {
			t.Skip("requires Wave2 commands")
		}

		root := newRootCmd("test")
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs([]string{"--config", configPath, "process", "start", "要件定義プロセス"})
		if err := root.Execute(); err != nil {
			t.Fatalf("process start failed: %v", err)
		}

		out.Reset()
		root = newRootCmd("test")
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs([]string{"--config", configPath, "audit", "list"})
		if err := root.Execute(); err != nil {
			t.Fatalf("audit list failed: %v", err)
		}
		if strings.TrimSpace(out.String()) == "" {
			t.Fatal("expected non-empty audit list output")
		}
	})

	t.Run("status role output", func(t *testing.T) {
		tmp := t.TempDir()
		configPath := setupTestProjectState(t, tmp, "development", "design")

		if !commandPathExists(newRootCmd("test"), "status") {
			t.Skip("requires Wave2 commands")
		}

		roles := []string{"pm", "dev", "qa"}
		for _, role := range roles {
			t.Run(role, func(t *testing.T) {
				root := newRootCmd("test")
				var out bytes.Buffer
				root.SetOut(&out)
				root.SetErr(&out)
				root.SetArgs([]string{"--config", configPath, "status", "--role", role})
				if err := root.Execute(); err != nil {
					t.Fatalf("status --role %s failed: %v", role, err)
				}
				if strings.TrimSpace(out.String()) == "" {
					t.Fatalf("status --role %s output is empty", role)
				}
			})
		}
	})
}

func commandPathExists(root *cobra.Command, names ...string) bool {
	current := root
	for _, name := range names {
		found := false
		for _, child := range current.Commands() {
			if child.Name() == name {
				current = child
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
