package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/agent"
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

func TestAgentAssignMissingInput(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	root := newRootCmd("test")
	root.SetArgs([]string{"agent", "assign", "--type", "review"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when input is missing")
	}
	if !strings.Contains(err.Error(), "input required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentAssignInputFileReadError(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	root := newRootCmd("test")
	root.SetArgs([]string{"agent", "assign", "--type", "review", "--input-file", "missing.txt"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when input file cannot be read")
	}
	if !strings.Contains(err.Error(), "read input file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentAssignInputFileUnknownType(t *testing.T) {
	tmp := t.TempDir()
	inputFile := filepath.Join(tmp, "input.txt")
	if err := os.WriteFile(inputFile, []byte("hello from file"), 0o644); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	root := newRootCmd("test")
	root.SetArgs([]string{
		"agent", "assign",
		"--type", "unknown",
		"--input-file", inputFile,
		"--api-key", "dummy",
		"--trust-level=",
	})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected unknown type error")
	}
	if !strings.Contains(err.Error(), "unknown agent type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentAssignSuccessTextAndJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"generated output"}],"usage":{"input_tokens":2,"output_tokens":3}}`))
	}))
	t.Cleanup(server.Close)

	oldBaseURL := agent.BaseURL
	agent.BaseURL = server.URL
	t.Cleanup(func() { agent.BaseURL = oldBaseURL })

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"agent", "assign", "--type", "review", "--api-key", "dummy", "request"})
	if err := root.Execute(); err != nil {
		t.Fatalf("agent assign text failed: %v", err)
	}

	textOut := out.String()
	if !strings.Contains(textOut, "Agent: review") || !strings.Contains(textOut, "generated output") {
		t.Fatalf("unexpected text output: %s", textOut)
	}
	if !strings.Contains(textOut, "Tokens used: 5") {
		t.Fatalf("expected token usage in output: %s", textOut)
	}

	out.Reset()
	root = newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--format", "json", "agent", "assign", "--type", "review", "--api-key", "dummy", "request"})
	if err := root.Execute(); err != nil {
		t.Fatalf("agent assign json failed: %v", err)
	}

	jsonOut := out.String()
	if !strings.Contains(jsonOut, `"type": "review"`) || !strings.Contains(jsonOut, `"tokens_used": 5`) {
		t.Fatalf("unexpected json output: %s", jsonOut)
	}
}
