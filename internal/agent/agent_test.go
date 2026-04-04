package agent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/taka-sho/teraflow/internal/agent"
)

// mockProvider はテスト用の Provider モック
type mockProvider struct {
	output string
	tokens int
	err    error
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Complete(_ context.Context, _, _ string, _ int) (string, int, error) {
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
