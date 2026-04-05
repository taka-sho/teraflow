package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/taka-sho/teraflow/internal/rbac"
	"gopkg.in/yaml.v3"
)

var allowedLocalRoles = map[string]struct{}{
	"pm":          {},
	"dev":         {},
	"qa":          {},
	"release_mgr": {},
}

type localUserConfig struct {
	User struct {
		Role string `yaml:"role"`
		Name string `yaml:"name,omitempty"`
	} `yaml:"user"`
}

func setLocalRole(configPath, role string) (string, error) {
	if _, ok := allowedLocalRoles[role]; !ok {
		return "", fmt.Errorf("invalid role: %q (expected: pm|dev|qa|release_mgr)", role)
	}

	path := localUserConfigPath(configPath)
	cfg := localUserConfig{}
	cfg.User.Role = role

	user, err := rbac.CurrentUser()
	if err == nil && user != "" {
		cfg.User.Name = user
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create local config dir: %w", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return "", fmt.Errorf("marshal local user config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write local user config: %w", err)
	}

	return path, nil
}

func loadLocalRole(configPath string) (string, error) {
	path := localUserConfigPath(configPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("role is not set; run `teraflow config set role <pm|dev|qa|release_mgr>`")
		}
		return "", fmt.Errorf("read local user config: %w", err)
	}

	var cfg localUserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parse local user config: %w", err)
	}
	if cfg.User.Role == "" {
		return "", fmt.Errorf("role is not set; run `teraflow config set role <pm|dev|qa|release_mgr>`")
	}
	if _, ok := allowedLocalRoles[cfg.User.Role]; !ok {
		return "", fmt.Errorf("invalid role in local user config: %q", cfg.User.Role)
	}

	return cfg.User.Role, nil
}

func localUserConfigPath(configPath string) string {
	projectRoot := filepath.Dir(filepath.Dir(configPath))
	return filepath.Join(projectRoot, ".teraflow", "user-config.yml")
}
