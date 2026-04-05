package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/config"
	"github.com/taka-sho/teraflow/internal/rbac"
)

var rbacExecCommand = exec.Command
var rbacLookPath = exec.LookPath

type rbacRoleOutput struct {
	Name        string   `json:"name"`
	Members     []string `json:"members"`
	Permissions []string `json:"permissions"`
	AdminRole   bool     `json:"admin_role"`
}

type roleIssueRequest struct {
	Operation  string
	TargetUser string
	Role       string
	OldRole    string
}

func newRbacCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rbac",
		Short: "Manage RBAC roles and permissions",
	}

	cmd.AddCommand(newRbacListCmd())
	cmd.AddCommand(newRbacCheckCmd())
	cmd.AddCommand(newRbacApplyCmd())
	return cmd
}

func newRbacListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List RBAC roles and members",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			adminRole := cfg.RBAC.AdminRole
			if adminRole == "" {
				adminRole = "admin"
			}

			out := make([]rbacRoleOutput, 0, len(cfg.RBAC.Roles))
			for _, role := range cfg.RBAC.Roles {
				perms := make([]string, 0, len(role.Permissions))
				for _, p := range role.Permissions {
					perms = append(perms, string(p))
				}
				out = append(out, rbacRoleOutput{
					Name:        role.Name,
					Members:     role.Members,
					Permissions: perms,
					AdminRole:   role.Name == adminRole,
				})
			}

			if format == "json" {
				return writeJSON(cmd, map[string]any{"roles": out})
			}

			for _, role := range out {
				if role.AdminRole {
					fmt.Fprintf(cmd.OutOrStdout(), "%s ★ (admin_role)\n", role.Name)
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), role.Name)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  members: %s\n", printableCSV(role.Members))
				fmt.Fprintf(cmd.OutOrStdout(), "  permissions: %s\n", printableCSV(role.Permissions))
			}
			return nil
		},
	}
}

