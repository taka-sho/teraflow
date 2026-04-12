package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveToAbsPath(t *testing.T) {
	tests := []struct {
		name        string
		projectRoot string
		path        string
		wantErr     bool
	}{
		{"absolute path", "/root", "/abs/path.go", false},
		{"relative path", "/root", "internal/pkg.go", false},
		{"empty path", "/root", "", true},
		{"spaces only", "/root", "   ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveToAbsPath(tt.projectRoot, tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveToAbsPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !filepath.IsAbs(got) {
				t.Fatalf("result should be absolute: %s", got)
			}
		})
	}
}

func TestResolveToAbsPathEmptyRoot(t *testing.T) {
	got, err := resolveToAbsPath("", "relative/path.go")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("result should be absolute: %s", got)
	}
}

func TestNormalizeModules(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int
	}{
		{
			name:  "string list",
			input: []any{"internal/a.go", "internal/b.go"},
			want:  2,
		},
		{
			name: "map list",
			input: []any{
				map[string]any{"path": "internal/a.go", "design_section": "auth"},
				map[string]any{"path": "internal/b.go", "depends_on": []any{"internal/a.go"}},
			},
			want: 2,
		},
		{
			name:  "mixed",
			input: []any{"internal/a.go", map[string]any{"path": "internal/b.go"}},
			want:  2,
		},
		{
			name:  "empty path filtered",
			input: []any{"", "  ", map[string]any{"path": ""}},
			want:  0,
		},
		{
			name:  "nil",
			input: nil,
			want:  0,
		},
		{
			name:  "wrong type",
			input: "not a list",
			want:  0,
		},
		{
			name:  "duplicates deduped",
			input: []any{"internal/a.go", "internal/a.go"},
			want:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeModules(tt.input)
			if len(got) != tt.want {
				t.Fatalf("normalizeModules() len = %d, want %d", len(got), tt.want)
			}
		})
	}
}

func TestNormalizeStrings(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int
	}{
		{"string slice", []any{"a", "b"}, 2},
		{"single string", "hello", 1},
		{"empty string", "", 0},
		{"nil", nil, 0},
		{"non-string items", []any{123, "x"}, 1},
		{"empty items filtered", []any{"", "  ", "valid"}, 1},
		{"number type", 42, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeStrings(tt.input)
			if len(got) != tt.want {
				t.Fatalf("normalizeStrings() len = %d, want %d", len(got), tt.want)
			}
		})
	}
}

func TestDedupeModulesMergesFields(t *testing.T) {
	in := []ModuleSpec{
		{Path: "internal/a.go", DesignSection: "original", TestSpec: "test1"},
		{Path: "internal/a.go", DesignSection: "", TestSpec: ""},
	}
	out := dedupeModules(in)
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	// Second entry has empty fields, should inherit from first
	if out[0].DesignSection != "original" {
		t.Fatalf("DesignSection = %q, want original", out[0].DesignSection)
	}
}

func TestDedupeModulesFiltersInvalid(t *testing.T) {
	in := []ModuleSpec{
		{Path: ""},
		{Path: "  "},
		{Path: "."},
		{Path: "valid.go"},
	}
	out := dedupeModules(in)
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
}

