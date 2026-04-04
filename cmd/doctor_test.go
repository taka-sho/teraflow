package cmd

import (
	"path/filepath"
	"testing"
)

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
