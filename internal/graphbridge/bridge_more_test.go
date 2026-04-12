package graphbridge_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/graphbridge"
)

func TestExecute_WithLegacyOKProtocol(t *testing.T) {
	tmpDir := t.TempDir()
	writeFakePython(t, tmpDir, `#!/bin/sh
cat >/dev/null
echo '{"ok":true,"data":{"result":"success"}}'
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	resp, err := bridge.Execute(graphbridge.Request{Command: "search", Args: map[string]any{"q": "test"}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !resp.OK {
		t.Fatal("OK should be true")
	}
	if resp.Data["result"] != "success" {
		t.Fatalf("unexpected data: %v", resp.Data)
	}
}

func TestExecute_WithLegacyOKFalse(t *testing.T) {
	tmpDir := t.TempDir()
	writeFakePython(t, tmpDir, `#!/bin/sh
cat >/dev/null
echo '{"ok":false,"error":"something wrong","code":"bad_input"}'
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	_, err := bridge.Execute(graphbridge.Request{Command: "query", Args: map[string]any{}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad_input") || !strings.Contains(err.Error(), "something wrong") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_WithLegacyOKFalseNoCode(t *testing.T) {
	tmpDir := t.TempDir()
	writeFakePython(t, tmpDir, `#!/bin/sh
cat >/dev/null
echo '{"ok":false,"error":"generic failure"}'
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	_, err := bridge.Execute(graphbridge.Request{Command: "query", Args: map[string]any{}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "generic failure") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_WithErrorProtocolNoCode(t *testing.T) {
	tmpDir := t.TempDir()
	writeFakePython(t, tmpDir, `#!/bin/sh
cat >/dev/null
echo '{"error":"just an error"}'
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	_, err := bridge.Execute(graphbridge.Request{Command: "query", Args: map[string]any{}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "just an error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_SubprocessFailure(t *testing.T) {
	tmpDir := t.TempDir()
	writeFakePython(t, tmpDir, `#!/bin/sh
cat >/dev/null
echo "crash" >&2
exit 1
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	_, err := bridge.Execute(graphbridge.Request{Command: "query", Args: map[string]any{}})
	if err == nil {
		t.Fatal("expected error on subprocess failure")
	}
	if !strings.Contains(err.Error(), "subprocess failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	writeFakePython(t, tmpDir, `#!/bin/sh
cat >/dev/null
echo 'not json'
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	_, err := bridge.Execute(graphbridge.Request{Command: "query", Args: map[string]any{}})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
	if !strings.Contains(err.Error(), "unmarshal response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAvailable_WithoutPyprojectOrPython(t *testing.T) {
	tmpDir := t.TempDir()
	// Override PATH to empty to ensure python3 not found
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })

	bridge := graphbridge.New(tmpDir)
	if bridge.Available() {
		t.Error("Available() should return false without pyproject.toml or python3")
	}
}

func TestNew_ReturnsNonNil(t *testing.T) {
	bridge := graphbridge.New("")
	if bridge == nil {
		t.Fatal("New() should not return nil")
	}
}

func TestExecute_WithLegacyUnmarshalError(t *testing.T) {
	tmpDir := t.TempDir()
	// Return JSON where "ok" is present but response can't be unmarshaled to legacy Response properly
	// This triggers the legacy "ok" path but with invalid type for "ok" value
	writeFakePython(t, tmpDir, `#!/bin/sh
cat >/dev/null
echo '{"ok":"not-a-bool","error":"type error"}'
`)
	setPathForTest(t, tmpDir)

	bridge := graphbridge.New(tmpDir)
	_, err := bridge.Execute(graphbridge.Request{Command: "query", Args: map[string]any{}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAvailable_WithPyprojectInSubdir(t *testing.T) {
	tmpDir := t.TempDir()
	graphragDir := filepath.Join(tmpDir, "graphrag")
	if err := os.MkdirAll(graphragDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(graphragDir, "pyproject.toml"), []byte("[project]"), 0o644); err != nil {
		t.Fatal(err)
	}

	bridge := graphbridge.New(tmpDir)
	if !bridge.Available() {
		t.Error("should detect pyproject.toml")
	}
}
