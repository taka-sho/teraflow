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

func TestLoadStateNotFound(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)
	// Don't create project-state.yml — LoadState should return error
	_, err := LoadState(cfgPath)
	if err == nil {
		t.Fatal("expected error when state file does not exist")
	}
}

func TestLoadStateInvalidYAML(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)
	writeTestFile(t, filepath.Join(root, ".github", "project-state.yml"), ":\n  bad: [\nyaml")
	_, err := LoadState(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
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

func TestLoadReworkLogInvalidYAML(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)
	reworkPath := filepath.Join(root, ".teraflow", "rework-log.yml")
	writeTestFile(t, reworkPath, ":\n  bad: [\ninvalid")
	_, err := LoadReworkLog(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadIncidentLogInvalidYAML(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)
	incidentPath := filepath.Join(root, ".teraflow", "incident-log.yml")
	writeTestFile(t, incidentPath, ":\n  bad: [\ninvalid")
	_, err := LoadIncidentLog(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestAppendReworkMultiple(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)

	for i, id := range []string{"rw-001", "rw-002"} {
		entry := ReworkEntry{
			ID:      id,
			Group:   "group-a",
			Status:  "open",
			Reason:  "reason",
		}
		if err := AppendRework(cfgPath, entry); err != nil {
			t.Fatalf("AppendRework %d: %v", i, err)
		}
	}

	log, err := LoadReworkLog(cfgPath)
	if err != nil {
		t.Fatalf("LoadReworkLog: %v", err)
	}
	if len(log.Reworks) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(log.Reworks))
	}
}

func TestSaveStateFailsOnBadPath(t *testing.T) {
	err := SaveState("/nonexistent/dir/teraflow.yml", &ProjectState{})
	if err == nil {
		t.Fatal("expected error for non-existent directory")
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

func TestLoadReworkLogNilSlice(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)
	reworkPath := filepath.Join(root, ".teraflow", "rework-log.yml")
	writeTestFile(t, reworkPath, "reworks:\n")

	log, err := LoadReworkLog(cfgPath)
	if err != nil {
		t.Fatalf("LoadReworkLog returned error: %v", err)
	}
	if log.Reworks == nil {
		t.Fatal("expected Reworks to be initialized to empty slice")
	}
	if len(log.Reworks) != 0 {
		t.Fatalf("expected empty Reworks slice, got %d", len(log.Reworks))
	}
}

func TestLoadIncidentLogNilSlice(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)
	incidentPath := filepath.Join(root, ".teraflow", "incident-log.yml")
	writeTestFile(t, incidentPath, "incidents:\n")

	log, err := LoadIncidentLog(cfgPath)
	if err != nil {
		t.Fatalf("LoadIncidentLog returned error: %v", err)
	}
	if log.Incidents == nil {
		t.Fatal("expected Incidents to be initialized to empty slice")
	}
	if len(log.Incidents) != 0 {
		t.Fatalf("expected empty Incidents slice, got %d", len(log.Incidents))
	}
}

func TestAppendReworkInvalidExistingLog(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)
	reworkPath := filepath.Join(root, ".teraflow", "rework-log.yml")
	writeTestFile(t, reworkPath, ":\n  bad: [\ninvalid")

	err := AppendRework(cfgPath, ReworkEntry{ID: "rw-001"})
	if err == nil {
		t.Fatal("expected error when existing rework log is invalid")
	}
}

func TestAppendReworkCreateDirError(t *testing.T) {
	err := AppendRework("/dev/null/.github/teraflow.yml", ReworkEntry{ID: "rw-001"})
	if err == nil {
		t.Fatal("expected error for invalid rework directory path")
	}
}

func TestSaveIncidentLogCreateDirError(t *testing.T) {
	err := SaveIncidentLog("/dev/null/.github/teraflow.yml", &IncidentLog{})
	if err == nil {
		t.Fatal("expected error for invalid incident directory path")
	}
}

func TestAppendReworkWriteError(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)
	reworkFilePath := filepath.Join(root, ".teraflow", "rework-log.yml")
	if err := os.MkdirAll(reworkFilePath, 0o755); err != nil {
		t.Fatalf("mkdir rework file path as directory: %v", err)
	}

	err := AppendRework(cfgPath, ReworkEntry{ID: "rw-001"})
	if err == nil {
		t.Fatal("expected write error when rework-log.yml is a directory")
	}
}

func TestSaveIncidentLogWriteError(t *testing.T) {
	root := t.TempDir()
	cfgPath := testConfigPath(t, root)
	incidentFilePath := filepath.Join(root, ".teraflow", "incident-log.yml")
	if err := os.MkdirAll(incidentFilePath, 0o755); err != nil {
		t.Fatalf("mkdir incident file path as directory: %v", err)
	}

	err := SaveIncidentLog(cfgPath, &IncidentLog{})
	if err == nil {
		t.Fatal("expected write error when incident-log.yml is a directory")
	}
}

func TestAppendReworkCoverageRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("version: \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entry := ReworkEntry{
		ID:        "RW-001",
		Group:     "phase:implementation",
		Reason:    "test",
		Status:    "open",
		CreatedAt: "2026-04-05",
	}
	if err := AppendRework(configPath, entry); err != nil {
		t.Fatalf("AppendRework: %v", err)
	}

	log, err := LoadReworkLog(configPath)
	if err != nil {
		t.Fatalf("LoadReworkLog: %v", err)
	}
	if len(log.Reworks) != 1 || log.Reworks[0].ID != "RW-001" {
		t.Fatalf("unexpected reworks: %+v", log.Reworks)
	}
}

func TestSaveStateRoundTripCoverage(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("version: \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &ProjectState{}
	s.Project.Name = "roundtrip"
	s.Lifecycle.CurrentStage = "initial_development"
	s.Phases.Current = "requirements"
	if err := SaveState(configPath, s); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	loaded, err := LoadState(configPath)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if loaded.Project.Name != "roundtrip" {
		t.Fatalf("name: %s", loaded.Project.Name)
	}
}

func TestSaveAndLoadIncidentLogCoverage(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("version: \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	log := &IncidentLog{
		Incidents: []IncidentEntry{{
			ID:        "INC-001",
			Title:     "test",
			Severity:  "high",
			Status:    "open",
			CreatedAt: "2026-04-05",
		}},
	}
	if err := SaveIncidentLog(configPath, log); err != nil {
		t.Fatalf("SaveIncidentLog: %v", err)
	}

	loaded, err := LoadIncidentLog(configPath)
	if err != nil {
		t.Fatalf("LoadIncidentLog: %v", err)
	}
	if len(loaded.Incidents) != 1 {
		t.Fatalf("expected 1: %+v", loaded)
	}
}
