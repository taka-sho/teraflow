package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/taka-sho/teraflow/internal/rbac"
	"gopkg.in/yaml.v3"
)

// TeraflowConfig represents .github/teraflow.yml.
type TeraflowConfig struct {
	Version      string          `yaml:"version"`
	Project      ProjectCfg      `yaml:"project"`
	Confirmation ConfirmationCfg `yaml:"confirmation,omitempty"`
	AI           AICfg           `yaml:"ai"`
	Agent        AgentCfg        `yaml:"agent"`
	Harness      HarnessCfg      `yaml:"harness"`
	RBAC         rbac.RBACConfig `yaml:"rbac,omitempty"`
}

type ProjectCfg struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Repository  string `yaml:"repository"`
}

type ConfirmationCfg struct {
	Trigger    string `yaml:"trigger"`
	ReqTrigger string `yaml:"req_trigger"`
}

type AICfg struct {
	DefaultProvider string `yaml:"default_provider"`
}

// AgentCfg is the teraflow.yml agent section.
type AgentCfg struct {
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

type HarnessCfg struct {
	ScoreThreshold int  `yaml:"score_threshold"`
	AutoIssue      bool `yaml:"auto_issue,omitempty"`
}

// Load loads teraflow config from path.
func Load(path string) (*TeraflowConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	var cfg TeraflowConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// Save writes teraflow config to path.
func Save(path string, cfg *TeraflowConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// GetValue gets a config value by key.
func GetValue(cfg *TeraflowConfig, key string) (string, error) {
	switch key {
	case "project.name":
		return cfg.Project.Name, nil
	case "project.description":
		return cfg.Project.Description, nil
	case "project.repository":
		return cfg.Project.Repository, nil
	case "ai.default_provider":
		return cfg.AI.DefaultProvider, nil
	case "harness.score_threshold":
		return strconv.Itoa(cfg.Harness.ScoreThreshold), nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// SetValue sets a config value by key.
func SetValue(cfg *TeraflowConfig, key, value string) error {
	switch key {
	case "project.name":
		cfg.Project.Name = value
	case "project.description":
		cfg.Project.Description = value
	case "project.repository":
		cfg.Project.Repository = value
	case "ai.default_provider":
		cfg.AI.DefaultProvider = value
	case "harness.score_threshold":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for %s: %w", key, err)
		}
		cfg.Harness.ScoreThreshold = n
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return nil
}
