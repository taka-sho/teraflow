package pipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
