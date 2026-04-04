package agent_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/taka-sho/teraflow/internal/agent"
)

type mockFallbackProvider struct {
	name   string
	output string
	tokens int
	err    error
}

func (m *mockFallbackProvider) Name() string {
	return m.name
}

func (m *mockFallbackProvider) Complete(_ context.Context, _, _ string, _ int) (string, int, error) {
	return m.output, m.tokens, m.err
}

func TestNewProviderFromConfigAnthropic(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	p, err := agent.NewProviderFromConfig(agent.ProviderConfig{Provider: "anthropic"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "anthropic" {
		t.Fatalf("expected anthropic, got %s", p.Name())
	}
}

func TestNewProviderFromConfigAnthropicMissingKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	_, err := agent.NewProviderFromConfig(agent.ProviderConfig{Provider: "anthropic"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY not set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewProviderFromConfigDefaultProvider(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	p, err := agent.NewProviderFromConfig(agent.ProviderConfig{Provider: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "anthropic" {
		t.Fatalf("expected anthropic, got %s", p.Name())
	}
}

func TestNewProviderFromConfigClaudeCode(t *testing.T) {
	p, err := agent.NewProviderFromConfig(agent.ProviderConfig{Provider: "claude-code"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "claude-code" {
		t.Fatalf("expected claude-code, got %s", p.Name())
	}
}

func TestNewProviderFromConfigOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	p, err := agent.NewProviderFromConfig(agent.ProviderConfig{Provider: "openai"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "openai" {
		t.Fatalf("expected openai, got %s", p.Name())
	}
}

func TestNewProviderFromConfigOpenAIMissingKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	_, err := agent.NewProviderFromConfig(agent.ProviderConfig{Provider: "openai"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "OPENAI_API_KEY not set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewProviderFromConfigCustomValidation(t *testing.T) {
	_, err := agent.NewProviderFromConfig(agent.ProviderConfig{Provider: "custom"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "custom_command is required") {
		t.Fatalf("unexpected error: %v", err)
	}

	p, err := agent.NewProviderFromConfig(agent.ProviderConfig{Provider: "custom", CustomCommand: "/usr/bin/true"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "custom" {
		t.Fatalf("expected custom, got %s", p.Name())
	}
}

func TestNewProviderFromConfigFallbackSecondaryError(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	_, err := agent.NewProviderFromConfig(agent.ProviderConfig{
		Provider: "fallback",
		Fallback: "anthropic",
	})
	if err == nil {
		t.Fatal("expected error when fallback inner provider fails")
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY not set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewProviderFromConfigFallbackSuccess(t *testing.T) {
	p, err := agent.NewProviderFromConfig(agent.ProviderConfig{
		Provider: "fallback",
		Fallback: "claude-code",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "claude-code" {
		t.Fatalf("unexpected provider name: %s", p.Name())
	}
}

func TestNewProviderFromConfigUnknown(t *testing.T) {
	_, err := agent.NewProviderFromConfig(agent.ProviderConfig{Provider: "unknown"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown agent provider") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClaudeCodeProviderName(t *testing.T) {
	p := agent.NewClaudeCodeProvider("")
	if p.Name() != "claude-code" {
		t.Fatalf("expected claude-code, got %s", p.Name())
	}
}

func TestNewAnthropicProviderWithClient(t *testing.T) {
	client := &http.Client{Timeout: 2 * time.Second}
	p := agent.NewAnthropicProviderWithClient("test-key", "claude-test", client)
	if p.Name() != "anthropic" {
		t.Fatalf("expected anthropic, got %s", p.Name())
	}
}

func TestNewAnthropicProviderWithClientDefaultModel(t *testing.T) {
	client := &http.Client{}
	p := agent.NewAnthropicProviderWithClient("test-key", "", client)
	if p.Name() != "anthropic" {
		t.Fatalf("expected anthropic, got %s", p.Name())
	}
}

func TestClaudeCodeProviderCompleteSuccess(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "claude")
	script := "#!/bin/sh\necho 'claude-output'\n"
	if err := os.WriteFile(claudePath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude script: %v", err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	p := agent.NewClaudeCodeProvider("haiku")
	out, tokens, err := p.Complete(context.Background(), "sys", "user", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "claude-output" {
		t.Fatalf("unexpected output: %q", out)
	}
	if tokens != 0 {
		t.Fatalf("unexpected tokens: %d", tokens)
	}
}

func TestClaudeCodeProviderCompleteCLIError(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "claude")
	script := "#!/bin/sh\nexit 1\n"
	if err := os.WriteFile(claudePath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude script: %v", err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	p := agent.NewClaudeCodeProvider("haiku")
	_, _, err := p.Complete(context.Background(), "sys", "user", 100)
	if err == nil {
		t.Fatal("expected error when claude exits with non-zero")
	}
}

func TestClaudeCodeProviderHealthCheckSuccess(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "claude")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  echo 'claude 1.0.0'\n  exit 0\nfi\necho 'ok'\n"
	if err := os.WriteFile(claudePath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude script: %v", err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	p := agent.NewClaudeCodeProvider("haiku")
	if err := p.HealthCheck(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClaudeCodeProviderHealthCheckMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	p := agent.NewClaudeCodeProvider("haiku")
	err := p.HealthCheck()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClaudeCodeProviderHealthCheckCLIError(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "claude")
	script := "#!/bin/sh\nexit 1\n"
	if err := os.WriteFile(claudePath, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude script: %v", err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	p := agent.NewClaudeCodeProvider("haiku")
	err := p.HealthCheck()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not working") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIProviderName(t *testing.T) {
	p := agent.NewOpenAIProvider("key", "")
	if p.Name() != "openai" {
		t.Fatalf("expected openai, got %s", p.Name())
	}
}

func TestOpenAIProviderCompleteSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("missing authorization header")
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		resp := map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "openai-ok"}}},
			"usage":   map[string]int{"total_tokens": 42},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	original := agent.OpenAIBaseURL
	agent.OpenAIBaseURL = server.URL
	defer func() { agent.OpenAIBaseURL = original }()

	p := agent.NewOpenAIProvider("test-key", "")
	out, tokens, err := p.Complete(context.Background(), "sys", "user", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "openai-ok" {
		t.Fatalf("unexpected output: %q", out)
	}
	if tokens != 42 {
		t.Fatalf("unexpected tokens: %d", tokens)
	}
}

func TestOpenAIProviderCompleteHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_api_key"}`))
	}))
	defer server.Close()

	original := agent.OpenAIBaseURL
	agent.OpenAIBaseURL = server.URL
	defer func() { agent.OpenAIBaseURL = original }()

	p := agent.NewOpenAIProvider("bad-key", "")
	_, _, err := p.Complete(context.Background(), "sys", "user", 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "E6002") || !strings.Contains(err.Error(), "401") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIProviderCompleteRequestFailure(t *testing.T) {
	original := agent.OpenAIBaseURL
	agent.OpenAIBaseURL = "http://127.0.0.1:1"
	defer func() { agent.OpenAIBaseURL = original }()

	p := agent.NewOpenAIProvider("bad-key", "")
	_, _, err := p.Complete(context.Background(), "sys", "user", 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "OpenAI API request failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIProviderCompleteEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{},
			"usage":   map[string]int{"total_tokens": 0},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	original := agent.OpenAIBaseURL
	agent.OpenAIBaseURL = server.URL
	defer func() { agent.OpenAIBaseURL = original }()

	p := agent.NewOpenAIProvider("key", "")
	_, _, err := p.Complete(context.Background(), "sys", "user", 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIProviderCompleteBadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-valid-json"))
	}))
	defer server.Close()

	original := agent.OpenAIBaseURL
	agent.OpenAIBaseURL = server.URL
	defer func() { agent.OpenAIBaseURL = original }()

	p := agent.NewOpenAIProvider("key", "")
	_, _, err := p.Complete(context.Background(), "sys", "user", 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIProviderCompleteInvalidURL(t *testing.T) {
	original := agent.OpenAIBaseURL
	agent.OpenAIBaseURL = "://invalid-url"
	defer func() { agent.OpenAIBaseURL = original }()

	p := agent.NewOpenAIProvider("key", "")
	_, _, err := p.Complete(context.Background(), "sys", "user", 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create request") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenAIProviderCompleteReadResponseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("response writer does not support hijacking")
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack failed: %v", err)
		}
		_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 20\r\n\r\n{}")
		_ = buf.Flush()
		_ = conn.Close()
	}))
	defer server.Close()

	original := agent.OpenAIBaseURL
	agent.OpenAIBaseURL = server.URL
	defer func() { agent.OpenAIBaseURL = original }()

	p := agent.NewOpenAIProvider("key", "")
	_, _, err := p.Complete(context.Background(), "sys", "user", 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "read response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCustomProviderName(t *testing.T) {
	p := agent.NewCustomProvider("/usr/bin/true")
	if p.Name() != "custom" {
		t.Fatalf("expected custom, got %s", p.Name())
	}
}

func TestCustomProviderCompleteRelativePath(t *testing.T) {
	p := agent.NewCustomProvider("./relative.sh")
	_, _, err := p.Complete(context.Background(), "sys", "user", 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "E6003") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCustomProviderCompleteMissingPath(t *testing.T) {
	p := agent.NewCustomProvider("/tmp/does-not-exist-teraflow-custom-provider")
	_, _, err := p.Complete(context.Background(), "sys", "user", 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "E6003") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCustomProviderCompleteExecutableScript(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "custom-provider.sh")
	script := "#!/bin/sh\necho 'hello from custom'\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	p := agent.NewCustomProvider(scriptPath)
	out, tokens, err := p.Complete(context.Background(), "sys", "user", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello from custom" {
		t.Fatalf("unexpected output: %q", out)
	}
	if tokens != 0 {
		t.Fatalf("unexpected tokens: %d", tokens)
	}
}

func TestCustomProviderCompleteCommandFails(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "custom-fail.sh")
	script := "#!/bin/sh\necho 'error output' >&2\nexit 1\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	p := agent.NewCustomProvider(scriptPath)
	_, _, err := p.Complete(context.Background(), "sys", "user", 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "E6002") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFallbackProviderNamePrimaryOnly(t *testing.T) {
	primary := &mockFallbackProvider{name: "primary"}
	p := agent.NewFallbackProvider(primary)
	if p.Name() != "primary" {
		t.Fatalf("expected primary, got %s", p.Name())
	}
}

func TestFallbackProviderNameWithSecondary(t *testing.T) {
	primary := &mockFallbackProvider{name: "primary"}
	secondary := &mockFallbackProvider{name: "secondary"}
	p := agent.NewFallbackProvider(primary).WithSecondary(secondary)
	if p.Name() != "primary+secondary(fallback)" {
		t.Fatalf("unexpected name: %s", p.Name())
	}
}

func TestFallbackProviderCompletePrimarySuccess(t *testing.T) {
	primary := &mockFallbackProvider{name: "primary", output: "ok", tokens: 7}
	secondary := &mockFallbackProvider{name: "secondary", output: "fallback", tokens: 3}
	p := agent.NewFallbackProvider(primary).WithSecondary(secondary)

	out, tokens, err := p.Complete(context.Background(), "sys", "user", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok" || tokens != 7 {
		t.Fatalf("unexpected result: out=%q tokens=%d", out, tokens)
	}
}

func TestFallbackProviderCompleteNoSecondary(t *testing.T) {
	primary := &mockFallbackProvider{name: "primary", err: errors.New("primary failed")}
	p := agent.NewFallbackProvider(primary)

	_, _, err := p.Complete(context.Background(), "sys", "user", 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no fallback configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFallbackProviderCompleteSecondarySuccess(t *testing.T) {
	primary := &mockFallbackProvider{name: "primary", err: errors.New("primary failed")}
	secondary := &mockFallbackProvider{name: "secondary", output: "fallback-ok", tokens: 12}
	p := agent.NewFallbackProvider(primary).WithSecondary(secondary)

	out, tokens, err := p.Complete(context.Background(), "sys", "user", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "fallback-ok" || tokens != 12 {
		t.Fatalf("unexpected result: out=%q tokens=%d", out, tokens)
	}
}
