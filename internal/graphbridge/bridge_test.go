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

func TestAvailable_WithPythonVersionFallback(t *testing.T) {
	tmpDir := t.TempDir()
	writeFakePython(t, tmpDir, `#!/bin/sh
if [ "$3" = "--version" ]; then
  echo "teraflow_graphrag 0.1.0"
  exit 0
fi
exit 1
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	if !bridge.Available() {
		t.Fatal("Available() should return true when python module --version succeeds")
	}
}

func TestNew(t *testing.T) {
	bridge := graphbridge.New("/some/path")
	if bridge == nil {
		t.Error("New() should not return nil")
	}
}

func TestExecute_WithDirectPayloadProtocol(t *testing.T) {
	tmpDir := t.TempDir()
	writeFakePython(t, tmpDir, `#!/bin/sh
cat >/dev/null
echo '{"answer":"ok","mode":"local","sources":[]}'
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	resp, err := bridge.Execute(graphbridge.Request{
		Command: "query",
		Args: map[string]any{
			"query": "auth",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := resp.Data["answer"]; got != "ok" {
		t.Fatalf("unexpected answer: %v", got)
	}
}

func TestExecute_WithErrorCodeProtocol(t *testing.T) {
	tmpDir := t.TempDir()
	writeFakePython(t, tmpDir, `#!/bin/sh
cat >/dev/null
echo '{"error":"boom","code":"invalid_argument"}'
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	_, err := bridge.Execute(graphbridge.Request{Command: "query", Args: map[string]any{"query": ""}})
	if err == nil {
		t.Fatal("expected Execute() to fail")
	}
	if !strings.Contains(err.Error(), "invalid_argument") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_WithLegacyProtocol(t *testing.T) {
	tmpDir := t.TempDir()
	writeFakePython(t, tmpDir, `#!/bin/sh
cat >/dev/null
echo '{"ok":true,"data":{"answer":"legacy"}}'
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	resp, err := bridge.Execute(graphbridge.Request{Command: "query", Args: map[string]any{"query": "x"}})
	if err != nil {
		t.Fatalf("Execute legacy protocol failed: %v", err)
	}
	if got := resp.Data["answer"]; got != "legacy" {
		t.Fatalf("unexpected legacy response: %+v", resp.Data)
	}
}

func TestExecute_WithLegacyProtocolError(t *testing.T) {
	tmpDir := t.TempDir()
	writeFakePython(t, tmpDir, `#!/bin/sh
cat >/dev/null
echo '{"ok":false,"error":"legacy-fail","code":"bad_request"}'
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	_, err := bridge.Execute(graphbridge.Request{Command: "query", Args: map[string]any{"query": "x"}})
	if err == nil || !strings.Contains(err.Error(), "bad_request") {
		t.Fatalf("expected legacy error with code, got: %v", err)
	}
}

func TestExecute_SubprocessAndUnmarshalErrors(t *testing.T) {
	tmpDir := t.TempDir()
	writeFakePython(t, tmpDir, `#!/bin/sh
echo "boom" 1>&2
exit 1
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	if _, err := bridge.Execute(graphbridge.Request{Command: "query"}); err == nil || !strings.Contains(err.Error(), "subprocess failed") {
		t.Fatalf("expected subprocess failure, got: %v", err)
	}

	tmpDir2 := t.TempDir()
	writeFakePython(t, tmpDir2, `#!/bin/sh
cat >/dev/null
echo "not-json"
`)
	setPathForTest(t, tmpDir2)
	bridge = graphbridge.New(tmpDir2)
	if _, err := bridge.Execute(graphbridge.Request{Command: "query"}); err == nil || !strings.Contains(err.Error(), "unmarshal response") {
		t.Fatalf("expected unmarshal response error, got: %v", err)
	}
}

func writeFakePython(t *testing.T, dir string, script string) {
	t.Helper()
	pythonPath := filepath.Join(dir, "python3")
	if err := os.WriteFile(pythonPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake python: %v", err)
	}
}

func setPathForTest(t *testing.T, prependDir string) {
	t.Helper()
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", prependDir+":"+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
	})
}
