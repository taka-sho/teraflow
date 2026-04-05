package constraint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type ConstraintType string

const (
	ConstraintRequiredPhase    ConstraintType = "required_phase"
	ConstraintRequiredCoverage ConstraintType = "required_coverage"
	ConstraintRequiredDocs     ConstraintType = "required_docs"
)

type Severity string

const (
	SeverityBlock Severity = "block"
	SeverityWarn  Severity = "warn"
)

type Constraint struct {
	Type     ConstraintType `yaml:"type"`
	Value    string         `yaml:"value,omitempty"`
	Severity Severity       `yaml:"severity"`
	Message  string         `yaml:"message,omitempty"`
}

type Result struct {
	Passed   bool
	Severity Severity
	Message  string
}

// Engine evaluates constraints.
type Engine struct{}

func (e *Engine) Check(constraints []Constraint, projectPath string) (blocked []Result, warnings []Result, err error) {
	for _, c := range constraints {
		result, violated, evalErr := evaluateConstraint(c, projectPath)
		if evalErr != nil {
			return nil, nil, evalErr
		}
		if !violated {
			continue
		}
		if result.Severity == SeverityWarn {
			warnings = append(warnings, result)
		} else {
			blocked = append(blocked, result)
		}
	}
	return blocked, warnings, nil
}

func evaluateConstraint(c Constraint, projectPath string) (Result, bool, error) {
	severity := c.Severity
	if severity == "" {
		severity = SeverityBlock
	}

	switch c.Type {
	case ConstraintRequiredPhase:
		current, err := loadCurrentPhase(projectPath)
		if err != nil {
			return Result{}, false, fmt.Errorf("evaluate required_phase: %w", err)
		}
		if phaseAtLeast(current, c.Value) {
			return Result{Passed: true, Severity: severity}, false, nil
		}
		msg := c.Message
		if strings.TrimSpace(msg) == "" {
			msg = fmt.Sprintf("required phase %q not reached (current: %q)", c.Value, current)
		}
		return Result{Passed: false, Severity: severity, Message: msg}, true, nil

	case ConstraintRequiredCoverage:
		required, err := parseCoverageValue(c.Value)
		if err != nil {
			return Result{}, false, fmt.Errorf("evaluate required_coverage: %w", err)
		}
		current, err := loadCoverage(projectPath)
		if err != nil {
			return Result{}, false, fmt.Errorf("evaluate required_coverage: %w", err)
		}
		if current >= required {
			return Result{Passed: true, Severity: severity}, false, nil
		}
		msg := c.Message
		if strings.TrimSpace(msg) == "" {
			msg = fmt.Sprintf("coverage %.1f%% is below required %.1f%%", current*100, required*100)
		}
		return Result{Passed: false, Severity: severity, Message: msg}, true, nil

	case ConstraintRequiredDocs:
		docs := parseDocList(c.Value)
		if len(docs) == 0 {
			return Result{}, false, fmt.Errorf("evaluate required_docs: value is empty")
		}
		missing := []string{}
		for _, doc := range docs {
			target := filepath.Join(projectPath, filepath.FromSlash(doc))
			if _, err := os.Stat(target); err != nil {
				if os.IsNotExist(err) {
					missing = append(missing, doc)
					continue
				}
				return Result{}, false, fmt.Errorf("evaluate required_docs: %w", err)
			}
		}
		if len(missing) == 0 {
			return Result{Passed: true, Severity: severity}, false, nil
		}
		msg := c.Message
		if strings.TrimSpace(msg) == "" {
			msg = fmt.Sprintf("required docs missing: %s", strings.Join(missing, ", "))
		}
		return Result{Passed: false, Severity: severity, Message: msg}, true, nil

	default:
		return Result{}, false, fmt.Errorf("unsupported constraint type: %s", c.Type)
	}
}

type stateView struct {
	Phases struct {
		Current string `yaml:"current"`
	} `yaml:"phases"`
}

func loadCurrentPhase(projectPath string) (string, error) {
	path := filepath.Join(projectPath, ".github", "project-state.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var s stateView
	if err := yaml.Unmarshal(data, &s); err != nil {
		return "", err
	}
	return s.Phases.Current, nil
}

var phaseOrder = []string{
	"requirements",
	"basic_design",
	"detailed_design",
	"implementation",
	"testing",
	"integration_test",
}

func phaseAtLeast(current, required string) bool {
	curIdx := phaseIndex(current)
	reqIdx := phaseIndex(required)
	if curIdx < 0 || reqIdx < 0 {
		return false
	}
	return curIdx >= reqIdx
}

func phaseIndex(phase string) int {
	for i, p := range phaseOrder {
		if p == phase {
			return i
		}
	}
	return -1
}

func loadCoverage(projectPath string) (float64, error) {
	candidates := []string{
		filepath.Join(projectPath, ".teraflow", "coverage-summary.txt"),
		filepath.Join(projectPath, ".teraflow", "coverage.txt"),
		filepath.Join(projectPath, ".teraflow", "coverage.out"),
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, err
		}
		if value, ok := parseCoverageFromText(string(data)); ok {
			return value, nil
		}
	}
	return 0, fmt.Errorf("coverage data not found (.teraflow/coverage-summary.txt|coverage.txt|coverage.out)")
}

func parseCoverageValue(raw string) (float64, error) {
	value, err := parseNumericCoverage(raw)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func parseCoverageFromText(text string) (float64, bool) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if value, err := parseNumericCoverage(line); err == nil {
			return value, true
		}
	}
	return 0, false
}

func parseNumericCoverage(raw string) (float64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("empty coverage value")
	}
	re := regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s*%?`)
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0, fmt.Errorf("invalid coverage value: %q", raw)
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, err
	}
	if v > 1.0 {
		v = v / 100.0
	}
	return v, nil
}

func parseDocList(raw string) []string {
	items := strings.Split(raw, ",")
	out := make([]string, 0, len(items))
	for _, item := range items {
		v := strings.TrimSpace(item)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
