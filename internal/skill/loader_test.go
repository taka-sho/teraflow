package skill_test

import (
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
