package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type discussionListOptions struct {
	Format string
}

var discussionSummarizer = summarizeDiscussionWithAnthropic

func newDiscussionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discussion",
		Short: "Manage GitHub Discussions",
	}

	cmd.AddCommand(newDiscussionListCmd())
	cmd.AddCommand(newDiscussionSummarizeCmd())
	return cmd
}

func newDiscussionListCmd() *cobra.Command {
	opts := &discussionListOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List discussions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := ghLookPath("gh"); err != nil {
				return fmt.Errorf("E5001: GitHub CLI (gh) is not installed.")
			}

			ghArgs := []string{"discussion", "list"}
			if opts.Format != "" {
				switch opts.Format {
				case "text":
				case "json":
					ghArgs = append(ghArgs, "--json", "number,title,state")
				default:
					return fmt.Errorf("unsupported format: %s", opts.Format)
				}
			}

			discussionCmd := ghExecCommand("gh", ghArgs...)
			discussionCmd.Stdin = os.Stdin
			discussionCmd.Stdout = os.Stdout
			discussionCmd.Stderr = os.Stderr
			if err := discussionCmd.Run(); err != nil {
				return fmt.Errorf("E5003: GitHub API error: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Format, "format", "", "Output format: text|json")
	return cmd
}

func newDiscussionSummarizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summarize <number>",
		Short: "Summarize a discussion",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := ghLookPath("gh"); err != nil {
				return fmt.Errorf("E5001: GitHub CLI (gh) is not installed.")
			}

			viewCmd := ghExecCommand("gh", "discussion", "view", args[0])
			body, err := viewCmd.Output()
			if err != nil {
				return fmt.Errorf("E5003: GitHub API error: %w", err)
			}

			content := strings.TrimSpace(string(body))
			if os.Getenv("ANTHROPIC_API_KEY") == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "AI summarization requires ANTHROPIC_API_KEY. Skipping.")
				fmt.Fprintln(cmd.OutOrStdout(), content)
				return nil
			}

			summary, err := discussionSummarizer(content, os.Getenv("ANTHROPIC_API_KEY"))
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), summary)
			return nil
		},
	}

	return cmd
}

func summarizeDiscussionWithAnthropic(content, apiKey string) (string, error) {
	payload := map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 300,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": "以下のGitHub Discussionを200字以内で要約してください：\n" + content,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("E6002: AI API error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("E6002: AI API error: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("E6002: AI API error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("E6002: AI API error: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("E6002: AI API error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("E6002: AI API error: %w", err)
	}
	if len(parsed.Content) == 0 || strings.TrimSpace(parsed.Content[0].Text) == "" {
		return "", fmt.Errorf("E6002: AI API error: empty response")
	}

	return strings.TrimSpace(parsed.Content[0].Text), nil
}
