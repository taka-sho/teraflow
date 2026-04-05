package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConditionType defines available gate condition types.
type ConditionType string

const (
	ConditionManualApproval ConditionType = "manual_approval"
	ConditionDocumentExists ConditionType = "document_exists"
	ConditionTestPassRate   ConditionType = "test_pass_rate"
	ConditionCoverage       ConditionType = "coverage"
)

// GateRule defines conditions for a process gate.
type GateRule struct {
	ProcessName  string      `yaml:"process_name"`
	Conditions   []Condition `yaml:"conditions"`
	ApproverRole string      `yaml:"approver_role"`
}

// Condition represents a single gate condition.
type Condition struct {
	Type  ConditionType `yaml:"type"`
	Value string        `yaml:"value,omitempty"`
}

// Evaluate checks if gate conditions are met.
func Evaluate(rule GateRule, projectPath string) (bool, []string, error) {
	messages := make([]string, 0, len(rule.Conditions))
	allPassed := true

	for _, c := range rule.Conditions {
		ok, msg, err := EvaluateCondition(c, projectPath)
		if err != nil {
			return false, nil, err
		}
		messages = append(messages, msg)
		if !ok {
			allPassed = false
		}
	}

	return allPassed, messages, nil
}

// EvaluateCondition checks a single condition.
func EvaluateCondition(c Condition, projectPath string) (bool, string, error) {
	switch c.Type {
	case ConditionManualApproval:
		return true, "manual approval granted", nil
	case ConditionDocumentExists:
		if strings.TrimSpace(c.Value) == "" {
			return false, "document path is empty", nil
		}
		target := filepath.Join(projectPath, c.Value)
		if _, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) {
				return false, fmt.Sprintf("document not found: %s", c.Value), nil
			}
			return false, "", fmt.Errorf("check document_exists: %w", err)
		}
		return true, fmt.Sprintf("document exists: %s", c.Value), nil
	case ConditionTestPassRate:
		threshold, err := parseThreshold(c.Value)
		if err != nil {
			return false, "", err
		}
		metrics, err := loadMetrics(projectPath)
		if err != nil {
			return false, "", err
		}
		ok := metrics.TestPassRate >= threshold
		return ok, fmt.Sprintf("test_pass_rate %.2f >= %.2f", metrics.TestPassRate, threshold), nil
	case ConditionCoverage:
		threshold, err := parseThreshold(c.Value)
		if err != nil {
			return false, "", err
		}
		metrics, err := loadMetrics(projectPath)
		if err != nil {
			return false, "", err
		}
		ok := metrics.Coverage >= threshold
		return ok, fmt.Sprintf("coverage %.2f >= %.2f", metrics.Coverage, threshold), nil
	default:
		return false, "", fmt.Errorf("unsupported gate condition type: %s", c.Type)
	}
}

type metrics struct {
	TestPassRate float64 `yaml:"test_pass_rate"`
	Coverage     float64 `yaml:"coverage"`
}

func parseThreshold(raw string) (float64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("condition threshold value is required")
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid threshold value %q: %w", value, err)
	}
	return f, nil
}

func loadMetrics(projectPath string) (*metrics, error) {
	path := filepath.Join(projectPath, ".teraflow", "gate-metrics.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load gate metrics: %w", err)
	}

	var m metrics
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse gate metrics: %w", err)
	}
	return &m, nil
}
