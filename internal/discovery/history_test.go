package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppendLocalHistory_CreateAndAppend(t *testing.T) {
	dir := t.TempDir()

	entries1 := []HistoryEntry{
		{Timestamp: "2026-01-01T00:00:00Z", Action: "add", FieldID: "f1", Source: "cli"},
	}
	if err := AppendLocalHistory(dir, entries1); err != nil {
		t.Fatalf("first append: %v", err)
	}

	h, err := ReadLocalHistory(dir)
	if err != nil {
		t.Fatalf("read after first append: %v", err)
	}
	if len(h.Entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(h.Entries))
	}

	entries2 := []HistoryEntry{
		{Timestamp: "2026-01-02T00:00:00Z", Action: "remove", FieldID: "f2", Source: "manual"},
	}
	if err := AppendLocalHistory(dir, entries2); err != nil {
		t.Fatalf("second append: %v", err)
	}

	h, err = ReadLocalHistory(dir)
	if err != nil {
		t.Fatalf("read after second append: %v", err)
	}
	if len(h.Entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(h.Entries))
	}
}

func TestAppendLocalHistory_SetsRepositoryInGlobal(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.yaml")

	origGlobal := GlobalHistoryPath
	GlobalHistoryPath = func() string { return globalPath }
	defer func() { GlobalHistoryPath = origGlobal }()

	entries := []HistoryEntry{
		{Timestamp: "2026-01-01T00:00:00Z", Action: "add", FieldID: "f1", Source: "cli"},
	}
	if err := AppendLocalHistory(dir, entries); err != nil {
		t.Fatalf("append: %v", err)
	}

	g, err := readHistory(globalPath)
	if err != nil {
		t.Fatalf("read global: %v", err)
	}
	if len(g.Entries) != 1 {
		t.Fatalf("want 1 global entry, got %d", len(g.Entries))
	}
	if g.Entries[0].Repository != dir {
		t.Errorf("want repository=%q, got %q", dir, g.Entries[0].Repository)
	}
}

func TestDetectAndRecordChanges(t *testing.T) {
	dir := t.TempDir()

	globalPath := filepath.Join(dir, "global.yaml")
	origGlobal := GlobalHistoryPath
	GlobalHistoryPath = func() string { return globalPath }
	defer func() { GlobalHistoryPath = origGlobal }()

	prev := RequirementTemplate{
		Items: []TemplateItem{
			{ID: "f1", Name: "Field1", Category: "required"},
			{ID: "f2", Name: "Field2", Category: "optional"},
		},
	}
	curr := RequirementTemplate{
		Items: []TemplateItem{
			{ID: "f1", Name: "Field1 Modified", Category: "required"},
			{ID: "f3", Name: "Field3", Category: "recommended"},
		},
	}

	entries, err := DetectAndRecordChanges(dir, prev, curr, "cli")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	actions := make(map[string]string, len(entries))
	for _, e := range entries {
		actions[e.FieldID] = e.Action
	}

	if actions["f2"] != "remove" {
		t.Errorf("f2 should be removed, got %q", actions["f2"])
	}
	if actions["f3"] != "add" {
		t.Errorf("f3 should be added, got %q", actions["f3"])
	}
	if actions["f1"] != "modify" {
		t.Errorf("f1 should be modified, got %q", actions["f1"])
	}

	h, err := ReadLocalHistory(dir)
	if err != nil {
		t.Fatalf("read local: %v", err)
	}
	if len(h.Entries) != 3 {
		t.Errorf("want 3 local entries, got %d", len(h.Entries))
	}
}

func TestDetectAndRecordChanges_NoChange(t *testing.T) {
	dir := t.TempDir()

	globalPath := filepath.Join(dir, "global.yaml")
	origGlobal := GlobalHistoryPath
	GlobalHistoryPath = func() string { return globalPath }
	defer func() { GlobalHistoryPath = origGlobal }()

	tmpl := RequirementTemplate{
		Items: []TemplateItem{
			{ID: "f1", Name: "Field1", Category: "required"},
		},
	}
	entries, err := DetectAndRecordChanges(dir, tmpl, tmpl, "cli")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("want 0 entries, got %d", len(entries))
	}
}

func TestReadLocalHistory_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	h, err := ReadLocalHistory(filepath.Join(dir, "nonexistent"))
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if h.Version != "1" {
		t.Errorf("want version 1, got %q", h.Version)
	}
	if len(h.Entries) != 0 {
		t.Errorf("want empty entries")
	}
}

func TestSyncToGlobal_DeduplicatesEntries(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.yaml")

	origGlobal := GlobalHistoryPath
	GlobalHistoryPath = func() string { return globalPath }
	defer func() { GlobalHistoryPath = origGlobal }()

	entries := []HistoryEntry{
		{Timestamp: "2026-01-01T00:00:00Z", Action: "add", FieldID: "f1", Source: "cli"},
		{Timestamp: "2026-01-02T00:00:00Z", Action: "remove", FieldID: "f2", Source: "cli"},
	}
	if err := AppendLocalHistory(dir, entries); err != nil {
		t.Fatalf("append: %v", err)
	}

	// first sync
	if err := SyncToGlobal(dir); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	g, _ := readHistory(globalPath)
	if len(g.Entries) != 2 {
		t.Fatalf("after first sync: want 2, got %d", len(g.Entries))
	}

	// second sync — should be no-op (dedup)
	if err := SyncToGlobal(dir); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	g, _ = readHistory(globalPath)
	if len(g.Entries) != 2 {
		t.Errorf("after second sync (dedup): want 2, got %d", len(g.Entries))
	}
}

func TestSyncToGlobal_AppendsNewEntries(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "global.yaml")

	origGlobal := GlobalHistoryPath
	GlobalHistoryPath = func() string { return globalPath }
	defer func() { GlobalHistoryPath = origGlobal }()

	// Write two batches directly to local only (simulates manual edits / global was wiped).
	localPath := LocalHistoryPath(dir)
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		t.Fatal(err)
	}
	allLocal := TemplateHistory{
		Version: "1",
		Entries: []HistoryEntry{
			{Timestamp: "2026-01-01T00:00:00Z", Action: "add", FieldID: "f1", Source: "cli"},
			{Timestamp: "2026-01-02T00:00:00Z", Action: "remove", FieldID: "f2", Source: "cli"},
			{Timestamp: "2026-01-03T00:00:00Z", Action: "add", FieldID: "f3", Source: "cli"},
		},
	}
	if err := writeHistory(localPath, allLocal); err != nil {
		t.Fatal(err)
	}

	if err := SyncToGlobal(dir); err != nil {
		t.Fatalf("sync: %v", err)
	}

	g, _ := readHistory(globalPath)
	if len(g.Entries) != 3 {
		t.Errorf("want 3 global entries, got %d", len(g.Entries))
	}

	// second sync must be no-op (dedup)
	if err := SyncToGlobal(dir); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	g, _ = readHistory(globalPath)
	if len(g.Entries) != 3 {
		t.Errorf("after second sync want 3, got %d", len(g.Entries))
	}
}

// cleanup helper — silence "declared but not used" if os import needed only here
var _ = os.DevNull
