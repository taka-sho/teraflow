package cmd

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestLabelList(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.AddCommand(newLabelCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"label", "list", "--config", cfgPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("label list failed: %v", err)
	}

	got := out.String()
	checks := []string{
		"Default teraflow labels:",
		"stage:initial-dev",
		"stage:release",
		"phase:requirements",
		"phase:basic-design",
		"phase:detailed-design",
		"phase:implementation",
		"phase:testing",
		"phase:integration",
		"type:rework",
		"type:incident",
		"type:confirmed",
	}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestLabelSyncNoGH(t *testing.T) {
	oldLookPath := ghLookPath
	ghLookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
	})

	root := newRootCmd("test")
	root.AddCommand(newLabelCmd())
	root.SetArgs([]string{"label", "sync"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "E5001") {
		t.Fatalf("expected E5001, got: %v", err)
	}
}
