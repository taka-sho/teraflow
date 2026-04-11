package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/agent"
	cfgpkg "github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/graphbridge"
	indexpkg "github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/pipeline"
	"github.com/taka-sho/teraflow/internal/validate"
)

var implementLoadConfig = cfgpkg.Load
var implementResolveProviderForType = cfgpkg.ResolveProviderForType
var implementNewProviderFromConfig = agent.NewProviderFromConfig
var implementNewEngine = func(provider agent.Provider, validator *validate.Validator, projectRoot string, idx *indexpkg.Index) *pipeline.ImplementEngine {
	return pipeline.NewImplementEngine(provider, nil, validator, projectRoot, idx)
}

func newImplementCmd() *cobra.Command {
	var designPath string
	var modulePaths []string
	var maxParallel int
	var createPR bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "implement",
		Short: "Generate implementation modules from detailed design docs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(designPath) == "" {
				return fmt.Errorf("--design is required")
			}
			if maxParallel < 1 {
				return fmt.Errorf("--parallel must be >= 1")
			}

			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			projectRoot, err := projectRootFromCmd(cmd)
			if err != nil {
				return err
			}

			cfg, err := implementLoadConfig(configPath)
			if err != nil {
				return err
			}

			idx, err := indexpkg.NewBuilder(projectRoot).LoadIndex()
			if err != nil {
				return err
			}

			providerType, model := implementResolveProviderForType(cfg, "implement")
			provider, providerErr := implementNewProviderFromConfig(agent.ProviderConfig{Provider: providerType, Model: model})
			if providerErr != nil && !dryRun {
				return fmt.Errorf("create provider: %w", providerErr)
			}

			validator := validate.NewValidator(idx, graphbridge.New(projectRoot), projectRoot)
			validator.SetLevel(2)
			engine := implementNewEngine(provider, validator, projectRoot, idx)

			req := pipeline.ImplementRequest{
				DesignDocPath: designPath,
				MaxParallel:   maxParallel,
				CreatePR:      createPR,
				DryRun:        dryRun,
			}
			if len(modulePaths) > 0 {
				req.Modules = make([]pipeline.ModuleSpec, 0, len(modulePaths))
				for _, path := range modulePaths {
					req.Modules = append(req.Modules, pipeline.ModuleSpec{Path: path})
				}
			}

			report, err := engine.Execute(cmd.Context(), req)
			if err != nil {
				return err
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				if err := writeJSON(cmd, report); err != nil {
					return err
				}
			} else {
				if providerErr != nil && dryRun {
					fmt.Fprintf(cmd.OutOrStdout(), "Warning: provider init skipped for dry-run: %v\n", providerErr)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Design: %s\n", report.DesignDocPath)
				fmt.Fprintf(cmd.OutOrStdout(), "Modules: total=%d completed=%d failed=%d skipped=%d\n",
					report.Summary["total"], report.Summary["completed"], report.Summary["failed"], report.Summary["skipped"])
				for _, r := range report.Modules {
					fmt.Fprintf(cmd.OutOrStdout(), "- %s status=%s review=%s\n", r.Module.Path, r.Status, r.ReviewRequired)
					if r.Error != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "  error: %s\n", r.Error)
					}
					for _, t := range r.TestFiles {
						fmt.Fprintf(cmd.OutOrStdout(), "  test: %s\n", t)
					}
				}
				if report.Validation != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Validation L1-2: valid=%t errors=%d warnings=%d\n",
						report.Validation.Valid, report.Validation.ErrorCount, report.Validation.WarningCount)
				}
			}

			if report.Summary["failed"] > 0 {
				return fmt.Errorf("implement completed with %d failed module(s)", report.Summary["failed"])
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&designPath, "design", "", "Detailed design doc path")
	cmd.Flags().StringArrayVar(&modulePaths, "module", nil, "Target module path (repeatable)")
	cmd.Flags().IntVar(&maxParallel, "parallel", 3, "Maximum parallel modules")
	cmd.Flags().BoolVar(&createPR, "create-pr", false, "Create integration PR metadata")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Plan generation without writing files")
	return cmd
}
