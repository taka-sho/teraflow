package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	discopkg "github.com/taka-sho/teraflow/internal/discovery"
	"gopkg.in/yaml.v3"
)

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage requirement template",
	}
	cmd.AddCommand(newTemplateInitCmd())
	cmd.AddCommand(newTemplateShowCmd())
	cmd.AddCommand(newTemplateListCmd())
	cmd.AddCommand(newTemplateValidateCmd())
	cmd.AddCommand(newTemplateDiffCmd())
	cmd.AddCommand(newTemplateAddCmd())
	cmd.AddCommand(newTemplateRemoveCmd())
	cmd.AddCommand(newTemplateSyncCmd())
	return cmd
}

func templatePaths(cmd *cobra.Command) (repoRoot, templatePath string, err error) {
	configPath, err := configPathFromCmd(cmd)
	if err != nil {
		return "", "", err
	}
	repoRoot = filepath.Dir(filepath.Dir(configPath))
	templatePath = filepath.Join(repoRoot, ".teraflow", "discovery", "requirement-template.yaml")
	return repoRoot, templatePath, nil
}

func loadTemplateFromPath(templatePath string) (discopkg.RequirementTemplate, error) {
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return discopkg.DefaultRequirementTemplate(), nil
	}
	return discopkg.LoadRequirementTemplate(templatePath)
}

func saveTemplate(templatePath string, tmpl discopkg.RequirementTemplate) error {
	if err := os.MkdirAll(filepath.Dir(templatePath), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(tmpl)
	if err != nil {
		return err
	}
	return os.WriteFile(templatePath, data, 0o644)
}

func newTemplateInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write default template to .teraflow/discovery/requirement-template.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, templatePath, err := templatePaths(cmd)
			if err != nil {
				return err
			}
			if _, err := os.Stat(templatePath); err == nil && !force {
				return fmt.Errorf("template already exists at %s (use --force to overwrite)", templatePath)
			}
			tmpl := discopkg.DefaultRequirementTemplate()
			if err := saveTemplate(templatePath, tmpl); err != nil {
				return fmt.Errorf("write template: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Template written to %s\n", templatePath)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing template")
	return cmd
}

func newTemplateShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current template as YAML",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, templatePath, err := templatePaths(cmd)
			if err != nil {
				return err
			}
			tmpl, err := loadTemplateFromPath(templatePath)
			if err != nil {
				return err
			}
			data, err := yaml.Marshal(tmpl)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
}

func newTemplateListCmd() *cobra.Command {
	var category string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List template fields as a table",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, templatePath, err := templatePaths(cmd)
			if err != nil {
				return err
			}
			tmpl, err := loadTemplateFromPath(templatePath)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tORIGIN")
			for _, item := range tmpl.Items {
				if category != "" && item.Category != category {
					continue
				}
				origin := item.Origin
				if origin == "" {
					origin = "builtin"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", item.ID, item.Name, item.Category, origin)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&category, "category", "", "Filter by category (required|recommended|optional)")
	return cmd
}

func newTemplateValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate template schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, templatePath, err := templatePaths(cmd)
			if err != nil {
				return err
			}
			tmpl, err := loadTemplateFromPath(templatePath)
			if err != nil {
				return err
			}
			if err := discopkg.ValidateTemplate(tmpl); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Validation failed: %v\n", err)
				return fmt.Errorf("validation failed: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Template is valid.")
			return nil
		},
	}
}

func newTemplateDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff",
		Short: "Show differences from the default template",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, templatePath, err := templatePaths(cmd)
			if err != nil {
				return err
			}
			tmpl, err := loadTemplateFromPath(templatePath)
			if err != nil {
				return err
			}
			diffs := discopkg.DiffFromDefault(tmpl)
			if len(diffs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No changes from default.")
				return nil
			}
			for _, d := range diffs {
				switch d.Action {
				case "add":
					fmt.Fprintf(cmd.OutOrStdout(), "+ %s (%s) [%s] %s\n",
						d.FieldID, d.Field.Name, d.Field.Category, d.Field.Origin)
				case "remove":
					fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", d.FieldID)
				case "modify":
					fmt.Fprintf(cmd.OutOrStdout(), "~ %s (%s) [%s]\n",
						d.FieldID, d.Field.Name, d.Field.Category)
				}
			}
			return nil
		},
	}
}

