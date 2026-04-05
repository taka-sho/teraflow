package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfigTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestLoad(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".github", "teraflow.yml")
	writeConfigTestFile(t, path, `version: "1"
project:
  name: "teraflow"
  description: "desc"
  repository: "https://example.com/repo.git"
confirmation:
  trigger: "確定"
  req_trigger: "要求確定"
constraints:
  - type: required_docs
    value: docs/requirements.md
    severity: block
ai:
  default_provider: anthropic
harness:
  score_threshold: 70
  auto_issue: false
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Version != "1" || cfg.Project.Name != "teraflow" || cfg.AI.DefaultProvider != "anthropic" || cfg.Harness.ScoreThreshold != 70 {
		t.Fatalf("unexpected config values: %+v", cfg)
	}
	if len(cfg.Constraints) != 1 || cfg.Constraints[0].Type != "required_docs" {
		t.Fatalf("constraints not loaded: %+v", cfg.Constraints)
	}
}

func TestGetSetValue(t *testing.T) {
	cfg := &TeraflowConfig{}

	pairs := map[string]string{
		"project.name":            "new-name",
		"project.description":     "new-desc",
		"project.repository":      "https://example.com/new.git",
		"ai.default_provider":     "openai",
		"harness.score_threshold": "88",
	}

	for key, value := range pairs {
		if err := SetValue(cfg, key, value); err != nil {
			t.Fatalf("SetValue(%s) returned error: %v", key, err)
		}
		got, err := GetValue(cfg, key)
		if err != nil {
			t.Fatalf("GetValue(%s) returned error: %v", key, err)
		}
		if got != value {
			t.Fatalf("value mismatch for %s: got %q, want %q", key, got, value)
		}
	}
}

func TestGetValue_invalid_key(t *testing.T) {
	cfg := &TeraflowConfig{}
	if _, err := GetValue(cfg, "unknown.key"); err == nil {
		t.Fatalf("expected error for unknown key")
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/teraflow.yml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "invalid.yml")
	writeConfigTestFile(t, path, ":\n  bad: [\nbroken yaml\n")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestSaveFailsOnBadPath(t *testing.T) {
	err := Save("/nonexistent/dir/teraflow.yml", &TeraflowConfig{Version: "1"})
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestSave(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".github", "teraflow.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := &TeraflowConfig{
		Version: "1",
	}
	cfg.Project.Name = "saved-project"
	cfg.AI.DefaultProvider = "openai"

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after Save returned error: %v", err)
	}
	if loaded.Project.Name != "saved-project" {
		t.Fatalf("unexpected project name: %q", loaded.Project.Name)
	}
	if loaded.AI.DefaultProvider != "openai" {
		t.Fatalf("unexpected AI provider: %q", loaded.AI.DefaultProvider)
	}
}

func TestSetValueInvalidInt(t *testing.T) {
	cfg := &TeraflowConfig{}
	err := SetValue(cfg, "harness.score_threshold", "not-a-number")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveProviderForType_AssignmentOverride(t *testing.T) {
	cfg := &TeraflowConfig{
		AI: AICfg{DefaultProvider: "anthropic"},
		Assignments: map[string]AssignmentConfig{
			"implement": {
				Provider: "openai",
				Model:    "gpt-4o",
			},
		},
	}

	provider, model := ResolveProviderForType(cfg, "implement")
	if provider != "openai" || model != "gpt-4o" {
		t.Fatalf("unexpected result: provider=%q model=%q", provider, model)
	}
}

func TestResolveProviderForType_AssignmentDefaultModel(t *testing.T) {
	cfg := &TeraflowConfig{
		Assignments: map[string]AssignmentConfig{
			"ci-fix": {
				Provider: "openai",
			},
		},
	}

	provider, model := ResolveProviderForType(cfg, "ci-fix")
	if provider != "openai" || model != "gpt-4o-mini" {
		t.Fatalf("unexpected result: provider=%q model=%q", provider, model)
	}
}

func TestResolveProviderForType_FallbackToDefaultProvider(t *testing.T) {
	cfg := &TeraflowConfig{
		AI: AICfg{DefaultProvider: "openai"},
	}

	provider, model := ResolveProviderForType(cfg, "review")
	if provider != "openai" || model != "gpt-4o-mini" {
		t.Fatalf("unexpected result: provider=%q model=%q", provider, model)
	}
}

func TestResolveProviderForType_FallbackToAnthropic(t *testing.T) {
	provider, model := ResolveProviderForType(&TeraflowConfig{}, "review")
	if provider != "anthropic" || model != "claude-haiku-4-5-20251001" {
		t.Fatalf("unexpected result: provider=%q model=%q", provider, model)
	}
}

func TestGetAPIKeyEnvName(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{name: "anthropic", provider: "anthropic", want: "ANTHROPIC_API_KEY"},
		{name: "openai", provider: "openai", want: "OPENAI_API_KEY"},
		{name: "claude-code", provider: "claude-code", want: ""},
		{name: "custom", provider: "custom", want: ""},
		{name: "unknown falls back to anthropic", provider: "other", want: "ANTHROPIC_API_KEY"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := GetAPIKeyEnvName(tc.provider); got != tc.want {
				t.Fatalf("GetAPIKeyEnvName(%q)=%q, want %q", tc.provider, got, tc.want)
			}
		})
	}
}
