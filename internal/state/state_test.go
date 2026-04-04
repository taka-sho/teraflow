package state

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func testConfigPath(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, ".github", "teraflow.yml")
	writeTestFile(t, path, "version: \"1\"\n")
	return path
}

func TestLoadState(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)

	writeTestFile(t, filepath.Join(root, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "initial_development"
phases:
  current: "requirements"
`)

	s, err := LoadState(cfgPath)
	if err != nil {
		t.Fatalf("LoadState returned error: %v", err)
	}
	if s.Project.Name != "test" {
		t.Fatalf("unexpected project name: %s", s.Project.Name)
	}
	if s.Lifecycle.CurrentStage != "initial_development" {
		t.Fatalf("unexpected current stage: %s", s.Lifecycle.CurrentStage)
	}
	if s.Phases.Current != "requirements" {
		t.Fatalf("unexpected current phase: %s", s.Phases.Current)
	}
}

func TestSaveState(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)

	expected := &ProjectState{
		Project:   ProjectInfo{Name: "save-test"},
		Lifecycle: LifecycleInfo{CurrentStage: "development"},
		Phases:    PhasesInfo{Current: "basic_design"},
	}

	if err := SaveState(cfgPath, expected); err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	actual, err := LoadState(cfgPath)
	if err != nil {
		t.Fatalf("LoadState returned error: %v", err)
	}

	if actual.Project.Name != expected.Project.Name ||
		actual.Lifecycle.CurrentStage != expected.Lifecycle.CurrentStage ||
		actual.Phases.Current != expected.Phases.Current {
		t.Fatalf("state mismatch: got %+v, want %+v", actual, expected)
	}
}

func TestLoadReworkLog_empty(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)

	log, err := LoadReworkLog(cfgPath)
	if err != nil {
		t.Fatalf("LoadReworkLog returned error: %v", err)
	}
	if len(log.Reworks) != 0 {
		t.Fatalf("expected empty rework log, got %d", len(log.Reworks))
	}
}

func TestAppendRework(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)

	entry := ReworkEntry{
		ID:          "rw-001",
		Group:       "group-api",
		TargetPhase: "basic_design",
		Reason:      "spec changed",
		CreatedAt:   "2026-04-05T00:00:00Z",
		Status:      "open",
	}

	if err := AppendRework(cfgPath, entry); err != nil {
		t.Fatalf("AppendRework returned error: %v", err)
	}

	log, err := LoadReworkLog(cfgPath)
	if err != nil {
		t.Fatalf("LoadReworkLog returned error: %v", err)
	}
	if len(log.Reworks) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(log.Reworks))
	}

	got := log.Reworks[0]
	if got != entry {
		t.Fatalf("entry mismatch: got %+v, want %+v", got, entry)
	}
}

func TestLoadIncidentLog_empty(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)

	log, err := LoadIncidentLog(cfgPath)
	if err != nil {
		t.Fatalf("LoadIncidentLog returned error: %v", err)
	}
	if len(log.Incidents) != 0 {
		t.Fatalf("expected empty incident log, got %d", len(log.Incidents))
	}
}

func TestSaveIncidentLog(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)

	expected := &IncidentLog{
		Incidents: []IncidentEntry{
			{
				ID:          "inc-001",
				Title:       "DB timeout",
				Severity:    "major",
				Description: "database does not respond",
				CreatedAt:   "2026-04-05T12:00:00",
				Status:      "open",
			},
		},
	}

	if err := SaveIncidentLog(cfgPath, expected); err != nil {
		t.Fatalf("SaveIncidentLog returned error: %v", err)
	}

	actual, err := LoadIncidentLog(cfgPath)
	if err != nil {
		t.Fatalf("LoadIncidentLog returned error: %v", err)
	}
	if len(actual.Incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(actual.Incidents))
	}
	if actual.Incidents[0] != expected.Incidents[0] {
		t.Fatalf("incident mismatch: got %+v, want %+v", actual.Incidents[0], expected.Incidents[0])
	}
}
