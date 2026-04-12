package actions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/taka-sho/teraflow/internal/hooks"
)

// HookWorkflowNames returns hook-derived workflow base names.
func HookWorkflowNames(hookCfg hooks.HookConfig) []string {
	names := make([]string, 0, 3)
	if hasAnyHook(hookCfg, hooks.EventDiscussionCreated, hooks.EventDiscussionComment, hooks.EventConfirmation) {
		names = append(names, "teraflow-hooks-discussion")
	}
	if hasAnyHook(hookCfg, hooks.EventPush) {
		names = append(names, "teraflow-hooks-push")
	}
	if hasAnyHook(hookCfg, hooks.EventPROpened) {
		names = append(names, "teraflow-hooks-pr")
	}
	return names
}

// GenerateHookWorkflows generates hook-specific GitHub Actions workflows.
func GenerateHookWorkflows(hookCfg hooks.HookConfig, outputDir, teraflowVersion string) error {
	version := strings.TrimSpace(teraflowVersion)
	if version == "" {
		version = "dev"
	}

	names := HookWorkflowNames(hookCfg)
	if len(names) == 0 {
		return nil
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create workflows directory: %w", err)
	}

	for _, name := range names {
		var content string
		switch name {
		case "teraflow-hooks-discussion":
			content = renderDiscussionWorkflow(hookCfg, version)
		case "teraflow-hooks-push":
			content = renderPushWorkflow(version)
		case "teraflow-hooks-pr":
			content = renderPRWorkflow(version)
		default:
			return fmt.Errorf("unknown hook workflow: %s", name)
		}

		dest := filepath.Join(outputDir, name+".yml")
		if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write workflow %s: %w", dest, err)
		}
	}

	return nil
}

func hasAnyHook(hookCfg hooks.HookConfig, events ...hooks.HookEvent) bool {
	for _, event := range events {
		if len(hookCfg[event]) > 0 {
			return true
		}
	}
	return false
}

func renderDiscussionWorkflow(hookCfg hooks.HookConfig, teraflowVersion string) string {
	hasDiscussionCreated := len(hookCfg[hooks.EventDiscussionCreated]) > 0
	hasDiscussionComment := len(hookCfg[hooks.EventDiscussionComment]) > 0
	hasConfirmation := len(hookCfg[hooks.EventConfirmation]) > 0

	var b strings.Builder
	b.WriteString("name: teraflow-hooks-discussion\n")
	b.WriteString("on:\n")
	if hasDiscussionCreated {
		b.WriteString("  discussion:\n")
		b.WriteString("    types: [created]\n")
	}
	if hasDiscussionComment || hasConfirmation {
		b.WriteString("  discussion_comment:\n")
		b.WriteString("    types: [created]\n")
	}
	b.WriteString("\n")
	b.WriteString("concurrency:\n")
	b.WriteString("  group: teraflow-hooks-discussion-${{ github.event.discussion.id || github.run_id }}\n")
	b.WriteString("  cancel-in-progress: false\n")
	b.WriteString("\n")
	b.WriteString("jobs:\n")
	b.WriteString("  hook-run:\n")
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    permissions:\n")
	b.WriteString("      contents: read\n")
	b.WriteString("      discussions: write\n")
	b.WriteString("    steps:\n")
	b.WriteString("      - uses: actions/checkout@v6\n")
	b.WriteString("      - uses: actions/setup-go@v6\n")
	b.WriteString("        with:\n")
	b.WriteString("          go-version-file: 'go.mod'\n")
	b.WriteString("          cache: false\n")
	b.WriteString("      - name: Install teraflow\n")
	b.WriteString(fmt.Sprintf("        run: go install github.com/taka-sho/teraflow@%s\n", teraflowVersion))
	if hasDiscussionCreated {
		appendRunHookStep(
			&b,
			"Run hook on_discussion_created",
			"github.event_name == 'discussion'",
			string(hooks.EventDiscussionCreated),
			"${{ github.event.sender.login }}",
			"${{ github.event.discussion.category.name }}",
			"${{ github.event.discussion.body }}",
			"${{ github.event.discussion.node_id }}",
		)
	}
	if hasDiscussionComment {
		appendRunHookStep(
			&b,
			"Run hook on_discussion_comment",
			"github.event_name == 'discussion_comment'",
			string(hooks.EventDiscussionComment),
			"${{ github.event.sender.login }}",
			"${{ github.event.discussion.category.name }}",
			"${{ github.event.comment.body || github.event.discussion.body }}",
			"${{ github.event.discussion.node_id }}",
		)
	}
	if hasConfirmation {
		appendRunHookStep(
			&b,
			"Run hook on_confirmation",
			"github.event_name == 'discussion_comment' && (contains(github.event.comment.body, '確定') || contains(github.event.comment.body, 'confirmed'))",
			string(hooks.EventConfirmation),
			"${{ github.event.sender.login }}",
			"${{ github.event.discussion.category.name }}",
			"${{ github.event.comment.body || github.event.discussion.body }}",
			"${{ github.event.discussion.node_id }}",
		)
	}
	return b.String()
}

