package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type scanCheck struct {
	File     string `json:"file"`
	Exists   bool   `json:"exists"`
	Optional bool   `json:"optional,omitempty"`
}

type scanJSONOutput struct {
	Status string      `json:"status"`
	Checks []scanCheck `json:"checks"`
}

func newScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan required project files",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			return runScan(cmd, configPath, format)
		},
	}

	return cmd
}

func runScan(cmd *cobra.Command, configPath, format string) error {
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return err
	}

	repoRoot := filepath.Dir(filepath.Dir(absConfigPath))
	checks := []scanCheck{
		{File: ".github/teraflow.yml", Exists: fileExists(filepath.Join(repoRoot, ".github", "teraflow.yml"))},
		{File: ".github/project-state.yml", Exists: fileExists(filepath.Join(repoRoot, ".github", "project-state.yml"))},
		{File: "docs/", Exists: dirExists(filepath.Join(repoRoot, "docs"))},
		{File: ".teraflow/rework-log.yml", Exists: fileExists(filepath.Join(repoRoot, ".teraflow", "rework-log.yml")), Optional: true},
	}

	if !checks[1].Exists {
		return errors.New("E0001: Not a teraflow project. Run `teraflow init` first.")
	}

	if format == "json" {
		out := scanJSONOutput{Status: "ok", Checks: checks}
		return writeJSON(cmd, out)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Scanning project...")
	for _, c := range checks {
		if c.Exists {
			fmt.Fprintf(cmd.OutOrStdout(), "  ✓ %s\n", c.File)
			continue
		}
		if c.Optional {
			fmt.Fprintf(cmd.OutOrStdout(), "  ○ %s (not found - no reworks yet)\n", c.File)
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  ✗ %s\n", c.File)
	}
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Scan complete: %d/%d required files present.\n", countRequiredChecks(checks), requiredChecksCount(checks))

	return nil
}

func countRequiredChecks(checks []scanCheck) int {
	count := 0
	for _, c := range checks {
		if c.Optional {
			continue
		}
		if c.Exists {
			count++
		}
	}
	return count
}

func requiredChecksCount(checks []scanCheck) int {
	count := 0
	for _, c := range checks {
		if !c.Optional {
			count++
		}
	}
	return count
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
