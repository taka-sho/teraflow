package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/state"
)

func TestIncidentCreate(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	command := newRootCmd("test")
	command.AddCommand(newIncidentCmd())
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"incident", "create", "--title", "DBが応答しない", "--severity", "major", "--description", "timeout", "--config", cfgPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("incident create execute error: %v", err)
	}

	log, err := state.LoadIncidentLog(cfgPath)
	if err != nil {
		t.Fatalf("LoadIncidentLog error: %v", err)
	}
	if len(log.Incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(log.Incidents))
	}
	if log.Incidents[0].ID != "inc-001" {
		t.Fatalf("unexpected incident id: %s", log.Incidents[0].ID)
	}
	if log.Incidents[0].Status != "open" {
		t.Fatalf("unexpected incident status: %s", log.Incidents[0].Status)
	}
}

func TestIncidentList(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	if err := state.SaveIncidentLog(cfgPath, &state.IncidentLog{Incidents: []state.IncidentEntry{{
		ID:        "inc-001",
		Title:     "DBが応答しない",
		Severity:  "major",
		CreatedAt: "2026-04-05T00:00:00",
		Status:    "open",
	}}}); err != nil {
		t.Fatalf("SaveIncidentLog error: %v", err)
	}

	command := newRootCmd("test")
	command.AddCommand(newIncidentCmd())
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"incident", "list", "--config", cfgPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("incident list execute error: %v", err)
	}
	if !strings.Contains(out.String(), "ID       Severity   Status   Title") {
		t.Fatalf("unexpected output: %s", out.String())
	}
	if !strings.Contains(out.String(), "inc-001") || !strings.Contains(out.String(), "DBが応答しない") {
		t.Fatalf("expected incident row in output, got:\n%s", out.String())
	}
}

func TestIncidentClose(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, cfgPath, "version: \"1\"\n")

	if err := state.SaveIncidentLog(cfgPath, &state.IncidentLog{Incidents: []state.IncidentEntry{{
		ID:        "inc-001",
		Title:     "DBが応答しない",
		Severity:  "major",
		CreatedAt: "2026-04-05T00:00:00",
		Status:    "open",
	}}}); err != nil {
		t.Fatalf("SaveIncidentLog error: %v", err)
	}

	command := newRootCmd("test")
	command.AddCommand(newIncidentCmd())
	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"incident", "close", "--id", "inc-001", "--config", cfgPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("incident close execute error: %v", err)
	}

	log, err := state.LoadIncidentLog(cfgPath)
	if err != nil {
		t.Fatalf("LoadIncidentLog error: %v", err)
	}
	if len(log.Incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(log.Incidents))
	}
	if log.Incidents[0].Status != "closed" {
		t.Fatalf("expected closed status, got %s", log.Incidents[0].Status)
	}
	if log.Incidents[0].ClosedAt == "" {
		t.Fatalf("expected closed_at to be set")
	}
}