func renderPushWorkflow(teraflowVersion string) string {
	var b strings.Builder
	b.WriteString("name: teraflow-hooks-push\n")
	b.WriteString("on:\n")
	b.WriteString("  push:\n")
	b.WriteString("    branches: ['**']\n")
	b.WriteString("\n")
	b.WriteString("concurrency:\n")
	b.WriteString("  group: teraflow-hooks-push-${{ github.ref }}\n")
	b.WriteString("  cancel-in-progress: false\n")
	b.WriteString("\n")
	b.WriteString("jobs:\n")
	b.WriteString("  hook-run:\n")
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    permissions:\n")
	b.WriteString("      contents: read\n")
	b.WriteString("    steps:\n")
	b.WriteString("      - uses: actions/checkout@v6\n")
	b.WriteString("      - uses: actions/setup-go@v6\n")
	b.WriteString("        with:\n")
	b.WriteString("          go-version-file: 'go.mod'\n")
	b.WriteString("          cache: false\n")
	b.WriteString("      - name: Install teraflow\n")
	b.WriteString(fmt.Sprintf("        run: go install github.com/taka-sho/teraflow@%s\n", teraflowVersion))
	appendRunHookStep(
		&b,
		"Run hook on_push",
		"github.event_name == 'push'",
		string(hooks.EventPush),
		"${{ github.actor }}",
		"",
		"${{ github.event.head_commit.message }}",
		"",
	)
	return b.String()
}

func renderPRWorkflow(teraflowVersion string) string {
	var b strings.Builder
	b.WriteString("name: teraflow-hooks-pr\n")
	b.WriteString("on:\n")
	b.WriteString("  pull_request:\n")
	b.WriteString("    types: [opened]\n")
	b.WriteString("\n")
	b.WriteString("concurrency:\n")
	b.WriteString("  group: teraflow-hooks-pr-${{ github.event.pull_request.number || github.run_id }}\n")
	b.WriteString("  cancel-in-progress: false\n")
	b.WriteString("\n")
	b.WriteString("jobs:\n")
	b.WriteString("  hook-run:\n")
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    permissions:\n")
	b.WriteString("      contents: read\n")
	b.WriteString("      pull-requests: read\n")
	b.WriteString("    steps:\n")
	b.WriteString("      - uses: actions/checkout@v6\n")
	b.WriteString("      - uses: actions/setup-go@v6\n")
	b.WriteString("        with:\n")
	b.WriteString("          go-version-file: 'go.mod'\n")
	b.WriteString("          cache: false\n")
	b.WriteString("      - name: Install teraflow\n")
	b.WriteString(fmt.Sprintf("        run: go install github.com/taka-sho/teraflow@%s\n", teraflowVersion))
	appendRunHookStep(
		&b,
		"Run hook on_pr_opened",
		"github.event_name == 'pull_request'",
		string(hooks.EventPROpened),
		"${{ github.event.pull_request.user.login }}",
		"",
		"${{ github.event.pull_request.body || github.event.pull_request.title }}",
		"",
	)
	return b.String()
}

func appendRunHookStep(
	b *strings.Builder,
	name string,
	ifExpr string,
	eventType string,
	authorExpr string,
	categoryExpr string,
	inputExpr string,
	discussionIDExpr string,
) {
	b.WriteString("      - name: ")
	b.WriteString(name)
	b.WriteString("\n")
	b.WriteString("        if: ")
	b.WriteString(ifExpr)
	b.WriteString("\n")
	b.WriteString("        env:\n")
	b.WriteString("          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n")
	b.WriteString("          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}\n")
	b.WriteString("          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}\n")
	b.WriteString("        run: |\n")
	b.WriteString("          teraflow hook run \"")
	b.WriteString(eventType)
	b.WriteString("\" \\\n")
	b.WriteString("            --author \"")
	b.WriteString(authorExpr)
	b.WriteString("\" \\\n")
	if categoryExpr != "" {
		b.WriteString("            --category \"")
		b.WriteString(categoryExpr)
		b.WriteString("\" \\\n")
	}
	b.WriteString("            --input \"")
	b.WriteString(inputExpr)
	b.WriteString("\" \\\n")
	if discussionIDExpr != "" {
		b.WriteString("            --discussion-id \"")
		b.WriteString(discussionIDExpr)
		b.WriteString("\" \\\n")
	}
	b.WriteString("            --config .github/teraflow.yml \\\n")
	b.WriteString("            --format json\n")
}