func newTemplateAddCmd() *cobra.Command {
	var label, category, hint string
	cmd := &cobra.Command{
		Use:   "add <field_id>",
		Short: "Add a new field to the template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fieldID := args[0]
			if label == "" {
				return fmt.Errorf("--label is required")
			}
			if category == "" {
				return fmt.Errorf("--category is required")
			}

			repoRoot, templatePath, err := templatePaths(cmd)
			if err != nil {
				return err
			}
			tmpl, err := loadTemplateFromPath(templatePath)
			if err != nil {
				return err
			}

			field := discopkg.TemplateItem{
				ID:         fieldID,
				Name:       label,
				Category:   category,
				PromptHint: hint,
			}
			if err := discopkg.AddField(&tmpl, field); err != nil {
				return err
			}
			if err := saveTemplate(templatePath, tmpl); err != nil {
				return fmt.Errorf("save template: %w", err)
			}

			entry := discopkg.HistoryEntry{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Action:    "add",
				FieldID:   fieldID,
				Details: map[string]string{
					"name":     label,
					"category": category,
				},
				Source: "cli",
			}
			if err := discopkg.AppendLocalHistory(repoRoot, []discopkg.HistoryEntry{entry}); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: history write failed: %v\n", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Field %q added.\n", fieldID)
			return nil
		},
	}
	cmd.Flags().StringVar(&label, "label", "", "Field label/name (required)")
	cmd.Flags().StringVar(&category, "category", "", "Category: required|recommended|optional (required)")
	cmd.Flags().StringVar(&hint, "hint", "", "Prompt hint")
	return cmd
}

func newTemplateRemoveCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <field_id>",
		Short: "Remove a field from the template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fieldID := args[0]

			repoRoot, templatePath, err := templatePaths(cmd)
			if err != nil {
				return err
			}
			tmpl, err := loadTemplateFromPath(templatePath)
			if err != nil {
				return err
			}

			// find field to check category
			var found *discopkg.TemplateItem
			for i := range tmpl.Items {
				if tmpl.Items[i].ID == fieldID {
					cp := tmpl.Items[i]
					found = &cp
					break
				}
			}
			if found == nil {
				return fmt.Errorf("field %q not found", fieldID)
			}

			if found.Category == discopkg.TemplateCategoryRequired && !force {
				fmt.Fprintf(cmd.OutOrStdout(), "Field %q is required. Remove? [y/N] ", fieldID)
				reader := bufio.NewReader(cmd.InOrStdin())
				answer, _ := reader.ReadString('\n')
				if !strings.EqualFold(strings.TrimSpace(answer), "y") {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			if _, err := discopkg.RemoveField(&tmpl, fieldID); err != nil {
				return err
			}
			if err := saveTemplate(templatePath, tmpl); err != nil {
				return fmt.Errorf("save template: %w", err)
			}

			entry := discopkg.HistoryEntry{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Action:    "remove",
				FieldID:   fieldID,
				Source:    "cli",
			}
			if err := discopkg.AppendLocalHistory(repoRoot, []discopkg.HistoryEntry{entry}); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: history write failed: %v\n", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Field %q removed.\n", fieldID)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation for required fields")
	return cmd
}

func newTemplateSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync local history to global history",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, _, err := templatePaths(cmd)
			if err != nil {
				return err
			}

			local, err := discopkg.ReadLocalHistory(repoRoot)
			if err != nil {
				return fmt.Errorf("read local history: %w", err)
			}
			beforeCount := len(local.Entries)

			if err := discopkg.SyncToGlobal(repoRoot); err != nil {
				return fmt.Errorf("sync: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Synced %d entries to global history.\n", beforeCount)
			return nil
		},
	}
}
