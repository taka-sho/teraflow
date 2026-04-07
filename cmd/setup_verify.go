package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var gitExecCommand = exec.Command

type setupVerifyResult struct {
	Repository setupVerifyRepository `json:"repository"`
	Checks     setupVerifyChecks     `json:"checks"`
	Issues     setupVerifyIssues     `json:"issues"`
}

type setupVerifyRepository struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

type setupVerifyChecks struct {
	DiscussionsEnabled             bool                    `json:"discussions_enabled"`
	DefaultWorkflowPermissions     string                  `json:"default_workflow_permissions"`
	CanApprovePullRequestReviews   bool                    `json:"can_approve_pull_request_reviews"`
	AllowActionsCreatePullRequests bool                    `json:"allow_actions_create_pull_requests"`
	Secrets                        setupVerifySecretsCheck `json:"secrets"`
}

type setupVerifySecretsCheck struct {
	AnthropicAPIKey bool `json:"anthropic_api_key"`
	OpenAIAPIKey    bool `json:"openai_api_key"`
	AnyProviderKey  bool `json:"any_provider_key"`
	Checked         bool `json:"checked"`
}

type setupVerifyIssues struct {
	Fatal        []string `json:"fatal"`
	Warnings     []string `json:"warnings"`
	FatalCount   int      `json:"fatal_count"`
	WarningCount int      `json:"warning_count"`
}

type workflowPermissionsResponse struct {
	DefaultWorkflowPermissions   string `json:"default_workflow_permissions"`
	CanApprovePullRequestReviews bool   `json:"can_approve_pull_request_reviews"`
}

type repositoryResponse struct {
	HasDiscussionsEnabled bool `json:"has_discussions_enabled"`
}

func newSetupVerifyCmd() *cobra.Command {
	var ownerFlag string
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify repository setup for teraflow agent workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, err := resolveSetupVerifyRepository(ownerFlag, repoFlag)
			if err != nil {
				return err
			}

			result, err := runSetupVerify(owner, repo)
			if err != nil {
				return err
			}

			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				if err := writeJSON(cmd, result); err != nil {
					return err
				}
			} else {
				printSetupVerifyText(cmd, result)
			}

			if result.Issues.FatalCount > 0 {
				return fmt.Errorf("setup verify failed: %d fatal issue(s) found", result.Issues.FatalCount)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&ownerFlag, "owner", "", "GitHub repository owner")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "GitHub repository name")
	return cmd
}

