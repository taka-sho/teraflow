package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/hooks"
)

var validHookEvents = map[hooks.HookEvent]struct{}{
	hooks.EventDiscussionCreated: {},
	hooks.EventDiscussionComment: {},
	hooks.EventConfirmation:      {},
	hooks.EventPush:              {},
	hooks.EventPROpened:          {},
}

var validHookActions = map[string]struct{}{
	"respond":        {},
	"summarize":      {},
	"index_update":   {},
	"summary_update": {},
	"generate":       {},
}

func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Manage event hooks",
	}
	cmd.AddCommand(newHookListCmd())
	cmd.AddCommand(newHookRunCmd())
	cmd.AddCommand(newHookValidateCmd())
	return cmd
}

func newHookListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List hooks defined in teraflow config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			parser := hooks.NewParser()
			hookCfg := parser.Parse(cfg)

			type hookListItem struct {
				Event      string   `json:"event"`
				Actions    int      `json:"actions"`
				Conditions []string `json:"conditions,omitempty"`
			}

			items := make([]hookListItem, 0, len(hookCfg))
			for event, actions := range hookCfg {
				item := hookListItem{
					Event:   string(event),
					Actions: len(actions),
				}
				conds := make([]string, 0, len(actions))
				for _, action := range actions {
					if action.Conditions == nil {
						continue
					}
					conds = append(conds, describeHookConditions(action.Conditions))
				}
				if len(conds) > 0 {
					item.Conditions = conds
				}
				items = append(items, item)
			}

			if format == "json" {
				return writeJSON(cmd, map[string]any{"hooks": items})
			}

			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No hooks defined")
				return nil
			}

			for _, item := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "- %s: actions=%d\n", item.Event, item.Actions)
				for _, c := range item.Conditions {
					fmt.Fprintf(cmd.OutOrStdout(), "  conditions: %s\n", c)
				}
			}
			return nil
		},
	}
}

func newHookRunCmd() *cobra.Command {
	var author string
	var category string
	var labels string
	var paths string
	var input string
	var discussionID string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "run <event>",
		Short: "Run matched hooks for an event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(author) == "" {
				return fmt.Errorf("--author is required")
			}

			cfgPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			event := hooks.HookEvent(args[0])
			if _, ok := validHookEvents[event]; !ok {
				return fmt.Errorf("unknown event: %s", args[0])
			}

			parser := hooks.NewParser()
			hookCfg := parser.Parse(cfg)
			ctx := hooks.HookContext{
				Event:        event,
				Author:       author,
				Category:     category,
				Labels:       splitCSV(labels),
				Paths:        splitCSV(paths),
				Input:        input,
				DiscussionID: discussionID,
			}

			matched := parser.MatchHooks(hookCfg, ctx)
			executor := hooks.NewExecutor(projectRootFromConfig(cfgPath), dryRun)
			result := executor.ExecuteAll(ctx, matched)

			if format == "json" {
				return writeJSON(cmd, result)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Event: %s\n", result.Event)
			fmt.Fprintf(cmd.OutOrStdout(), "Matched actions: %d\n", len(result.Matched))
			for _, r := range result.Results {
				fmt.Fprintf(cmd.OutOrStdout(), "- %s: success=%t message=%s\n", r.Action, r.Success, r.Message)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&author, "author", "", "Event author login (required)")
	cmd.Flags().StringVar(&category, "category", "", "Discussion category")
	cmd.Flags().StringVar(&labels, "labels", "", "Comma-separated labels")
	cmd.Flags().StringVar(&paths, "paths", "", "Comma-separated changed paths")
	cmd.Flags().StringVar(&input, "input", "", "Input text")
	cmd.Flags().StringVar(&discussionID, "discussion-id", "", "Discussion node_id")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show commands without executing")
	return cmd
}

func newHookValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate hooks events and actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			validationErrors := make([]string, 0)
			for eventName, actions := range cfg.Hooks {
				event := hooks.HookEvent(eventName)
				if _, ok := validHookEvents[event]; !ok {
					validationErrors = append(validationErrors, fmt.Sprintf("unknown event: %s", eventName))
				}
				for i, action := range actions {
					if _, ok := validHookActions[action.Action]; !ok {
						validationErrors = append(validationErrors, fmt.Sprintf("unknown action at %s[%d]: %s", eventName, i, action.Action))
					}
				}
			}

			if len(validationErrors) > 0 {
				if format == "json" {
					_ = writeJSON(cmd, map[string]any{"valid": false, "errors": validationErrors})
				}
				return fmt.Errorf("%s", strings.Join(validationErrors, "; "))
			}

			if format == "json" {
				return writeJSON(cmd, map[string]any{"valid": true})
			}
			fmt.Fprintln(cmd.OutOrStdout(), "hooks configuration is valid")
			return nil
		},
	}
}

func describeHookConditions(cond *hooks.HookConditions) string {
	parts := make([]string, 0, 4)
	if len(cond.NotAuthor) > 0 {
		parts = append(parts, "not_author="+strings.Join(cond.NotAuthor, ","))
	}
	if len(cond.Categories) > 0 {
		parts = append(parts, "categories="+strings.Join(cond.Categories, ","))
	}
	if len(cond.Labels) > 0 {
		parts = append(parts, "labels="+strings.Join(cond.Labels, ","))
	}
	if len(cond.Paths) > 0 {
		parts = append(parts, "paths="+strings.Join(cond.Paths, ","))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " ")
}

func splitCSV(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
