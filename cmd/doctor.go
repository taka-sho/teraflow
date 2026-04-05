package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/state"
)

// CheckResult is one health check result in the doctor command.
type CheckResult struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Message  string `json:"message"`
}

type doctorJSONOutput struct {
	Status string        `json:"status"`
	Checks []CheckResult `json:"checks"`
	Issues int           `json:"issues"`
}

var doctorStdout io.Writer = os.Stdout

func newDoctorCmd() *cobra.Command {
	var checkAI bool
	var ciMode bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check project health and environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			doctorStdout = cmd.OutOrStdout()
			effectiveCIMode := ciMode || isCIEnvironment()
			results := runChecks(configPath, checkAI, effectiveCIMode)
			if err := printDoctorResults(results, format, checkAI); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&checkAI, "check-ai", false, "Also check AI integration")
	cmd.Flags().BoolVar(&ciMode, "ci", false, "Skip external auth checks for CI environments")
	return cmd
}

func runChecks(configPath string, checkAI, ciMode bool) []CheckResult {
	results := make([]CheckResult, 0, 12)
	results = append(results, checkEnvironment(ciMode)...)
	results = append(results, checkConfiguration(configPath)...)
	results = append(results, checkIntegrity(configPath)...)
	results = append(results, checkTemplates(configPath)...)
	if checkAI {
		results = append(results, checkAIIntegration(ciMode)...)
	}
	results = append(results, checkAgentProvider(configPath, ciMode)...)
	return results
}

func checkEnvironment(ciMode bool) []CheckResult {
	results := make([]CheckResult, 0, 3)

	if goPath, err := exec.LookPath("go"); err != nil {
		results = append(results, CheckResult{Category: "environment", Name: "go", OK: false, Message: "go not found"})
	} else {
		message := "go found"
		if out, err := exec.Command(goPath, "version").Output(); err == nil {
			fields := strings.Fields(strings.TrimSpace(string(out)))
			if len(fields) >= 3 {
				message = fields[2]
			}
		}
		results = append(results, CheckResult{Category: "environment", Name: "go", OK: true, Message: message})
	}

	if ghPath, err := exec.LookPath("gh"); err != nil {
		results = append(results, CheckResult{Category: "environment", Name: "gh", OK: false, Message: "gh not found"})
	} else {
		version := "gh found"
		if out, err := exec.Command(ghPath, "--version").Output(); err == nil {
			line := strings.Split(strings.TrimSpace(string(out)), "\n")[0]
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				version = fields[0] + " " + fields[2]
			}
		}

		if ciMode {
			results = append(results, CheckResult{Category: "environment", Name: "gh", OK: true, Message: version + " (auth check skipped in CI mode)"})
		} else {
			authErr := exec.Command(ghPath, "auth", "status").Run()
			if authErr != nil {
				results = append(results, CheckResult{Category: "environment", Name: "gh", OK: false, Message: "gh found (not authenticated)"})
			} else {
				results = append(results, CheckResult{Category: "environment", Name: "gh", OK: true, Message: version + " (authenticated)"})
			}
		}
	}

	if _, err := exec.LookPath("git"); err != nil {
		results = append(results, CheckResult{Category: "environment", Name: "git", OK: false, Message: "git not found"})
	} else {
		results = append(results, CheckResult{Category: "environment", Name: "git", OK: true, Message: "git found"})
	}

	return results
}

func checkConfiguration(configPath string) []CheckResult {
	results := make([]CheckResult, 0, 2)

	if _, err := os.Stat(configPath); err != nil {
		results = append(results, CheckResult{Category: "configuration", Name: "teraflow.yml", OK: false, Message: err.Error()})
	} else if _, err := config.Load(configPath); err != nil {
		results = append(results, CheckResult{Category: "configuration", Name: "teraflow.yml", OK: false, Message: err.Error()})
	} else {
		results = append(results, CheckResult{Category: "configuration", Name: "teraflow.yml", OK: true, Message: "teraflow.yml (valid)"})
	}

	statePath := filepath.Join(filepath.Dir(configPath), "project-state.yml")
	if _, err := os.Stat(statePath); err != nil {
		if os.IsNotExist(err) {
			results = append(results, CheckResult{Category: "configuration", Name: "project-state.yml", OK: false, Message: "not initialized (run teraflow init)"})
		} else {
			results = append(results, CheckResult{Category: "configuration", Name: "project-state.yml", OK: false, Message: err.Error()})
		}
	} else if _, err := state.LoadState(configPath); err != nil {
		results = append(results, CheckResult{Category: "configuration", Name: "project-state.yml", OK: false, Message: err.Error()})
	} else {
		results = append(results, CheckResult{Category: "configuration", Name: "project-state.yml", OK: true, Message: "project-state.yml (valid)"})
	}

	return results
}

