package agent

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// ClaudeCodeProvider は Claude Code CLI を使う Provider 実装
type ClaudeCodeProvider struct {
	model string
}

// NewClaudeCodeProvider は ClaudeCodeProvider を作成する
func NewClaudeCodeProvider(model string) *ClaudeCodeProvider {
	if model == "" {
		model = "haiku"
	}
	return &ClaudeCodeProvider{model: model}
}

func (p *ClaudeCodeProvider) Name() string { return "claude-code" }

// Complete は Claude Code CLI を非対話モードで実行する
func (p *ClaudeCodeProvider) Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, int, error) {
	prompt := fmt.Sprintf("System: %s\n\nUser: %s", systemPrompt, userPrompt)

	args := []string{"-p", prompt, "--model", p.model}

	cmd := exec.CommandContext(ctx, "claude", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", 0, fmt.Errorf("E6002: claude CLI failed: %s: %w", stderr.String(), err)
	}

	// Claude Code CLI はトークン使用量を返さないため 0 を返す
	return stdout.String(), 0, nil
}

// HealthCheck は claude CLI の存在と動作を確認する
func (p *ClaudeCodeProvider) HealthCheck() error {
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude CLI not found in PATH")
	}
	cmd := exec.Command("claude", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude CLI not working: %w", err)
	}
	return nil
}
