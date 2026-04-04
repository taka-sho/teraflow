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

// OpenAIBaseURL は OpenAI API のベースURL（テスト時に差し替え可能）
var OpenAIBaseURL = "https://api.openai.com/v1/chat/completions"

// OpenAIProvider は OpenAI Chat Completions API を使う Provider 実装
type OpenAIProvider struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenAIProvider は OpenAIProvider を作成する
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIProvider{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

// Complete は OpenAI Chat Completions API を呼び出す
func (p *OpenAIProvider) Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, int, error) {
	reqBody := map[string]any{
		"model":      p.model,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		OpenAIBaseURL, bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("E6002: OpenAI API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != 200 {
		return "", 0, fmt.Errorf("E6002: OpenAI API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", 0, fmt.Errorf("parse response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", 0, fmt.Errorf("empty response from OpenAI API")
	}
	return result.Choices[0].Message.Content, result.Usage.TotalTokens, nil
}