func checkIntegrity(configPath string) []CheckResult {
	results := make([]CheckResult, 0, 2)

	reworkLog, err := state.LoadReworkLog(configPath)
	if err != nil {
		results = append(results, CheckResult{Category: "integrity", Name: "rework-log.yml", OK: false, Message: err.Error()})
	} else {
		results = append(results, CheckResult{Category: "integrity", Name: "rework-log.yml", OK: true, Message: fmt.Sprintf("%d entries", len(reworkLog.Reworks))})
	}

	incidentLog, err := state.LoadIncidentLog(configPath)
	if err != nil {
		results = append(results, CheckResult{Category: "integrity", Name: "incident-log.yml", OK: false, Message: err.Error()})
	} else {
		results = append(results, CheckResult{Category: "integrity", Name: "incident-log.yml", OK: true, Message: fmt.Sprintf("%d entries", len(incidentLog.Incidents))})
	}

	return results
}

func checkTemplates(configPath string) []CheckResult {
	results := make([]CheckResult, 0, 2)
	cwd := filepath.Dir(filepath.Dir(configPath))

	issueDir := filepath.Join(cwd, ".github", "ISSUE_TEMPLATE")
	if entries, err := os.ReadDir(issueDir); err != nil {
		results = append(results, CheckResult{
			Category: "templates",
			Name:     "issue-templates",
			OK:       false,
			Message:  "not found; run: teraflow setup templates",
		})
	} else {
		results = append(results, CheckResult{
			Category: "templates",
			Name:     "issue-templates",
			OK:       true,
			Message:  fmt.Sprintf("%d templates found", len(entries)),
		})
	}

	discussionDir := filepath.Join(cwd, ".github", "DISCUSSION_TEMPLATE")
	if entries, err := os.ReadDir(discussionDir); err != nil {
		results = append(results, CheckResult{
			Category: "templates",
			Name:     "discussion-templates",
			OK:       false,
			Message:  "not found; run: teraflow setup templates",
		})
	} else {
		results = append(results, CheckResult{
			Category: "templates",
			Name:     "discussion-templates",
			OK:       true,
			Message:  fmt.Sprintf("%d templates found", len(entries)),
		})
	}

	return results
}

func checkAIIntegration(ciMode bool) []CheckResult {
	if ciMode {
		return []CheckResult{{Category: "ai", Name: "ANTHROPIC_API_KEY", OK: true, Message: "skipped in CI mode"}}
	}
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		return []CheckResult{{Category: "ai", Name: "ANTHROPIC_API_KEY", OK: false, Message: "not set"}}
	}
	return []CheckResult{{Category: "ai", Name: "ANTHROPIC_API_KEY", OK: true, Message: "set"}}
}

