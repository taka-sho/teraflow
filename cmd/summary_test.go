package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	indexpkg "github.com/taka-sho/teraflow/internal/index"
)

func TestNewSummaryUpdateCmdConfigFlagError(t *testing.T) {
	cmd := newSummaryUpdateCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when config flag is missing")
	}
	if !strings.Contains(err.Error(), "config") {
		t.Fatalf("expected config-related error, got: %v", err)
	}
}

func TestNewSummaryShowCmdConfigFlagError(t *testing.T) {
	cmd := newSummaryShowCmd()
	cmd.SetArgs([]string{"req:alpha"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when config flag is missing")
	}
	if !strings.Contains(err.Error(), "config") {
		t.Fatalf("expected config-related error, got: %v", err)
	}
}

func TestResolveSummaryProviderFallbackWhenConfigMissing(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	provider, err := resolveSummaryProvider("/non/existent/path.yml")
	if err != nil {
		t.Fatalf("resolveSummaryProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("resolveSummaryProvider() returned nil provider")
	}
}

func TestProjectRootFromConfigAndSummaryFileName(t *testing.T) {
	cfg := filepath.Join("/tmp", "sample", ".github", "teraflow.yml")
	if got := projectRootFromConfig(cfg); got != filepath.Join("/tmp", "sample") {
		t.Fatalf("projectRootFromConfig() = %q", got)
	}
	if got := summaryFileName("req:alpha/design"); got != "req--alpha-design.txt" {
		t.Fatalf("summaryFileName() = %q", got)
	}
}

func TestSummaryUpdateMissingIndexReturnsE7001(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
ai:
  default_provider: anthropic
`)

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "summary", "update"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when index is missing")
	}
	if !strings.Contains(err.Error(), "E7001") {
		t.Fatalf("expected E7001 error, got: %v", err)
	}
}

func TestSummaryUpdateDryRunAndShow(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
ai:
  default_provider: anthropic
`)
	mustWrite(t, filepath.Join(tmp, "docs", "alpha.md"), `---
codd:
  node_id: req:alpha
  title: Alpha
---
# Alpha
`)

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "index", "build"})
	if err := root.Execute(); err != nil {
		t.Fatalf("index build error = %v", err)
	}

	var dryOut bytes.Buffer
	root = newRootCmd("test")
	root.SetOut(&dryOut)
	root.SetErr(&dryOut)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "summary", "update", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("summary update --dry-run error = %v", err)
	}
	if !strings.Contains(dryOut.String(), `"count":1`) || !strings.Contains(dryOut.String(), `"req:alpha"`) {
		t.Fatalf("unexpected dry-run output: %q", dryOut.String())
	}

	mustWrite(t, filepath.Join(tmp, ".teraflow", "summaries", summaryFileName("req:alpha")), "# hash:any\nhello summary\n")

	var showOut bytes.Buffer
	root = newRootCmd("test")
	root.SetOut(&showOut)
	root.SetErr(&showOut)
	root.SetArgs([]string{"--config", cfgPath, "summary", "show", "req:alpha"})
	if err := root.Execute(); err != nil {
		t.Fatalf("summary show error = %v", err)
	}
	if !strings.Contains(showOut.String(), "hello summary") {
		t.Fatalf("unexpected summary show output: %q", showOut.String())
	}
}

func TestSummaryUpdateNoChangesJSON(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
ai:
  default_provider: anthropic
`)
	mustWrite(t, filepath.Join(tmp, "docs", "alpha.md"), `---
codd:
  node_id: req:alpha
  title: Alpha
---
# Alpha
`)

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "index", "build"})
	if err := root.Execute(); err != nil {
		t.Fatalf("index build error = %v", err)
	}

	idx, err := indexpkg.NewBuilder(tmp).LoadIndex()
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	if len(idx.Entries) != 1 {
		t.Fatalf("expected 1 index entry, got %d", len(idx.Entries))
	}

	cache := "# hash:" + idx.Entries[0].ContentHash + "\ncached summary\n"
	mustWrite(t, filepath.Join(tmp, ".teraflow", "summaries", summaryFileName("req:alpha")), cache)

	var out bytes.Buffer
	root = newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "summary", "update"})
	if err := root.Execute(); err != nil {
		t.Fatalf("summary update error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, `"updated":0`) || !strings.Contains(got, `"skipped":1`) {
		t.Fatalf("unexpected summary update output: %q", got)
	}
}

func TestSummaryUpdateForceWithEmptyIndex(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, `version: "1"
ai:
  default_provider: anthropic
`)

	if err := indexpkg.NewBuilder(tmp).Save(&indexpkg.Index{Version: "1", Entries: nil}); err != nil {
		t.Fatalf("save empty index: %v", err)
	}
	mustWrite(t, filepath.Join(tmp, ".teraflow", "summaries", "stale.txt"), "stale")

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "summary", "update", "--force"})
	if err := root.Execute(); err != nil {
		t.Fatalf("summary update --force error = %v", err)
	}
	if !strings.Contains(out.String(), "updated 0, skipped 0") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestSummaryShowJSON(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "summaries", summaryFileName("req:alpha")), "# hash:any\nhello summary\n")

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "summary", "show", "req:alpha"})
	if err := root.Execute(); err != nil {
		t.Fatalf("summary show json error = %v", err)
	}
	if !strings.Contains(out.String(), `"node_id":"req:alpha"`) || !strings.Contains(out.String(), `"summary":"hello summary"`) {
		t.Fatalf("unexpected json output: %q", out.String())
	}
}
