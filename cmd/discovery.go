package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	discopkg "github.com/taka-sho/teraflow/internal/discovery"
)

var discoveryArgPattern = regexp.MustCompile(`^(?:discussion-)?(\d+)$`)

func newDiscoveryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discovery",
		Short: "Inspect grill-me requirements discovery sessions",
	}
	cmd.AddCommand(newDiscoveryStatusCmd())
	cmd.AddCommand(newDiscoveryTreeCmd())
	return cmd
}

func newDiscoveryStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <discussion-N>",
		Short: "Show discovery session progress",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			discussionNum, err := parseDiscussionArg(args[0])
			if err != nil {
				return err
			}
			state, err := loadDiscoverySession(cmd, discussionNum)
			if err != nil {
				return err
			}
			state.RecalculateSummary()

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"discussion_number": discussionNum,
					"title":             state.Title,
					"summary":           state.Summary,
					"complete":          state.IsComplete(),
					"completion_report": state.CompletionReport(),
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Discussion: #%d\n", discussionNum)
			if strings.TrimSpace(state.Title) != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Title: %s\n", state.Title)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Progress: %d/%d (%d%%)\n", state.Summary.Answered, state.Summary.Total, state.Summary.ProgressPercent)
			fmt.Fprintf(cmd.OutOrStdout(), "Pending: %d\n", state.Summary.Pending)
			fmt.Fprintf(cmd.OutOrStdout(), "Skipped: %d\n", state.Summary.Skipped)
			fmt.Fprintf(cmd.OutOrStdout(), "Blocked: %d\n", state.Summary.Blocked)
			if state.IsComplete() {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", state.CompletionReport())
			}
			return nil
		},
	}
}

func newDiscoveryTreeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tree <discussion-N>",
		Short: "Show discovery decision tree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			discussionNum, err := parseDiscussionArg(args[0])
			if err != nil {
				return err
			}
			state, err := loadDiscoverySession(cmd, discussionNum)
			if err != nil {
				return err
			}
			state.RecalculateSummary()

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"discussion_number": discussionNum,
					"title":             state.Title,
					"tree":              state.Tree,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Discovery tree: discussion-%d\n", discussionNum)
			printDiscoveryTree(cmd, state.Tree, "")
			return nil
		},
	}
}

func printDiscoveryTree(cmd *cobra.Command, nodes []discopkg.Branch, indent string) {
	for _, node := range nodes {
		line := fmt.Sprintf("%s- [%s] %s", indent, normalizeDiscoveryStatus(node.Status), node.ID)
		if strings.TrimSpace(node.Question) != "" {
			line += ": " + strings.TrimSpace(node.Question)
		}
		if len(node.DependsOn) > 0 {
			deps := append([]string(nil), node.DependsOn...)
			sort.Strings(deps)
			line += fmt.Sprintf(" (depends_on=%s)", strings.Join(deps, ","))
		}
		fmt.Fprintln(cmd.OutOrStdout(), line)
		if len(node.Children) > 0 {
			printDiscoveryTree(cmd, node.Children, indent+"  ")
		}
	}
}

func parseDiscussionArg(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	m := discoveryArgPattern.FindStringSubmatch(trimmed)
	if len(m) != 2 {
		return 0, fmt.Errorf("invalid discussion id: %s", raw)
	}
	var num int
	_, err := fmt.Sscanf(m[1], "%d", &num)
	if err != nil || num <= 0 {
		return 0, fmt.Errorf("invalid discussion id: %s", raw)
	}
	return num, nil
}

func loadDiscoverySession(cmd *cobra.Command, discussionNum int) (*discopkg.SessionState, error) {
	projectRoot, err := projectRootFromCmd(cmd)
	if err != nil {
		return nil, err
	}
	statePath := filepath.Join(projectRoot, ".teraflow", "discovery", fmt.Sprintf("discussion-%d.yaml", discussionNum))
	state, err := discopkg.LoadSessionState(statePath)
	if err != nil {
		return nil, fmt.Errorf("read discovery session: %w", err)
	}
	return state, nil
}

func normalizeDiscoveryStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case discopkg.StatusAnswered, discopkg.StatusResolved:
		return discopkg.StatusAnswered
	case discopkg.StatusSkipped:
		return discopkg.StatusSkipped
	case discopkg.StatusBlocked:
		return discopkg.StatusBlocked
	default:
		return discopkg.StatusPending
	}
}
