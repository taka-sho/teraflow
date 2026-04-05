package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDoctorCmd(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "development"
phases:
  current: "implementation"
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "doctor"})

	// doctor may return error if issues found (e.g. go/git not in PATH on CI)
	// We just verify the command runs and prints results
	_ = root.Execute()
	got := out.String()
	if !strings.Contains(got, "teraflow doctor") && !strings.Contains(got, "status") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestNewDoctorCmdJSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "development"
phases:
  current: "implementation"
`)

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--config", configPath, "--format", "json", "doctor"})

	_ = root.Execute()
	got := out.String()
	if !strings.Contains(got, `"status"`) {
		t.Fatalf("expected JSON output with status: %q", got)
	}
}

func TestNewDoctorCmdInvalidFormat(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "development"
phases:
  current: "implementation"
`)

	root := newRootCmd("test")
	root.SetArgs([]string{"--config", configPath, "--format", "xml", "doctor"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunChecksInitialized(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "development"
phases:
  current: "implementation"
`)

	results := runChecks(configPath, false, false)
	cfg := byCategory(results, "configuration")
	if len(cfg) != 2 {
		t.Fatalf("expected 2 configuration checks, got %d", len(cfg))
	}
	for _, r := range cfg {
		if !r.OK {
			t.Fatalf("expected configuration check %q to be OK, got NG: %s", r.Name, r.Message)
		}
	}
}

func TestRunChecksNotInitialized(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)

	results := runChecks(configPath, false, false)
	cfg := byCategory(results, "configuration")
	found := false
	for _, r := range cfg {
		if r.Name == "project-state.yml" {
			found = true
			if r.OK {
				t.Fatalf("expected project-state.yml check to fail when missing")
			}
		}
	}
	if !found {
		t.Fatal("project-state.yml check was not found")
	}
}

func TestRunChecksNoAI(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "development"
phases:
  current: "implementation"
`)

	results := runChecks(configPath, false, false)
	if got := byCategory(results, "ai"); len(got) != 0 {
		t.Fatalf("expected no ai checks when checkAI=false, got %d", len(got))
	}
}

func TestRunChecksInvalidConfig(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	// File exists but has invalid YAML (unmarshal fails)
	mustWrite(t, configPath, ":\n  bad: [\nbroken\n")

	results := runChecks(configPath, false, false)
	cfg := byCategory(results, "configuration")
	found := false
	for _, r := range cfg {
		if r.Name == "teraflow.yml" {
			found = true
			if r.OK {
				t.Fatal("expected teraflow.yml check to fail for invalid YAML")
			}
		}
	}
	if !found {
		t.Fatal("teraflow.yml check not found")
	}
}

func TestCheckConfigurationInvalidStateYAML(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), ":\n  bad: [\nbroken\n")

	results := checkConfiguration(configPath)
	found := false
	for _, r := range results {
		if r.Name == "project-state.yml" {
			found = true
			if r.OK {
				t.Fatal("expected project-state.yml check to fail for invalid YAML")
			}
			if !strings.Contains(r.Message, "parse project state") {
				t.Fatalf("unexpected message: %s", r.Message)
			}
		}
	}
	if !found {
		t.Fatal("project-state.yml check not found")
	}
}

func TestCheckIntegrityInvalidLogs(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")
	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".teraflow", "rework-log.yml"), ":\n  bad: [\nbroken\n")
	mustWrite(t, filepath.Join(tmp, ".teraflow", "incident-log.yml"), ":\n  bad: [\nbroken\n")

	results := checkIntegrity(configPath)

	var reworkFound, incidentFound bool
	for _, r := range results {
		if r.Name == "rework-log.yml" {
			reworkFound = true
			if r.OK {
				t.Fatal("expected rework-log.yml check to fail")
			}
			if !strings.Contains(r.Message, "parse rework log") {
				t.Fatalf("unexpected rework message: %s", r.Message)
			}
		}
		if r.Name == "incident-log.yml" {
			incidentFound = true
			if r.OK {
				t.Fatal("expected incident-log.yml check to fail")
			}
			if !strings.Contains(r.Message, "parse incident log") {
				t.Fatalf("unexpected incident message: %s", r.Message)
			}
		}
	}
	if !reworkFound || !incidentFound {
		t.Fatalf("missing integrity checks, rework=%v incident=%v", reworkFound, incidentFound)
	}
}

func TestCheckAIIntegrationNoKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	results := checkAIIntegration(false)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].OK {
		t.Fatal("expected AI check to fail without key")
	}
}

func TestCheckAIIntegrationSkippedInCI(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	results := checkAIIntegration(true)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].OK {
		t.Fatal("expected AI check to be skipped in CI mode")
	}
	if !strings.Contains(results[0].Message, "skipped in CI mode") {
		t.Fatalf("unexpected message: %s", results[0].Message)
	}
}

func TestRunChecksWithAI(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "dummy-key")
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".github", "teraflow.yml")

	mustWrite(t, configPath, `version: "1"
project:
  name: "test"
`)
	mustWrite(t, filepath.Join(tmp, ".github", "project-state.yml"), `project:
  name: "test"
lifecycle:
  current_stage: "development"
phases:
  current: "implementation"
`)

	results := runChecks(configPath, true, false)
	ai := byCategory(results, "ai")
	if len(ai) != 1 {
		t.Fatalf("expected 1 ai check, got %d", len(ai))
	}
	if !ai[0].OK {
		t.Fatalf("expected ai check to be OK, got message: %s", ai[0].Message)
	}
}

func byCategory(results []CheckResult, category string) []CheckResult {
	filtered := make([]CheckResult, 0, len(results))
	for _, r := range results {
		if r.Category == category {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func TestCountIssues(t *testing.T) {
	results := []CheckResult{
		{OK: true},
		{OK: false},
		{OK: false},
	}
	if got := countIssues(results); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}

	if got := countIssues(nil); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestPrintCategory(t *testing.T) {
	var buf bytes.Buffer
	orig := doctorStdout
	doctorStdout = &buf
	defer func() { doctorStdout = orig }()

	results := []CheckResult{
		{Category: "environment", Name: "go", OK: true, Message: "ok"},
		{Category: "environment", Name: "missing", OK: false, Message: "not found"},
		{Category: "other", Name: "skip", OK: true, Message: "skip"},
	}
	printCategory("Environment", "environment", results)

	got := buf.String()
	if !strings.Contains(got, "✓ go") {
		t.Fatalf("expected ✓ go in output: %q", got)
	}
	if !strings.Contains(got, "✗ missing") {
		t.Fatalf("expected ✗ missing in output: %q", got)
	}
	if strings.Contains(got, "skip") {
		t.Fatalf("should not include other category: %q", got)
	}
}

func TestPrintDoctorResultsText(t *testing.T) {
	var buf bytes.Buffer
	orig := doctorStdout
	doctorStdout = &buf
	defer func() { doctorStdout = orig }()

	results := []CheckResult{
		{Category: "environment", Name: "go", OK: true, Message: "ok"},
		{Category: "configuration", Name: "config", OK: true, Message: "ok"},
		{Category: "integrity", Name: "state", OK: true, Message: "ok"},
	}
	err := printDoctorResults(results, "text", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "all checks passed") {
		t.Fatalf("expected all checks passed: %q", got)
	}
}

func TestPrintDoctorResultsTextWithIssue(t *testing.T) {
	var buf bytes.Buffer
	orig := doctorStdout
	doctorStdout = &buf
	defer func() { doctorStdout = orig }()

	results := []CheckResult{
		{Category: "environment", Name: "go", OK: false, Message: "missing"},
	}
	err := printDoctorResults(results, "text", false)
	if err == nil {
		t.Fatal("expected error when issues found")
	}
}

func TestPrintDoctorResultsJSON(t *testing.T) {
	var buf bytes.Buffer
	orig := doctorStdout
	doctorStdout = &buf
	defer func() { doctorStdout = orig }()

	results := []CheckResult{
		{Category: "environment", Name: "go", OK: true, Message: "ok"},
	}
	err := printDoctorResults(results, "json", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), `"status"`) {
		t.Fatalf("expected JSON with status: %q", buf.String())
	}
}

func TestCheckEnvironmentGoMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	results := checkEnvironment(false)
	goResult, ok := findCheck(results, "environment", "go")
	if !ok {
		t.Fatal("go check not found")
	}
	if goResult.OK {
		t.Fatalf("expected go check to fail, got: %+v", goResult)
	}
	if !strings.Contains(goResult.Message, "go not found") {
		t.Fatalf("unexpected message: %s", goResult.Message)
	}
}

func TestCheckEnvironmentToolsFoundAndAuthenticated(t *testing.T) {
	bin := t.TempDir()
	mustWriteExecutable(t, filepath.Join(bin, "go"), "#!/bin/sh\necho \"go version go1.24.0 darwin/amd64\"\n")
	mustWriteExecutable(t, filepath.Join(bin, "gh"), "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo \"gh version 2.70.0\"; exit 0; fi\nif [ \"$1\" = \"auth\" ] && [ \"$2\" = \"status\" ]; then exit 0; fi\nexit 0\n")
	mustWriteExecutable(t, filepath.Join(bin, "git"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", bin)

	results := checkEnvironment(false)

	goResult, ok := findCheck(results, "environment", "go")
	if !ok || !goResult.OK || goResult.Message != "go1.24.0" {
		t.Fatalf("unexpected go result: %+v (found=%v)", goResult, ok)
	}

	ghResult, ok := findCheck(results, "environment", "gh")
	if !ok || !ghResult.OK || !strings.Contains(ghResult.Message, "(authenticated)") {
		t.Fatalf("unexpected gh result: %+v (found=%v)", ghResult, ok)
	}

	gitResult, ok := findCheck(results, "environment", "git")
	if !ok || !gitResult.OK {
		t.Fatalf("unexpected git result: %+v (found=%v)", gitResult, ok)
	}
}

func TestCheckEnvironmentSkipsAuthInCI(t *testing.T) {
	bin := t.TempDir()
	mustWriteExecutable(t, filepath.Join(bin, "go"), "#!/bin/sh\necho \"go version go1.24.0 darwin/amd64\"\n")
	mustWriteExecutable(t, filepath.Join(bin, "gh"), "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo \"gh version 2.70.0\"; exit 0; fi\nif [ \"$1\" = \"auth\" ] && [ \"$2\" = \"status\" ]; then exit 1; fi\nexit 0\n")
	mustWriteExecutable(t, filepath.Join(bin, "git"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", bin)

	results := checkEnvironment(true)
	ghResult, ok := findCheck(results, "environment", "gh")
	if !ok || !ghResult.OK {
		t.Fatalf("unexpected gh result: %+v (found=%v)", ghResult, ok)
	}
	if !strings.Contains(ghResult.Message, "skipped in CI mode") {
		t.Fatalf("unexpected gh message: %s", ghResult.Message)
	}
}

func TestCheckAgentProviderBranches(t *testing.T) {
	tmp := t.TempDir()
	customPath := filepath.Join(tmp, "custom-agent.sh")
	mustWriteExecutable(t, customPath, "#!/bin/sh\nexit 0\n")
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-secret-key")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("PATH", t.TempDir()) // no claude by default

	t.Run("config load error", func(t *testing.T) {
		results := checkAgentProvider(filepath.Join(tmp, "missing.yml"), false)
		provider, ok := findCheck(results, "agent_provider", "provider")
		if !ok || provider.OK {
			t.Fatalf("expected provider load error, got: %+v", provider)
		}
	})

	t.Run("anthropic default with fallback", func(t *testing.T) {
		configPath := writeDoctorConfigForProviderTest(t, `
ai:
  default_provider: anthropic
agent:
  fallback: openai
`)
		results := checkAgentProvider(configPath, false)

		provider, ok := findCheck(results, "agent_provider", "provider")
		if !ok || provider.Message != "anthropic" {
			t.Fatalf("unexpected provider result: %+v", provider)
		}
		key, ok := findCheck(results, "agent_provider", "ANTHROPIC_API_KEY")
		if !ok || !key.OK || !strings.Contains(key.Message, "set (anthropi") {
			t.Fatalf("unexpected key result: %+v", key)
		}
		fallback, ok := findCheck(results, "agent_provider", "fallback")
		if !ok || fallback.Message != "openai" {
			t.Fatalf("unexpected fallback result: %+v", fallback)
		}
	})

	t.Run("openai without key", func(t *testing.T) {
		configPath := writeDoctorConfigForProviderTest(t, `
ai:
  default_provider: anthropic
agent:
  provider: openai
`)
		results := checkAgentProvider(configPath, false)
		openai, ok := findCheck(results, "agent_provider", "OPENAI_API_KEY")
		if !ok || openai.OK || openai.Message != "not set" {
			t.Fatalf("unexpected OPENAI_API_KEY result: %+v", openai)
		}
	})

	t.Run("anthropic in ci skips api key", func(t *testing.T) {
		t.Setenv("ANTHROPIC_API_KEY", "")
		configPath := writeDoctorConfigForProviderTest(t, `
ai:
  default_provider: anthropic
agent:
  provider: anthropic
`)
		results := checkAgentProvider(configPath, true)
		key, ok := findCheck(results, "agent_provider", "ANTHROPIC_API_KEY")
		if !ok || !key.OK || key.Message != "skipped in CI mode" {
			t.Fatalf("unexpected ANTHROPIC_API_KEY result: %+v", key)
		}
	})

	t.Run("claude-code missing", func(t *testing.T) {
		configPath := writeDoctorConfigForProviderTest(t, `
ai:
  default_provider: anthropic
agent:
  provider: claude-code
`)
		results := checkAgentProvider(configPath, false)
		claude, ok := findCheck(results, "agent_provider", "claude CLI")
		if !ok || claude.OK {
			t.Fatalf("expected missing claude CLI, got: %+v", claude)
		}
	})

	t.Run("custom not configured", func(t *testing.T) {
		configPath := writeDoctorConfigForProviderTest(t, `
ai:
  default_provider: anthropic
agent:
  provider: custom
`)
		results := checkAgentProvider(configPath, false)
		custom, ok := findCheck(results, "agent_provider", "custom_command")
		if !ok || custom.OK || custom.Message != "not configured" {
			t.Fatalf("unexpected custom result: %+v", custom)
		}
	})

	t.Run("custom command missing path", func(t *testing.T) {
		configPath := writeDoctorConfigForProviderTest(t, `
ai:
  default_provider: anthropic
agent:
  provider: custom
  custom_command: "/path/does/not/exist"
`)
		results := checkAgentProvider(configPath, false)
		custom, ok := findCheck(results, "agent_provider", "custom_command")
		if !ok || custom.OK || !strings.Contains(custom.Message, "not found") {
			t.Fatalf("unexpected custom result: %+v", custom)
		}
	})

	t.Run("claude-code available", func(t *testing.T) {
		bin := t.TempDir()
		mustWriteExecutable(t, filepath.Join(bin, "claude"), "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", bin)

		configPath := writeDoctorConfigForProviderTest(t, `
ai:
  default_provider: anthropic
agent:
  provider: claude-code
`)
		results := checkAgentProvider(configPath, false)
		claude, ok := findCheck(results, "agent_provider", "claude CLI")
		if !ok || !claude.OK || claude.Message != "available" {
			t.Fatalf("unexpected claude CLI result: %+v", claude)
		}
	})

	t.Run("custom command found", func(t *testing.T) {
		body := fmt.Sprintf(`
ai:
  default_provider: anthropic
agent:
  provider: custom
  custom_command: %q
`, customPath)
		configPath := writeDoctorConfigForProviderTest(t, body)
		results := checkAgentProvider(configPath, false)
		custom, ok := findCheck(results, "agent_provider", "custom_command")
		if !ok || !custom.OK || custom.Message != customPath {
			t.Fatalf("unexpected custom result: %+v", custom)
		}
	})
}

func writeDoctorConfigForProviderTest(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "teraflow.yml")
	mustWrite(t, configPath, "version: \"1\"\nproject:\n  name: \"test\"\n"+body)
	return configPath
}

func mustWriteExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

func findCheck(results []CheckResult, category, name string) (CheckResult, bool) {
	for _, r := range results {
		if r.Category == category && r.Name == name {
			return r, true
		}
	}
	return CheckResult{}, false
}

func TestIsCIEnvironment(t *testing.T) {
	t.Setenv("CI", "true")
	if !isCIEnvironment() {
		t.Fatal("expected CI=true to be detected")
	}
	t.Setenv("CI", "false")
	if isCIEnvironment() {
		t.Fatal("expected CI=false to disable CI mode")
	}
}
