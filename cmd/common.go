package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

const notProjectError = "E0001: Not a teraflow project. Run `teraflow init` first."

func configPathFromCmd(cmd *cobra.Command) (string, error) {
	configPath, err := cmd.Root().PersistentFlags().GetString("config")
	if err != nil {
		return "", err
	}
	return configPath, nil
}

func outputFormatFromCmd(cmd *cobra.Command) (string, error) {
	format, err := cmd.Root().PersistentFlags().GetString("format")
	if err != nil {
		return "", err
	}
	switch format {
	case "text", "json":
		return format, nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func statePathFromConfig(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "project-state.yml")
}

func writeJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	return enc.Encode(v)
}
