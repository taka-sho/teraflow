package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoveryStatusTextAndJSON(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "discovery", "discussion-42.yaml"), `version: "1"
discussion_number: 42
title: "ユーザー認証要件"
mode: "sequential"
tree:
  - id: scope.platform
    question: 対象プラットフォームは？
    category: scope
    status: answered
    answer: Web + API
  - id: nfr.performance
    question: 性能要件は？
    category: non_functional
    status: pending
summary:
  total: 2
  answered: 1
  pending: 1
  skipped: 0
  progress_percent: 50
`)

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "discovery", "status", "discussion-42"})
	if err := root.Execute(); err != nil {
		t.Fatalf("discovery status text error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Discussion: #42") || !strings.Contains(got, "Progress: 1/2 (50%)") {
		t.Fatalf("unexpected text output: %q", got)
	}

	out.Reset()
	root = newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "discovery", "status", "42"})
	if err := root.Execute(); err != nil {
		t.Fatalf("discovery status json error: %v", err)
	}
	if !strings.Contains(out.String(), `"discussion_number":42`) || !strings.Contains(out.String(), `"progress_percent":50`) {
		t.Fatalf("unexpected json output: %q", out.String())
	}
}

func TestDiscoveryTreeTextAndJSON(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "discovery", "discussion-7.yaml"), `version: "1"
discussion_number: 7
title: "API要件"
tree:
  - id: scope.platform
    question: 対象プラットフォームは？
    category: scope
    status: answered
    children:
      - id: scope.mobile.auth
        question: モバイル認証方式は？
        category: functional
        status: blocked
        depends_on: [scope.platform]
summary:
  total: 2
  answered: 1
  pending: 0
  skipped: 0
  blocked: 1
  progress_percent: 50
`)

	var out bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "discovery", "tree", "discussion-7"})
	if err := root.Execute(); err != nil {
		t.Fatalf("discovery tree text error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "- [answered] scope.platform") || !strings.Contains(got, "- [blocked] scope.mobile.auth") {
		t.Fatalf("unexpected tree output: %q", got)
	}

	out.Reset()
	root = newRootCmd("test")
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "discovery", "tree", "7"})
	if err := root.Execute(); err != nil {
		t.Fatalf("discovery tree json error: %v", err)
	}
	if !strings.Contains(out.String(), `"id":"scope.platform"`) {
		t.Fatalf("unexpected json output: %q", out.String())
	}
}

func TestDiscoveryStatusInvalidArg(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "discovery", "status", "foo"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid discussion id") {
		t.Fatalf("unexpected error: %v", err)
	}
}