func resolveSetupVerifyRepository(ownerFlag, repoFlag string) (string, string, error) {
	if ownerFlag != "" || repoFlag != "" {
		if ownerFlag == "" || repoFlag == "" {
			return "", "", fmt.Errorf("both --owner and --repo are required when one is specified")
		}
		return ownerFlag, repoFlag, nil
	}

	gitCmd := gitExecCommand("git", "remote", "get-url", "origin")
	var stderr bytes.Buffer
	gitCmd.Stderr = &stderr
	out, err := gitCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("resolve repository from git remote failed: %v (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	owner, repo, parseErr := parseGitHubRepository(strings.TrimSpace(string(out)))
	if parseErr != nil {
		return "", "", fmt.Errorf("parse origin remote URL: %w", parseErr)
	}
	return owner, repo, nil
}

func runSetupVerify(owner, repo string) (*setupVerifyResult, error) {
	result := &setupVerifyResult{
		Repository: setupVerifyRepository{Owner: owner, Repo: repo},
	}

	workflowPerms, err := fetchWorkflowPermissions(owner, repo)
	if err != nil {
		return nil, err
	}
	result.Checks.DefaultWorkflowPermissions = workflowPerms.DefaultWorkflowPermissions
	result.Checks.CanApprovePullRequestReviews = workflowPerms.CanApprovePullRequestReviews
	result.Checks.AllowActionsCreatePullRequests = workflowPerms.CanApprovePullRequestReviews

	if workflowPerms.DefaultWorkflowPermissions != "write" {
		result.Issues.Fatal = append(result.Issues.Fatal,
			"Workflow permissions is not write")
	}
	if !workflowPerms.CanApprovePullRequestReviews {
		result.Issues.Fatal = append(result.Issues.Fatal,
			"Allow GitHub Actions to create and approve pull requests is disabled")
	}

	repoInfo, err := fetchRepositoryInfo(owner, repo)
	if err != nil {
		return nil, err
	}
	result.Checks.DiscussionsEnabled = repoInfo.HasDiscussionsEnabled
	if !repoInfo.HasDiscussionsEnabled {
		result.Issues.Warnings = append(result.Issues.Warnings,
			"GitHub Discussions is disabled")
	}

	secrets := checkRepoSecrets(owner, repo)
	result.Checks.Secrets = secrets
	if !secrets.Checked {
		result.Issues.Warnings = append(result.Issues.Warnings,
			"Could not verify repository secrets (gh secret list failed)")
	} else if !secrets.AnyProviderKey {
		result.Issues.Warnings = append(result.Issues.Warnings,
			"Neither ANTHROPIC_API_KEY nor OPENAI_API_KEY is set")
	}

	result.Issues.FatalCount = len(result.Issues.Fatal)
	result.Issues.WarningCount = len(result.Issues.Warnings)
	return result, nil
}

func fetchWorkflowPermissions(owner, repo string) (*workflowPermissionsResponse, error) {
	var response workflowPermissionsResponse
	if err := ghAPIJSON(&response, "repos/%s/%s/actions/permissions/workflow", owner, repo); err != nil {
		return nil, fmt.Errorf("read workflow permissions: %w", err)
	}
	return &response, nil
}

func fetchRepositoryInfo(owner, repo string) (*repositoryResponse, error) {
	var response repositoryResponse
	if err := ghAPIJSON(&response, "repos/%s/%s", owner, repo); err != nil {
		return nil, fmt.Errorf("read repository settings: %w", err)
	}
	return &response, nil
}

func ghAPIJSON(dst any, pathFmt string, args ...any) error {
	path := fmt.Sprintf(pathFmt, args...)
	ghCmd := ghExecCommand("gh", "api", path)
	var stderr bytes.Buffer
	ghCmd.Stderr = &stderr
	out, err := ghCmd.Output()
	if err != nil {
		return fmt.Errorf("gh api %s failed: %v (stderr: %s)", path, err, strings.TrimSpace(stderr.String()))
	}
	if err := json.Unmarshal(out, dst); err != nil {
		return fmt.Errorf("parse gh api %s response: %w", path, err)
	}
	return nil
}

func checkRepoSecrets(owner, repo string) setupVerifySecretsCheck {
	ghCmd := ghExecCommand("gh", "secret", "list", "--repo", owner+"/"+repo)
	var stderr bytes.Buffer
	ghCmd.Stderr = &stderr
	out, err := ghCmd.Output()
	if err != nil {
		return setupVerifySecretsCheck{Checked: false}
	}

	upper := strings.ToUpper(string(out))
	anthropic := strings.Contains(upper, "ANTHROPIC_API_KEY")
	openai := strings.Contains(upper, "OPENAI_API_KEY")
	return setupVerifySecretsCheck{
		AnthropicAPIKey: anthropic,
		OpenAIAPIKey:    openai,
		AnyProviderKey:  anthropic || openai,
		Checked:         true,
	}
}

func printSetupVerifyText(cmd *cobra.Command, result *setupVerifyResult) {
	out := cmd.OutOrStdout()
	owner := result.Repository.Owner
	repo := result.Repository.Repo

	fmt.Fprintf(out, "Repository: %s/%s\n\n", owner, repo)

	if result.Checks.DiscussionsEnabled {
		fmt.Fprintln(out, "✅ Discussions enabled")
	} else {
		fmt.Fprintln(out, "⚠️  Discussions disabled")
		fmt.Fprintf(out, "   -> fix: enable Discussions at https://github.com/%s/%s/settings\n", owner, repo)
	}

	if result.Checks.DefaultWorkflowPermissions == "write" {
		fmt.Fprintln(out, "✅ Workflow permissions: write")
	} else {
		fmt.Fprintf(out, "❌ Workflow permissions: %s\n", result.Checks.DefaultWorkflowPermissions)
		fmt.Fprintf(out, "   -> fix: gh api -X PUT repos/%s/%s/actions/permissions/workflow \\\n", owner, repo)
		fmt.Fprintln(out, "            -F can_approve_pull_request_reviews=true \\")
		fmt.Fprintln(out, "            -F default_workflow_permissions=write")
	}

	if result.Checks.CanApprovePullRequestReviews {
		fmt.Fprintln(out, "✅ Allow Actions to create PRs: true")
	} else {
		fmt.Fprintln(out, "❌ Allow Actions to create PRs: false")
		fmt.Fprintf(out, "   -> fix: gh api -X PUT repos/%s/%s/actions/permissions/workflow \\\n", owner, repo)
		fmt.Fprintln(out, "            -F can_approve_pull_request_reviews=true \\")
		fmt.Fprintln(out, "            -F default_workflow_permissions=write")
	}

	if !result.Checks.Secrets.Checked {
		fmt.Fprintln(out, "⚠️  ANTHROPIC_API_KEY / OPENAI_API_KEY: could not verify (gh secret list failed)")
	} else {
		if result.Checks.Secrets.AnthropicAPIKey {
			fmt.Fprintln(out, "✅ ANTHROPIC_API_KEY: set")
		} else {
			fmt.Fprintln(out, "⚠️  ANTHROPIC_API_KEY: not detected (may be set but unreadable)")
		}
		if result.Checks.Secrets.OpenAIAPIKey {
			fmt.Fprintln(out, "✅ OPENAI_API_KEY: set")
		} else {
			fmt.Fprintln(out, "⚠️  OPENAI_API_KEY: not detected (may be set but unreadable)")
		}
	}

	if result.Issues.FatalCount == 1 {
		fmt.Fprintln(out, "\n1 issue found.")
	} else {
		fmt.Fprintf(out, "\n%d issues found.\n", result.Issues.FatalCount)
	}
	if result.Issues.WarningCount > 0 {
		fmt.Fprintf(out, "%d warning(s) found.\n", result.Issues.WarningCount)
	}
}
