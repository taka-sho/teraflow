package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	indexpkg "github.com/taka-sho/teraflow/internal/index"
)

func TestNewIndexBuildCmdConfigFlagError(t *testing.T) {
	cmd := newIndexBuildCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when root flags are missing")
	}
	if !strings.Contains(err.Error(), "format") && !strings.Contains(err.Error(), "config") {
		t.Fatalf("expected missing flag error, got: %v", err)
	}
}

func TestNewIndexStatusCmdConfigFlagError(t *testing.T) {
	cmd := newIndexStatusCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when root flags are missing")
	}
	if !strings.Contains(err.Error(), "format") && !strings.Contains(err.Error(), "config") {
		t.Fatalf("expected missing flag error, got: %v", err)
	}
}

func TestProjectRootFromCmd(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")

	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().String("config", "", "")
	if err := root.PersistentFlags().Set("config", cfgPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	child := &cobra.Command{Use: "child"}
	root.AddCommand(child)

	got, err := projectRootFromCmd(child)
	if err != nil {
		t.Fatalf("projectRootFromCmd() error = %v", err)
	}
	if got != tmp {
		t.Fatalf("projectRootFromCmd() = %q, want %q", got, tmp)
	}
}

func TestCalcIndexDiff(t *testing.T) {
	added, updated := calcIndexDiff(&indexpkg.Index{}, nil)
	if added != 0 || updated != 0 {
		t.Fatalf("calcIndexDiff(old,nil) = (%d,%d), want (0,0)", added, updated)
	}

	newIdx := &indexpkg.Index{Entries: []indexpkg.Entry{{NodeID: "a", ContentHash: "1"}, {NodeID: "b", ContentHash: "2"}}}
	added, updated = calcIndexDiff(nil, newIdx)
	if added != 2 || updated != 0 {
		t.Fatalf("calcIndexDiff(nil,new) = (%d,%d), want (2,0)", added, updated)
	}

	oldIdx := &indexpkg.Index{Entries: []indexpkg.Entry{{NodeID: "a", ContentHash: "1"}, {NodeID: "b", ContentHash: "old"}}}
	added, updated = calcIndexDiff(oldIdx, newIdx)
	if added != 0 || updated != 1 {
		t.Fatalf("calcIndexDiff(old,new) = (%d,%d), want (0,1)", added, updated)
	}
}

func TestIsIndexNotFound(t *testing.T) {
	err := &os.PathError{Op: "open", Path: "x", Err: os.ErrNotExist}
	if !isIndexNotFound(err) {
		t.Fatal("expected true for os.ErrNotExist path error")
	}
	if isIndexNotFound(errors.New("other error")) {
		t.Fatal("expected false for non-path error")
	}
}

func TestIndexBuildAndStatusCommands(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, "docs", "alpha.md"), `---
codd:
  node_id: req:alpha
  title: Alpha
---
# Alpha
`)

	var buildOut bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&buildOut)
	root.SetErr(&buildOut)
	root.SetArgs([]string{"--config", cfgPath, "index", "build"})
	if err := root.Execute(); err != nil {
		t.Fatalf("index build error = %v", err)
	}
	if !strings.Contains(buildOut.String(), "Index built:") {
		t.Fatalf("unexpected build output: %q", buildOut.String())
	}

	mustWrite(t, filepath.Join(tmp, "docs", "alpha.md"), `---
codd:
  node_id: req:alpha
  title: Alpha
---
# Alpha changed
`)
	var buildJSONOut bytes.Buffer
	root = newRootCmd("test")
	root.SetOut(&buildJSONOut)
	root.SetErr(&buildJSONOut)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "index", "build"})
	if err := root.Execute(); err != nil {
		t.Fatalf("index build(json) error = %v", err)
	}
	if !strings.Contains(buildJSONOut.String(), `"updated":1`) {
		t.Fatalf("unexpected build json output: %q", buildJSONOut.String())
	}

	var statusOut bytes.Buffer
	root = newRootCmd("test")
	root.SetOut(&statusOut)
	root.SetErr(&statusOut)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "index", "status"})
	if err := root.Execute(); err != nil {
		t.Fatalf("index status error = %v", err)
	}
	got := statusOut.String()
	if !strings.Contains(got, `"status":"ok"`) || !strings.Contains(got, `"entries":1`) {
		t.Fatalf("unexpected status JSON: %q", got)
	}

	var statusTextOut bytes.Buffer
	root = newRootCmd("test")
	root.SetOut(&statusTextOut)
	root.SetErr(&statusTextOut)
	root.SetArgs([]string{"--config", cfgPath, "index", "status"})
	if err := root.Execute(); err != nil {
		t.Fatalf("index status(text) error = %v", err)
	}
	if !strings.Contains(statusTextOut.String(), "Summary coverage:") {
		t.Fatalf("unexpected status text output: %q", statusTextOut.String())
	}
}

func TestIndexStatusMissingIndex(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	var textOut bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&textOut)
	root.SetErr(&textOut)
	root.SetArgs([]string{"--config", cfgPath, "index", "status"})
	if err := root.Execute(); err != nil {
		t.Fatalf("index status(text) error = %v", err)
	}
	if !strings.Contains(textOut.String(), "Index not found") {
		t.Fatalf("unexpected text output: %q", textOut.String())
	}

	var jsonOut bytes.Buffer
	root = newRootCmd("test")
	root.SetOut(&jsonOut)
	root.SetErr(&jsonOut)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "index", "status"})
	if err := root.Execute(); err != nil {
		t.Fatalf("index status(json) error = %v", err)
	}
	if !strings.Contains(jsonOut.String(), `"status":"missing"`) {
		t.Fatalf("unexpected json output: %q", jsonOut.String())
	}
}

func TestIndexBuildLoadError(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, "docs", "alpha.md"), `---
codd:
  node_id: req:alpha
  title: Alpha
---
# Alpha
`)
	if err := os.MkdirAll(filepath.Join(tmp, ".teraflow", "index.yml"), 0o755); err != nil {
		t.Fatalf("prepare index path directory: %v", err)
	}

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "index", "build"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected load index error")
	}
	if !strings.Contains(err.Error(), "read index") {
		t.Fatalf("unexpected error: %v", err)
	}
}
