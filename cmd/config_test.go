package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigShow(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
project:
  name: "my-project"
  description: ""
  repository: ""
ai:
  default_provider: anthropic
harness:
  score_threshold: 70
`)

	command := newRootCmd("test")
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"config", "show", "--config", cfgPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("config show execute error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "project.name:") || !strings.Contains(got, "my-project") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "harness.score_threshold:") || !strings.Contains(got, "70") {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestConfigShowJSON(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
project:
  name: "json-project"
  description: "desc"
  repository: ""
ai:
  default_provider: anthropic
harness:
  score_threshold: 70
`)

	command := newRootCmd("test")
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"--format", "json", "config", "show", "--config", cfgPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("config show --format json execute error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"project.name"`) || !strings.Contains(got, "json-project") {
		t.Fatalf("unexpected JSON output: %s", got)
	}
}

func TestConfigSetInvalidKey(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	command := newRootCmd("test")
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"config", "set", "unknown.key", "value", "--config", cfgPath})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigSet(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
project:
  name: "my-project"
  description: ""
  repository: ""
ai:
  default_provider: anthropic
harness:
  score_threshold: 70
`)

	setCmd := newRootCmd("test")
	setCmd.SetArgs([]string{"config", "set", "project.description", "注文管理システム", "--config", cfgPath})
	if err := setCmd.Execute(); err != nil {
		t.Fatalf("config set execute error: %v", err)
	}

	showCmd := newRootCmd("test")
	var out bytes.Buffer
	showCmd.SetOut(&out)
	showCmd.SetErr(&out)
	showCmd.SetArgs([]string{"config", "show", "--config", cfgPath})
	if err := showCmd.Execute(); err != nil {
		t.Fatalf("config show execute error: %v", err)
	}

	if !strings.Contains(out.String(), "注文管理システム") {
		t.Fatalf("updated value not found in show output:\n%s", out.String())
	}
}

func TestConfigShowInvalidFormat(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	command := newRootCmd("test")
	command.SetArgs([]string{"--format", "xml", "config", "show", "--config", cfgPath})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigSetMissingConfig(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "missing.yml")

	command := newRootCmd("test")
	command.SetArgs([]string{"config", "set", "project.name", "myproject", "--config", cfgPath})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected error for missing config")
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigShowMissingConfig(t *testing.T) {
	tmp := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	command := newRootCmd("test")
	command.SetArgs([]string{"config", "show"})

	err = command.Execute()
	if err == nil {
		t.Fatal("expected error for default missing config")
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigSetInvalidThresholdValue(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
project:
  name: "my-project"
  description: ""
  repository: ""
ai:
  default_provider: anthropic
harness:
  score_threshold: 70
`)

	command := newRootCmd("test")
	command.SetArgs([]string{"config", "set", "harness.score_threshold", "NaN", "--config", cfgPath})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected error for invalid score threshold")
	}
	if !strings.Contains(err.Error(), "invalid value for harness.score_threshold") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigShowConfigFlagError(t *testing.T) {
	cmd := newConfigShowCmd()
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when config flag is not defined on root")
	}
	if !strings.Contains(err.Error(), "flag accessed but not defined: config") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigSetConfigFlagError(t *testing.T) {
	cmd := newConfigSetCmd()
	err := cmd.RunE(cmd, []string{"project.name", "x"})
	if err == nil {
		t.Fatal("expected error when config flag is not defined on root")
	}
	if !strings.Contains(err.Error(), "flag accessed but not defined: config") {
		t.Fatalf("unexpected error: %v", err)
	}
}