func newRbacCheckCmd() *cobra.Command {
	var user string
	var action string

	cmd := &cobra.Command{
		Use:   "check --action <action>",
		Short: "Check whether a user is allowed to perform an action",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			engine := rbac.NewEngine(cfg.RBAC)
			resolvedUser, err := engine.ResolveUser(user)
			if err != nil {
				return err
			}

			allowed := engine.CheckPermission(resolvedUser, action)
			roles := engine.GetUserRoles(resolvedUser)

			if format == "json" {
				if jsonErr := writeJSON(cmd, map[string]any{
					"user":    resolvedUser,
					"action":  action,
					"allowed": allowed,
					"roles":   roles,
				}); jsonErr != nil {
					return jsonErr
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "user: %s\n", resolvedUser)
				fmt.Fprintf(cmd.OutOrStdout(), "action: %s\n", action)
				fmt.Fprintf(cmd.OutOrStdout(), "allowed: %t\n", allowed)
				fmt.Fprintf(cmd.OutOrStdout(), "roles: %s\n", printableCSV(roles))
			}

			if !allowed {
				return fmt.Errorf("permission denied: user %q lacks %s", resolvedUser, action)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&user, "user", "", "User id (defaults to git user)")
	cmd.Flags().StringVar(&action, "action", "", "Action to verify (e.g. gate.approve.planning)")
	_ = cmd.MarkFlagRequired("action")
	return cmd
}

func newRbacApplyCmd() *cobra.Command {
	var fromIssue int
	var user string
	var createBranch bool

	cmd := &cobra.Command{
		Use:   "apply --from-issue <number>",
		Short: "Apply RBAC role change from a GitHub issue form",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			engine := rbac.NewEngine(cfg.RBAC)
			actor, err := engine.ResolveUser(user)
			if err != nil {
				return err
			}
			if cfg.RBAC.Enabled && !engine.CheckPermission(actor, "rbac.apply") {
				return fmt.Errorf("permission denied: user %q lacks rbac.apply", actor)
			}

			projectRoot := projectPathFromConfig(configPath)
			body, err := fetchIssueBody(projectRoot, fromIssue)
			if err != nil {
				return err
			}

			req, err := parseRoleIssueRequest(body)
			if err != nil {
				return err
			}
			if err := applyRoleIssueRequest(cfg, req); err != nil {
				return err
			}

			if err := config.Save(configPath, cfg); err != nil {
				return err
			}

			if createBranch {
				if err := createRBACPR(projectRoot, configPath, fromIssue, req); err != nil {
					return err
				}
			}

			if format == "json" {
				return writeJSON(cmd, map[string]any{
					"issue":       fromIssue,
					"operator":    actor,
					"operation":   req.Operation,
					"target_user": req.TargetUser,
					"role":        req.Role,
					"old_role":    req.OldRole,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Applied RBAC change from issue #%d\n", fromIssue)
			fmt.Fprintf(cmd.OutOrStdout(), "operator: %s\n", actor)
			fmt.Fprintf(cmd.OutOrStdout(), "operation: %s\n", req.Operation)
			fmt.Fprintf(cmd.OutOrStdout(), "target_user: %s\n", req.TargetUser)
			fmt.Fprintf(cmd.OutOrStdout(), "role: %s\n", req.Role)
			if req.OldRole != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "old_role: %s\n", req.OldRole)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&fromIssue, "from-issue", 0, "GitHub issue number")
	cmd.Flags().StringVar(&user, "user", "", "User id (defaults to git user)")
	cmd.Flags().BoolVar(&createBranch, "create-branch", false, "Create branch/commit/push/PR")
	_ = cmd.MarkFlagRequired("from-issue")
	return cmd
}

func fetchIssueBody(projectRoot string, issueNum int) (string, error) {
	if issueNum <= 0 {
		return "", errors.New("--from-issue must be a positive number")
	}
	if _, err := rbacLookPath("gh"); err != nil {
		return "", fmt.Errorf("E5001: GitHub CLI (gh) is not installed")
	}

	repo, err := readRemoteOriginRepo(projectRoot)
	if err != nil {
		return "", err
	}
	out, err := runExternalCommand(projectRoot, "gh", "api", fmt.Sprintf("repos/%s/issues/%d", repo, issueNum), "--jq", ".body")
	if err != nil {
		return "", fmt.Errorf("fetch issue body: %w", err)
	}
	if strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("issue #%d body is empty", issueNum)
	}
	return out, nil
}

func parseRoleIssueRequest(body string) (roleIssueRequest, error) {
	sections := map[string]string{}
	var currentKey string
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "### ") {
			currentKey = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "### ")))
			continue
		}
		if currentKey == "" || line == "" {
			continue
		}
		if strings.HasPrefix(line, "```") {
			continue
		}
		if sections[currentKey] == "" {
			sections[currentKey] = line
		}
	}

	req := roleIssueRequest{
		Operation:  normalizeOperation(sectionValue(sections, "operation", "操作")),
		TargetUser: strings.TrimSpace(sectionValue(sections, "target_user", "対象ユーザー")),
		Role:       strings.TrimSpace(sectionValue(sections, "role", "ロール")),
		OldRole:    normalizeOldRole(sectionValue(sections, "old_role", "変更元ロール（change時のみ）", "変更元ロール")),
	}
	if req.Operation == "" {
		return roleIssueRequest{}, errors.New("issue form parse error: operation is missing")
	}
	if req.TargetUser == "" {
		return roleIssueRequest{}, errors.New("issue form parse error: target_user is missing")
	}
	if req.Role == "" {
		return roleIssueRequest{}, errors.New("issue form parse error: role is missing")
	}
	if req.Operation == "change" && req.OldRole == "" {
		return roleIssueRequest{}, errors.New("issue form parse error: old_role is required for change")
	}
	return req, nil
}

func sectionValue(sections map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(sections[strings.ToLower(k)]); v != "" {
			return v
		}
	}
	return ""
}

func normalizeOperation(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.HasPrefix(v, "add"), strings.Contains(v, "追加"):
		return "add"
	case strings.HasPrefix(v, "change"), strings.Contains(v, "変更"):
		return "change"
	case strings.HasPrefix(v, "remove"), strings.Contains(v, "削除"):
		return "remove"
	default:
		return ""
	}
}

func normalizeOldRole(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	if strings.EqualFold(v, "n/a") || strings.EqualFold(v, "na") {
		return ""
	}
	return v
}

