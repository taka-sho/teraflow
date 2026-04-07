package cmd

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestSetupVerifyFatalIssueReturnsError(t *testing.T) {
	oldGHExec := ghExecCommand
	oldGitExec := gitExecCommand

	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		switch {
		case len(args) >= 2 && args[0] == "api" && args[1] == "repos/acme/rocket/actions/permissions/workflow":
			return exec.Command("sh", "-c", `printf '{"default_workflow_permissions":"write","can_approve_pull_request_reviews":false}'`)
		case len(args) >= 2 && args[0] == "api" && args[1] == "repos/acme/rocket":
			return exec.Command("sh", "-c", `printf '{"has_discussions_enabled":true}'`)
		case len(args) >= 3 && args[0] == "secret" && args[1] == "list":
			return exec.Command("sh", "-c", "printf 'OPENAI_API_KEY 2026-01-01' ")
		default:
			return exec.Command("sh", "-c", "exit 1")
		}
	}
	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'git@github.com:acme/rocket.git'")
	}
	t.Cleanup(func() {
		ghExecCommand = oldGHExec
		gitExecCommand = oldGitExec
	})

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"setup", "verify"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected fatal issue error")
	}
	if !strings.Contains(err.Error(), "fatal issue") {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "❌ Allow Actions to create PRs: false") {
		t.Fatalf("expected fatal output, got: %s", got)
	}
}

func TestSetupVerifyWarningsOnlyReturnsNil(t *testing.T) {
	oldGHExec := ghExecCommand
	oldGitExec := gitExecCommand

	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		switch {
		case len(args) >= 2 && args[0] == "api" && args[1] == "repos/acme/rocket/actions/permissions/workflow":
			return exec.Command("sh", "-c", `printf '{"default_workflow_permissions":"write","can_approve_pull_request_reviews":true}'`)
		case len(args) >= 2 && args[0] == "api" && args[1] == "repos/acme/rocket":
			return exec.Command("sh", "-c", `printf '{"has_discussions_enabled":false}'`)
		case len(args) >= 3 && args[0] == "secret" && args[1] == "list":
			return exec.Command("sh", "-c", "printf ''")
		default:
			return exec.Command("sh", "-c", "exit 1")
		}
	}
	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'https://github.com/acme/rocket.git'")
	}
	t.Cleanup(func() {
		ghExecCommand = oldGHExec
		gitExecCommand = oldGitExec
	})

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"setup", "verify"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected warnings-only run to succeed, got: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "⚠️  Discussions disabled") {
		t.Fatalf("expected discussions warning, got: %s", got)
	}
	if !strings.Contains(got, "0 issues found.") {
		t.Fatalf("expected 0 fatal issues, got: %s", got)
	}
}

func TestSetupVerifyJSONOutput(t *testing.T) {
	oldGHExec := ghExecCommand
	oldGitExec := gitExecCommand

	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		switch {
		case len(args) >= 2 && args[0] == "api" && args[1] == "repos/acme/rocket/actions/permissions/workflow":
			return exec.Command("sh", "-c", `printf '{"default_workflow_permissions":"write","can_approve_pull_request_reviews":true}'`)
		case len(args) >= 2 && args[0] == "api" && args[1] == "repos/acme/rocket":
			return exec.Command("sh", "-c", `printf '{"has_discussions_enabled":true}'`)
		case len(args) >= 3 && args[0] == "secret" && args[1] == "list":
			return exec.Command("sh", "-c", "printf 'ANTHROPIC_API_KEY 2026-01-01' ")
		default:
			return exec.Command("sh", "-c", "exit 1")
		}
	}
	gitExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf 'git@github.com:acme/rocket.git'")
	}
	t.Cleanup(func() {
		ghExecCommand = oldGHExec
		gitExecCommand = oldGitExec
	})

	root := newRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--format", "json", "setup", "verify"})

	if err := root.Execute(); err != nil {
		t.Fatalf("setup verify --format json: %v", err)
	}

	var parsed setupVerifyResult
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput=%s", err, out.String())
	}
	if parsed.Repository.Owner != "acme" || parsed.Repository.Repo != "rocket" {
		t.Fatalf("unexpected repository: %+v", parsed.Repository)
	}
	if parsed.Issues.FatalCount != 0 {
		t.Fatalf("expected no fatal issues, got: %+v", parsed.Issues)
	}
	if !parsed.Checks.Secrets.AnthropicAPIKey {
		t.Fatalf("expected anthropic secret detected, got: %+v", parsed.Checks.Secrets)
	}
}

func TestResolveSetupVerifyRepositoryRequiresBothFlags(t *testing.T) {
	_, _, err := resolveSetupVerifyRepository("acme", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "both --owner and --repo") {
		t.Fatalf("unexpected error: %v", err)
	}
}
