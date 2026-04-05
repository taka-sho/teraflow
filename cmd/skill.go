package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/skill"
)

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage AI skill definitions",
	}
	cmd.AddCommand(newSkillListCmd())
	cmd.AddCommand(newSkillShowCmd())
	cmd.AddCommand(newSkillValidateCmd())
	return cmd
}

func newSkillListCmd() *cobra.Command {
	var skillsDir string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			loader := skill.NewFileLoader(skillsDir)
			skills, err := loader.LoadAll()
			if err != nil {
				return err
			}
			sort.Slice(skills, func(i, j int) bool {
				return skills[i].Name < skills[j].Name
			})

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			if format == "json" {
				type skillListItem struct {
					Name        string   `json:"name"`
					Description string   `json:"description"`
					Labels      []string `json:"labels"`
				}

				out := make([]skillListItem, 0, len(skills))
				for _, s := range skills {
					out = append(out, skillListItem{
						Name:        s.Name,
						Description: s.Description,
						Labels:      s.Trigger.Labels,
					})
				}
				return writeJSON(cmd, out)
			}

			if len(skills) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No skills found.")
				return nil
			}

			for _, s := range skills {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"- %s\n  description: %s\n  labels: %s\n",
					s.Name,
					defaultIfEmpty(s.Description, "-"),
					joinOrDash(s.Trigger.Labels),
				)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&skillsDir, "skills-dir", "skills", "Skills directory")
	return cmd
}

func newSkillShowCmd() *cobra.Command {
	var skillsDir string

	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show skill details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			loader := skill.NewFileLoader(skillsDir)
			s, err := loader.LoadByName(args[0])
			if err != nil {
				return err
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, s)
			}

			printSkillDetails(cmd, s)
			return nil
		},
	}

	cmd.Flags().StringVar(&skillsDir, "skills-dir", "skills", "Skills directory")
	return cmd
}

func newSkillValidateCmd() *cobra.Command {
	var skillsDir string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate all skill files",
		RunE: func(cmd *cobra.Command, args []string) error {
			loader := skill.NewFileLoader(skillsDir)
			skills, err := loader.LoadAll()
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "All %d skills are valid.\n", len(skills))
			return nil
		},
	}

	cmd.Flags().StringVar(&skillsDir, "skills-dir", "skills", "Skills directory")
	return cmd
}

func printSkillDetails(cmd *cobra.Command, s *skill.Skill) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "name: %s\n", s.Name)
	fmt.Fprintf(out, "version: %s\n", s.Version)
	fmt.Fprintf(out, "description: %s\n", defaultIfEmpty(s.Description, "-"))
	fmt.Fprintf(out, "trigger.labels: %s\n", joinOrDash(s.Trigger.Labels))
	fmt.Fprintf(out, "trigger.categories: %s\n", joinOrDash(s.Trigger.Categories))
	fmt.Fprintf(out, "trigger.agent_types: %s\n", joinOrDash(s.Trigger.AgentTypes))
	fmt.Fprintf(out, "prompts.system: %s\n", defaultIfEmpty(s.Prompts.System, "-"))
	fmt.Fprintf(out, "prompts.confirm: %s\n", defaultIfEmpty(s.Prompts.Confirm, "-"))
	fmt.Fprintf(out, "context.include: %s\n", joinOrDash(s.Context.Include))
	fmt.Fprintf(out, "context.exclude: %s\n", joinOrDash(s.Context.Exclude))
	fmt.Fprintf(out, "context.max_context_tokens: %d\n", s.Context.MaxContextTokens)
	fmt.Fprintf(out, "output.dialogue.header: %s\n", defaultIfEmpty(s.Output.Dialogue.Header, "-"))
	fmt.Fprintf(out, "output.dialogue.footer: %s\n", defaultIfEmpty(s.Output.Dialogue.Footer, "-"))
	fmt.Fprintf(out, "output.confirm.header: %s\n", defaultIfEmpty(s.Output.Confirm.Header, "-"))
	fmt.Fprintf(out, "output.confirm.footer: %s\n", defaultIfEmpty(s.Output.Confirm.Footer, "-"))
	fmt.Fprintf(out, "options.max_tokens: %d\n", s.Options.MaxTokens)
	fmt.Fprintf(out, "options.temperature: %g\n", s.Options.Temperature)
}

func joinOrDash(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, ", ")
}

func defaultIfEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
