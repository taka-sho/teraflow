package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BaseURL is the Anthropic API endpoint. Override in tests.
var BaseURL = "https://api.anthropic.com/v1/messages"

// AnthropicProvider は Anthropic Claude API を使う Provider 実装
type AnthropicProvider struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewAnthropicProvider は AnthropicProvider を作成する
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}
	return &AnthropicProvider{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// NewAnthropicProviderWithClient creates an AnthropicProvider with a custom HTTP client (for testing).
func NewAnthropicProviderWithClient(apiKey, model string, httpClient *http.Client) *AnthropicProvider {
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}
	return &AnthropicProvider{
		apiKey:     apiKey,
		model:      model,
		httpClient: httpClient,
	}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

// Complete は Anthropic Messages API を呼び出す
func (p *AnthropicProvider) Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, int, error) {
	reqBody := map[string]any{
		"model":      p.model,
		"max_tokens": maxTokens,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", BaseURL, bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("E6002: API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", 0, fmt.Errorf("E6002: API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", 0, fmt.Errorf("parse response: %w", err)
	}
	if len(result.Content) == 0 {
		return "", 0, fmt.Errorf("empty response from API")
	}

	totalTokens := result.Usage.InputTokens + result.Usage.OutputTokens
	return result.Content[0].Text, totalTokens, nil
}
