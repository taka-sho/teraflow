package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanCmd(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), "project:\n  name: test\n")
	if err := os.MkdirAll(filepath.Join(tmp, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}

	command := newRootCmd("test")
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"scan", "--config", cfgPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("scan execute error: %v", err)
	}

	got := out.String()
	checks := []string{
		"Scanning project...",
		"✓ .github/teraflow.yml",
		"✓ .github/project-state.yml",
		"✓ docs/",
		"○ .teraflow/rework-log.yml",
		"Scan complete: 3/3 required files present.",
	}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Fatalf("output should contain %q, got:\n%s", want, got)
		}
	}
}

func TestScanCmdNoProject(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	command := newRootCmd("test")
	command.SetArgs([]string{"scan", "--config", cfgPath})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected error when project-state is missing")
	}
	if !strings.Contains(err.Error(), "E0001") {
		t.Fatalf("expected E0001 error, got: %v", err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
