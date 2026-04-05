package agent_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/taka-sho/teraflow/internal/agent"
)

// mockProvider はテスト用の Provider モック
type mockProvider struct {
	output        string
	tokens        int
	err           error
	gotUserPrompt string
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Complete(_ context.Context, _, userPrompt string, _ int) (string, int, error) {
	m.gotUserPrompt = userPrompt
	return m.output, m.tokens, m.err
}

func TestAgentManagerRun(t *testing.T) {
	mock := &mockProvider{output: "テスト要約", tokens: 100}
	mgr := agent.NewAgentManager(mock)

	ctx := agent.AgentContext{
		Type:  agent.AgentTypeRequirements,
		Input: "ユーザーがログイン機能を求めている",
	}

	result, err := mgr.Run(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got failure: %s", result.Error)
	}
	if result.Output != "テスト要約" {
		t.Fatalf("unexpected output: %s", result.Output)
	}
	if result.TokensUsed != 100 {
		t.Fatalf("unexpected tokens: %d", result.TokensUsed)
	}
}

func TestAgentManagerRunProviderError(t *testing.T) {
	mock := &mockProvider{err: errors.New("connection refused")}
	mgr := agent.NewAgentManager(mock)

	ctx := agent.AgentContext{
		Type:  agent.AgentTypeReview,
		Input: "some diff",
	}

	result, err := mgr.Run(context.Background(), ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if result.Success {
		t.Fatal("expected failure")
	}
}

func TestAgentManagerUnknownType(t *testing.T) {
	mock := &mockProvider{output: "ok"}
	mgr := agent.NewAgentManager(mock)

	ctx := agent.AgentContext{
		Type:  agent.AgentType("unknown"),
		Input: "test",
	}

	_, err := mgr.Run(context.Background(), ctx)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestGetSystemPrompt(t *testing.T) {
	types := []agent.AgentType{
		agent.AgentTypeRequirements,
		agent.AgentTypeReview,
		agent.AgentTypeImplement,
		agent.AgentTypeCIFix,
		agent.AgentTypeConflict,
		agent.AgentTypeIncident,
		agent.AgentTypeMaintenance,
	}
	for _, typ := range types {
		prompt := agent.GetSystemPrompt(typ)
		if prompt == "" {
			t.Errorf("empty prompt for agent type: %s", typ)
		}
	}
}

func TestGetSystemPromptRequirementsDialogueMode(t *testing.T) {
	prompt := agent.GetSystemPrompt(agent.AgentTypeRequirements)
	requiredPhrases := []string{
		"ユーザーと壁打ちしながら要件を深めてください。",
		"ユーザーが「要求確定」と発言した場合:",
		"通常の対話応答は「💬 AI壁打ち」ヘッダーで始めること。",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("requirements prompt must contain %q", phrase)
		}
	}
}

func TestAnthropicProviderName(t *testing.T) {
	p := agent.NewAnthropicProvider("test-key", "")
	if p.Name() != "anthropic" {
		t.Fatalf("expected 'anthropic', got %s", p.Name())
	}
}

func TestAnthropicProviderCompleteSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Errorf("expected anthropic-version header")
		}
		resp := map[string]any{
			"content": []map[string]string{{"type": "text", "text": "AI応答テスト"}},
			"usage":   map[string]int{"input_tokens": 10, "output_tokens": 5},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	agent.BaseURL = server.URL
	defer func() { agent.BaseURL = "https://api.anthropic.com/v1/messages" }()

	p := agent.NewAnthropicProvider("test-key", "")
	out, tokens, err := p.Complete(context.Background(), "system prompt", "user input", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "AI応答テスト" {
		t.Fatalf("unexpected output: %q", out)
	}
	if tokens != 15 {
		t.Fatalf("unexpected tokens: %d", tokens)
	}
}

func TestAnthropicProviderCompleteHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid_api_key"}}`))
	}))
	defer server.Close()

	agent.BaseURL = server.URL
	defer func() { agent.BaseURL = "https://api.anthropic.com/v1/messages" }()

	p := agent.NewAnthropicProvider("bad-key", "")
	_, _, err := p.Complete(context.Background(), "s", "u", 100)
	if err == nil {
		t.Fatal("expected error for HTTP 401")
	}
}

