package graphbridge_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/taka-sho/teraflow/internal/graphbridge"
)

func TestAvailable_WithPyproject(t *testing.T) {
	tmpDir := t.TempDir()
	graphragDir := filepath.Join(tmpDir, "graphrag")
	if err := os.MkdirAll(graphragDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pyproject := filepath.Join(graphragDir, "pyproject.toml")
	if err := os.WriteFile(pyproject, []byte("[project]\nname = \"teraflow-graphrag\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	bridge := graphbridge.New(tmpDir)
	if !bridge.Available() {
		t.Error("Available() should return true when graphrag/pyproject.toml exists")
	}
}

func TestAvailable_WithoutPyproject(t *testing.T) {
	tmpDir := t.TempDir()
	bridge := graphbridge.New(tmpDir)

	// python3 -m teraflow_graphrag likely also fails in CI.
	// Just verify it doesn't panic.
	_ = bridge.Available()
}

func TestNew(t *testing.T) {
	bridge := graphbridge.New("/some/path")
	if bridge == nil {
		t.Error("New() should not return nil")
	}
}
