package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDoctorCmd(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "development"
phases:
  current: "implementation"
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "doctor"})

	// doctor may return error if issues found (e.g. go/git not in PATH on CI)
	// We just verify the command runs and prints results
	_ = root.Execute()
	got := out.String()
	if !strings.Contains(got, "teraflow doctor") && !strings.Contains(got, "status") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestNewDoctorCmdJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "development"
phases:
  current: "implementation"
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "doctor"})

	_ = root.Execute()
	got := out.String()
	if !strings.Contains(got, `"status"`) {
		t.Fatalf("expected JSON output with status: %q", got)
	}
}

func TestRunChecksInitialized(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "development"
phases:
  current: "implementation"
`)

	results := runChecks(configPath, false)
	cfg := byCategory(results, "configuration")
	if len(cfg) != 2 {
		t.Fatalf("expected 2 configuration checks, got %d", len(cfg))
	}
	for _, r := range cfg {
		if !r.OK {
			t.Fatalf("expected configuration check %q to be OK, got NG: %s", r.Name, r.Message)
		}
	}
}

func TestRunChecksNotInitialized(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)

	results := runChecks(configPath, false)
	cfg := byCategory(results, "configuration")
	found := false
	for _, r := range cfg {
		if r.Name == "project-state.yml" {
			found = true
			if r.OK {
				t.Fatalf("expected project-state.yml check to fail when missing")
			}
		}
	}
	if !found {
		t.Fatal("project-state.yml check was not found")
	}
}

func TestRunChecksNoAI(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "development"
phases:
  current: "implementation"
`)

	results := runChecks(configPath, false)
	if got := byCategory(results, "ai"); len(got) != 0 {
		t.Fatalf("expected no ai checks when checkAI=false, got %d", len(got))
	}
}

func TestRunChecksWithAI(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "dummy-key")
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "development"
phases:
  current: "implementation"
`)

	results := runChecks(configPath, true)
	ai := byCategory(results, "ai")
	if len(ai) != 1 {
		t.Fatalf("expected 1 ai check, got %d", len(ai))
	}
	if !ai[0].OK {
		t.Fatalf("expected ai check to be OK, got message: %s", ai[0].Message)
	}
}

func byCategory(results []CheckResult, category string) []CheckResult {
	filtered := make([]CheckResult, 0, len(results))
	for _, r := range results {
		if r.Category == category {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func TestCountIssues(t *testing.T) {
	results := []CheckResult{
		{OK: true},
		{OK: false},
		{OK: false},
	}
	if got := countIssues(results); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}

	if got := countIssues(nil); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestPrintCategory(t *testing.T) {
	var buf bytes.Buffer
	orig := doctorStdout
	doctorStdout = &buf
	defer func() { doctorStdout = orig }()

	results := []CheckResult{
		{Category: "environment", Name: "go", OK: true, Message: "ok"},
		{Category: "environment", Name: "missing", OK: false, Message: "not found"},
		{Category: "other", Name: "skip", OK: true, Message: "skip"},
	}
	printCategory("Environment", "environment", results)

	got := buf.String()
	if !strings.Contains(got, "✓ go") {
		t.Fatalf("expected ✓ go in output: %q", got)
	}
	if !strings.Contains(got, "✗ missing") {
		t.Fatalf("expected ✗ missing in output: %q", got)
	}
	if strings.Contains(got, "skip") {
		t.Fatalf("should not include other category: %q", got)
	}
}

func TestPrintDoctorResultsText(t *testing.T) {
	var buf bytes.Buffer
	orig := doctorStdout
	doctorStdout = &buf
	defer func() { doctorStdout = orig }()

	results := []CheckResult{
		{Category: "environment", Name: "go", OK: true, Message: "ok"},
		{Category: "configuration", Name: "config", OK: true, Message: "ok"},
		{Category: "integrity", Name: "state", OK: true, Message: "ok"},
	}
	err := printDoctorResults(results, "text", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "all checks passed") {
		t.Fatalf("expected all checks passed: %q", got)
	}
}

func TestPrintDoctorResultsTextWithIssue(t *testing.T) {
	var buf bytes.Buffer
	orig := doctorStdout
	doctorStdout = &buf
	defer func() { doctorStdout = orig }()

	results := []CheckResult{
		{Category: "environment", Name: "go", OK: false, Message: "missing"},
	}
	err := printDoctorResults(results, "text", false)
	if err == nil {
		t.Fatal("expected error when issues found")
	}
}

func TestPrintDoctorResultsJSON(t *testing.T) {
	var buf bytes.Buffer
	orig := doctorStdout
	doctorStdout = &buf
	defer func() { doctorStdout = orig }()

	results := []CheckResult{
		{Category: "environment", Name: "go", OK: true, Message: "ok"},
	}
	err := printDoctorResults(results, "json", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), `"status"`) {
		t.Fatalf("expected JSON with status: %q", buf.String())
	}
}
