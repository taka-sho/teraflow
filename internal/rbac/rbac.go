package rbac

import (
	"os"
	"os/exec"
	"strings"

	tferrors "github.com/taka-sho/teraflow/internal/errors"
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
	Enabled           bool   `yaml:"enabled"`
	AdminRole         string `yaml:"admin_role,omitempty"`
	GitHubEnforcement bool   `yaml:"github_enforcement,omitempty"`
	Roles             []Role `yaml:"roles"`
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

// IsAdmin returns true if the user belongs to the configured admin role.
// If RBAC is disabled, everyone is treated as admin for backward compatibility.
func (e *Engine) IsAdmin(user string) bool {
	if e == nil || !e.config.Enabled {
		return true
	}

	adminRole := e.config.AdminRole
	if adminRole == "" {
		adminRole = "admin"
	}

	for _, role := range e.config.Roles {
		if role.Name == adminRole && containsUser(role.Members, user) {
			return true
		}
	}

	return false
}

// ResolveUser resolves an RBAC user.
// If flagUser is set, it is used directly.
// If github_enforcement is enabled and flagUser is empty, an error is returned.
func (e *Engine) ResolveUser(flagUser string) (string, error) {
	if flagUser != "" {
		return flagUser, nil
	}
	if e != nil && e.config.GitHubEnforcement {
		return "", &tferrors.AppError{
			Code:     tferrors.CodeTFCL01,
			Category: tferrors.CatCLI,
			Message:  "rbac.github_enforcement is enabled: --user flag is required",
			ExitCode: 2,
		}
	}
	return CurrentUser()
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

	return "", &tferrors.AppError{
		Code:     tferrors.CodeTFRB02,
		Category: tferrors.CatRBAC,
		Message:  "unable to determine current user from git config user.name",
		ExitCode: 3,
	}
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
