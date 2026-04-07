package cmd

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteNonAppErrorOutputsToStderr(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join(".."))
	run := exec.Command("go", "run", ".", "unknown-subcommand")
	run.Dir = repoRoot

	var stderr bytes.Buffer
	run.Stderr = &stderr
	run.Stdout = &bytes.Buffer{}

	err := run.Run()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	if _, ok := err.(*exec.ExitError); !ok {
		t.Fatalf("expected ExitError, got %T (%v)", err, err)
	}

	got := stderr.String()
	if !strings.Contains(got, "Error:") {
		t.Fatalf("expected Error prefix in stderr, got: %s", got)
	}
	if !strings.Contains(got, "unknown command") {
		t.Fatalf("expected unknown command in stderr, got: %s", got)
	}
}
