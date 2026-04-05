package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateConditionDocumentExists(t *testing.T) {
	project := t.TempDir()
	path := filepath.Join(project, "docs", "design")
	writeTestFile(t, filepath.Join(path, "spec.md"), "# spec")

	ok, msg, err := EvaluateCondition(Condition{Type: ConditionDocumentExists, Value: "docs/design"}, project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected document_exists to pass, msg=%s", msg)
	}
}

func TestEvaluateConditionManualApproval(t *testing.T) {
	ok, _, err := EvaluateCondition(Condition{Type: ConditionManualApproval}, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("manual approval should always pass")
	}
}

func TestEvaluateConditionTestPassRate(t *testing.T) {
	project := t.TempDir()
	writeTestFile(t, filepath.Join(project, ".teraflow", "gate-metrics.yml"), "test_pass_rate: 0.90\ncoverage: 0.75\n")

	ok, _, err := EvaluateCondition(Condition{Type: ConditionTestPassRate, Value: "0.85"}, project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected test_pass_rate condition to pass")
	}

	ok, _, err = EvaluateCondition(Condition{Type: ConditionTestPassRate, Value: "0.95"}, project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected test_pass_rate condition to fail")
	}
}

func TestEvaluateConditionCoverage(t *testing.T) {
	project := t.TempDir()
	writeTestFile(t, filepath.Join(project, ".teraflow", "gate-metrics.yml"), "test_pass_rate: 1.0\ncoverage: 0.80\n")

	ok, _, err := EvaluateCondition(Condition{Type: ConditionCoverage, Value: "0.70"}, project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected coverage condition to pass")
	}
}

func TestEvaluateAggregatesConditionFailures(t *testing.T) {
	project := t.TempDir()
	rule := GateRule{
		ProcessName: "detailed_design",
		Conditions: []Condition{
			{Type: ConditionDocumentExists, Value: "docs/design"},
			{Type: ConditionManualApproval},
		},
	}

	ok, messages, err := Evaluate(rule, project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected gate evaluation to fail")
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if !strings.Contains(messages[0], "document not found") {
		t.Fatalf("unexpected message: %v", messages)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
