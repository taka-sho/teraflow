package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTraceCmdConfigFlagError(t *testing.T) {
	cmd := newTraceCmd()
	cmd.SetArgs([]string{"req-user-auth"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when root flags are missing")
	}
	if !strings.Contains(err.Error(), "format") && !strings.Contains(err.Error(), "config") {
		t.Fatalf("expected missing flag error, got: %v", err)
	}
}

func TestTraceCmdIndexNotFound(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "trace", "req-user-auth"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when index is missing")
	}
	if !strings.Contains(err.Error(), "read index") {
		t.Fatalf("expected read index error, got: %v", err)
	}
}

func TestTraceCmdTextAndJSONOutput(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "index.yml"), `version: "1"
generated_at: 2026-04-06T00:00:00Z
entries:
  - node_id: req-core-auth
    title: コア認証基盤
    path: docs/requirements/core-auth.md
    depends_on: []
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "a"
    summary_available: false
  - node_id: req-session-management
    title: セッション管理要件
    path: docs/requirements/session-management.md
    depends_on: ["req-core-auth"]
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "b"
    summary_available: false
  - node_id: req-user-auth
    title: ユーザー認証要件
    path: docs/requirements/user-auth.md
    depends_on: ["req-session-management"]
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "c"
    summary_available: false
  - node_id: design-api-gateway
    title: APIゲートウェイ設計
    path: docs/design/api-gateway.md
    depends_on: ["req-user-auth"]
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "d"
    summary_available: false
  - node_id: review-api-gateway
    title: APIゲートウェイレビュー
    path: docs/review/api-gateway.md
    depends_on: ["design-api-gateway"]
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "e"
    summary_available: false
`)

	var textOut bytes.Buffer
	root := newRootCmd("test")
	root.SetOut(&textOut)
	root.SetErr(&textOut)
	root.SetArgs([]string{"--config", cfgPath, "trace", "req-user-auth"})
	if err := root.Execute(); err != nil {
		t.Fatalf("trace text error = %v", err)
	}
	gotText := textOut.String()
	if !strings.Contains(gotText, "▲ Upstream (depends on):") ||
		!strings.Contains(gotText, "▼ Downstream (depended by):") ||
		!strings.Contains(gotText, "● req-user-auth (ユーザー認証要件)") ||
		!strings.Contains(gotText, "req-session-management (セッション管理要件)") ||
		!strings.Contains(gotText, "design-api-gateway (APIゲートウェイ設計)") {
		t.Fatalf("unexpected trace text output: %q", gotText)
	}

	var jsonOut bytes.Buffer
	root = newRootCmd("test")
	root.SetOut(&jsonOut)
	root.SetErr(&jsonOut)
	root.SetArgs([]string{"--config", cfgPath, "--format", "json", "trace", "req-user-auth", "--direction", "down", "--depth", "1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("trace json error = %v", err)
	}
	gotJSON := jsonOut.String()
	if !strings.Contains(gotJSON, `"RootNodeID":"req-user-auth"`) ||
		!strings.Contains(gotJSON, `"Direction":"down"`) ||
		!strings.Contains(gotJSON, `"NodeID":"design-api-gateway"`) {
		t.Fatalf("unexpected trace json output: %q", gotJSON)
	}
	if strings.Contains(gotJSON, `"NodeID":"review-api-gateway"`) {
		t.Fatalf("depth filter not applied: %q", gotJSON)
	}
}

func TestTraceCmdInvalidDirection(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "index.yml"), `version: "1"
generated_at: 2026-04-06T00:00:00Z
entries:
  - node_id: req-user-auth
    title: ユーザー認証要件
    path: docs/requirements/user-auth.md
    depends_on: []
    updated_at: 2026-04-06T00:00:00Z
    content_hash: "c"
    summary_available: false
`)

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", cfgPath, "trace", "req-user-auth", "--direction", "sideways"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected invalid direction error")
	}
	if !strings.Contains(err.Error(), "invalid direction") {
		t.Fatalf("unexpected error: %v", err)
	}
}
