package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/validate"
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

func TestValidateCmdInputValidation(t *testing.T) {
	root := newRootCmd("test")
	root.SetArgs([]string{"validate", "--level", "0"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "--level must be 1..4") {
		t.Fatalf("expected level error, got: %v", err)
	}

	root = newRootCmd("test")
	root.SetArgs([]string{"validate", "--phase", "p", "--node", "x"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "--phase and --node cannot be used together") {
		t.Fatalf("expected phase/node conflict, got: %v", err)
	}
}

func TestValidateCmdJSONMode(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "name: test\n")
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
	cmd.SetArgs([]string{"--config", cfgPath, "--format", "json", "validate", "--full"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected warning exit code")
	}

	var payload map[string]any
	if jErr := json.Unmarshal(out.Bytes(), &payload); jErr != nil {
		t.Fatalf("json unmarshal failed: %v raw=%s", jErr, out.String())
	}
	if _, ok := payload["valid"]; !ok {
		t.Fatalf("missing valid field in payload: %+v", payload)
	}
}

func TestSortErrorsAndWarnings(t *testing.T) {
	errs := []validate.ValidationError{
		{Level: 2, NodeID: "b", ErrorType: "z"},
		{Level: 1, NodeID: "a", ErrorType: "m"},
		{Level: 1, NodeID: "a", ErrorType: "a"},
	}
	gotErrs := sortErrors(errs)
	wantErrTypes := []string{"a", "m", "z"}
	types := []string{gotErrs[0].ErrorType, gotErrs[1].ErrorType, gotErrs[2].ErrorType}
	if !reflect.DeepEqual(types, wantErrTypes) {
		t.Fatalf("sorted error types=%v want=%v", types, wantErrTypes)
	}

	warns := []validate.ValidationWarning{
		{Level: 2, NodeID: "x", WarningType: "b"},
		{Level: 1, NodeID: "x", WarningType: "c"},
		{Level: 1, NodeID: "x", WarningType: "a"},
	}
	gotWarns := sortWarnings(warns)
	wantWarnTypes := []string{"a", "c", "b"}
	wtypes := []string{gotWarns[0].WarningType, gotWarns[1].WarningType, gotWarns[2].WarningType}
	if !reflect.DeepEqual(wtypes, wantWarnTypes) {
		t.Fatalf("sorted warning types=%v want=%v", wtypes, wantWarnTypes)
	}
}
