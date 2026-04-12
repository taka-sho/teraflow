package skill_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/taka-sho/teraflow/internal/skill"
)

func TestFileLoaderLoadAll(t *testing.T) {
	loader := skill.NewFileLoader(filepath.Join("testdata", "skills"))

	skills, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("LoadAll() count = %d, want 2", len(skills))
	}
}

func TestFileLoaderLoadByName(t *testing.T) {
	loader := skill.NewFileLoader(filepath.Join("testdata", "skills"))

	got, err := loader.LoadByName("review-skill")
	if err != nil {
		t.Fatalf("LoadByName() error = %v", err)
	}
	if got.Name != "review-skill" {
		t.Fatalf("LoadByName() name = %q, want review-skill", got.Name)
	}
}

func TestFileLoaderValidateRequiredFields(t *testing.T) {
	loader := skill.NewFileLoader(filepath.Join("testdata", "skills"))

	_, err := loader.Load(filepath.Join("testdata", "invalid", "invalid-missing-name.yml"))
	if err == nil {
		t.Fatal("Load() expected validation error, got nil")
	}
}

func TestFileLoaderFallsBackToEmbeddedWhenSkillDirMissing(t *testing.T) {
	loader := skill.NewFileLoader(filepath.Join("testdata", "missing"))

	got, err := loader.LoadByName("code-review")
	if err != nil {
		t.Fatalf("LoadByName() with missing dir error = %v", err)
	}
	if got.Name != "code-review" {
		t.Fatalf("LoadByName() name = %q, want code-review", got.Name)
	}
}

func TestFileLoaderPrefersFilesystemOverEmbedded(t *testing.T) {
	tmp := t.TempDir()
	customDir := filepath.Join(tmp, "skills")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("mkdir custom skills dir: %v", err)
	}
	customSkill := `name: code-review
version: "1"
description: "local override"
prompts:
  system: LOCAL_ONLY
`
	if err := os.WriteFile(filepath.Join(customDir, "code-review.yml"), []byte(customSkill), 0o644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	loader := skill.NewFileLoader(customDir)
	got, err := loader.LoadByName("code-review")
	if err != nil {
		t.Fatalf("LoadByName() error = %v", err)
	}
	if got.Description != "local override" {
		t.Fatalf("expected filesystem skill, got description %q", got.Description)
	}
}
