package graphbridge_test

import (
	"os"
	"path/filepath"
	"strings"
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

func TestExecute_Success(t *testing.T) {
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "teraflow_graphrag")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainPy := filepath.Join(moduleDir, "__main__.py")
	script := `import json
import sys
_ = json.loads(sys.stdin.read())
print(json.dumps({"ok": True, "data": {"node_count": 3, "edge_count": 2}}))
`
	if err := os.WriteFile(mainPy, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PYTHONPATH", tmpDir)

	bridge := graphbridge.New(tmpDir)
	resp, err := bridge.Execute(graphbridge.Request{
		Command: "build",
		Args: map[string]any{
			"project_root": tmpDir,
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp == nil || !resp.OK {
		t.Fatalf("Execute() response = %#v", resp)
	}
	if got := resp.Data["node_count"]; got != float64(3) {
		t.Fatalf("node_count = %v, want 3", got)
	}
}

func TestExecute_PythonErrorResponse(t *testing.T) {
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "teraflow_graphrag")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainPy := filepath.Join(moduleDir, "__main__.py")
	script := `import json
print(json.dumps({"ok": False, "error": "boom"}))
`
	if err := os.WriteFile(mainPy, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PYTHONPATH", tmpDir)

	bridge := graphbridge.New(tmpDir)
	_, err := bridge.Execute(graphbridge.Request{Command: "build", Args: map[string]any{}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "python error: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}
