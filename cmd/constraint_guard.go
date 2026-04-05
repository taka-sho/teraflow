package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/constraint"
	"github.com/taka-sho/teraflow/internal/rbac"
)

func runConstraintGuard(cmd *cobra.Command, configPath, operation string, force bool, reason string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	if len(cfg.Constraints) == 0 {
		return nil
	}

	if force {
		if strings.TrimSpace(reason) == "" {
			return errors.New("--force requires --reason")
		}
		user, err := rbac.CurrentUser()
		if err != nil {
			return err
		}
		engine := rbac.NewEngine(cfg.RBAC)
		if !engine.CheckPermission(user, "constraint.override") {
			return fmt.Errorf("--force requires permission: constraint.override (user: %s)", user)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "WARNING: constraints skipped for %s (--force): %s\n", operation, reason)
		return nil
	}

	projectPath := projectPathFromConfig(configPath)
	cEngine := &constraint.Engine{}
	blocked, warnings, err := cEngine.Check(cfg.Constraints, projectPath)
	if err != nil {
		return err
	}

	for _, result := range warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "WARNING: %s\n", result.Message)
	}
	if len(blocked) > 0 {
		for _, result := range blocked {
			fmt.Fprintf(cmd.ErrOrStderr(), "BLOCKED: %s\n", result.Message)
		}
		return errors.New("operation blocked by constraints")
	}
	return nil
}

func projectPathFromConfig(configPath string) string {
	dir := filepath.Dir(configPath)
	if filepath.Base(dir) == ".github" {
		return filepath.Dir(dir)
	}
	return dir
}
