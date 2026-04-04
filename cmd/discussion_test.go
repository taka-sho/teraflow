package cmd

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDiscussionSummarizeWithAI(t *testing.T) {
	oldLookPath := ghLookPath
	oldExecCommand := ghExecCommand
	oldSummarizer := discussionSummarizer

	ghLookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'Discussion content'")
	}
	discussionSummarizer = func(content, apiKey string) (string, error) {
		return "Summary: " + content, nil
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
		ghExecCommand = oldExecCommand
		discussionSummarizer = oldSummarizer
	})

	_ = os.Setenv("ANTHROPIC_API_KEY", "dummy-key")
	t.Cleanup(func() { _ = os.Unsetenv("ANTHROPIC_API_KEY") })

	root := newRootCmd("test")
	root.AddCommand(newDiscussionCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"discussion", "summarize", "123"})

	if err := root.Execute(); err != nil {
		t.Fatalf("discussion summarize with AI failed: %v", err)
	}

	if !strings.Contains(out.String(), "Summary:") {
		t.Fatalf("expected summary in output, got: %s", out.String())
	}
}

func TestDiscussionSummarizeWithAIError(t *testing.T) {
	oldLookPath := ghLookPath
	oldExecCommand := ghExecCommand
	oldSummarizer := discussionSummarizer

	ghLookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'content'")
	}
	discussionSummarizer = func(content, apiKey string) (string, error) {
		return "", errors.New("AI error")
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
		ghExecCommand = oldExecCommand
		discussionSummarizer = oldSummarizer
	})

	_ = os.Setenv("ANTHROPIC_API_KEY", "dummy-key")
	t.Cleanup(func() { _ = os.Unsetenv("ANTHROPIC_API_KEY") })

	root := newRootCmd("test")
	root.AddCommand(newDiscussionCmd())
	root.SetArgs([]string{"discussion", "summarize", "123"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from AI summarizer")
	}
	if !strings.Contains(err.Error(), "AI error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

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

func TestDiscussionListText(t *testing.T) {
	oldLookPath := ghLookPath
	oldExecCommand := ghExecCommand

	ghLookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'Discussion list output'")
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
		ghExecCommand = oldExecCommand
	})

	root := newRootCmd("test")
	root.AddCommand(newDiscussionCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"discussion", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("discussion list failed: %v", err)
	}
}

func TestDiscussionListJSONFormat(t *testing.T) {
	oldLookPath := ghLookPath
	oldExecCommand := ghExecCommand

	ghLookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", `echo '[{"number":1,"title":"test","state":"OPEN"}]'`)
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
		ghExecCommand = oldExecCommand
	})

	root := newRootCmd("test")
	root.AddCommand(newDiscussionCmd())

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"discussion", "list", "--format", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("discussion list --format json failed: %v", err)
	}
}

func TestDiscussionListUnsupportedFormat(t *testing.T) {
	oldLookPath := ghLookPath
	ghLookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
	})

	root := newRootCmd("test")
	root.AddCommand(newDiscussionCmd())
	root.SetArgs([]string{"discussion", "list", "--format", "xml"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("expected unsupported format error, got: %v", err)
	}
}

func TestDiscussionSummarizeGHError(t *testing.T) {
	oldLookPath := ghLookPath
	oldExecCommand := ghExecCommand

	ghLookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	ghExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 1")
	}
	t.Cleanup(func() {
		ghLookPath = oldLookPath
		ghExecCommand = oldExecCommand
	})

	root := newRootCmd("test")
	root.AddCommand(newDiscussionCmd())
	root.SetArgs([]string{"discussion", "summarize", "123"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when gh fails")
	}
	if !strings.Contains(err.Error(), "E5003") {
		t.Fatalf("expected E5003 error, got: %v", err)
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

func TestSummarizeDiscussionSuccess(t *testing.T) {
	old := httpDoer
	httpDoer = func(req *http.Request) (*http.Response, error) {
		body := `{"content":[{"type":"text","text":"テスト要約"}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	t.Cleanup(func() { httpDoer = old })

	summary, err := summarizeDiscussionWithAnthropic("content", "key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary != "テスト要約" {
		t.Fatalf("got: %s", summary)
	}
}

func TestSummarizeDiscussionHTTPError(t *testing.T) {
	old := httpDoer
	httpDoer = func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	}
	t.Cleanup(func() { httpDoer = old })

	_, err := summarizeDiscussionWithAnthropic("content", "key")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "E6002") {
		t.Fatalf("expected E6002: %v", err)
	}
}

func TestSummarizeDiscussionNon200(t *testing.T) {
	old := httpDoer
	httpDoer = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 401, Body: io.NopCloser(strings.NewReader("unauthorized"))}, nil
	}
	t.Cleanup(func() { httpDoer = old })

	_, err := summarizeDiscussionWithAnthropic("content", "key")
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "E6002") {
		t.Fatalf("expected E6002: %v", err)
	}
}

func TestSummarizeDiscussionEmptyContent(t *testing.T) {
	old := httpDoer
	httpDoer = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"content":[]}`))}, nil
	}
	t.Cleanup(func() { httpDoer = old })

	_, err := summarizeDiscussionWithAnthropic("content", "key")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestSummarizeDiscussionInvalidJSON(t *testing.T) {
	old := httpDoer
	httpDoer = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("not json"))}, nil
	}
	t.Cleanup(func() { httpDoer = old })

	_, err := summarizeDiscussionWithAnthropic("content", "key")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
