package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	indexpkg "github.com/taka-sho/teraflow/internal/index"
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

func TestGraphSearchRequiresGraphRAG(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "graph", "search", "auth"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected graphrag availability error")
	}
	if !strings.Contains(err.Error(), "GraphRAG module not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphImpactFallsBackToCoDDWhenUnavailable(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "index.yml"), `version: "1"
generated_at: 2026-04-06T00:00:00Z
entries:
  - node_id: req-auth
    title: Auth Requirement
    path: docs/requirements/auth.md
    status: confirmed
    depends_on: ["design-auth"]
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "a"
    summary_available: false
  - node_id: design-auth
    title: Auth Design
    path: docs/design/auth.md
    status: review
    depends_on: []
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "b"
    summary_available: false
`)

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "graph", "impact", "design-auth", "--depth", "2"})
	if err := root.Execute(); err != nil {
		t.Fatalf("graph impact fallback error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"root_node_id":"design-auth"`) {
		t.Fatalf("unexpected graph impact output: %q", got)
	}
	if !strings.Contains(got, `"source":"codd"`) {
		t.Fatalf("expected codd source fallback, got: %q", got)
	}
	if !strings.Contains(got, "graphrag not available, showing CoDD explicit dependencies only") {
		t.Fatalf("expected fallback warning, got: %q", got)
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

func TestGraphBuildSearchImpactWithFakeGraphRAG(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, "graphrag", "pyproject.toml"), "[project]\nname = \"fake\"\n")

	fakeBin := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}
	fakePython := filepath.Join(fakeBin, "python3")
	script := `#!/bin/sh
cat >/dev/null
printf '{"ok":true,"data":{"answer":"ok","sources":[{"node_id":"n1","label":"Node1","source":"doc"}],"root_node_id":"req:a","total_count":1,"affected_nodes":[{"node_id":"design:b","depth":1,"edge_type":"DEPENDS_ON","source":"graphrag","label":"B"}]}}'
`
	if err := os.WriteFile(fakePython, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake python: %v", err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", fakeBin+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "graph", "build", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("graph build failed: %v", err)
	}
	if !strings.Contains(out.String(), `"answer":"ok"`) {
		t.Fatalf("unexpected build output: %s", out.String())
	}

	out.Reset()
	root = newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "graph", "search", "auth", "--mode", "global"})
	if err := root.Execute(); err != nil {
		t.Fatalf("graph search failed: %v", err)
	}
	if !strings.Contains(out.String(), "Answer: ok") || !strings.Contains(out.String(), "Sources:") {
		t.Fatalf("unexpected search output: %s", out.String())
	}

	out.Reset()
	root = newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "graph", "impact", "req:a", "--depth", "2"})
	if err := root.Execute(); err != nil {
		t.Fatalf("graph impact failed: %v", err)
	}
	if !strings.Contains(out.String(), "Total affected: 1") || !strings.Contains(out.String(), "design:b") {
		t.Fatalf("unexpected impact output: %s", out.String())
	}
}

func TestGraphSearchAndImpactValidation(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "graph", "search", "q", "--mode", "invalid"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "invalid mode") {
		t.Fatalf("expected invalid mode error, got: %v", err)
	}

	root = newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "graph", "impact", "node", "--depth", "0"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "depth must be >= 1") {
		t.Fatalf("expected depth validation error, got: %v", err)
	}
}

func TestAnalyzeCoDDImpactAndHelpers(t *testing.T) {
	idx := &indexpkg.Index{
		Entries: []indexpkg.Entry{
			{NodeID: "req:a", Title: "A"},
			{NodeID: "design:b", Title: "B", DependsOn: []string{"req:a"}},
			{NodeID: "impl:c", Title: "C", DependsOn: []string{"design:b"}},
		},
	}

	out := analyzeCoDDImpact(idx, "req:a", 2)
	if intFromMap(out, "total_count") != 2 {
		t.Fatalf("unexpected total_count: %+v", out)
	}
	if got := firstString(map[string]any{"x": 7}, "x"); got != "7" {
		t.Fatalf("firstString mismatch: %q", got)
	}
	if got := intFromMap(map[string]any{"n": "3"}, "n"); got != 3 {
		t.Fatalf("intFromMap mismatch: %d", got)
	}
	if got := intFromMap(map[string]any{"n": "bad"}, "n"); got != 0 {
		t.Fatalf("intFromMap bad string mismatch: %d", got)
	}
}