func TestSelectModulesNoRequested(t *testing.T) {
	design := []ModuleSpec{
		{Path: "a.go"},
		{Path: "b.go"},
	}
	got, err := selectModules(design, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func TestSelectModulesWithDeps(t *testing.T) {
	design := []ModuleSpec{
		{Path: "a.go"},
		{Path: "b.go", DependsOn: []string{"a.go"}},
		{Path: "c.go"},
	}
	// Request b.go, should auto-include a.go (dependency)
	got, err := selectModules(design, []ModuleSpec{{Path: "b.go"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (b.go + dep a.go)", len(got))
	}
}

func TestSelectModulesEmptyRequested(t *testing.T) {
	design := []ModuleSpec{{Path: "a.go"}}
	_, err := selectModules(design, []ModuleSpec{{Path: ""}})
	if err == nil {
		t.Fatal("empty module path should error")
	}
}

func TestSelectModulesNotInDesign(t *testing.T) {
	design := []ModuleSpec{{Path: "a.go"}}
	got, err := selectModules(design, []ModuleSpec{{Path: "new.go"}})
	if err != nil {
		t.Fatal(err)
	}
	// new.go not in design but still included
	found := false
	for _, m := range got {
		if m.Path == "new.go" {
			found = true
		}
	}
	if !found {
		t.Fatal("new.go should be included even if not in design")
	}
}

func TestExecutionLevelsCycle(t *testing.T) {
	modules := []ModuleSpec{
		{Path: "a.go", DependsOn: []string{"b.go"}},
		{Path: "b.go", DependsOn: []string{"a.go"}},
	}
	_, err := executionLevels(modules)
	if err == nil {
		t.Fatal("cycle should error")
	}
}

func TestExecutionLevelsLinear(t *testing.T) {
	modules := []ModuleSpec{
		{Path: "a.go"},
		{Path: "b.go", DependsOn: []string{"a.go"}},
		{Path: "c.go", DependsOn: []string{"b.go"}},
	}
	levels, err := executionLevels(modules)
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 3 {
		t.Fatalf("levels = %d, want 3", len(levels))
	}
}

func TestExecutionLevelsParallel(t *testing.T) {
	modules := []ModuleSpec{
		{Path: "a.go"},
		{Path: "b.go"},
		{Path: "c.go", DependsOn: []string{"a.go", "b.go"}},
	}
	levels, err := executionLevels(modules)
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 2 {
		t.Fatalf("levels = %d, want 2", len(levels))
	}
	if len(levels[0]) != 2 {
		t.Fatalf("level 0 = %d modules, want 2", len(levels[0]))
	}
}

func TestCreateIntegrationPR(t *testing.T) {
	engine := NewImplementEngine(nil, nil, nil, "/root", nil)
	report := ImplementReport{
		DesignDocPath: "docs/design/auth.md",
		Modules: []ImplementResult{
			{Module: ModuleSpec{Path: "internal/a.go"}},
			{Module: ModuleSpec{Path: "internal/b.go"}},
		},
	}
	pr := engine.CreateIntegrationPR(report, true)
	if pr == nil {
		t.Fatal("pr should not be nil")
	}
	if !pr.DryRun {
		t.Fatal("DryRun should be true")
	}
	if pr.Created {
		t.Fatal("Created should be false on dry run")
	}
	if len(pr.Modules) != 2 {
		t.Fatalf("modules = %d, want 2", len(pr.Modules))
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello-world", "hello_world"},
		{"a:b/c.d", "a_b_c_d"},
		{"__double__", "double"},
		{"", "generated"},
		{"   ", "generated"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeIdentifier(tt.input); got != tt.want {
				t.Fatalf("sanitizeIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToExportedIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello_world", "HelloWorld"},
		{"", "Generated"},
		{"a", "A"},
		{"already_upper_A", "AlreadyUpperA"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := toExportedIdentifier(tt.input); got != tt.want {
				t.Fatalf("toExportedIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRenderTestSkeleton(t *testing.T) {
	goTest := renderTestSkeleton("go", "test:handler")
	if len(goTest) == 0 {
		t.Fatal("go test skeleton should not be empty")
	}
	pyTest := renderTestSkeleton("python", "test:handler")
	if len(pyTest) == 0 {
		t.Fatal("python test skeleton should not be empty")
	}
}

func TestInferTestLanguage(t *testing.T) {
	if got := inferTestLanguage("pytest:auth"); got != "python" {
		t.Fatalf("got %q, want python", got)
	}
	if got := inferTestLanguage("test:go-auth"); got != "go" {
		t.Fatalf("got %q, want go", got)
	}
}

func TestDefaultTestPath(t *testing.T) {
	got := defaultTestPath("internal/handler.go", "test:x", "go")
	if got != filepath.Join("internal", "handler_test.go") {
		t.Fatalf("go path = %q", got)
	}
	got = defaultTestPath("internal/handler.py", "test:x", "python")
	if got != filepath.Join("internal", "test_handler.py") {
		t.Fatalf("python path = %q", got)
	}
	// Non-.py module with python lang
	got = defaultTestPath("internal/handler.go", "test:x", "python")
	if got != filepath.Join("tests", "handler_test.py") {
		t.Fatalf("non-py python path = %q", got)
	}
}

func TestLoadDesignDocSpec(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "design.md")
	if err := os.WriteFile(path, []byte(`---
codd:
  node_id: detail:auth
  title: auth
  review_required: approve
  modules:
    - internal/auth.go
  verified_by:
    - test:auth
---
# Auth
`), 0o644); err != nil {
		t.Fatal(err)
	}

	spec, err := loadDesignDocSpec(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.NodeID != "detail:auth" {
		t.Fatalf("NodeID = %q", spec.NodeID)
	}
	if len(spec.Modules) != 1 {
		t.Fatalf("modules = %d", len(spec.Modules))
	}
	if len(spec.VerifiedBy) != 1 {
		t.Fatalf("verified_by = %d", len(spec.VerifiedBy))
	}
}

func TestLoadDesignDocSpecNoFrontmatter(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.md")
	if err := os.WriteFile(path, []byte("no frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := loadDesignDocSpec(path)
	if err == nil {
		t.Fatal("should error without frontmatter")
	}
}

func TestLoadDesignDocSpecNoModules(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "no-mod.md")
	if err := os.WriteFile(path, []byte("---\ncodd:\n  node_id: x\n  title: t\n---\n# T\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := loadDesignDocSpec(path)
	if err == nil {
		t.Fatal("should error without modules")
	}
}

func TestLoadDesignDocSpecNotFound(t *testing.T) {
	_, err := loadDesignDocSpec("/nonexistent/design.md")
	if err == nil {
		t.Fatal("should error on missing file")
	}
}

func TestParseDesignFrontmatterInlineFields(t *testing.T) {
	content := `---
node_id: inline:test
title: Inline Test
modules:
  - inline/mod.go
---
# Test
`
	spec, err := parseDesignFrontmatter(content)
	if err != nil {
		t.Fatal(err)
	}
	if spec.NodeID != "inline:test" {
		t.Fatalf("NodeID = %q, want inline:test", spec.NodeID)
	}
}

func TestBuildImplementPrompt(t *testing.T) {
	design := designDocSpec{Raw: "design content"}
	spec := ModuleSpec{
		Path:          "internal/a.go",
		DesignSection: "section content",
		DependsOn:     []string{"internal/b.go"},
		TestSpec:      "test:a",
	}
	prompt := buildImplementPrompt(design, spec)
	if len(prompt) == 0 {
		t.Fatal("prompt should not be empty")
	}
}

func TestNormalizeReviewLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"auto", "auto"},
		{"review", "review"},
		{"approve", "approve"},
		{"APPROVE", "approve"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeReviewLevel(tt.input); got != tt.want {
				t.Fatalf("normalizeReviewLevel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExecuteEmptyDesignPath(t *testing.T) {
	engine := NewImplementEngine(nil, nil, nil, "/root", nil)
	_, err := engine.Execute(context.TODO(), ImplementRequest{})
	if err == nil {
		t.Fatal("empty design path should error")
	}
}

func TestMaxReviewLevel(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want string
	}{
		{"both empty", "", "", ""},
		{"a empty b review", "", "review", "review"},
		{"a auto b review", "auto", "review", "review"},
		{"a review b auto", "review", "auto", "review"},
		{"a review b approve", "review", "approve", "approve"},
		{"a approve b review", "approve", "review", "approve"},
		{"same level", "review", "review", "review"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maxReviewLevel(tt.a, tt.b)
			if got != tt.want {
				t.Fatalf("maxReviewLevel(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDetermineReviewLevelExplicitOverride(t *testing.T) {
	got := DetermineReviewLevel(ModuleSpec{Path: "internal/core.go", ReviewRequired: "approve"}, "auto", 1)
	if got != "approve" {
		t.Fatalf("explicit approve should override: got %q", got)
	}
}

func TestDetermineReviewLevelEmptyDefault(t *testing.T) {
	got := DetermineReviewLevel(ModuleSpec{Path: "internal/core.go"}, "", 1)
	if got != "review" {
		t.Fatalf("empty default should be review: got %q", got)
	}
}

func TestGenerateModulePythonOutput(t *testing.T) {
	root := t.TempDir()
	designPath := filepath.Join(root, "docs", "design", "py.md")
	mustWriteImplementFile(t, designPath, `---
codd:
  node_id: detail:py
  title: py
  modules:
    - internal/handler.py
  verified_by:
    - pytest:handler
---
# py
`)
	engine := NewImplementEngine(&fakeImplementProvider{}, nil, nil, root, nil)
	report, err := engine.Execute(context.TODO(), ImplementRequest{DesignDocPath: designPath})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary["completed"] != 1 {
		t.Fatalf("completed = %d", report.Summary["completed"])
	}
}

func TestExecuteWithCreatePR(t *testing.T) {
	root := t.TempDir()
	designPath := filepath.Join(root, "docs", "design", "pr.md")
	mustWriteImplementFile(t, designPath, `---
codd:
  node_id: detail:pr
  title: pr
  modules:
    - internal/mod.go
---
# pr
`)
	engine := NewImplementEngine(&fakeImplementProvider{}, nil, nil, root, nil)
	report, err := engine.Execute(context.TODO(), ImplementRequest{DesignDocPath: designPath, CreatePR: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.IntegrationPR == nil {
		t.Fatal("PR should be created")
	}
}

func TestDefaultTestPathEmptyBase(t *testing.T) {
	got := defaultTestPath("", "test:gen", "go")
	if got == "" {
		t.Fatal("should not be empty")
	}
}

func TestToExportedIdentifierEmpty(t *testing.T) {
	got := toExportedIdentifier("_")
	if got == "" {
		t.Fatal("should not be empty")
	}
}
