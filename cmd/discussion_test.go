package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDiscussionListNoGH(t *testing.T) {
	oldLookPath := ghLookPath
	ghLookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
	})

	root := newRootCmd("test")
	root.AddCommand(newDiscussionCmd())
	root.SetArgs([]string{"discussion", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "E5001") {
		t.Fatalf("expected E5001, got: %v", err)
	}
}

func TestDiscussionSummarizeNoAI(t *testing.T) {
	oldLookPath := ghLookPath
	oldExecCommand := ghExecCommand
	oldAnthropic := os.Getenv("ANTHROPIC_API_KEY")

	ghLookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		if name == "gh" && len(args) >= 3 && args[0] == "discussion" && args[1] == "view" {
			return exec.Command("sh", "-c", "printf 'Discussion body from gh' ")
		}
		return exec.Command("sh", "-c", "exit 1")
	}
	_ = os.Unsetenv("ANTHROPIC_API_KEY")
	defer func() {
		ghLookPath = oldLookPath
		ghExecCommand = oldExecCommand
		if oldAnthropic == "" {
			_ = os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			_ = os.Setenv("ANTHROPIC_API_KEY", oldAnthropic)
		}
	}()

	root := newRootCmd("test")
	root.AddCommand(newDiscussionCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"discussion", "summarize", "123"})

	if err := root.Execute(); err != nil {
		t.Fatalf("discussion summarize failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "AI summarization requires ANTHROPIC_API_KEY. Skipping.") {
		t.Fatalf("expected skip message, got:\n%s", got)
	}
	if !strings.Contains(got, "Discussion body from gh") {
		t.Fatalf("expected discussion body output, got:\n%s", got)
	}
}
