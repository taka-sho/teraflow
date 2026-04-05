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
	SLCPJCF   SLCPJCFState  `yaml:"slcp_jcf,omitempty"`
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

// SLCPJCFProcess represents SLCP-JCF process tracking state.
type SLCPJCFProcess struct {
	Name        string `yaml:"name"`
	Status      string `yaml:"status"` // not_started | in_progress | completed
	StartedAt   string `yaml:"started_at,omitempty"`
	CompletedAt string `yaml:"completed_at,omitempty"`
}

// SLCPJCFState tracks SLCP-JCF process progress.
type SLCPJCFState struct {
	Processes      []SLCPJCFProcess `yaml:"processes,omitempty"`
	CurrentProcess string           `yaml:"current_process,omitempty"`
	UpdatedAt      string           `yaml:"updated_at,omitempty"`
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

// DefaultSLCPJCFProcesses returns the standard 8 SLCP-JCF processes.
func DefaultSLCPJCFProcesses() []SLCPJCFProcess {
	return []SLCPJCFProcess{
		{Name: "企画プロセス", Status: "not_started"},
		{Name: "要件定義プロセス", Status: "not_started"},
		{Name: "システム設計プロセス", Status: "not_started"},
		{Name: "ソフトウェア設計プロセス", Status: "not_started"},
		{Name: "ソフトウェア構築プロセス", Status: "not_started"},
		{Name: "ソフトウェアテストプロセス", Status: "not_started"},
		{Name: "システム結合テスト", Status: "not_started"},
		{Name: "運用・保守プロセス", Status: "not_started"},
	}
}

// NewProjectState builds a default project state for initialization flows.
func NewProjectState(projectName, stage string) *ProjectState {
	if stage == "" {
		stage = "initial_development"
	}
	return &ProjectState{
		Project:   ProjectInfo{Name: projectName},
		Lifecycle: LifecycleInfo{CurrentStage: stage},
		Phases:    PhasesInfo{Current: "requirements"},
		SLCPJCF: SLCPJCFState{
			Processes: DefaultSLCPJCFProcesses(),
		},
	}
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
