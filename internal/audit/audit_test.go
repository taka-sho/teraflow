package audit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/taka-sho/teraflow/internal/rbac"
)

func TestAppendAuditAndLoadAuditLogRoundTrip(t *testing.T) {
	projectDir := t.TempDir()
	entry := AuditEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		User:      "test-user",
		Action:    "gate.approve",
		Target:    "detailed_design",
		Result:    "approved",
	}

	if err := AppendAudit(projectDir, entry); err != nil {
		t.Fatalf("AppendAudit() error = %v", err)
	}

	logData, err := LoadAuditLog(projectDir)
	if err != nil {
		t.Fatalf("LoadAuditLog() error = %v", err)
	}
	if len(logData.Entries) != 1 {
		t.Fatalf("LoadAuditLog() entries = %d, want 1", len(logData.Entries))
	}
	if got := logData.Entries[0]; got != entry {
		t.Fatalf("entry mismatch: got %+v, want %+v", got, entry)
	}

	logPath := filepath.Join(projectDir, AuditLogPath)
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("audit log file not created: %v", err)
	}
}

func TestLoadAuditLogReturnsEmptyWhenMissing(t *testing.T) {
	projectDir := t.TempDir()

	logData, err := LoadAuditLog(projectDir)
	if err != nil {
		t.Fatalf("LoadAuditLog() error = %v", err)
	}
	if len(logData.Entries) != 0 {
		t.Fatalf("LoadAuditLog() entries = %d, want 0", len(logData.Entries))
	}
}

func TestNewEntrySetsTimestampAndUser(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USER", "audit-test-user")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(tmpHome, "global.gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(tmpHome, "system.gitconfig"))

	expectedUser, err := rbac.CurrentUser()
	if err != nil {
		t.Fatalf("CurrentUser() error = %v", err)
	}

	entry := NewEntry("process.start", "planning", "ok")
	if entry.User != expectedUser {
		t.Fatalf("entry.User = %q, want %q", entry.User, expectedUser)
	}
	if entry.Action != "process.start" || entry.Target != "planning" || entry.Result != "ok" {
		t.Fatalf("entry fields mismatch: %+v", entry)
	}
	if _, err := time.Parse(time.RFC3339, entry.Timestamp); err != nil {
		t.Fatalf("entry.Timestamp is not RFC3339: %q", entry.Timestamp)
	}
}
