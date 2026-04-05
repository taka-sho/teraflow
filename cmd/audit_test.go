package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditListNoEntries(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, "version: \"1\"\n")

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "audit", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("audit list failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "📋 監査ログ") {
		t.Fatalf("expected header in output: %s", got)
	}
	if !strings.Contains(got, "(記録なし)") {
		t.Fatalf("expected empty message in output: %s", got)
	}
}

func TestAuditListWithFilters(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, "version: \"1\"\n")
	writeCmdTestFile(t, filepath.Join(tmp, ".teraflow", "audit-log.yml"), `entries:
  - timestamp: "2026-04-05T17:15:00+09:00"
    user: "Alice"
    action: "gate.approve"
    target: "planning"
    result: "success"
  - timestamp: "2026-04-05T17:20:00+09:00"
    user: "Bob"
    action: "process.start"
    target: "requirements"
    result: "success"
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{
		"--config", configPath,
		"audit", "list",
		"--user", "Alice",
		"--action", "gate.approve",
		"--since", "2026-04-05",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("audit list with filters failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Alice") {
		t.Fatalf("expected Alice in output: %s", got)
	}
	if strings.Contains(got, "Bob") {
		t.Fatalf("did not expect Bob in output: %s", got)
	}
	if !strings.Contains(got, "gate.approve") {
		t.Fatalf("expected action in output: %s", got)
	}
}

func TestAuditListJSONOutput(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, "version: \"1\"\n")
	writeCmdTestFile(t, filepath.Join(tmp, ".teraflow", "audit-log.yml"), `entries:
  - timestamp: "2026-04-05T17:30:00+09:00"
    user: "Alice"
    action: "process.complete"
    target: "planning"
    result: "success"
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "audit", "list", "--output", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("audit list --output json failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, `"entries"`) {
		t.Fatalf("expected entries key in JSON output: %s", got)
	}
	if !strings.Contains(got, `"process.complete"`) {
		t.Fatalf("expected action in JSON output: %s", got)
	}
}

func TestAuditListInvalidSince(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	writeCmdTestFile(t, configPath, "version: \"1\"\n")

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "audit", "list", "--since", "2026/04/05"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid --since")
	}
	if !strings.Contains(err.Error(), "--since must be YYYY-MM-DD") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatAuditTimestamp(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: "-"},
		{name: "rfc3339", in: "2026-04-05T17:15:00+09:00", want: "2026-04-05 17:15"},
		{name: "space_seconds", in: "2026-04-05 17:15:00", want: "2026-04-05 17:15"},
		{name: "space_minute", in: "2026-04-05 17:15", want: "2026-04-05 17:15"},
		{name: "fallback_slice", in: "2026-04-05 17:15 unknown", want: "2026-04-05 17:15"},
		{name: "short_passthrough", in: "n/a", want: "n/a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatAuditTimestamp(tt.in); got != tt.want {
				t.Fatalf("formatAuditTimestamp(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestEmptyFallback(t *testing.T) {
	if got := emptyFallback("Alice", "-"); got != "Alice" {
		t.Fatalf("expected original value, got %q", got)
	}
	if got := emptyFallback("   ", "-"); got != "-" {
		t.Fatalf("expected fallback for spaces, got %q", got)
	}
}
