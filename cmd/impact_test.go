package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/pipeline"
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

func TestImpactCmdValidationErrors(t *testing.T) {
	rootDir := t.TempDir()
	cfg := filepath.Join(rootDir, ".github", "teraflow.yml")
	mustWrite(t, cfg, "name: test\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfg, "impact", "--output", "yaml", "--node", "req:a"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "unsupported --output format") {
		t.Fatalf("expected unsupported output format error, got: %v", err)
	}

	root = newRootCmd("test")
	root.SetArgs([]string{"--config", cfg, "impact"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "is required") {
		t.Fatalf("expected required node-id error, got: %v", err)
	}

	root = newRootCmd("test")
	root.SetArgs([]string{"--config", cfg, "impact", "--node", "req:a", "--depth", "0"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "--depth must be >= 1") {
		t.Fatalf("expected depth validation error, got: %v", err)
	}
}

func TestLoadImpactResultsModes(t *testing.T) {
	tmp := t.TempDir()

	emptyPath := filepath.Join(tmp, "empty.json")
	mustWrite(t, emptyPath, " \n")
	results, err := loadImpactResults(emptyPath)
	if err != nil || len(results) != 0 {
		t.Fatalf("empty results err=%v len=%d", err, len(results))
	}

	arrayPath := filepath.Join(tmp, "array.json")
	mustWrite(t, arrayPath, `[{"changed_node":"req:a"},{"changed_node":"design:b"}]`)
	results, err = loadImpactResults(arrayPath)
	if err != nil || len(results) != 2 {
		t.Fatalf("array results err=%v len=%d", err, len(results))
	}

	singlePath := filepath.Join(tmp, "single.json")
	mustWrite(t, singlePath, `{"changed_node":"req:c"}`)
	results, err = loadImpactResults(singlePath)
	if err != nil || len(results) != 1 || results[0].ChangedNode != "req:c" {
		t.Fatalf("single result err=%v results=%+v", err, results)
	}

	ndjsonPath := filepath.Join(tmp, "lines.ndjson")
	mustWrite(t, ndjsonPath, `{"changed_node":"req:x"}
{"changed_node":""}
{"changed_node":"req:y"}
`)
	results, err = loadImpactResults(ndjsonPath)
	if err != nil || len(results) != 2 {
		t.Fatalf("ndjson results err=%v len=%d", err, len(results))
	}
}

func TestLoadImpactResultsInvalidNDJSON(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "bad.ndjson")
	mustWrite(t, p, "{not-json}\n")
	_, err := loadImpactResults(p)
	if err == nil || !strings.Contains(err.Error(), "invalid NDJSON line") {
		t.Fatalf("expected invalid NDJSON error, got: %v", err)
	}
}

func TestCreateIssueFallbackWhenLabelsFail(t *testing.T) {
	oldExec := impactExecCommand
	call := 0
	impactExecCommand = func(name string, args ...string) *exec.Cmd {
		call++
		if call == 1 {
			return exec.Command("sh", "-c", "echo label-failed 1>&2; exit 1")
		}
		return exec.Command("sh", "-c", "printf 'https://example.com/issues/42\\n'")
	}
	t.Cleanup(func() { impactExecCommand = oldExec })

	got, err := createIssue(t.TempDir(), "title", "body", []string{"impact", "amber"})
	if err != nil {
		t.Fatalf("createIssue fallback failed: %v", err)
	}
	if got != "https://example.com/issues/42" {
		t.Fatalf("unexpected url: %q", got)
	}
	if call != 2 {
		t.Fatalf("expected 2 gh calls with fallback, got %d", call)
	}
}

func TestCreateIssueFailureNoFallback(t *testing.T) {
	oldExec := impactExecCommand
	impactExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo plain-failed 1>&2; exit 1")
	}
	t.Cleanup(func() { impactExecCommand = oldExec })

	_, err := createIssue(t.TempDir(), "title", "body", []string{"custom"})
	if err == nil || !strings.Contains(err.Error(), "gh issue create failed") {
		t.Fatalf("expected gh issue create failed, got: %v", err)
	}
}

func TestRunImpactApplyErrors(t *testing.T) {
	rootDir := t.TempDir()
	cfg := filepath.Join(rootDir, ".github", "teraflow.yml")
	mustWrite(t, cfg, "name: test\n")

	empty := filepath.Join(rootDir, "empty.json")
	mustWrite(t, empty, "[]")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfg, "impact", "--apply", empty})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "no impact results found") {
		t.Fatalf("expected empty apply error, got: %v", err)
	}

	results := filepath.Join(rootDir, "result.json")
	mustWrite(t, results, `[{"changed_node":"req:a","review_needed":["design:b"]}]`)
	oldLookPath := impactLookPath
	impactLookPath = func(file string) (string, error) { return "", errors.New("not found") }
	t.Cleanup(func() { impactLookPath = oldLookPath })

	root = newRootCmd("test")
	root.SetArgs([]string{"--config", cfg, "impact", "--apply", results})
	err = root.Execute()
	if err == nil || !strings.Contains(err.Error(), "gh CLI not found") {
		t.Fatalf("expected gh missing error, got: %v", err)
	}
}

func TestImpactHelpers(t *testing.T) {
	if got := colorBand("gray"); !strings.Contains(got, "gray") {
		t.Fatalf("gray color conversion failed: %q", got)
	}
	if got := colorBand("unknown"); got != "unknown" {
		t.Fatalf("unexpected passthrough color band: %q", got)
	}

	m := map[string]any{"a": "", "b": 123}
	if got := firstString(m, "x", "b"); got != "123" {
		t.Fatalf("firstString fallback failed: %q", got)
	}

	numbers := []struct {
		val  any
		want int
	}{
		{3, 3},
		{int64(4), 4},
		{5.9, 5},
		{"6", 6},
		{"bad", 0},
		{nil, 0},
	}
	for _, tc := range numbers {
		data := map[string]any{"n": tc.val}
		if got := intFromMap(data, "n"); got != tc.want {
			t.Fatalf("intFromMap(%v)=%d want=%d", tc.val, got, tc.want)
		}
	}
}

func TestGenerateBodiesIncludeNodeInformation(t *testing.T) {
	result := pipeline.ImpactResult{ChangedNode: "req:auth"}
	amber := generateAmberReviewBody(result, "design:auth")
	gray := generateGrayRegenBody(result, "impl:auth")

	if !strings.Contains(amber, "req:auth") || !strings.Contains(amber, "design:auth") {
		t.Fatalf("amber body missing expected fields: %s", amber)
	}
	if !strings.Contains(gray, "req:auth") || !strings.Contains(gray, "impl:auth") {
		t.Fatalf("gray body missing expected fields: %s", gray)
	}
}

func TestCreateIssueSuccessOutputTrim(t *testing.T) {
	oldExec := impactExecCommand
	impactExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'https://example.com/issues/7\\n\\n'")
	}
	t.Cleanup(func() { impactExecCommand = oldExec })

	got, err := createIssue(os.TempDir(), fmt.Sprintf("t-%d", 1), "body", nil)
	if err != nil {
		t.Fatalf("createIssue success failed: %v", err)
	}
	if got != "https://example.com/issues/7" {
		t.Fatalf("trimmed output mismatch: %q", got)
	}
}
