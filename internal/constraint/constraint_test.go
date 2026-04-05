package constraint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckRequiredPhase(t *testing.T) {
	root := t.TempDir()
	writeConstraintFile(t, filepath.Join(root, ".github", "project-state.yml"), `project:
  name: "test"
phases:
  current: "testing"
`)
	engine := &Engine{}
	blocked, warnings, err := engine.Check([]Constraint{
		{Type: ConstraintRequiredPhase, Value: "implementation", Severity: SeverityBlock},
		{Type: ConstraintRequiredPhase, Value: "integration_test", Severity: SeverityWarn},
	}, root)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if len(blocked) != 0 {
		t.Fatalf("expected no blocked, got %d", len(blocked))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestCheckRequiredCoverage(t *testing.T) {
	root := t.TempDir()
	writeConstraintFile(t, filepath.Join(root, ".github", "project-state.yml"), "phases:\n  current: \"testing\"\n")
	writeConstraintFile(t, filepath.Join(root, ".teraflow", "coverage-summary.txt"), "total: 78.5%\n")

	engine := &Engine{}
	blocked, warnings, err := engine.Check([]Constraint{
		{Type: ConstraintRequiredCoverage, Value: "80", Severity: SeverityBlock},
		{Type: ConstraintRequiredCoverage, Value: "70%", Severity: SeverityWarn},
	}, root)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if len(blocked) != 1 {
		t.Fatalf("expected 1 blocked, got %d", len(blocked))
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(warnings))
	}
}

func TestCheckRequiredDocs(t *testing.T) {
	root := t.TempDir()
	writeConstraintFile(t, filepath.Join(root, ".github", "project-state.yml"), "phases:\n  current: \"testing\"\n")
	writeConstraintFile(t, filepath.Join(root, "docs", "a.md"), "ok\n")

	engine := &Engine{}
	blocked, warnings, err := engine.Check([]Constraint{
		{Type: ConstraintRequiredDocs, Value: "docs/a.md", Severity: SeverityWarn},
		{Type: ConstraintRequiredDocs, Value: "docs/a.md, docs/b.md", Severity: SeverityBlock},
	}, root)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if len(blocked) != 1 {
		t.Fatalf("expected 1 blocked, got %d", len(blocked))
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(warnings))
	}
	if !strings.Contains(blocked[0].Message, "docs/b.md") {
		t.Fatalf("unexpected message: %q", blocked[0].Message)
	}
}

func TestCheckInvalidConstraint(t *testing.T) {
	root := t.TempDir()
	writeConstraintFile(t, filepath.Join(root, ".github", "project-state.yml"), "phases:\n  current: \"testing\"\n")
	engine := &Engine{}
	_, _, err := engine.Check([]Constraint{{Type: "unknown", Value: "x", Severity: SeverityBlock}}, root)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func writeConstraintFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
