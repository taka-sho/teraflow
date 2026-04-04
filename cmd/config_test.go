package cmd

import (
	"bytes"
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
