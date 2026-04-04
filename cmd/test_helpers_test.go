package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCmdTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func setupTestProjectState(t *testing.T, root, stage, phase string) string {
	t.Helper()
	configPath := filepath.Join(root, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, "version: \"1\"\n")
	writeCmdTestFile(t, filepath.Join(root, ".github", "project-state.yml"), "project:\n  name: \"test\"\nlifecycle:\n  current_stage: \""+stage+"\"\nphases:\n  current: \""+phase+"\"\n")
	return configPath
}
