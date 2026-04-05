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
	cmd.AddCommand(newConfigGetProviderCmd())
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
			if key == "role" {
				path, setErr := setLocalRole(configPath, value)
				if setErr != nil {
					return setErr
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Set role = %s (%s)\n", value, path)
				return nil
			}

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

func newConfigGetProviderCmd() *cobra.Command {
	var agentType string

	cmd := &cobra.Command{
		Use:   "get-provider",
		Short: "Get provider configuration for an agent type",
		RunE: func(cmd *cobra.Command, args []string) error {
			if agentType == "" {
				return fmt.Errorf("--type is required")
			}

			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}

			var cfg *cfgpkg.TeraflowConfig
			if configPath != "" {
				loadedCfg, loadErr := cfgpkg.Load(configPath)
				if loadErr == nil {
					cfg = loadedCfg
				}
			}

			provider, model := cfgpkg.ResolveProviderForType(cfg, agentType)
			apiKeyEnv := cfgpkg.GetAPIKeyEnvName(provider)

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, map[string]string{
					"provider":    provider,
					"model":       model,
					"api_key_env": apiKeyEnv,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "PROVIDER=%s\n", provider)
			fmt.Fprintf(cmd.OutOrStdout(), "MODEL=%s\n", model)
			fmt.Fprintf(cmd.OutOrStdout(), "API_KEY_ENV=%s\n", apiKeyEnv)
			return nil
		},
	}

	cmd.Flags().StringVar(&agentType, "type", "", "Agent type (requirements/review/implement/...)")
	return cmd
}
