package cmd

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestImpactCmdNodeFlagJSON(t *testing.T) {
	rootDir := t.TempDir()
	mustWrite(t, filepath.Join(rootDir, ".github", "teraflow.yml"), "name: test\n")
	mustWrite(t, filepath.Join(rootDir, ".teraflow", "index.yml"), `version: "1"
entries:
  - node_id: req:a
    title: "A"
    path: docs/a.md
  - node_id: design:b
    title: "B"
    path: docs/b.md
    depends_on: [req:a]
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", filepath.Join(rootDir, ".github", "teraflow.yml"), "--format", "json", "impact", "--node", "req:a"})

	if err := root.Execute(); err != nil {
		t.Fatalf("impact --node failed: %v", err)
	}
	if !strings.Contains(out.String(), `"changed_node":"req:a"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestImpactCmdOutputAlias(t *testing.T) {
	rootDir := t.TempDir()
	mustWrite(t, filepath.Join(rootDir, ".github", "teraflow.yml"), "name: test\n")
	mustWrite(t, filepath.Join(rootDir, ".teraflow", "index.yml"), `version: "1"
entries:
  - node_id: req:a
    title: "A"
    path: docs/a.md
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", filepath.Join(rootDir, ".github", "teraflow.yml"), "impact", "--node", "req:a", "--output", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("impact --output json failed: %v", err)
	}
	if !strings.Contains(out.String(), `"changed_node":"req:a"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestImpactApplyCreatesIssuesFromNDJSON(t *testing.T) {
	oldExec := impactExecCommand
	oldLookPath := impactLookPath
	impactExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'https://github.com/acme/teraflow/issues/123\\n'")
	}
	impactLookPath = func(file string) (string, error) { return "/usr/bin/gh", nil }
	t.Cleanup(func() {
		impactExecCommand = oldExec
		impactLookPath = oldLookPath
	})

	rootDir := t.TempDir()
	mustWrite(t, filepath.Join(rootDir, ".github", "teraflow.yml"), "name: test\n")
	resultPath := filepath.Join(rootDir, "impact-results.ndjson")
	mustWrite(t, resultPath, `{"changed_node":"req:a","review_needed":["design:b"],"regen_required":["impl:c"]}
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", filepath.Join(rootDir, ".github", "teraflow.yml"), "--format", "json", "impact", "--apply", resultPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("impact --apply failed: %v", err)
	}
	if !strings.Contains(out.String(), `"band":"amber"`) || !strings.Contains(out.String(), `"band":"gray"`) {
		t.Fatalf("expected amber/gray issue records, got: %s", out.String())
	}
}
