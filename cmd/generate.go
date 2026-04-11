package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/agent"
	cfgpkg "github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/graphbridge"
	indexpkg "github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/validate"
	"github.com/taka-sho/teraflow/internal/wave"
)

type waveExecutor interface {
	ExecutePhase(ctx context.Context, phase string, waves []wave.WaveDefinition, opts wave.ExecuteOptions) ([]wave.WaveResult, error)
}

var generateLoadConfig = cfgpkg.Load
var generateResolveProviderForType = cfgpkg.ResolveProviderForType
var generateNewProviderFromConfig = agent.NewProviderFromConfig
var generateNewWaveEngine = func(provider agent.Provider, validator *validate.Validator, bridge *graphbridge.Bridge, idx *indexpkg.Index, projectRoot string) waveExecutor {
	return wave.NewEngine(provider, validator, bridge, idx, projectRoot)
}

func newGenerateCmd() *cobra.Command {
	var phase string
	var waveNumber int
	var createPR bool
	var dryRun bool
	var templateOverride string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate CoDD artifacts from wave definitions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(phase) == "" {
				return fmt.Errorf("--phase is required")
			}
			if waveNumber < 0 {
				return fmt.Errorf("--wave must be >= 0")
			}

			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			projectRoot, err := projectRootFromCmd(cmd)
			if err != nil {
				return err
			}

			cfg, err := generateLoadConfig(configPath)
			if err != nil {
				return err
			}

			defs := wave.DefaultWaveDefinitions(phase)
			if len(defs) == 0 {
				return fmt.Errorf("no wave definitions for phase %q", phase)
			}
			if waveNumber > 0 {
				filtered := make([]wave.WaveDefinition, 0, 1)
				for _, def := range defs {
					if def.Number == waveNumber {
						filtered = append(filtered, def)
						break
					}
				}
				if len(filtered) == 0 {
					return fmt.Errorf("wave %d not found in phase %s", waveNumber, phase)
				}
				defs = filtered
			}

			idx, err := indexpkg.NewBuilder(projectRoot).LoadIndex()
			if err != nil {
				return err
			}

			providerType, model := generateResolveProviderForType(cfg, providerTypeForPhase(phase))
			provider, providerErr := generateNewProviderFromConfig(agent.ProviderConfig{Provider: providerType, Model: model})
			if providerErr != nil && !dryRun {
				return fmt.Errorf("create provider: %w", providerErr)
			}

			bridge := graphbridge.New(projectRoot)
			validator := validate.NewValidator(idx, bridge, projectRoot)
			engine := generateNewWaveEngine(provider, validator, bridge, idx, projectRoot)

			results, err := engine.ExecutePhase(cmd.Context(), phase, defs, wave.ExecuteOptions{
				DryRun:           dryRun,
				TemplateOverride: templateOverride,
			})
			if err != nil {
				return err
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"phase":      phase,
					"wave_count": len(results),
					"dry_run":    dryRun,
					"create_pr":  createPR,
					"results":    results,
				})
			}

			if providerErr != nil && dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "Warning: provider init skipped for dry-run: %v\n", providerErr)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Phase: %s\n", phase)
			for _, r := range results {
				fmt.Fprintf(cmd.OutOrStdout(), "- Wave %d %s status=%s\n", r.WaveNumber, r.WaveName, r.Status)
				for _, a := range r.Artifacts {
					fmt.Fprintf(cmd.OutOrStdout(), "  artifact: %s (%s)\n", a.Path, a.ArtifactType)
				}
				for _, w := range r.Warnings {
					fmt.Fprintf(cmd.OutOrStdout(), "  warning: %s\n", w)
				}
			}
			if createPR {
				fmt.Fprintln(cmd.OutOrStdout(), "create-pr requested: use CI workflow teraflow-wave-generate.yml for automated PR handling")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&phase, "phase", "", "Target phase")
	cmd.Flags().IntVar(&waveNumber, "wave", 0, "Wave number (0 = all waves)")
	cmd.Flags().BoolVar(&createPR, "create-pr", false, "Create PR after generation (workflow-driven)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Generate preview without writing files")
	cmd.Flags().StringVar(&templateOverride, "template", "", "Override template file path")
	return cmd
}

func providerTypeForPhase(phase string) string {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "requirements":
		return "requirements"
	case "basic-design", "basic_design", "detailed-design", "detailed_design":
		return "design"
	default:
		return "implement"
	}
}
