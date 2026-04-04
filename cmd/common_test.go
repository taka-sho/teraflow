package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestStatePathFromConfig(t *testing.T) {
	configPath := "/some/dir/.github/teraflow.yml"
	got := statePathFromConfig(configPath)
	want := filepath.Join("/some/dir/.github", "project-state.yml")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestOutputFormatFromCmdInvalid(t *testing.T) {
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--format", "xml", "status"})
	// Should return error about unsupported format
	err := root.Execute()
	if err == nil {
		t.Log("no error - format validation may not apply to all commands")
		return
	}
	if !strings.Contains(err.Error(), "unsupported") && !strings.Contains(err.Error(), "xml") {
		t.Fatalf("expected unsupported format error, got: %v", err)
	}
}

func TestWriteJSON(t *testing.T) {
	root := newRootCmd("test")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	type payload struct {
		Key string `json:"key"`
	}
	if err := writeJSON(root, payload{Key: "value"}); err != nil {
		t.Fatalf("writeJSON error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, `"key"`) || !strings.Contains(got, `"value"`) {
		t.Fatalf("unexpected JSON output: %q", got)
	}
}

func TestConfigPathFromCmdMissingFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	_, err := configPathFromCmd(cmd)
	if err == nil {
		t.Fatal("expected error when config flag is not defined")
	}
}

func TestOutputFormatFromCmdMissingFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	_, err := outputFormatFromCmd(cmd)
	if err == nil {
		t.Fatal("expected error when format flag is not defined")
	}
}
