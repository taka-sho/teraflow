package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestHookList(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, `version: "1"
hooks:
  on_push:
    - action: index_update
  on_discussion_comment:
    - action: respond
      conditions:
        labels: ["urgent"]
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "hook", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("hook list failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "on_push") {
		t.Fatalf("expected on_push in output, got:\n%s", got)
	}
	if !strings.Contains(got, "actions=1") {
		t.Fatalf("expected actions count in output, got:\n%s", got)
	}
}

func TestHookValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, ".github", "teraflow.yml")
		writeCmdTestFile(t, configPath, `version: "1"
hooks:
  on_push:
    - action: index_update
`)

		root := newRootCmd("test")
		root.SetArgs([]string{"--config", configPath, "hook", "validate"})
		if err := root.Execute(); err != nil {
			t.Fatalf("expected valid config, got: %v", err)
		}
	})

	t.Run("unknown event", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, ".github", "teraflow.yml")
		writeCmdTestFile(t, configPath, `version: "1"
hooks:
  on_unknown_event:
    - action: respond
`)

		root := newRootCmd("test")
		root.SetArgs([]string{"--config", configPath, "hook", "validate"})
		err := root.Execute()
		if err == nil {
			t.Fatal("expected validation error")
		}
		if !strings.Contains(err.Error(), "unknown event") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestHookRunDryRun(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, `version: "1"
hooks:
  on_discussion_comment:
    - action: respond
      skill: requirements
      conditions:
        categories: ["bug"]
        labels: ["urgent"]
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{
		"--config", configPath,
		"--format", "json",
		"hook", "run", "on_discussion_comment",
		"--author", "alice",
		"--category", "bug",
		"--labels", "urgent,triage",
		"--discussion-id", "D_kwDOXYZ",
		"--input", "please help",
		"--dry-run",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("hook run dry-run failed: %v", err)
	}

	var got struct {
		Event   string `json:"Event"`
		DryRun  bool   `json:"DryRun"`
		Matched []any  `json:"Matched"`
		Results []struct {
			Action  string `json:"Action"`
			Success bool   `json:"Success"`
			Message string `json:"Message"`
		} `json:"Results"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v\noutput=%s", err, out.String())
	}

	if got.Event != "on_discussion_comment" {
		t.Fatalf("unexpected event: %s", got.Event)
	}
	if !got.DryRun {
		t.Fatalf("expected dry run true")
	}
	if len(got.Matched) != 1 {
		t.Fatalf("expected 1 matched action, got %d", len(got.Matched))
	}
	if len(got.Results) != 1 || !got.Results[0].Success {
		t.Fatalf("expected one successful result, got %+v", got.Results)
	}
	if !strings.Contains(got.Results[0].Message, "would run:") {
		t.Fatalf("expected dry-run message, got %q", got.Results[0].Message)
	}
}