func applyRoleIssueRequest(cfg *config.TeraflowConfig, req roleIssueRequest) error {
	targetIdx := findRoleIndex(cfg.RBAC.Roles, req.Role)
	if targetIdx < 0 {
		return fmt.Errorf("role not found: %s", req.Role)
	}

	switch req.Operation {
	case "add":
		addMember(&cfg.RBAC.Roles[targetIdx].Members, req.TargetUser)
	case "change":
		oldIdx := findRoleIndex(cfg.RBAC.Roles, req.OldRole)
		if oldIdx < 0 {
			return fmt.Errorf("old role not found: %s", req.OldRole)
		}
		removeMember(&cfg.RBAC.Roles[oldIdx].Members, req.TargetUser)
		addMember(&cfg.RBAC.Roles[targetIdx].Members, req.TargetUser)
	case "remove":
		removeMember(&cfg.RBAC.Roles[targetIdx].Members, req.TargetUser)
	default:
		return fmt.Errorf("unsupported operation: %s", req.Operation)
	}

	adminRole := cfg.RBAC.AdminRole
	if adminRole == "" {
		adminRole = "admin"
	}
	adminIdx := findRoleIndex(cfg.RBAC.Roles, adminRole)
	if adminIdx >= 0 && len(cfg.RBAC.Roles[adminIdx].Members) == 0 {
		return fmt.Errorf("cannot remove last admin member from role %q", adminRole)
	}
	return nil
}

func addMember(members *[]string, user string) {
	for _, member := range *members {
		if member == user {
			return
		}
	}
	*members = append(*members, user)
}

func removeMember(members *[]string, user string) {
	filtered := make([]string, 0, len(*members))
	for _, member := range *members {
		if member != user {
			filtered = append(filtered, member)
		}
	}
	*members = filtered
}

func findRoleIndex(roles []rbac.Role, name string) int {
	for i, role := range roles {
		if role.Name == name {
			return i
		}
	}
	return -1
}

func createRBACPR(projectRoot, configPath string, issueNum int, req roleIssueRequest) error {
	branch := fmt.Sprintf("rbac/issue-%d", issueNum)
	if _, err := rbacLookPath("gh"); err != nil {
		return fmt.Errorf("E5001: GitHub CLI (gh) is not installed")
	}
	if _, err := runExternalCommand(projectRoot, "git", "checkout", "-b", branch); err != nil {
		return fmt.Errorf("create branch: %w", err)
	}
	rel, err := filepath.Rel(projectRoot, configPath)
	if err != nil {
		rel = configPath
	}
	if _, err := runExternalCommand(projectRoot, "git", "add", rel); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	if _, err := runExternalCommand(projectRoot, "git", "commit", "-m", fmt.Sprintf("chore(rbac): apply role change from issue #%d", issueNum)); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	if _, err := runExternalCommand(projectRoot, "git", "push", "-u", "origin", branch); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	title := fmt.Sprintf("chore(rbac): apply role change from issue #%d", issueNum)
	body := fmt.Sprintf("Apply RBAC change from issue #%d\n\n- operation: %s\n- target_user: %s\n- role: %s\n- old_role: %s\n", issueNum, req.Operation, req.TargetUser, req.Role, req.OldRole)
	if _, err := runExternalCommand(projectRoot, "gh", "pr", "create", "--title", title, "--body", body, "--base", "main"); err != nil {
		return fmt.Errorf("gh pr create: %w", err)
	}
	return nil
}

func readRemoteOriginRepo(projectRoot string) (string, error) {
	url, err := runExternalCommand(projectRoot, "git", "config", "--get", "remote.origin.url")
	if err != nil {
		return "", fmt.Errorf("detect repository: %w", err)
	}
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return "", errors.New("remote.origin.url is empty")
	}

	switch {
	case strings.HasPrefix(trimmed, "git@github.com:"):
		return strings.TrimSuffix(strings.TrimPrefix(trimmed, "git@github.com:"), ".git"), nil
	case strings.HasPrefix(trimmed, "https://github.com/"):
		return strings.TrimSuffix(strings.TrimPrefix(trimmed, "https://github.com/"), ".git"), nil
	default:
		return "", fmt.Errorf("unsupported remote URL format: %s", trimmed)
	}
}

func runExternalCommand(dir, name string, args ...string) (string, error) {
	c := rbacExecCommand(name, args...)
	c.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

func printableCSV(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}
