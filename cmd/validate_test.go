package cmd

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCmdWarningsExitCode(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".github", "teraflow.yml"), "name: test\n")
	mustWrite(t, filepath.Join(root, ".teraflow", "index.yml"), `version: "1"
entries:
  - node_id: req:a
    title: "A"
    path: docs/a.md
`)

	cmd := newRootCmd("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", filepath.Join(root, ".github", "teraflow.yml"), "validate", "--full"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected warning exit error")
	}
	var exitErr exitCodeCarrier
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %v", err)
	}
	if !strings.Contains(out.String(), "Warnings:") {
		t.Fatalf("expected warning output, got: %s", out.String())
	}
}

func TestValidateCmdErrorsExitCode(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".github", "teraflow.yml"), "name: test\n")
	mustWrite(t, filepath.Join(root, ".teraflow", "index.yml"), `version: "1"
entries:
  - node_id: req:a
    title: "A"
    path: docs/a.md
    depends_on: [missing:b]
`)

	cmd := newRootCmd("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", filepath.Join(root, ".github", "teraflow.yml"), "validate", "--full", "--level", "2"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error exit")
	}
	var exitErr exitCodeCarrier
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got %v", err)
	}
}

func TestImpactCmdJSON(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".github", "teraflow.yml"), "name: test\n")
	mustWrite(t, filepath.Join(root, ".teraflow", "index.yml"), `version: "1"
entries:
  - node_id: req:a
    title: "A"
    path: docs/a.md
  - node_id: design:b
    title: "B"
    path: docs/b.md
    depends_on: [req:a]
`)

	cmd := newRootCmd("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", filepath.Join(root, ".github", "teraflow.yml"), "--format", "json", "impact", "req:a"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("impact command failed: %v", err)
	}
	if !strings.Contains(out.String(), `"changed_node":"req:a"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
