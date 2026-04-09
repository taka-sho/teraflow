package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraphStatusTextAndJSON(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "index.yml"), `version: "1"
generated_at: 2026-04-06T00:00:00Z
entries:
  - node_id: req-auth
    title: Auth
    path: docs/requirements/auth.md
    status: confirmed
    depends_on: ["design-auth"]
    tags: ["core", "auth"]
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "a"
    summary_available: false
  - node_id: design-auth
    title: Auth Design
    path: docs/design/auth.md
    status: review
    depends_on: []
    tags: ["design"]
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "b"
    summary_available: false
`)

	var textOut bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&textOut)
	root.SetErr(&textOut)
	root.SetArgs([]string{"--config", cfgPath, "graph", "status"})
	if err := root.Execute(); err != nil {
		t.Fatalf("graph status text error = %v", err)
	}
	gotText := textOut.String()
	if !strings.Contains(gotText, "Nodes: 2") || !strings.Contains(gotText, "Edges: 1") || !strings.Contains(gotText, "Isolated nodes: 0") {
		t.Fatalf("unexpected graph status text output: %q", gotText)
	}

	var jsonOut bytes.Buffer
	root = newRootCmd("test")
	root.SetOut(&jsonOut)
	root.SetErr(&jsonOut)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "graph", "status"})
	if err := root.Execute(); err != nil {
		t.Fatalf("graph status json error = %v", err)
	}
	gotJSON := jsonOut.String()
	if !strings.Contains(gotJSON, `"total_nodes":2`) || !strings.Contains(gotJSON, `"total_edges":1`) {
		t.Fatalf("unexpected graph status json output: %q", gotJSON)
	}
}

func TestGraphCheckDetectsIssues(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "index.yml"), `version: "1"
generated_at: 2026-04-06T00:00:00Z
entries:
  - node_id: A
    title: A
    path: docs/a.md
    status: confirmed
    depends_on: ["B", "MISSING"]
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "a"
    summary_available: false
  - node_id: B
    title: B
    path: docs/b.md
    status: draft
    depends_on: ["A"]
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "b"
    summary_available: false
`)

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "graph", "check"})
	if err := root.Execute(); err != nil {
		t.Fatalf("graph check error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "[error] broken_ref") || !strings.Contains(got, "[warning] status_conflict") || !strings.Contains(got, "[error] cyclic_dep") {
		t.Fatalf("unexpected graph check output: %q", got)
	}
}

func TestGraphExportMermaidAndDot(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "index.yml"), `version: "1"
generated_at: 2026-04-06T00:00:00Z
entries:
  - node_id: A
    title: A
    path: docs/a.md
    status: confirmed
    depends_on: ["B"]
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "a"
    summary_available: false
  - node_id: B
    title: B
    path: docs/b.md
    status: review
    depends_on: []
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "b"
    summary_available: false
`)

	var mermaidOut bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&mermaidOut)
	root.SetErr(&mermaidOut)
	root.SetArgs([]string{"--config", cfgPath, "graph", "export", "--format", "mermaid"})
	if err := root.Execute(); err != nil {
		t.Fatalf("graph export mermaid error = %v", err)
	}
	if got := mermaidOut.String(); !strings.Contains(got, "flowchart TD") || !strings.Contains(got, "A") || !strings.Contains(got, "B") {
		t.Fatalf("unexpected mermaid output: %q", got)
	}

	dotPath := filepath.Join(tmp, "graph.dot")
	var dotOut bytes.Buffer
	root = newRootCmd("test")
	root.SetOut(&dotOut)
	root.SetErr(&dotOut)
	root.SetArgs([]string{"--config", cfgPath, "graph", "export", "--format", "dot", "--output", dotPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("graph export dot error = %v", err)
	}
	if !strings.Contains(dotOut.String(), "Exported graph to") {
		t.Fatalf("unexpected dot export output: %q", dotOut.String())
	}
	data := mustRead(t, dotPath)
	if !strings.Contains(string(data), "digraph G") || !strings.Contains(string(data), "\"A\" -> \"B\"") {
		t.Fatalf("unexpected dot file content: %q", string(data))
	}
}

func TestGraphExportInvalidFormat(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "index.yml"), `version: "1"
generated_at: 2026-04-06T00:00:00Z
entries: []
`)

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "graph", "export", "--format", "svg"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected invalid format error")
	}
	if !strings.Contains(err.Error(), "unsupported export format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphCommandsIndexMissing(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "graph", "status"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected missing index error")
	}
	if !strings.Contains(err.Error(), "read index") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}
