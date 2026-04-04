package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CustomProvider は外部コマンドを使う Provider 実装
type CustomProvider struct {
	command string
}

// NewCustomProvider は CustomProvider を作成する
func NewCustomProvider(command string) *CustomProvider {
	return &CustomProvider{command: command}
}

func (p *CustomProvider) Name() string { return "custom" }

// Complete は外部コマンドを実行してレスポンスを取得する
// セキュリティ: コマンドパスは絶対パスのみ許可
func (p *CustomProvider) Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, int, error) {
	// セキュリティ検証: 絶対パスのみ許可
	if !filepath.IsAbs(p.command) {
		return "", 0, fmt.Errorf("E6003: custom_command must be an absolute path: %s", p.command)
	}
	if _, err := os.Stat(p.command); err != nil {
		return "", 0, fmt.Errorf("E6003: custom_command not found: %s", p.command)
	}

	// プロンプトを JSON 形式で stdin に渡す
	input := map[string]any{
		"system_prompt": systemPrompt,
		"user_prompt":   userPrompt,
		"max_tokens":    maxTokens,
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", 0, fmt.Errorf("marshal input: %w", err)
	}

	// シェル展開を防ぐため exec.CommandContext でコマンドを直接実行
	cmd := exec.CommandContext(ctx, p.command)
	cmd.Stdin = bytes.NewReader(inputJSON)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", 0, fmt.Errorf("E6002: custom command failed: %s: %w", stderr.String(), err)
	}

	// stdout の内容をそのままレスポンスとして返す
	return strings.TrimSpace(stdout.String()), 0, nil
}
