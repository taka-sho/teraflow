package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestAgentStatus(t *testing.T) {
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"agent", "status"})

	if err := root.Execute(); err != nil {
		t.Fatalf("agent status: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "requirements") {
		t.Fatalf("expected agent types in output: %s", got)
	}
}

func TestAgentStatusJSON(t *testing.T) {
	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"agent", "status", "--format", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("agent status --format json: %v", err)
	}

	if !strings.Contains(out.String(), "available_types") {
		t.Fatalf("expected JSON output: %s", out.String())
	}
}

func TestAgentAssignMissingType(t *testing.T) {
	root := newRootCmd("test")
	root.SetArgs([]string{"agent", "assign", "some input"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --type is missing")
	}
}

func TestAgentAssignMissingAPIKey(t *testing.T) {
	root := newRootCmd("test")
	root.SetArgs([]string{"agent", "assign", "--type", "review", "some input"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when API key is missing")
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Fatalf("unexpected error: %v", err)
	}
}
