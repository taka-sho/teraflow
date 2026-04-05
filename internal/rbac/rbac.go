package rbac

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// Permission represents an RBAC permission string (e.g. "gate.approve.planning").
type Permission string

// Role represents a project role.
type Role struct {
	Name        string       `yaml:"name"`
	Members     []string     `yaml:"members"`
	Permissions []Permission `yaml:"permissions"`
}

// RBACConfig is the top-level RBAC configuration.
type RBACConfig struct {
	Enabled bool   `yaml:"enabled"`
	Roles   []Role `yaml:"roles"`
}

// Engine checks RBAC permissions.
type Engine struct {
	config RBACConfig
}

// NewEngine creates a new RBAC engine.
func NewEngine(config RBACConfig) *Engine {
	return &Engine{config: config}
}

// CheckPermission returns true if user has the given permission.
// If RBAC is disabled, all operations are allowed for backward compatibility.
func (e *Engine) CheckPermission(user, permission string) bool {
	if e == nil || !e.config.Enabled {
		return true
	}

	for _, role := range e.config.Roles {
		if !containsUser(role.Members, user) {
			continue
		}
		for _, granted := range role.Permissions {
			if permissionMatch(string(granted), permission) {
				return true
			}
		}
	}

	return false
}

// GetUserRoles returns all roles for a given user.
func (e *Engine) GetUserRoles(user string) []string {
	if e == nil {
		return nil
	}

	roles := []string{}
	for _, role := range e.config.Roles {
		if containsUser(role.Members, user) {
			roles = append(roles, role.Name)
		}
	}
	return roles
}

// CurrentUser returns the current git user.name.
func CurrentUser() (string, error) {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err == nil {
		user := strings.TrimSpace(string(out))
		if user != "" {
			return user, nil
		}
	}

	if user := strings.TrimSpace(os.Getenv("USER")); user != "" {
		return user, nil
	}

	return "", errors.New("unable to determine current user from git config user.name")
}

func containsUser(members []string, user string) bool {
	for _, member := range members {
		if strings.TrimSpace(member) == user {
			return true
		}
	}
	return false
}

func permissionMatch(granted, required string) bool {
	if granted == "*" || granted == required {
		return true
	}

	gParts := strings.Split(granted, ".")
	rParts := strings.Split(required, ".")
	if len(gParts) != len(rParts) {
		return false
	}

	for i := range gParts {
		if gParts[i] == "*" {
			continue
		}
		if gParts[i] != rParts[i] {
			return false
		}
	}

	return true
}
