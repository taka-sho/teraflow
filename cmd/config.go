package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	cfgpkg "github.com/taka-sho/teraflow/internal/config"
)

var configShowKeys = []string{
	"project.name",
	"project.description",
	"project.repository",
	"ai.default_provider",
	"harness.score_threshold",
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and update teraflow configuration",
	}

	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigSetCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show config values",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			cfg, err := cfgpkg.Load(configPath)
			if err != nil {
				return err
			}

			if format == "json" {
				result := map[string]string{}
				for _, key := range configShowKeys {
					val, getErr := cfgpkg.GetValue(cfg, key)
					if getErr != nil {
						return getErr
					}
					result[key] = val
				}

				return writeJSON(cmd, result)
			}

			for _, key := range configShowKeys {
				val, getErr := cfgpkg.GetValue(cfg, key)
				if getErr != nil {
					return getErr
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%-27s %s\n", key+":", val)
			}
			return nil
		},
	}

	return cmd
}

func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			key := args[0]
			value := args[1]

			cfg, err := cfgpkg.Load(configPath)
			if err != nil {
				return err
			}

			if err := cfgpkg.SetValue(cfg, key, value); err != nil {
				return err
			}

			if err := cfgpkg.Save(configPath, cfg); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", key, value)
			return nil
		},
	}

	return cmd
}
