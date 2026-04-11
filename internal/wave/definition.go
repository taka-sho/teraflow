package wave

import "strings"

// WaveDefinition defines one generation unit in a phase.
type WaveDefinition struct {
	Number       int      `yaml:"number" json:"number"`
	Name         string   `yaml:"name" json:"name"`
	ArtifactType string   `yaml:"artifact_type" json:"artifact_type"`
	Template     string   `yaml:"template" json:"template"`
	DependsOn    []int    `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Inputs       []string `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	ReviewReq    string   `yaml:"review_required,omitempty" json:"review_required,omitempty"`
	Parallel     bool     `yaml:"parallel,omitempty" json:"parallel,omitempty"`
}

// GeneratedArtifact represents one generated file.
type GeneratedArtifact struct {
	Path         string `json:"path"`
	NodeID       string `json:"node_id,omitempty"`
	ArtifactType string `json:"artifact_type"`
	ReviewReq    string `json:"review_required,omitempty"`
}

// WaveResult is the execution result for one wave.
type WaveResult struct {
	WaveNumber int                 `json:"wave_number"`
	WaveName   string              `json:"wave_name"`
	Artifacts  []GeneratedArtifact `json:"artifacts"`
	PRNumber   int                 `json:"pr_number,omitempty"`
	Status     string              `json:"status"`
	Warnings   []string            `json:"warnings,omitempty"`
}

// ExecuteOptions controls wave execution behavior.
type ExecuteOptions struct {
	DryRun           bool
	TemplateOverride string
}

// DefaultWaveDefinitions returns built-in wave plans per phase.
func DefaultWaveDefinitions(phase string) []WaveDefinition {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "basic-design", "basic_design":
		return []WaveDefinition{
			{Number: 1, Name: "Acceptance Criteria", ArtifactType: "acceptance-criteria", Template: "wave-acceptance-criteria.tmpl", ReviewReq: "review"},
			{Number: 2, Name: "ADR", ArtifactType: "adr", Template: "wave-adr.tmpl", DependsOn: []int{1}, ReviewReq: "review"},
			{Number: 3, Name: "System Design", ArtifactType: "system-design", Template: "wave-system-design.tmpl", DependsOn: []int{2}, ReviewReq: "review"},
			{Number: 4, Name: "DB Design", ArtifactType: "db-design", Template: "wave-db-design.tmpl", DependsOn: []int{3}, ReviewReq: "approve", Parallel: true},
			{Number: 5, Name: "API Design", ArtifactType: "api-design", Template: "wave-api-design.tmpl", DependsOn: []int{3}, ReviewReq: "approve", Parallel: true},
			{Number: 6, Name: "UI Design", ArtifactType: "ui-design", Template: "wave-ui-design.tmpl", DependsOn: []int{5}, ReviewReq: "review"},
			{Number: 7, Name: "Implementation Plan", ArtifactType: "implementation-plan", Template: "wave-implementation-plan.tmpl", DependsOn: []int{4, 5, 6}, ReviewReq: "approve"},
		}
	case "detailed-design", "detailed_design":
		return []WaveDefinition{
			{Number: 1, Name: "Module Split", ArtifactType: "module-split", Template: "wave-module-split.tmpl", ReviewReq: "review"},
			{Number: 2, Name: "Dataflow", ArtifactType: "dataflow", Template: "wave-dataflow.tmpl", DependsOn: []int{1}, ReviewReq: "review"},
			{Number: 3, Name: "Test Spec", ArtifactType: "test-spec", Template: "wave-test-spec.tmpl", DependsOn: []int{1, 2}, ReviewReq: "approve"},
		}
	default:
		return nil
	}
}
