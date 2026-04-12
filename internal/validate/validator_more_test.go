package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/graphbridge"
	"github.com/taka-sho/teraflow/internal/index"
)

func TestSetLevelOutOfRangeIsIgnored(t *testing.T) {
	v := NewValidator(&index.Index{}, nil, t.TempDir())
	v.SetLevel(2)
	v.SetLevel(9)
	if v.maxLevel != 2 {
		t.Fatalf("maxLevel changed unexpectedly: %d", v.maxLevel)
	}
}

func TestValidateWithNilIndexErrors(t *testing.T) {
	v := NewValidator(nil, nil, t.TempDir())
	if res := v.ValidatePhaseTransition("requirements", "detailed-design"); res.Valid || len(res.Errors) == 0 {
		t.Fatalf("expected invalid result with errors for nil index: %+v", res)
	}
	if res := v.ValidateArtifact("req:a"); res.Valid || len(res.Errors) == 0 {
		t.Fatalf("expected invalid artifact result with errors for nil index: %+v", res)
	}
}

func TestValidateArtifactMissingNode(t *testing.T) {
	v := NewValidator(&index.Index{Entries: []index.Entry{{NodeID: "req:a", Title: "A", Path: "docs/a.md"}}}, nil, t.TempDir())
	res := v.ValidateArtifact("req:missing")
	if res.Valid || len(res.Errors) == 0 || res.Errors[0].ErrorType != "missing_node" {
		t.Fatalf("expected missing_node error, got: %+v", res)
	}
}

func TestValidateSchemaPhaseAndReferences(t *testing.T) {
	root := t.TempDir()
	idx := &index.Index{Entries: []index.Entry{
		{
			NodeID:         "Bad ID",
			Title:          "",
			Path:           "",
			ReviewRequired: "manual",
			ChangeImpact:   "red",
			Phase:          "requirements",
			Status:         "",
			DependsOn:      []string{"missing:dep"},
			Modules:        []string{"/abs/path.go", "../escape.go"},
			Verifies:       []string{"target:1"},
			VerifiedBy:     []string{"source:1"},
		},
		{NodeID: "target:1", Title: "Target", Path: "docs/target.md"},
		{NodeID: "source:1", Title: "Source", Path: "docs/source.md"},
		{
			NodeID:     "req:missing-links",
			Title:      "Missing Links",
			Path:       "docs/missing.md",
			Verifies:   []string{"target:missing"},
			VerifiedBy: []string{"source:missing"},
		},
	}}
	v := NewValidator(idx, nil, root)
	v.SetLevel(3)
	res := v.ValidateArtifact("Bad ID")
	if res.Valid {
		t.Fatalf("expected invalid result, got valid: %+v", res)
	}
	if !hasErrorType(res.Errors, "invalid_node_id") ||
		!hasErrorType(res.Errors, "missing_title") ||
		!hasErrorType(res.Errors, "missing_path") ||
		!hasErrorType(res.Errors, "invalid_review_required") ||
		!hasErrorType(res.Errors, "invalid_change_impact") ||
		!hasErrorType(res.Errors, "missing_dependency") ||
		!hasErrorType(res.Errors, "invalid_module_path") ||
		!hasErrorType(res.Errors, "verifies_mismatch") ||
		!hasErrorType(res.Errors, "verified_by_mismatch") {
		t.Fatalf("expected multiple schema/reference errors, got: %+v", res.Errors)
	}
	if !hasWarningType(res.Warnings, "missing_status") {
		t.Fatalf("expected missing_status warning, got: %+v", res.Warnings)
	}

	allRes := v.ValidatePhaseTransition("", "")
	if !hasErrorType(allRes.Errors, "missing_verifies_target") || !hasErrorType(allRes.Errors, "missing_verified_by_source") {
		t.Fatalf("expected missing link errors in phase transition validation, got: %+v", allRes.Errors)
	}
}

func TestValidatePhaseTransitionStatusNotReady(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{
			NodeID: "req:a", Title: "A", Path: "docs/a.md",
			Phase: "requirements", Status: "draft",
		},
	}}
	v := NewValidator(idx, nil, t.TempDir())
	v.SetLevel(3)
	res := v.ValidatePhaseTransition("requirements", "detailed-design")
	if !hasErrorType(res.Errors, "status_not_ready") {
		t.Fatalf("expected status_not_ready error, got: %+v", res.Errors)
	}
}

func TestValidateGraphBridgePaths(t *testing.T) {
	root := t.TempDir()
	idx := &index.Index{Entries: []index.Entry{{NodeID: "req:a", Title: "A", Path: "docs/a.md"}}}

	vUnavailable := NewValidator(idx, graphbridge.New(root), root)
	vUnavailable.SetLevel(4)
	resUnavailable := vUnavailable.ValidateArtifact("req:a")
	if !hasWarningType(resUnavailable.Warnings, "graphrag_unavailable") {
		t.Fatalf("expected graphrag_unavailable warning, got: %+v", resUnavailable.Warnings)
	}

	if err := os.MkdirAll(filepath.Join(root, "graphrag"), 0o755); err != nil {
		t.Fatalf("mkdir graphrag: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "graphrag", "pyproject.toml"), []byte("[project]\nname='x'\n"), 0o644); err != nil {
		t.Fatalf("write pyproject: %v", err)
	}

	vFail := NewValidator(idx, graphbridge.New(root), root)
	vFail.SetLevel(4)
	resFail := vFail.ValidateArtifact("req:a")
	if !hasWarningType(resFail.Warnings, "graphrag_check_failed") {
		t.Fatalf("expected graphrag_check_failed warning, got: %+v", resFail.Warnings)
	}
}

func TestValidateGraphBridgeExecuteReturnsIssues(t *testing.T) {
	root := t.TempDir()
	moduleDir := filepath.Join(root, "teraflow_graphrag")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}
	mainPy := `import json,sys
if "--version" in sys.argv:
    print("0.0.1")
    raise SystemExit(0)
print(json.dumps({"issues":[
    {"node_id":"req:a","severity":"error","type":"bridge_error","message":"bad","source":""},
    {"node_id":"req:a","severity":"warning","type":"bridge_warn","message":"warn"}
]}))
`
	if err := os.WriteFile(filepath.Join(moduleDir, "__main__.py"), []byte(mainPy), 0o644); err != nil {
		t.Fatalf("write __main__.py: %v", err)
	}
	oldPyPath := os.Getenv("PYTHONPATH")
	t.Setenv("PYTHONPATH", root+string(os.PathListSeparator)+oldPyPath)

	idx := &index.Index{Entries: []index.Entry{{NodeID: "req:a", Title: "A", Path: "docs/a.md"}}}
	v := NewValidator(idx, graphbridge.New(root), root)
	v.SetLevel(4)
	res := v.ValidateArtifact("req:a")
	if !hasErrorType(res.Errors, "bridge_error") {
		t.Fatalf("expected bridge_error in errors, got: %+v", res.Errors)
	}
	if !hasWarningType(res.Warnings, "bridge_warn") {
		t.Fatalf("expected bridge_warn in warnings, got: %+v", res.Warnings)
	}
}

func hasErrorType(errs []ValidationError, want string) bool {
	for _, err := range errs {
		if strings.EqualFold(err.ErrorType, want) {
			return true
		}
	}
	return false
}

func hasWarningType(warns []ValidationWarning, want string) bool {
	for _, warn := range warns {
		if strings.EqualFold(warn.WarningType, want) {
			return true
		}
	}
	return false
}
