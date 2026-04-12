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
	if hasAnyHook(hookCfg, hooks.EventPROpened) {
		return []string{"teraflow-hooks-pr"}
	}
	return nil
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
	fmt.Fprintf(&b, "        run: go install github.com/taka-sho/teraflow@%s\n", teraflowVersion)
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
