package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateWithExplicitVersion(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("v0.5.4")
	root.SetArgs([]string{"update", "--config", cfgPath, "--version", "v0.5.5"})
	if err := root.Execute(); err != nil {
		t.Fatalf("update: %v", err)
	}

	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(cfgData), "teraflow_version: v0.5.5") {
		t.Fatalf("config should contain teraflow_version: v0.5.5, got:\n%s", string(cfgData))
	}

	wfData, err := os.ReadFile(filepath.Join(tmp, ".github", "workflows", "teraflow-phase-gate.yml"))
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	if !strings.Contains(string(wfData), `go install "github.com/taka-sho/teraflow@v0.5.5"`) {
		t.Fatalf("workflow should be pinned to v0.5.5, got:\n%s", string(wfData))
	}
}

func TestUpdateFetchLatestTagWhenVersionNotSpecified(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	oldExec := ghExecCommand
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'v0.6.0\\n'")
	}
	t.Cleanup(func() { ghExecCommand = oldExec })

	root := newRootCmd("v0.5.4")
	root.SetArgs([]string{"update", "--config", cfgPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("update without --version: %v", err)
	}

	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(cfgData), "teraflow_version: v0.6.0") {
		t.Fatalf("expected fetched version in config, got:\n%s", string(cfgData))
	}
}

func TestSetupActionsUsesConfiguredVersionAndWarnsMismatch(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\nteraflow_version: v0.5.3\n")

	root := newRootCmd("v0.5.4")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"setup", "actions", "--config", cfgPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("setup actions: %v", err)
	}

	s := out.String()
	if !strings.Contains(s, "warning: running teraflow v0.5.4 but teraflow.yml specifies v0.5.3") {
		t.Fatalf("expected mismatch warning, got:\n%s", s)
	}
	if !strings.Contains(s, "hint: run 'teraflow update' to align versions") {
		t.Fatalf("expected mismatch hint, got:\n%s", s)
	}

	wfData, err := os.ReadFile(filepath.Join(tmp, ".github", "workflows", "teraflow-phase-gate.yml"))
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	if !strings.Contains(string(wfData), `go install "github.com/taka-sho/teraflow@v0.5.3"`) {
		t.Fatalf("workflow should use configured version, got:\n%s", string(wfData))
	}
}
