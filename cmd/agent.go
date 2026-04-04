package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/agent"
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

			if apiKey == "" {
				apiKey = os.Getenv("ANTHROPIC_API_KEY")
			}
			if apiKey == "" {
				return fmt.Errorf("ANTHROPIC_API_KEY not set")
			}

			provider := agent.NewAnthropicProvider(apiKey, "")
			mgr := agent.NewAgentManager(provider)

			agentCtx := agent.AgentContext{
				Type:       agent.AgentType(agentType),
				TrustLevel: agent.TrustLevel(trustLevel),
				Input:      input,
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
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Anthropic API key (or set ANTHROPIC_API_KEY env var)")
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
