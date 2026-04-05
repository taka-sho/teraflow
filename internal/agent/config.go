package agent

import (
	"fmt"
	"os"

	tferrors "github.com/taka-sho/teraflow/internal/errors"
)

// ProviderConfig はteraflow.ymlのagentセクションを表す
type ProviderConfig struct {
	Provider      string `yaml:"provider"`
	Model         string `yaml:"model"`
	MaxTokens     int    `yaml:"max_tokens"`
	Timeout       int    `yaml:"timeout"`
	CustomCommand string `yaml:"custom_command"`
	Fallback      string `yaml:"fallback"`
	TrustLevel    string `yaml:"trust_level"`
	RateLimit     struct {
		MaxCallsPerHour int `yaml:"max_calls_per_hour"`
	} `yaml:"rate_limit"`
}

// NewProviderFromConfig は設定に基づいてProviderを生成する
func NewProviderFromConfig(cfg ProviderConfig) (Provider, error) {
	switch cfg.Provider {
	case "anthropic", "":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, &tferrors.AppError{
				Code:     tferrors.CodeTFAI01,
				Category: tferrors.CatAI,
				Message:  "E6001: ANTHROPIC_API_KEY not set",
				ExitCode: 4,
			}
		}
		return NewAnthropicProvider(apiKey, cfg.Model), nil

	case "claude-code":
		return NewClaudeCodeProvider(cfg.Model), nil

	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, &tferrors.AppError{
				Code:     tferrors.CodeTFAI01,
				Category: tferrors.CatAI,
				Message:  "E6001: OPENAI_API_KEY not set",
				ExitCode: 4,
			}
		}
		return NewOpenAIProvider(apiKey, cfg.Model), nil

	case "custom":
		if cfg.CustomCommand == "" {
			return nil, &tferrors.AppError{
				Code:     tferrors.CodeTFCL02,
				Category: tferrors.CatCLI,
				Message:  "E6003: custom_command is required for provider=custom",
				ExitCode: 2,
			}
		}
		return NewCustomProvider(cfg.CustomCommand), nil

	case "fallback":
		inner, err := NewProviderFromConfig(ProviderConfig{Provider: cfg.Fallback, Model: cfg.Model})
		if err != nil {
			return nil, err
		}
		return NewFallbackProvider(inner), nil

	default:
		return nil, &tferrors.AppError{
			Code:     tferrors.CodeTFCL02,
			Category: tferrors.CatCLI,
			Message:  fmt.Sprintf("E6003: unknown agent provider: %s", cfg.Provider),
			ExitCode: 2,
		}
	}
}
