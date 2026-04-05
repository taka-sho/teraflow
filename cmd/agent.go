package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/agent"
	cfgpkg "github.com/taka-sho/teraflow/internal/config"
	ctxpkg "github.com/taka-sho/teraflow/internal/context"
	"github.com/taka-sho/teraflow/internal/index"
	"github.com/taka-sho/teraflow/internal/rbac"
	"github.com/taka-sho/teraflow/internal/skill"
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage AI agents for automated tasks",
	}
	cmd.AddCommand(newAgentAssignCmd())
	cmd.AddCommand(newAgentStatusCmd())
	return cmd
}

func newAgentAssignCmd() *cobra.Command {
	var agentType string
	var trustLevel string
	var inputFile string
	var apiKey string
	var skillName string
	var userFlag string

	cmd := &cobra.Command{
		Use:   "assign",
		Short: "Assign an AI agent to a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if agentType == "" {
				return fmt.Errorf("--type is required")
			}

			var input string
			if inputFile != "" {
				data, err := os.ReadFile(inputFile)
				if err != nil {
					return fmt.Errorf("read input file: %w", err)
				}
				input = string(data)
			} else if len(args) > 0 {
				input = args[0]
			} else {
				return fmt.Errorf("input required: provide as argument or --input-file")
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
			engine := rbac.NewEngine(rbac.RBACConfig{})
			if cfg != nil {
				engine = rbac.NewEngine(cfg.RBAC)
			}
			if _, err := engine.ResolveUser(userFlag); err != nil {
				return fmt.Errorf("resolve user: %w", err)
			}

			resolvedProvider, resolvedModel := cfgpkg.ResolveProviderForType(cfg, agentType)
			if apiKey != "" {
				if keyEnv := cfgpkg.GetAPIKeyEnvName(resolvedProvider); keyEnv != "" {
					_ = os.Setenv(keyEnv, apiKey)
				}
			}

			skillsDir := "skills"
			if configPath != "" {
				skillsDir = filepath.Clean(filepath.Join(filepath.Dir(configPath), "..", "skills"))
			}

			loader := skill.NewFileLoader(skillsDir)
			selector := &skill.DefaultSelector{}
			var selectedSkill *skill.Skill

			if skillName != "" {
				if _, statErr := os.Stat(skillsDir); statErr != nil {
					if os.IsNotExist(statErr) {
						fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not load skills: %v\n", statErr)
					} else {
						return fmt.Errorf("check skills directory: %w", statErr)
					}
				} else {
					selectedSkill, err = loader.LoadByName(skillName)
					if err != nil {
						return fmt.Errorf("load skill %q: %w", skillName, err)
					}
				}
			} else if agentType != "" {
				skills, err := loader.LoadAll()
				if err != nil {
					if _, statErr := os.Stat(skillsDir); os.IsNotExist(statErr) {
						fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not load skills: %v\n", err)
						skills = nil
					} else {
						return fmt.Errorf("load skills: %w", err)
					}
				}
				selectedSkill = selector.Select(skills, skill.SelectContext{AgentType: agentType})
			}

			systemPrompt := agent.GetSystemPrompt(agent.AgentType(agentType))
			if selectedSkill != nil {
				systemPrompt = selectedSkill.Prompts.System
			}

			var additionalCtx string
			if selectedSkill != nil && selectedSkill.Context.MaxContextTokens > 0 {
				projectRoot := "."
				if configPath != "" {
					projectRoot = projectRootFromConfig(configPath)
				}
				if idx, loadErr := index.NewBuilder(projectRoot).LoadIndex(); loadErr == nil {
					asm := ctxpkg.NewAssembler(
						idx,
						filepath.Join(projectRoot, ".teraflow", "summaries"),
						projectRoot,
					)
					assembled, assembleErr := asm.Assemble(selectedSkill.Context, input, "")
					if assembleErr == nil && assembled != nil {
						additionalCtx = assembled.Context
					}
				}
			}

			provider, err := agent.NewProviderFromConfig(agent.ProviderConfig{
				Provider: resolvedProvider,
				Model:    resolvedModel,
			})
			if err != nil {
				return fmt.Errorf("create provider: %w", err)
			}
			mgr := agent.NewAgentManager(provider)

			agentCtx := agent.AgentContext{
				Type:              agent.AgentType(agentType),
				TrustLevel:        agent.TrustLevel(trustLevel),
				Input:             input,
				AdditionalContext: additionalCtx,
				SystemPrompt:      systemPrompt,
			}
			if agentCtx.TrustLevel == "" {
				agentCtx.TrustLevel = agent.TrustLevelSupervised
			}

			result, err := mgr.Run(cmd.Context(), agentCtx)
			if err != nil {
				return err
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Agent: %s\n", result.Type)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: success\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Tokens used: %d\n\n", result.TokensUsed)
			fmt.Fprintln(cmd.OutOrStdout(), result.Output)
			return nil
		},
	}

	cmd.Flags().StringVar(&agentType, "type", "", "Agent type: requirements|review|implement|ci-fix|conflict|incident|maintenance")
	cmd.Flags().StringVar(&trustLevel, "trust-level", "supervised", "Trust level: supervised|autonomous")
	cmd.Flags().StringVar(&inputFile, "input-file", "", "Path to input file")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key override for the resolved provider")
	cmd.Flags().StringVar(&skillName, "skill", "", "Skill name override (loads from skills/*.yml)")
	cmd.Flags().StringVar(&userFlag, "user", "", "GitHub username for RBAC check")
	return cmd
}

func newAgentStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show available agent types",
		RunE: func(cmd *cobra.Command, args []string) error {
			types := []string{
				"requirements - Organize and clarify requirements from discussions",
				"review       - Code review for PRs",
				"implement    - Implement features from requirements",
				"ci-fix       - Analyze CI failures and suggest fixes",
				"conflict     - Resolve merge conflicts",
				"incident     - Investigate and analyze incidents",
				"maintenance  - Assess maintenance health score",
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"available_types": []string{
						"requirements", "review", "implement",
						"ci-fix", "conflict", "incident", "maintenance",
					},
				})
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Available agent types:")
			for _, t := range types {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", t)
			}
			return nil
		},
	}
}
