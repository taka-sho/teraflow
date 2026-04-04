package agent

import (
	"context"
	"fmt"
)

// FallbackProvider は primary Provider 失敗時に secondary にフォールバックする Provider
type FallbackProvider struct {
	primary   Provider
	secondary Provider
}

// NewFallbackProvider は FallbackProvider を作成する
func NewFallbackProvider(primary Provider) *FallbackProvider {
	return &FallbackProvider{primary: primary}
}

// WithSecondary は secondary Provider を設定する
func (p *FallbackProvider) WithSecondary(secondary Provider) *FallbackProvider {
	p.secondary = secondary
	return p
}

func (p *FallbackProvider) Name() string {
	if p.secondary != nil {
		return fmt.Sprintf("%s+%s(fallback)", p.primary.Name(), p.secondary.Name())
	}
	return p.primary.Name()
}

// Complete は primary を試み、失敗時に secondary にフォールバックする
func (p *FallbackProvider) Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, int, error) {
	output, tokens, err := p.primary.Complete(ctx, systemPrompt, userPrompt, maxTokens)
	if err == nil {
		return output, tokens, nil
	}
	if p.secondary == nil {
		return "", 0, fmt.Errorf("primary provider failed and no fallback configured: %w", err)
	}
	return p.secondary.Complete(ctx, systemPrompt, userPrompt, maxTokens)
}