func TestAnthropicProviderCompleteRequestFailure(t *testing.T) {
	original := agent.BaseURL
	agent.BaseURL = "http://127.0.0.1:1"
	defer func() { agent.BaseURL = original }()

	p := agent.NewAnthropicProvider("key", "")
	_, _, err := p.Complete(context.Background(), "s", "u", 100)
	if err == nil {
		t.Fatal("expected request failure")
	}
}

func TestAnthropicProviderCompleteEmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]string{},
			"usage":   map[string]int{"input_tokens": 5, "output_tokens": 0},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	agent.BaseURL = server.URL
	defer func() { agent.BaseURL = "https://api.anthropic.com/v1/messages" }()

	p := agent.NewAnthropicProvider("key", "")
	_, _, err := p.Complete(context.Background(), "s", "u", 100)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestAnthropicProviderCompleteInvalidURL(t *testing.T) {
	original := agent.BaseURL
	agent.BaseURL = "://invalid-url"
	defer func() { agent.BaseURL = original }()

	p := agent.NewAnthropicProvider("key", "")
	_, _, err := p.Complete(context.Background(), "s", "u", 100)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if err != nil && err.Error() == "" {
		t.Fatal("expected non-empty error")
	}
}

func TestAnthropicProviderCompleteBadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-valid-json"))
	}))
	defer server.Close()

	original := agent.BaseURL
	agent.BaseURL = server.URL
	defer func() { agent.BaseURL = original }()

	p := agent.NewAnthropicProvider("key", "")
	_, _, err := p.Complete(context.Background(), "s", "u", 100)
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestAnthropicProviderCompleteReadResponseError(t *testing.T) {
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

	original := agent.BaseURL
	agent.BaseURL = server.URL
	defer func() { agent.BaseURL = original }()

	p := agent.NewAnthropicProvider("key", "")
	_, _, err := p.Complete(context.Background(), "s", "u", 100)
	if err == nil {
		t.Fatal("expected read response error")
	}
}

func TestAgentManagerAllTypes(t *testing.T) {
	types := []agent.AgentType{
		agent.AgentTypeRequirements,
		agent.AgentTypeReview,
		agent.AgentTypeImplement,
		agent.AgentTypeCIFix,
		agent.AgentTypeConflict,
		agent.AgentTypeIncident,
		agent.AgentTypeMaintenance,
	}
	for _, typ := range types {
		mock := &mockProvider{output: "テスト出力", tokens: 50}
		mgr := agent.NewAgentManager(mock)
		ctx := agent.AgentContext{Type: typ, Input: "テスト入力"}
		result, err := mgr.Run(context.Background(), ctx)
		if err != nil {
			t.Errorf("type %s: unexpected error: %v", typ, err)
		}
		if !result.Success {
			t.Errorf("type %s: expected success, got error: %s", typ, result.Error)
		}
		if result.Type != typ {
			t.Errorf("type %s: result.Type mismatch: %s", typ, result.Type)
		}
	}
}

func TestAgentManagerMaxTokensDefault(t *testing.T) {
	mock := &mockProvider{output: "ok", tokens: 10}
	mgr := agent.NewAgentManager(mock)
	ctx := agent.AgentContext{
		Type:      agent.AgentTypeReview,
		Input:     "test input",
		MaxTokens: 0,
	}
	result, err := mgr.Run(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
}

func TestAgentManagerRunWithAdditionalContext(t *testing.T) {
	mock := &mockProvider{output: "ok", tokens: 10}
	mgr := agent.NewAgentManager(mock)

	ctx := agent.AgentContext{
		Type:              agent.AgentTypeReview,
		Input:             "please review this diff",
		AdditionalContext: "doc summary here",
	}

	_, err := mgr.Run(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(mock.gotUserPrompt, "## コンテキスト（関連文書）") {
		t.Fatalf("expected context section in user prompt: %q", mock.gotUserPrompt)
	}
	if !strings.Contains(mock.gotUserPrompt, "doc summary here") {
		t.Fatalf("expected additional context in user prompt: %q", mock.gotUserPrompt)
	}
	if !strings.Contains(mock.gotUserPrompt, "## ユーザー入力") {
		t.Fatalf("expected user input section in prompt: %q", mock.gotUserPrompt)
	}
	if !strings.Contains(mock.gotUserPrompt, "please review this diff") {
		t.Fatalf("expected original input in prompt: %q", mock.gotUserPrompt)
	}
}
