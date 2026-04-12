package pipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/index"
)

type fakeImplementProvider struct{}

func (f *fakeImplementProvider) Complete(_ context.Context, _ string, userPrompt string, _ int) (string, int, error) {
	if strings.Contains(userPrompt, "\nTarget module:\ninternal/fail.go\n") {
		return "", 0, errors.New("forced failure")
	}
	if strings.Contains(userPrompt, ".py") {
		return "print('ok')", 12, nil
	}
	return "package internal\n", 12, nil
}

func (f *fakeImplementProvider) Name() string { return "fake" }

func TestImplementEngineDryRun(t *testing.T) {
	root := t.TempDir()
	designPath := filepath.Join(root, "docs", "design", "auth.md")
	mustWriteImplementFile(t, designPath, `---
codd:
  node_id: detail:auth
  title: auth
  review_required: review
  modules:
    - internal/auth/handler.go
    - internal/auth/token.go
  verified_by:
    - test:ut-auth
---
# auth
`)

	engine := NewImplementEngine(nil, nil, nil, root, nil)
	report, err := engine.Execute(context.Background(), ImplementRequest{
		DesignDocPath: designPath,
		MaxParallel:   2,
		DryRun:        true,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got, want := report.Summary["total"], 2; got != want {
		t.Fatalf("total=%d, want %d", got, want)
	}
	for _, module := range report.Modules {
		if module.Status != "dry_run" {
			t.Fatalf("status=%s, want dry_run", module.Status)
		}
	}
}

func TestImplementEngineContinueOnFailure(t *testing.T) {
	root := t.TempDir()
	designPath := filepath.Join(root, "docs", "design", "core.md")
	mustWriteImplementFile(t, designPath, `---
codd:
  node_id: detail:core
  title: core
  review_required: review
  modules:
    - path: internal/ok.go
    - path: internal/fail.go
---
# core
`)

	engine := NewImplementEngine(&fakeImplementProvider{}, nil, nil, root, nil)
	report, err := engine.Execute(context.Background(), ImplementRequest{
		DesignDocPath: designPath,
		MaxParallel:   2,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got, want := report.Summary["failed"], 1; got != want {
		t.Fatalf("failed=%d, want %d", got, want)
	}
	if got, want := report.Summary["completed"], 1; got != want {
		t.Fatalf("completed=%d, want %d", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "ok.go")); err != nil {
		t.Fatalf("expected generated module: %v", err)
	}
}

func TestImplementEngineGeneratesTestSkeletonFromVerifiedBy(t *testing.T) {
	root := t.TempDir()
	designPath := filepath.Join(root, "docs", "design", "mod.md")
	mustWriteImplementFile(t, designPath, `---
codd:
  node_id: detail:mod
  title: mod
  review_required: review
  modules:
    - internal/mod/handler.go
  verified_by:
    - test:ut-handler
---
# mod
`)
	idx := &index.Index{Entries: []index.Entry{{
		NodeID: "test:ut-handler",
		Path:   "tests/generated/handler_test.go",
	}}}

	engine := NewImplementEngine(&fakeImplementProvider{}, nil, nil, root, idx)
	report, err := engine.Execute(context.Background(), ImplementRequest{DesignDocPath: designPath})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if report.Summary["completed"] != 1 {
		t.Fatalf("completed=%d, want 1", report.Summary["completed"])
	}
	if _, err := os.Stat(filepath.Join(root, "tests", "generated", "handler_test.go")); err != nil {
		t.Fatalf("expected generated test skeleton: %v", err)
	}
}

func mustWriteImplementFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestImplementEngineValidationErrors(t *testing.T) {
	root := t.TempDir()
	engine := NewImplementEngine(nil, nil, nil, root, nil)

	if _, err := engine.Execute(context.Background(), ImplementRequest{}); err == nil {
		t.Fatal("expected error for empty design doc path")
	}

	designPath := filepath.Join(root, "docs", "design", "empty.md")
	mustWriteImplementFile(t, designPath, `---
codd:
  node_id: detail:none
---
# empty
`)
	if _, err := engine.Execute(context.Background(), ImplementRequest{DesignDocPath: designPath}); err == nil || !strings.Contains(err.Error(), "modules field is required") {
		t.Fatalf("expected modules required error, got: %v", err)
	}
}

func TestImplementEngineProviderRequiredForNonDryRun(t *testing.T) {
	root := t.TempDir()
	designPath := filepath.Join(root, "docs", "design", "one.md")
	mustWriteImplementFile(t, designPath, `---
codd:
  modules:
    - internal/mod.go
---
`)

	engine := NewImplementEngine(nil, nil, nil, root, nil)
	report, err := engine.Execute(context.Background(), ImplementRequest{
		DesignDocPath: designPath,
		DryRun:        false,
	})
	if err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
	if report.Summary["failed"] != 1 {
		t.Fatalf("failed=%d want=1", report.Summary["failed"])
	}
	if !strings.Contains(report.Modules[0].Error, "provider is required") {
		t.Fatalf("unexpected module error: %q", report.Modules[0].Error)
	}
}

func TestImplementEngineCreatePR(t *testing.T) {
	root := t.TempDir()
	designPath := filepath.Join(root, "docs", "design", "pr.md")
	mustWriteImplementFile(t, designPath, `---
codd:
  modules:
    - internal/a.go
---
`)

	engine := NewImplementEngine(nil, nil, nil, root, nil)
	report, err := engine.Execute(context.Background(), ImplementRequest{
		DesignDocPath: designPath,
		DryRun:        true,
		CreatePR:      true,
	})
	if err != nil {
		t.Fatalf("execute with CreatePR failed: %v", err)
	}
	if report.IntegrationPR == nil || report.IntegrationPR.Created {
		t.Fatalf("expected dry-run integration PR metadata only, got: %+v", report.IntegrationPR)
	}
	if len(report.IntegrationPR.Modules) != 1 || report.IntegrationPR.Modules[0] != "internal/a.go" {
		t.Fatalf("unexpected integration PR modules: %+v", report.IntegrationPR)
	}
}

func TestResolveToAbsPath(t *testing.T) {
	root := t.TempDir()
	got, err := resolveToAbsPath(root, "internal/a.go")
	if err != nil {
		t.Fatalf("resolve relative path failed: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(got), "/internal/a.go") {
		t.Fatalf("unexpected resolved path: %s", got)
	}
	abs := filepath.Join(root, "x.go")
	got, err = resolveToAbsPath(root, abs)
	if err != nil || got != abs {
		t.Fatalf("resolve abs path got=%q err=%v", got, err)
	}
	if _, err := resolveToAbsPath(root, " "); err == nil {
		t.Fatal("expected path required error")
	}
}

func TestParseDesignFrontmatter(t *testing.T) {
	content := `---
codd:
  node_id: detail:auth
  title: Auth
  review_required: require
  modules:
    - path: internal/auth/handler.go
      design_section: "Auth section"
      test_spec: test:ut-auth
      depends_on: [internal/auth/base.go]
  verified_by: [test:ut-auth]
---
# body`
	spec, err := parseDesignFrontmatter(content)
	if err != nil {
		t.Fatalf("parseDesignFrontmatter failed: %v", err)
	}
	if spec.NodeID != "detail:auth" || len(spec.Modules) != 1 || spec.ReviewRequired != "" {
		t.Fatalf("unexpected parsed spec: %+v", spec)
	}
	if len(spec.VerifiedBy) != 1 || spec.VerifiedBy[0] != "test:ut-auth" {
		t.Fatalf("unexpected verified_by: %+v", spec.VerifiedBy)
	}

	// Inline fallback should work when codd.* is absent.
	inline, err := parseDesignFrontmatter(`---
node_id: detail:inline
modules:
  - internal/a.go
---
`)
	if err != nil || inline.NodeID != "detail:inline" || len(inline.Modules) != 1 {
		t.Fatalf("inline parse mismatch spec=%+v err=%v", inline, err)
	}
}

func TestParseDesignFrontmatterErrors(t *testing.T) {
	if _, err := parseDesignFrontmatter("# no frontmatter"); err == nil {
		t.Fatal("expected frontmatter missing error")
	}
	if _, err := parseDesignFrontmatter(`---
: bad
---
`); err == nil {
		t.Fatal("expected yaml parse error")
	}
}

func TestNormalizeModulesAndDedupe(t *testing.T) {
	raw := []any{
		" internal/a.go ",
		map[string]any{
			"path":          "internal/b.go",
			"design_section": "sec",
			"depends_on":     []any{"internal/a.go"},
		},
		map[string]any{
			"path":           "internal/b.go",
			"test_spec":      "test:ut-b",
			"review_required": "review",
		},
	}
	mods := normalizeModules(raw)
	if len(mods) != 2 {
		t.Fatalf("normalizeModules len=%d want=2", len(mods))
	}
	if mods[1].Path != "internal/b.go" {
		t.Fatalf("unexpected normalized path: %+v", mods[1])
	}
	if mods[1].DependsOn[0] != "internal/a.go" {
		t.Fatalf("depends_on should normalize slash: %+v", mods[1].DependsOn)
	}
}

func TestSelectModulesAndDependencies(t *testing.T) {
	design := []ModuleSpec{
		{Path: "a.go"},
		{Path: "b.go", DependsOn: []string{"a.go"}},
		{Path: "c.go", DependsOn: []string{"b.go"}},
	}

	selected, err := selectModules(design, []ModuleSpec{{Path: "c.go"}})
	if err != nil {
		t.Fatalf("selectModules failed: %v", err)
	}
	got := make([]string, 0, len(selected))
	for _, s := range selected {
		got = append(got, s.Path)
	}
	want := []string{"a.go", "b.go", "c.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selected paths=%v want=%v", got, want)
	}

	if _, err := selectModules(design, []ModuleSpec{{Path: " "}}); err == nil {
		t.Fatal("expected no valid module error")
	}
}

func TestExecutionLevels(t *testing.T) {
	levels, err := executionLevels([]ModuleSpec{
		{Path: "a.go"},
		{Path: "b.go", DependsOn: []string{"a.go"}},
		{Path: "c.go", DependsOn: []string{"a.go"}},
		{Path: "d.go", DependsOn: []string{"b.go", "c.go"}},
	})
	if err != nil {
		t.Fatalf("executionLevels failed: %v", err)
	}
	if len(levels) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(levels))
	}
	if levels[0][0].Path != "a.go" {
		t.Fatalf("unexpected first level: %+v", levels[0])
	}

	_, err = executionLevels([]ModuleSpec{
		{Path: "a.go", DependsOn: []string{"b.go"}},
		{Path: "b.go", DependsOn: []string{"a.go"}},
	})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got: %v", err)
	}
}

func TestPathAndIdentifierHelpers(t *testing.T) {
	if got := inferTestLanguage("test:python:api"); got != "python" {
		t.Fatalf("inferTestLanguage python mismatch: %q", got)
	}
	if got := inferTestLanguage("test:go:api"); got != "go" {
		t.Fatalf("inferTestLanguage go mismatch: %q", got)
	}
	if got := defaultTestPath("internal/auth/handler.go", "test:ut-auth", "go"); got != filepath.Join("internal", "auth", "handler_test.go") {
		t.Fatalf("unexpected go test path: %q", got)
	}
	if got := defaultTestPath("internal/mod.py", "test:py", "python"); got != filepath.Join("internal", "test_mod.py") {
		t.Fatalf("unexpected python test path: %q", got)
	}
	if got := sanitizeIdentifier("  a//b::c--d  "); got != "a_b_c_d" {
		t.Fatalf("unexpected sanitizeIdentifier: %q", got)
	}
	if got := sanitizeIdentifier("___"); got != "generated" {
		t.Fatalf("empty sanitize fallback mismatch: %q", got)
	}
	if got := toExportedIdentifier("some_func_name"); got != "SomeFuncName" {
		t.Fatalf("unexpected exported identifier: %q", got)
	}
	if got := toExportedIdentifier(""); got != "Generated" {
		t.Fatalf("empty exported identifier fallback mismatch: %q", got)
	}
}
