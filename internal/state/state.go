package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectState represents .github/project-state.yml.
type ProjectState struct {
	Project   ProjectInfo   `yaml:"project"`
	Lifecycle LifecycleInfo `yaml:"lifecycle"`
	Phases    PhasesInfo    `yaml:"phases"`
}

type ProjectInfo struct {
	Name string `yaml:"name"`
}

type LifecycleInfo struct {
	CurrentStage string `yaml:"current_stage"`
}

type PhasesInfo struct {
	Current string `yaml:"current"`
}

// ReworkEntry represents a single rework record.
type ReworkEntry struct {
	ID          string `yaml:"id"`
	Group       string `yaml:"group"`
	TargetPhase string `yaml:"target_phase"`
	Reason      string `yaml:"reason"`
	CreatedAt   string `yaml:"created_at"`
	Status      string `yaml:"status"` // open | approved | rejected
}

// ReworkLog represents .teraflow/rework-log.yml.
type ReworkLog struct {
	Reworks []ReworkEntry `yaml:"reworks"`
}

// IncidentEntry represents a single incident record.
type IncidentEntry struct {
	ID          string `yaml:"id"`
	Title       string `yaml:"title"`
	Severity    string `yaml:"severity"` // critical | major | minor
	Description string `yaml:"description"`
	CreatedAt   string `yaml:"created_at"`
	ClosedAt    string `yaml:"closed_at,omitempty"`
	Status      string `yaml:"status"` // open | closed
}

// IncidentLog represents .teraflow/incident-log.yml.
type IncidentLog struct {
	Incidents []IncidentEntry `yaml:"incidents"`
}

func stateFilePath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "project-state.yml")
}

func reworkLogPath(configPath string) string {
	root := filepath.Dir(filepath.Dir(configPath))
	return filepath.Join(root, ".teraflow", "rework-log.yml")
}

func incidentLogPath(configPath string) string {
	root := filepath.Dir(filepath.Dir(configPath))
	return filepath.Join(root, ".teraflow", "incident-log.yml")
}

// LoadState loads .github/project-state.yml derived from configPath.
func LoadState(configPath string) (*ProjectState, error) {
	path := stateFilePath(configPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load project state: %w", err)
	}

	var s ProjectState
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse project state: %w", err)
	}

	return &s, nil
}

// SaveState writes .github/project-state.yml derived from configPath.
func SaveState(configPath string, s *ProjectState) error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal project state: %w", err)
	}

	path := stateFilePath(configPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write project state: %w", err)
	}

	return nil
}

// LoadReworkLog loads .teraflow/rework-log.yml derived from configPath.
// If the file does not exist, it returns an empty log.
func LoadReworkLog(configPath string) (*ReworkLog, error) {
	path := reworkLogPath(configPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &ReworkLog{Reworks: []ReworkEntry{}}, nil
		}
		return nil, fmt.Errorf("load rework log: %w", err)
	}

	var log ReworkLog
	if err := yaml.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("parse rework log: %w", err)
	}
	if log.Reworks == nil {
		log.Reworks = []ReworkEntry{}
	}

	return &log, nil
}

// AppendRework appends an entry to .teraflow/rework-log.yml.
func AppendRework(configPath string, entry ReworkEntry) error {
	log, err := LoadReworkLog(configPath)
	if err != nil {
		return err
	}

	log.Reworks = append(log.Reworks, entry)

	path := reworkLogPath(configPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create rework directory: %w", err)
	}

	data, err := yaml.Marshal(log)
	if err != nil {
		return fmt.Errorf("marshal rework log: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write rework log: %w", err)
	}

	return nil
}

// LoadIncidentLog loads .teraflow/incident-log.yml.
// If the file does not exist, it returns an empty log.
func LoadIncidentLog(configPath string) (*IncidentLog, error) {
	path := incidentLogPath(configPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &IncidentLog{Incidents: []IncidentEntry{}}, nil
		}
		return nil, fmt.Errorf("load incident log: %w", err)
	}
	var log IncidentLog
	if err := yaml.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("parse incident log: %w", err)
	}
	if log.Incidents == nil {
		log.Incidents = []IncidentEntry{}
	}
	return &log, nil
}

// SaveIncidentLog saves .teraflow/incident-log.yml.
func SaveIncidentLog(configPath string, log *IncidentLog) error {
	path := incidentLogPath(configPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create incident directory: %w", err)
	}
	data, err := yaml.Marshal(log)
	if err != nil {
		return fmt.Errorf("marshal incident log: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write incident log: %w", err)
	}
	return nil
}