func checkAgentProvider(configPath string, ciMode bool) []CheckResult {
	cfg, err := config.Load(configPath)
	if err != nil {
		return []CheckResult{{
			Category: "agent_provider",
			Name:     "provider",
			OK:       false,
			Message:  "could not load config: " + err.Error(),
		}}
	}

	provider := cfg.Agent.Provider
	if provider == "" {
		provider = cfg.AI.DefaultProvider
	}
	if provider == "" {
		provider = "anthropic"
	}

	results := []CheckResult{{
		Category: "agent_provider",
		Name:     "provider",
		OK:       true,
		Message:  provider,
	}}

	switch provider {
	case "anthropic", "":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey != "" {
			results = append(results, CheckResult{
				Category: "agent_provider",
				Name:     "ANTHROPIC_API_KEY",
				OK:       true,
				Message:  "set (" + maskSecret(apiKey) + ")",
			})
		} else if ciMode {
			results = append(results, CheckResult{
				Category: "agent_provider",
				Name:     "ANTHROPIC_API_KEY",
				OK:       true,
				Message:  "skipped in CI mode",
			})
		} else {
			results = append(results, CheckResult{
				Category: "agent_provider",
				Name:     "ANTHROPIC_API_KEY",
				OK:       false,
				Message:  "not set (AI features disabled)",
			})
		}
	case "claude-code":
		if _, err := exec.LookPath("claude"); err != nil {
			results = append(results, CheckResult{
				Category: "agent_provider",
				Name:     "claude CLI",
				OK:       false,
				Message:  "not found in PATH",
			})
		} else {
			results = append(results, CheckResult{
				Category: "agent_provider",
				Name:     "claude CLI",
				OK:       true,
				Message:  "available",
			})
		}
	case "openai":
		if os.Getenv("OPENAI_API_KEY") != "" {
			results = append(results, CheckResult{
				Category: "agent_provider",
				Name:     "OPENAI_API_KEY",
				OK:       true,
				Message:  "set",
			})
		} else if ciMode {
			results = append(results, CheckResult{
				Category: "agent_provider",
				Name:     "OPENAI_API_KEY",
				OK:       true,
				Message:  "skipped in CI mode",
			})
		} else {
			results = append(results, CheckResult{
				Category: "agent_provider",
				Name:     "OPENAI_API_KEY",
				OK:       false,
				Message:  "not set",
			})
		}
	case "custom":
		customCmd := cfg.Agent.CustomCommand
		if customCmd == "" {
			results = append(results, CheckResult{
				Category: "agent_provider",
				Name:     "custom_command",
				OK:       false,
				Message:  "not configured",
			})
		} else if _, err := os.Stat(customCmd); err != nil {
			results = append(results, CheckResult{
				Category: "agent_provider",
				Name:     "custom_command",
				OK:       false,
				Message:  "not found: " + customCmd,
			})
		} else {
			results = append(results, CheckResult{
				Category: "agent_provider",
				Name:     "custom_command",
				OK:       true,
				Message:  customCmd,
			})
		}
	}

	if cfg.Agent.Fallback != "" {
		results = append(results, CheckResult{
			Category: "agent_provider",
			Name:     "fallback",
			OK:       true,
			Message:  cfg.Agent.Fallback,
		})
	} else {
		results = append(results, CheckResult{
			Category: "agent_provider",
			Name:     "fallback",
			OK:       true,
			Message:  "none (no fallback configured)",
		})
	}

	return results
}

func isCIEnvironment() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("CI")), "true")
}

func maskSecret(v string) string {
	n := len(v)
	if n > 8 {
		n = 8
	}
	return v[:n] + "****"
}

func printDoctorResults(results []CheckResult, format string, checkAI bool) error {
	issues := countIssues(results)

	if format == "json" {
		status := "ok"
		if issues > 0 {
			status = "issues_found"
		}
		out := doctorJSONOutput{Status: status, Checks: results, Issues: issues}
		enc := json.NewEncoder(doctorStdout)
		if err := enc.Encode(out); err != nil {
			return err
		}
		if issues > 0 {
			return fmt.Errorf("doctor found %d issue(s)", issues)
		}
		return nil
	}

	fmt.Fprintln(doctorStdout, "teraflow doctor")
	fmt.Fprintln(doctorStdout)
	printCategory("Environment", "environment", results)
	fmt.Fprintln(doctorStdout)
	printCategory("Configuration", "configuration", results)
	fmt.Fprintln(doctorStdout)
	printCategory("Project Integrity", "integrity", results)
	fmt.Fprintln(doctorStdout)
	printCategory("Templates", "templates", results)
	fmt.Fprintln(doctorStdout)
	if checkAI {
		printCategory("AI Integration", "ai", results)
	} else {
		fmt.Fprintln(doctorStdout, "AI Integration (skipped — use --check-ai to test)")
	}
	fmt.Fprintln(doctorStdout)
	printCategory("Agent Provider Checks", "agent_provider", results)
	fmt.Fprintln(doctorStdout)

	if issues > 0 {
		word := "issues"
		if issues == 1 {
			word = "issue"
		}
		fmt.Fprintf(doctorStdout, "Result: %d %s found\n", issues, word)
		return fmt.Errorf("doctor found %d issue(s)", issues)
	}

	fmt.Fprintln(doctorStdout, "Result: all checks passed")
	return nil
}

func printCategory(title, category string, results []CheckResult) {
	fmt.Fprintln(doctorStdout, title)
	for _, r := range results {
		if r.Category != category {
			continue
		}
		mark := "✗"
		if r.OK {
			mark = "✓"
		}
		fmt.Fprintf(doctorStdout, "  %s %s — %s\n", mark, r.Name, r.Message)
	}
}

func countIssues(results []CheckResult) int {
	issues := 0
	for _, r := range results {
		if !r.OK {
			issues++
		}
	}
	return issues
}
